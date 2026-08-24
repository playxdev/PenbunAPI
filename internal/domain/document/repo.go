// Path: internal/domain/document/repo.go
package document

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

// batchSize จำกัดจำนวนแถวต่อคำสั่ง INSERT
//
// SQL Server รับพารามิเตอร์ได้สูงสุด 2,100 ตัวต่อคำสั่ง รายการเอกสารมีได้ถึง
// ~12 คอลัมน์ต่อแถว จึงตั้งไว้ที่ 150 เพื่อกันชนเพดานแม้ในกรณีที่คอลัมน์เยอะที่สุด
//
// ทางที่เร็วกว่านี้คือ Table-Valued Parameter ซึ่งจะเหลือ round trip เดียว
// แต่ต้องมี user-defined table type ในฐานข้อมูลก่อน
const batchSize = 150

type Repo struct{ db *repository.DB }

func NewRepo(db *repository.DB) *Repo { return &Repo{db: db} }

// Meta คือข้อมูลขั้นต่ำที่ต้องรู้ก่อนเปลี่ยนสถานะหรือโพสต์เอกสาร
type Meta struct {
	AutoID     int
	BusinessID string
	DocStatus  string
	ItemCount  int
}

// Item คือรายการหนึ่งบรรทัด เก็บทั้งค่าที่ผ่านการแปลงแล้วและ autoID ที่ resolve ได้
type Item struct {
	LineNo int
	Refs   map[string]int // ref_sku_auto ฯลฯ
	Values map[string]any
}

// InsertHeader คืน autoID ของเอกสารที่เพิ่งสร้าง
//
// OUTPUT ได้เฉพาะ autoID เท่านั้น
// Trigger ที่สร้าง Business ID เป็นชนิด AFTER INSERT ซึ่ง UPDATE แถวตามหลัง
// ค่าที่ OUTPUT คืนจึงเป็นค่า ณ ตอน INSERT ซึ่งยังเป็น NULL
func (r *Repo) InsertHeader(
	ctx context.Context, tx *sql.Tx, s *Spec,
	vals map[string]any, refs map[string]int, updateBy string,
) (int, error) {
	a := &schema.Args{}
	cols := make([]string, 0, len(vals)+len(refs)+3)
	phs := make([]string, 0, len(vals)+len(refs)+3)

	for _, n := range schema.SortedKeys(vals) {
		cols = append(cols, n)
		phs = append(phs, a.Add(vals[n]))
	}
	for _, n := range schema.SortedKeys(refs) {
		cols = append(cols, n)
		phs = append(phs, a.Add(refs[n]))
	}

	// doc_status ถูกกำหนดโดย engine เสมอ ไม่รับจาก client
	// การให้ client ตั้งสถานะเองเท่ากับข้ามด่านทั้งหมดของวงจรเอกสาร
	cols = append(cols, "doc_status", "update_by")
	phs = append(phs, a.Add(StatusDraft), a.Add(updateBy))

	q := fmt.Sprintf("INSERT INTO dbo.%s (%s) OUTPUT INSERTED.autoID VALUES (%s)",
		s.HeaderTable, strings.Join(cols, ", "), strings.Join(phs, ", "))

	var autoID int
	err := tx.QueryRowContext(ctx, q, a.Values()...).Scan(&autoID)
	return autoID, err
}

