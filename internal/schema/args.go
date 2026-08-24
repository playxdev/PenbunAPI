// Path: internal/schema/args.go
package schema

import (
	"sort"
	"strconv"
)

// Args สะสมพารามิเตอร์และคืน placeholder แบบ @pN ตามลำดับที่เพิ่ม
//
// ค่าจาก request ต้องผ่านตรงนี้เสมอ ห้ามต่อเข้า SQL string โดยตรงไม่ว่ากรณีใด
// ชื่อตารางและคอลัมน์เท่านั้นที่ต่อเข้า string ได้ และต้องมาจาก descriptor
type Args struct{ vals []any }

func (a *Args) Add(v any) string {
	a.vals = append(a.vals, v)
	return "@p" + strconv.Itoa(len(a.vals))
}

func (a *Args) Values() []any { return a.vals }
func (a *Args) Len() int      { return len(a.vals) }

// Snapshot คัดลอกค่าที่สะสมไว้ ณ ตอนนี้
// ใช้ตอนต้องแยก query นับจำนวนออกจาก query ดึงรายการ ซึ่งใช้ WHERE ชุดเดียวกัน
// แต่ query นับไม่มี OFFSET / FETCH
func (a *Args) Snapshot() []any {
	out := make([]any, len(a.vals))
	copy(out, a.vals)
	return out
}

// SortedKeys คืน key ของ map เรียงตามตัวอักษร
//
// ใช้เพื่อให้ SQL ที่ประกอบขึ้นจากชุด input เดียวกันมีหน้าตาเหมือนเดิมทุกครั้ง
// SQL Server จะได้ใช้ execution plan ที่ cache ไว้ซ้ำแทนที่จะ compile ใหม่
func SortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
