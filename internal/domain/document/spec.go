// Path: internal/domain/document/spec.go
package document

import (
	"fmt"

	"penbun/api/internal/schema"
)

// สถานะที่ใช้ร่วมกันทุกเอกสาร
const (
	StatusDraft     = "DRAFT"
	StatusConfirmed = "CONFIRMED"
	StatusCancelled = "CANCELLED"
)

// Spec คือคำอธิบายของเอกสารหนึ่งชนิด
//
// เอกสารทั้งสี่ชนิด (ใบรับ / ใบส่ง / ใบรับคืน / ใบส่งคืนเจ้าของ) มีวงจรชีวิต
// เหมือนกันคือ DRAFT → CONFIRMED → โพสต์ผ่าน Stored Procedure ต่างกันแค่
// ชื่อตาราง ชื่อ proc สถานะปลายทาง และคลังที่สินค้าเข้าหรือออก
//
// จึงเขียน engine ตัวเดียวแล้วอธิบายความต่างผ่าน Spec แทนการทำ handler สี่ชุด
type Spec struct {
	Name  string // path segment เช่น "order"
	Label string // ชื่อไทย เช่น "ใบส่งหนังสือ"

	HeaderTable string // "tb_order"
	ItemTable   string // "tb_order_item"

	// HeaderSource / ItemSource คือสิ่งที่ตามหลัง FROM
	// ใช้ View เมื่อมี ใช้ derived table เมื่อยังไม่มี (คอมเมนต์ TEMP กำกับไว้)
	HeaderSource string
	ItemSource   string

	IDColumn     string // "order_id"
	AutoColumn   string // "order_auto" — ชื่อใน HeaderSource
	ItemFKColumn string // "ref_order_auto"

	// PostProc คือ Stored Procedure ที่ทำให้เอกสารมีผลต่อสต็อกจริง
	// API ไม่คำนวณสต็อกเองแม้แต่ขั้นตอนเดียว
	PostProc      string // "USP_POST_ORDER"
	PostParamName string // "@OrderID"
	PostedStatus  string // สถานะหลังโพสต์สำเร็จ

	// PostedStatuses คือสถานะทั้งหมดที่แปลว่า "โพสต์ไปแล้ว"
	// ใช้แยก 409 ALREADY_POSTED ออกจาก 422 ที่สถานะยังไม่พร้อมโพสต์
	PostedStatuses []string

	// LockPairsSQL คืนคู่ (sku autoID, warehouse autoID) ที่ต้องจองล็อกก่อนโพสต์
	// รับ @p1 เป็น autoID ของ header และต้อง ORDER BY ให้ผลลัพธ์เรียงเสมอ
	//
	// แต่ละเอกสารกำหนดคลังปลายทางคนละแบบ ใบรับ/ใบส่ง/ใบส่งคืนใช้คลังใน header
	// ส่วนใบรับคืนแยกคลังตามสภาพสินค้าเป็นรายบรรทัด จึงให้ Spec เขียน SQL เอง
	LockPairsSQL string

	// TotalsSQL คำนวณยอดรวมของ header ใหม่จากรายการจริง รับ @p1 = header autoID, @p2 = update_by
	TotalsSQL string

	HeaderRefs   []schema.Ref
	HeaderFields []schema.Field
	ItemRefs     []schema.Ref
	ItemFields   []schema.Field

	Filters []schema.Filter

	// DefaultsFrom ให้ Spec เติมค่าเริ่มต้นจากข้อมูลอ้างอิงก่อน insert
	// เช่น ใบรับที่ไม่ได้ส่ง trade_type มา ให้ยึดตามที่ตั้งไว้ที่คู่ค้า
	DefaultsFrom func(body map[string]any, refs map[string]int) error

	// ValidateHeader ตรวจกฎเฉพาะของเอกสารชนิดนั้นหลังผ่านการตรวจทั่วไปแล้ว
	ValidateHeader func(vals map[string]any, items []Item) error
}

func (s *Spec) headerField(name string) (schema.Field, bool) {
	for _, f := range s.HeaderFields {
		if f.Name == name {
			return f, true
		}
	}
	return schema.Field{}, false
}

func (s *Spec) isPosted(status string) bool {
	for _, v := range s.PostedStatuses {
		if v == status {
			return true
		}
	}
	return false
}

func (s *Spec) Validate() error {
	switch {
	case s.Name == "":
		return fmt.Errorf("document: Name is required")
	case s.HeaderTable == "" || s.ItemTable == "":
		return fmt.Errorf("document %s: HeaderTable and ItemTable are required", s.Name)
	case s.HeaderSource == "" || s.ItemSource == "":
		return fmt.Errorf("document %s: HeaderSource and ItemSource are required", s.Name)
	case s.IDColumn == "" || s.AutoColumn == "" || s.ItemFKColumn == "":
		return fmt.Errorf("document %s: IDColumn, AutoColumn and ItemFKColumn are required", s.Name)
	case s.PostProc == "" || s.PostParamName == "":
		return fmt.Errorf("document %s: PostProc and PostParamName are required", s.Name)
	case s.PostedStatus == "":
		return fmt.Errorf("document %s: PostedStatus is required", s.Name)
	case len(s.PostedStatuses) == 0:
		return fmt.Errorf("document %s: PostedStatuses is required", s.Name)
	case s.LockPairsSQL == "":
		// ไม่มี LockPairsSQL แปลว่าโพสต์ได้โดยไม่จองล็อก ซึ่งเปิดช่องให้สต็อกติดลบ
		return fmt.Errorf("document %s: LockPairsSQL is required", s.Name)
	case s.TotalsSQL == "":
		return fmt.Errorf("document %s: TotalsSQL is required", s.Name)
	}

	seen := map[string]bool{}
	for _, f := range s.HeaderFields {
		if seen[f.Name] {
			return fmt.Errorf("document %s: duplicate header field %q", s.Name, f.Name)
		}
		seen[f.Name] = true
	}
	for _, r := range s.HeaderRefs {
		if seen[r.Field] {
			return fmt.Errorf("document %s: header ref %q collides with a field", s.Name, r.Field)
		}
		seen[r.Field] = true
	}

	hasSku := false
	for _, r := range s.ItemRefs {
		if r.Column == "ref_sku_auto" {
			hasSku = true
		}
	}
	if !hasSku {
		return fmt.Errorf("document %s: ItemRefs must contain a ref to ref_sku_auto", s.Name)
	}
	return nil
}