func (r *Repo) UpdateHeader(
	ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int,
	vals map[string]any, refs map[string]int, updateBy string,
) (int64, error) {
	a := &schema.Args{}
	sets := make([]string, 0, len(vals)+len(refs)+1)

	for _, n := range schema.SortedKeys(vals) {
		sets = append(sets, n+" = "+a.Add(vals[n]))
	}
	for _, n := range schema.SortedKeys(refs) {
		sets = append(sets, n+" = "+a.Add(refs[n]))
	}
	sets = append(sets, "update_by = "+a.Add(updateBy))

	q := fmt.Sprintf("UPDATE dbo.%s SET %s WHERE autoID = %s AND is_delete = 0",
		s.HeaderTable, strings.Join(sets, ", "), a.Add(headerAuto))

	res, err := tx.ExecContext(ctx, q, a.Values()...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// InsertItems เขียนรายการทั้งหมดเป็นชุด
func (r *Repo) InsertItems(
	ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int, items []Item, updateBy string,
) error {
	if len(items) == 0 {
		return nil
	}
	// ทุกแถวต้องมีคอลัมน์ชุดเดียวกัน ไม่งั้นประกอบ multi-row VALUES ไม่ได้
	// จึงรวม key ของทุกแถวเป็นชุดเดียวแล้วเติม NULL ให้แถวที่ขาด
	valueCols := unionKeys(items, func(it Item) map[string]any { return it.Values })
	refCols := unionKeysInt(items, func(it Item) map[string]int { return it.Refs })

	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		if err := r.insertItemBatch(ctx, tx, s, headerAuto,
			items[start:end], valueCols, refCols, updateBy); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) insertItemBatch(
	ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int,
	items []Item, valueCols, refCols []string, updateBy string,
) error {
	cols := make([]string, 0, len(valueCols)+len(refCols)+3)
	cols = append(cols, s.ItemFKColumn, "line_no")
	cols = append(cols, refCols...)
	cols = append(cols, valueCols...)
	cols = append(cols, "update_by")

	a := &schema.Args{}
	var sb strings.Builder
	fmt.Fprintf(&sb, "INSERT INTO dbo.%s (%s) VALUES ", s.ItemTable, strings.Join(cols, ", "))

	for i, it := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		phs := make([]string, 0, len(cols))
		phs = append(phs, a.Add(headerAuto), a.Add(it.LineNo))
		for _, c := range refCols {
			if v, ok := it.Refs[c]; ok {
				phs = append(phs, a.Add(v))
			} else {
				phs = append(phs, a.Add(nil))
			}
		}
		for _, c := range valueCols {
			phs = append(phs, a.Add(it.Values[c]))
		}
		phs = append(phs, a.Add(updateBy))
		fmt.Fprintf(&sb, "(%s)", strings.Join(phs, ", "))
	}

	_, err := tx.ExecContext(ctx, sb.String(), a.Values()...)
	return err
}

// DeleteItems soft delete รายการทั้งหมดของเอกสาร ใช้ตอนแทนที่รายการทั้งใบ
func (r *Repo) DeleteItems(ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int, updateBy string) error {
	q := fmt.Sprintf(
		"UPDATE dbo.%s SET is_delete = 1, update_by = @p2 WHERE %s = @p1 AND is_delete = 0",
		s.ItemTable, s.ItemFKColumn)
	_, err := tx.ExecContext(ctx, q, headerAuto, updateBy)
	return err
}

// RecalcTotals คำนวณยอดรวมของ header ใหม่จากรายการจริง
//
// ยอดรวมไม่รับจาก client เพราะถ้ายอดใน header ไม่ตรงกับผลรวมของรายการ
// เอกสาร ใบกำกับ และรายงานทั้งหมดจะเพี้ยนตามกันไปโดยไม่มีอะไรฟ้อง
func (r *Repo) RecalcTotals(ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int, updateBy string) error {
	_, err := tx.ExecContext(ctx, s.TotalsSQL, headerAuto, updateBy)
	return err
}

func (r *Repo) MetaByID(ctx context.Context, ex repository.Executor, s *Spec, businessID string) (*Meta, error) {
	q := fmt.Sprintf(`
SELECT TOP 1 h.autoID, h.%s, h.doc_status,
       (SELECT COUNT(1) FROM dbo.%s i WHERE i.%s = h.autoID AND i.is_delete = 0)
  FROM dbo.%s h
 WHERE h.%s = @p1 AND h.is_delete = 0`,
		s.IDColumn, s.ItemTable, s.ItemFKColumn, s.HeaderTable, s.IDColumn)

	var m Meta
	err := ex.QueryRowContext(ctx, q, businessID).
		Scan(&m.AutoID, &m.BusinessID, &m.DocStatus, &m.ItemCount)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repo) BusinessIDByAuto(ctx context.Context, ex repository.Executor, s *Spec, autoID int) (string, error) {
	q := fmt.Sprintf("SELECT %s FROM dbo.%s WHERE autoID = @p1", s.IDColumn, s.HeaderTable)
	var id string
	err := ex.QueryRowContext(ctx, q, autoID).Scan(&id)
	return id, err
}

// SetStatus เปลี่ยนสถานะแบบมีเงื่อนไข
//
// เงื่อนไข doc_status = @from ใน WHERE คือสิ่งที่ทำให้การเปลี่ยนสถานะปลอดภัย
// เมื่อมีคนสองคนกดพร้อมกัน คนที่สองจะได้ RowsAffected = 0 แทนที่จะเขียนทับ
func (r *Repo) SetStatus(
	ctx context.Context, ex repository.Executor, s *Spec,
	headerAuto int, from, to, updateBy string,
) (int64, error) {
	q := fmt.Sprintf(`
UPDATE dbo.%s SET doc_status = @p3, update_by = @p4
 WHERE autoID = @p1 AND doc_status = @p2 AND is_delete = 0`, s.HeaderTable)

	res, err := ex.ExecContext(ctx, q, headerAuto, from, to, updateBy)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Repo) SoftDeleteHeader(ctx context.Context, ex repository.Executor, s *Spec, headerAuto int, updateBy string) (int64, error) {
	q := fmt.Sprintf(
		"UPDATE dbo.%s SET is_delete = 1, update_by = @p2 WHERE autoID = @p1 AND is_delete = 0",
		s.HeaderTable)
	res, err := ex.ExecContext(ctx, q, headerAuto, updateBy)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// LockPairs คืนคู่ (sku, warehouse) ที่ต้องจองล็อกก่อนโพสต์ เรียงลำดับแน่นอน
type LockPair struct{ SkuAuto, WarehouseAuto int }

func (r *Repo) LockPairs(ctx context.Context, tx *sql.Tx, s *Spec, headerAuto int) ([]LockPair, error) {
	rows, err := tx.QueryContext(ctx, s.LockPairsSQL, headerAuto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LockPair
	for rows.Next() {
		var p LockPair
		if err := rows.Scan(&p.SkuAuto, &p.WarehouseAuto); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Post เรียก Stored Procedure ที่ทำให้เอกสารมีผลต่อสต็อกจริง
//
// proc เป็นผู้ตัดสต็อก ปรับยอดฝากขาย บันทึกประวัติการจัดส่ง และตั้งสถานะปลายทาง
// ทั้งหมดในคราวเดียว API ไม่คำนวณอะไรเองเลยแม้แต่ขั้นตอนเดียว
func (r *Repo) Post(ctx context.Context, tx *sql.Tx, s *Spec, businessID, updateBy string) error {
	q := fmt.Sprintf("EXEC dbo.%s %s = @p1, @UpdateBy = @p2", s.PostProc, s.PostParamName)
	_, err := tx.ExecContext(ctx, q, businessID, updateBy)
	return err
}

func (r *Repo) HeaderView(ctx context.Context, ex repository.Executor, s *Spec, businessID string) (repository.Row, error) {
	q := fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE %s = @p1", s.HeaderSource, s.IDColumn)
	return repository.QueryOne(ctx, ex, q, []any{businessID})
}

func (r *Repo) ItemsView(ctx context.Context, ex repository.Executor, s *Spec, businessID string) ([]repository.Row, error) {
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s = @p1 ORDER BY line_no ASC", s.ItemSource, s.IDColumn)
	return repository.QueryMany(ctx, ex, q, []any{businessID})
}

func (r *Repo) List(ctx context.Context, ex repository.Executor, s *Spec, p ListParams) ([]repository.Row, int64, error) {
	a := &schema.Args{}
	var where strings.Builder
	where.WriteString(" WHERE 1 = 1")

	if p.DocStatus != "" {
		fmt.Fprintf(&where, " AND doc_status = %s", a.Add(p.DocStatus))
	}
	if p.DocNo != "" {
		fmt.Fprintf(&where, " AND doc_no LIKE N'%%' + %s + N'%%'", a.Add(p.DocNo))
	}
	if p.DateFrom != "" {
		fmt.Fprintf(&where, " AND doc_date >= %s", a.Add(p.DateFrom))
	}
	if p.DateTo != "" {
		fmt.Fprintf(&where, " AND doc_date < DATEADD(DAY, 1, %s)", a.Add(p.DateTo))
	}
	for _, name := range schema.SortedKeys(p.Filters) {
		for _, f := range s.Filters {
			if f.Param == name {
				fmt.Fprintf(&where, " AND %s = %s", f.Column, a.Add(p.Filters[name]))
				break
			}
		}
	}

	var total int64
	countSQL := "SELECT COUNT_BIG(1) FROM " + s.HeaderSource + where.String()
	if err := ex.QueryRowContext(ctx, countSQL, a.Snapshot()...).Scan(&total); err != nil {
		return nil, 0, err
	}

	off := a.Add((p.Page - 1) * p.Limit)
	lim := a.Add(p.Limit)
	listSQL := fmt.Sprintf(
		"SELECT * FROM %s%s ORDER BY doc_date DESC, %s DESC OFFSET %s ROWS FETCH NEXT %s ROWS ONLY",
		s.HeaderSource, where.String(), s.AutoColumn, off, lim)

	items, err := repository.QueryMany(ctx, ex, listSQL, a.Values())
	return items, total, err
}

func unionKeys(items []Item, get func(Item) map[string]any) []string {
	set := map[string]struct{}{}
	for _, it := range items {
		for k := range get(it) {
			set[k] = struct{}{}
		}
	}
	return schema.SortedKeys(set)
}

func unionKeysInt(items []Item, get func(Item) map[string]int) []string {
	set := map[string]struct{}{}
	for _, it := range items {
		for k := range get(it) {
			set[k] = struct{}{}
		}
	}
	return schema.SortedKeys(set)
}
