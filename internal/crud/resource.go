// Path: internal/crud/resource.go
package crud

import (
	"fmt"

	"penbun/api/internal/schema"
)

// Resource คือคำอธิบายทั้งหมดของ master resource หนึ่งตัว
//
// เพิ่ม resource ใหม่ = เขียน descriptor ตัวหนึ่ง ไม่ต้องเขียน handler, SQL,
// หรือ test ของ CRUD เลย engine สร้าง 5 endpoint ให้จาก descriptor นี้
type Resource struct {
	Name  string // path segment เช่น "customer"
	Label string // ชื่อไทยสำหรับข้อความ เช่น "ลูกค้า"

	// Source คือสิ่งที่ตามหลัง FROM ในทุก SELECT
	//
	// ปกติคือชื่อ View ของ PenbunSQL เช่น "dbo.vw_customer" ซึ่งแปลง autoID
	// เป็น Business ID ให้แล้ว ตารางที่ยังไม่มี View รองรับให้ใส่เป็น derived
	// table แล้วคอมเมนต์ TEMP กำกับ จะได้ไล่เปลี่ยนเป็นชื่อ View ทีเดียวเมื่อ
	// ฝั่งฐานข้อมูลเพิ่มให้ครบ
	Source string

	Table      string // ตารางจริงสำหรับ INSERT / UPDATE / DELETE
	IDColumn   string // Business ID เช่น "customer_id"
	AutoColumn string // ชื่อคอลัมน์ autoID ที่ Source คืนออกมา เช่น "customer_auto"

	SearchColumns []string          // ใช้กับ ?q=
	SortColumns   map[string]string // ชื่อที่ client ส่ง -> ชื่อคอลัมน์จริง
	DefaultSort   string            // ต้องเป็น key ที่มีใน SortColumns

	Filters []schema.Filter
	Refs    []schema.Ref
	Fields  []schema.Field

	ReadOnly bool // true = mount เฉพาะ GET

	// RequireLevel จำกัดทุก endpoint ของ resource นี้ไว้ที่ user_level ที่ระบุ
	// ว่างไว้ = ผู้ใช้ที่ login แล้วทุกคนเข้าได้ ซึ่งเป็นค่าปกติของ master data
	// resource ที่คืนข้อมูลของคนอื่น เช่น ผู้ใช้งาน ต้องระบุเสมอ
	RequireLevel []string
}

func (r *Resource) field(name string) (schema.Field, bool) {
	for _, f := range r.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return schema.Field{}, false
}

func (r *Resource) ref(name string) (schema.Ref, bool) {
	for _, rf := range r.Refs {
		if rf.Field == name {
			return rf, true
		}
	}
	return schema.Ref{}, false
}

// Validate ตรวจ descriptor ตอน process start ไม่ใช่ตอนมี request เข้ามา
// descriptor ที่ผิดควรทำให้ start ไม่ขึ้น ดีกว่าปล่อยให้พังตอนมีคนใช้งานจริง
func (r *Resource) Validate() error {
	switch {
	case r.Name == "":
		return fmt.Errorf("resource: Name is required")
	case r.Source == "":
		return fmt.Errorf("resource %s: Source is required", r.Name)
	case r.Table == "":
		return fmt.Errorf("resource %s: Table is required", r.Name)
	case r.IDColumn == "":
		return fmt.Errorf("resource %s: IDColumn is required", r.Name)
	case r.AutoColumn == "":
		return fmt.Errorf("resource %s: AutoColumn is required", r.Name)
	case len(r.SortColumns) == 0:
		return fmt.Errorf("resource %s: SortColumns is required", r.Name)
	}
	if _, ok := r.SortColumns[r.DefaultSort]; !ok {
		return fmt.Errorf("resource %s: DefaultSort %q is not in SortColumns", r.Name, r.DefaultSort)
	}

	seen := map[string]bool{}
	for _, f := range r.Fields {
		if seen[f.Name] {
			return fmt.Errorf("resource %s: duplicate field %q", r.Name, f.Name)
		}
		seen[f.Name] = true
	}
	for _, rf := range r.Refs {
		if seen[rf.Field] {
			return fmt.Errorf("resource %s: ref %q collides with a field", r.Name, rf.Field)
		}
		if rf.Table == "" || rf.Column == "" {
			return fmt.Errorf("resource %s: ref %q needs Table and Column", r.Name, rf.Field)
		}
		seen[rf.Field] = true
	}
	if !r.ReadOnly {
		for _, f := range r.Filters {
			if f.Column == "" {
				return fmt.Errorf("resource %s: filter %q needs Column", r.Name, f.Param)
			}
		}
	}
	return nil
}
