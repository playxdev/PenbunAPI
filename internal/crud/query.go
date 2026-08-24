// Path: internal/crud/query.go
package crud

import (
	"fmt"
	"strings"

	"penbun/api/internal/schema"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

// ListParams คือ query string ที่ผ่านการตรวจและ clamp แล้ว
type ListParams struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	SortAsc  bool
	IsActive *bool // nil = ไม่กรอง
	Filters  map[string]any
}

// buildList คืน query ดึงรายการ และ query นับจำนวนทั้งหมด
// ทั้งสองใช้ WHERE ชุดเดียวกัน ต่างกันแค่ ORDER BY / OFFSET
func (r *Resource) buildList(p ListParams) (listSQL, countSQL string, listArgs, countArgs []any) {
	var where strings.Builder
	a := &schema.Args{}

	where.WriteString(" WHERE 1 = 1")

	if p.Search != "" && len(r.SearchColumns) > 0 {
		ph := a.Add(p.Search)
		parts := make([]string, 0, len(r.SearchColumns))
		for _, col := range r.SearchColumns {
			parts = append(parts, fmt.Sprintf("%s LIKE N'%%' + %s + N'%%'", col, ph))
		}
		fmt.Fprintf(&where, " AND (%s)", strings.Join(parts, " OR "))
	}

	if p.IsActive != nil {
		fmt.Fprintf(&where, " AND is_active = %s", a.Add(*p.IsActive))
	}

	for _, name := range schema.SortedKeys(p.Filters) {
		for _, f := range r.Filters {
			if f.Param == name {
				fmt.Fprintf(&where, " AND %s = %s", f.Column, a.Add(p.Filters[name]))
				break
			}
		}
	}

	countSQL = "SELECT COUNT_BIG(1) FROM " + r.Source + where.String()
	countArgs = a.Snapshot()

	dir := "DESC"
	if p.SortAsc {
		dir = "ASC"
	}
	sortCol := r.SortColumns[r.DefaultSort]
	if col, ok := r.SortColumns[p.Sort]; ok {
		sortCol = col
	}

	offPH := a.Add((p.Page - 1) * p.Limit)
	limPH := a.Add(p.Limit)

	// tiebreaker บังคับ
	//
	// update_date ซ้ำกันได้ เพราะแถวที่ insert ในทรานแซกชันเดียวกันได้เวลาเดียวกัน
	// ถ้าไม่มีตัวตัดสินลำดับ ผลของ OFFSET จะไม่นิ่ง แถวเดิมจะโผล่สองหน้าและบางแถว
	// จะหายไปเลยโดยไม่มีอะไรฟ้อง
	listSQL = fmt.Sprintf(
		"SELECT * FROM %s%s ORDER BY %s %s, %s DESC OFFSET %s ROWS FETCH NEXT %s ROWS ONLY",
		r.Source, where.String(), sortCol, dir, r.AutoColumn, offPH, limPH)

	return listSQL, countSQL, a.Values(), countArgs
}

func (r *Resource) buildGetByID(businessID string) (string, []any) {
	a := &schema.Args{}
	ph := a.Add(businessID)
	return fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE %s = %s", r.Source, r.IDColumn, ph), a.Values()
}

func (r *Resource) buildGetByAuto(autoID int) (string, []any) {
	a := &schema.Args{}
	ph := a.Add(autoID)
	return fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE %s = %s", r.Source, r.AutoColumn, ph), a.Values()
}

// buildInsert สร้าง INSERT พร้อม OUTPUT INSERTED.autoID
//
// ห้าม OUTPUT คอลัมน์ Business ID เด็ดขาด
//
// Trigger ที่สร้าง Business ID เป็นชนิด AFTER INSERT ซึ่ง UPDATE แถวตามหลัง
// ค่าที่ OUTPUT คืนคือค่า ณ ตอน INSERT ซึ่งยังเป็น NULL อยู่
// ต้องเอา autoID ไป SELECT จาก Source ต่อในทรานแซกชันเดียวกันเสมอ
func (r *Resource) buildInsert(vals map[string]any, refs map[string]int, updateBy string) (string, []any) {
	a := &schema.Args{}
	cols := make([]string, 0, len(vals)+len(refs)+1)
	phs := make([]string, 0, len(vals)+len(refs)+1)

	for _, n := range schema.SortedKeys(vals) {
		cols = append(cols, n)
		phs = append(phs, a.Add(vals[n]))
	}
	for _, n := range schema.SortedKeys(refs) {
		cols = append(cols, n)
		phs = append(phs, a.Add(refs[n]))
	}
	cols = append(cols, "update_by")
	phs = append(phs, a.Add(updateBy))

	// ไม่ส่ง prefix / <table>_id / update_date / is_active / is_delete / id_status
	// ทั้งหมดมี DEFAULT constraint หรือ Trigger ดูแลอยู่แล้วครบทุกตาราง
	q := fmt.Sprintf("INSERT INTO dbo.%s (%s) OUTPUT INSERTED.autoID VALUES (%s)",
		r.Table, strings.Join(cols, ", "), strings.Join(phs, ", "))
	return q, a.Values()
}

// buildUpdate ตั้งค่าเฉพาะคอลัมน์ที่ client ส่งมาจริงในรอบนี้
//
// ไม่ใช้ COALESCE กับทุกคอลัมน์ เพราะแบบนั้นจะตั้งค่าเป็น NULL หรือสตริงว่าง
// ไม่ได้เลย และเขียนทับทุกคอลัมน์ทุกครั้งที่ UPDATE
func (r *Resource) buildUpdate(id string, vals map[string]any, refs map[string]int, updateBy string) (string, []any) {
	a := &schema.Args{}
	sets := make([]string, 0, len(vals)+len(refs)+1)

	for _, n := range schema.SortedKeys(vals) {
		sets = append(sets, n+" = "+a.Add(vals[n]))
	}
	for _, n := range schema.SortedKeys(refs) {
		sets = append(sets, n+" = "+a.Add(refs[n]))
	}
	sets = append(sets, "update_by = "+a.Add(updateBy))

	q := fmt.Sprintf("UPDATE dbo.%s SET %s WHERE %s = %s AND is_delete = 0",
		r.Table, strings.Join(sets, ", "), r.IDColumn, a.Add(id))
	return q, a.Values()
}

// buildSoftDelete ตั้งเฉพาะ is_delete
// Trigger จะบังคับ is_active = 0 และ id_status = 'DELETED' ให้เองตามมา
// API จึงไม่ต้องตั้งซ้ำ และไม่ควรตั้ง เพราะถ้ากฎฝั่งฐานข้อมูลเปลี่ยน โค้ดจะไม่ค้าง
func (r *Resource) buildSoftDelete(id, updateBy string) (string, []any) {
	a := &schema.Args{}
	byPH := a.Add(updateBy)
	idPH := a.Add(id)
	q := fmt.Sprintf(
		"UPDATE dbo.%s SET is_delete = 1, update_by = %s WHERE %s = %s AND is_delete = 0",
		r.Table, byPH, r.IDColumn, idPH)
	return q, a.Values()
}
