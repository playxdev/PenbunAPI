// Path: internal/repository/resolver.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrRefNotFound คืนเมื่อ Business ID ที่อ้างถึงไม่มีในระบบ
// ชั้นบนแปลงเป็น httpx.RefNotFound เพื่อบอกได้ว่าฟิลด์ไหนผิด
var ErrRefNotFound = errors.New("reference not found")

// Resolver แปลง Business ID (CUSA000041) เป็น autoID (41)
//
// v7 ผูก FK ด้วย ref_*_auto INT ทั้ง 53 จุด แต่โลกภายนอกรู้จักแต่ Business ID
// ทางเลือกอื่นคือ resolve ด้วย subquery ใน SQL ซึ่งเร็วกว่าแต่ ID ผิดจะกลายเป็น
// FK violation 547 ที่บอกไม่ได้ว่าฟิลด์ไหนผิด — เราเลือกความชัดเจนของ error
type Resolver struct {
	db  *DB
	ttl time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	autoID  int
	expires time.Time
}

// cacheable — เฉพาะตาราง Layer 0-1 ที่ข้อมูลน้อยและแทบไม่เปลี่ยน
// ตาราง master/product ไม่ cache เพราะแถวเยอะและแก้บ่อย การ cache จะกิน
// หน่วยความจำมากโดยได้ hit rate ต่ำ
var cacheable = map[string]bool{
	"tb_company":             true,
	"tb_customer_type":       true,
	"tb_vendor_type":         true,
	"tb_discount_type":       true,
	"tb_product_category":    true,
	"tb_product_format_type": true,
	"tb_unit_type":           true,
	"tb_book_type":           true,
	"tb_warehouse":           true,
	"tb_route":               true,
	"tb_discount_group":      true,
}

// idColumn บอกว่าตารางไหนใช้คอลัมน์อะไรเป็น Business ID
// ทำเป็น allow-list เพราะชื่อตาราง/คอลัมน์ถูกต่อเข้า SQL string โดยตรง
// ค่าที่ไม่อยู่ใน map นี้จะถูกปฏิเสธ ไม่มีทางที่ค่าจาก request จะหลุดเข้า SQL
var idColumn = map[string]string{
	"tb_company":             "company_id",
	"tb_customer_type":       "customer_type_id",
	"tb_vendor_type":         "vendor_type_id",
	"tb_discount_type":       "discount_type_id",
	"tb_discount_group":      "discount_group_id",
	"tb_product_category":    "product_category_id",
	"tb_product_format_type": "product_format_type_id",
	"tb_unit_type":           "unit_type_id",
	"tb_book_type":           "book_type_id",
	"tb_product_group":       "product_group_id",
	"tb_warehouse":           "warehouse_id",
	"tb_vendor":              "vendor_id",
	"tb_customer":            "customer_id",
	"tb_discount":            "discount_id",
	"tb_route":               "route_id",
	"tb_customer_route":      "customer_route_id",
	"tb_product":             "product_id",
	"tb_product_sku":         "sku_id",
	"tb_book":                "book_id",
	"tb_order":               "order_id",
	"tb_receive_note":        "receive_note_id",
	"tb_return_note":         "return_note_id",
	"tb_vendor_return_note":  "vendor_return_note_id",
	"tb_users":               "user_id",
}

func NewResolver(db *DB, ttl time.Duration) *Resolver {
	return &Resolver{db: db, ttl: ttl, cache: make(map[string]cacheEntry)}
}

// Resolve คืน autoID ของ businessID ในตารางที่ระบุ
func (r *Resolver) Resolve(ctx context.Context, table, businessID string) (int, error) {
	return r.resolveWith(ctx, r.db.Exec(), table, businessID)
}

// ResolveTx เหมือน Resolve แต่ทำงานในทรานแซกชันที่กำลังเปิดอยู่
// ใช้เมื่อ reference ชี้ไปแถวที่เพิ่งถูกสร้างในทรานแซกชันเดียวกัน
func (r *Resolver) ResolveTx(ctx context.Context, tx *sql.Tx, table, businessID string) (int, error) {
	return r.resolveWith(ctx, tx, table, businessID)
}

func (r *Resolver) resolveWith(ctx context.Context, ex Executor, table, businessID string) (int, error) {
	col, ok := idColumn[table]
	if !ok {
		return 0, fmt.Errorf("resolver: unknown table %q", table)
	}
	if businessID == "" {
		return 0, ErrRefNotFound
	}

	key := table + "\x00" + businessID
	useCache := cacheable[table]

	if useCache {
		if id, ok := r.get(key); ok {
			return id, nil
		}
	}

	// ชื่อตารางและคอลัมน์มาจาก allow-list ด้านบนเท่านั้น ค่าจากผู้ใช้เป็น parameter
	q := fmt.Sprintf(
		"SELECT TOP 1 autoID FROM dbo.%s WHERE %s = @p1 AND is_delete = 0", table, col)

	var autoID int
	if err := ex.QueryRowContext(ctx, q, businessID).Scan(&autoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrRefNotFound
		}
		return 0, err
	}

	if useCache {
		r.put(key, autoID)
	}
	return autoID, nil
}

// Invalidate ล้าง cache ของทั้งตาราง เรียกหลังเขียนตารางที่ cacheable
func (r *Resolver) Invalidate(table string) {
	if !cacheable[table] {
		return
	}
	prefix := table + "\x00"
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.cache {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(r.cache, k)
		}
	}
}

func (r *Resolver) get(key string) (int, bool) {
	r.mu.RLock()
	e, ok := r.cache[key]
	r.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return 0, false
	}
	return e.autoID, true
}

func (r *Resolver) put(key string, autoID int) {
	r.mu.Lock()
	r.cache[key] = cacheEntry{autoID: autoID, expires: time.Now().Add(r.ttl)}
	r.mu.Unlock()
}
