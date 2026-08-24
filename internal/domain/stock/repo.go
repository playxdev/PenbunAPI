// Path: internal/domain/stock/repo.go
package stock

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"penbun/api/internal/repository"
)

type Repo struct{ db *repository.DB }

func NewRepo(db *repository.DB) *Repo { return &Repo{db: db} }

// MovementTypes ตรงกับ CK_tb_stock_movement_type ของ v7
var MovementTypes = []string{
	"RECEIVE", "ISSUE", "RETURN_IN", "RETURN_OUT",
	"TRANSFER_IN", "TRANSFER_OUT", "ADJUST",
}

// ApplyMovement คือทางเดียวที่โค้ดฝั่ง Go แตะสต็อกได้
//
// ห้าม UPDATE tb_product_stock ตรง ๆ ที่ใดในระบบเด็ดขาด เพราะ Ledger
// (tb_stock_movement) คือความจริง ส่วน tb_product_stock เป็นแค่ cache
// การเขียน cache โดยไม่ผ่าน Ledger จะทำให้สองตารางไม่ตรงกันถาวรและ
// USP_REBUILD_STOCK_CACHE จะลบผลลัพธ์นั้นทิ้งในการรันครั้งถัดไป
type Movement struct {
	SkuAuto       int
	WarehouseAuto int
	MovementType  string
	QtyChange     float64
	DocTable      *string
	DocAuto       *int
	DocNo         *string
	CustomerAuto  *int
	VendorAuto    *int
	UnitCost      *float64
	Remark        *string
}

func ApplyMovement(ctx context.Context, ex repository.Executor, m Movement, updateBy string) error {
	const q = `
EXEC dbo.USP_APPLY_STOCK_MOVEMENT
     @SkuAuto       = @p1,
     @WarehouseAuto = @p2,
     @MovementType  = @p3,
     @QtyChange     = @p4,
     @DocTable      = @p5,
     @DocAuto       = @p6,
     @DocNo         = @p7,
     @CustomerAuto  = @p8,
     @VendorAuto    = @p9,
     @UnitCost      = @p10,
     @UpdateBy      = @p11`

	_, err := ex.ExecContext(ctx, q,
		m.SkuAuto, m.WarehouseAuto, m.MovementType, m.QtyChange,
		nullString(m.DocTable), nullInt(m.DocAuto), nullString(m.DocNo),
		nullInt(m.CustomerAuto), nullInt(m.VendorAuto), nullFloat(m.UnitCost),
		updateBy)
	return err
}

// OnHandFilter สะท้อน query parameter ของ GET /stock/onhand
type OnHandFilter struct {
	WarehouseCode string
	SkuID         string
	ProductID     string
	Search        string
	OnlyPositive  bool
	Below         *float64
	Page, Limit   int
}

func (r *Repo) OnHand(ctx context.Context, f OnHandFilter) ([]repository.Row, int64, error) {
	var w strings.Builder
	var args []any
	add := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("@p%d", len(args))
	}

	w.WriteString(" WHERE 1 = 1")
	if f.WarehouseCode != "" {
		fmt.Fprintf(&w, " AND warehouse_code = %s", add(f.WarehouseCode))
	}
	if f.SkuID != "" {
		fmt.Fprintf(&w, " AND sku_id = %s", add(f.SkuID))
	}
	if f.ProductID != "" {
		fmt.Fprintf(&w, " AND product_id = %s", add(f.ProductID))
	}
	if f.Search != "" {
		ph := add(f.Search)
		fmt.Fprintf(&w, " AND (product_name LIKE N'%%' + %s + N'%%' OR sku_code LIKE N'%%' + %s + N'%%')", ph, ph)
	}
	if f.OnlyPositive {
		w.WriteString(" AND qty_onhand <> 0")
	}
	if f.Below != nil {
		fmt.Fprintf(&w, " AND qty_available < %s", add(*f.Below))
	}

	countArgs := append([]any{}, args...)
	var total int64
	countSQL := "SELECT COUNT_BIG(1) FROM dbo.vw_stock_onhand" + w.String()
	if err := r.db.Exec().QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := add((f.Page - 1) * f.Limit)
	limit := add(f.Limit)
	listSQL := "SELECT * FROM dbo.vw_stock_onhand" + w.String() +
		" ORDER BY warehouse_code ASC, sku_code ASC, stock_auto ASC" +
		" OFFSET " + offset + " ROWS FETCH NEXT " + limit + " ROWS ONLY"

	rows, err := r.db.Exec().QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := repository.ScanRows(rows)
	return items, total, err
}

type MovementFilter struct {
	SkuID         string
	WarehouseCode string
	CustomerID    string
	VendorID      string
	MovementType  string
	DocNo         string
	DateFrom      *string
	DateTo        *string
	Page, Limit   int
}

func (r *Repo) Movements(ctx context.Context, f MovementFilter) ([]repository.Row, int64, error) {
	var w strings.Builder
	var args []any
	add := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("@p%d", len(args))
	}

	w.WriteString(" WHERE 1 = 1")
	if f.SkuID != "" {
		fmt.Fprintf(&w, " AND sku_id = %s", add(f.SkuID))
	}
	if f.WarehouseCode != "" {
		fmt.Fprintf(&w, " AND warehouse_code = %s", add(f.WarehouseCode))
	}
	if f.CustomerID != "" {
		fmt.Fprintf(&w, " AND customer_id = %s", add(f.CustomerID))
	}
	if f.VendorID != "" {
		fmt.Fprintf(&w, " AND vendor_id = %s", add(f.VendorID))
	}
	if f.MovementType != "" {
		fmt.Fprintf(&w, " AND movement_type = %s", add(f.MovementType))
	}
	if f.DocNo != "" {
		fmt.Fprintf(&w, " AND doc_no = %s", add(f.DocNo))
	}
	if f.DateFrom != nil {
		fmt.Fprintf(&w, " AND movement_date >= %s", add(*f.DateFrom))
	}
	if f.DateTo != nil {
		fmt.Fprintf(&w, " AND movement_date < DATEADD(DAY, 1, %s)", add(*f.DateTo))
	}

	countArgs := append([]any{}, args...)
	var total int64
	countSQL := "SELECT COUNT_BIG(1) FROM dbo.vw_stock_movement" + w.String()
	if err := r.db.Exec().QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := add((f.Page - 1) * f.Limit)
	limit := add(f.Limit)
	listSQL := "SELECT * FROM dbo.vw_stock_movement" + w.String() +
		" ORDER BY movement_date DESC, stock_movement_auto DESC" +
		" OFFSET " + offset + " ROWS FETCH NEXT " + limit + " ROWS ONLY"

	rows, err := r.db.Exec().QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := repository.ScanRows(rows)
	return items, total, err
}

func (r *Repo) RebuildStockCache(ctx context.Context, updateBy string) error {
	_, err := r.db.Exec().ExecContext(ctx, "EXEC dbo.USP_REBUILD_STOCK_CACHE @UpdateBy = @p1", updateBy)
	return err
}

func (r *Repo) RebuildConsignBalance(ctx context.Context, updateBy string) error {
	_, err := r.db.Exec().ExecContext(ctx, "EXEC dbo.USP_REBUILD_CONSIGN_BALANCE @UpdateBy = @p1", updateBy)
	return err
}

func nullString(p *string) any {
	if p == nil {
		return sql.NullString{}
	}
	return *p
}

func nullInt(p *int) any {
	if p == nil {
		return sql.NullInt32{}
	}
	return *p
}

func nullFloat(p *float64) any {
	if p == nil {
		return sql.NullFloat64{}
	}
	return *p
}
