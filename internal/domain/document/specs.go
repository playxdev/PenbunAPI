// Path: internal/domain/document/specs.go
package document

import (
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/schema"
)

// All คืน spec ของเอกสารทุกชนิดที่ระบบรองรับ
func All() []*Spec {
	return []*Spec{ReceiveNote, Order, ReturnNote, VendorReturnNote}
}

func Validate() error {
	for _, s := range All() {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// docDate เป็นฟิลด์ร่วมของทุกเอกสาร
func docFields(extra ...schema.Field) []schema.Field {
	base := []schema.Field{
		{Name: "doc_no", Kind: schema.KindString, Required: true, MaxLen: 50,
			Label: "เลขที่เอกสาร", NoUpdate: true},
		{Name: "doc_date", Kind: schema.KindDate, Label: "วันที่เอกสาร"},
		{Name: "remark", Kind: schema.KindString, MaxLen: 500, Label: "หมายเหตุ"},
	}
	return append(base, extra...)
}

// ─────────────────────────────────────────────────────────────────────────────
// ใบรับสินค้าเข้าคลัง
// ─────────────────────────────────────────────────────────────────────────────

var ReceiveNote = &Spec{
	Name:  "receive-note",
	Label: "ใบรับสินค้า",

	HeaderTable: "tb_receive_note",
	ItemTable:   "tb_receive_item",

	// TEMP: ยังไม่มี View รองรับเอกสารชุดนี้ ใช้ derived table ไปก่อน
	HeaderSource: `(SELECT h.autoID AS receive_note_auto, h.receive_note_id, h.doc_no, h.doc_date,
	                       h.doc_status, h.trade_type, h.vendor_doc_no,
	                       h.total_qty, h.total_amount, h.remark,
	                       v.vendor_id, v.vendor_name,
	                       w.warehouse_id, w.warehouse_code, w.warehouse_name,
	                       h.is_active, h.id_status, h.update_by, h.update_date
	                  FROM dbo.tb_receive_note h
	                  INNER JOIN dbo.tb_vendor    v ON v.autoID = h.ref_vendor_auto
	                  INNER JOIN dbo.tb_warehouse w ON w.autoID = h.ref_warehouse_auto
	                 WHERE h.is_delete = 0) AS src`,

	ItemSource: `(SELECT i.autoID AS receive_item_auto, h.receive_note_id, i.line_no,
	                     s.sku_id, s.sku_code, s.issue_no,
	                     p.product_id, p.product_name,
	                     i.qty, i.unit_cost, i.cover_price, i.amount, i.remark,
	                     i.update_by, i.update_date
	                FROM dbo.tb_receive_item i
	                INNER JOIN dbo.tb_receive_note h ON h.autoID = i.ref_receive_note_auto
	                INNER JOIN dbo.tb_product_sku  s ON s.autoID = i.ref_sku_auto
	                INNER JOIN dbo.tb_product      p ON p.autoID = s.ref_product_auto
	               WHERE i.is_delete = 0) AS src`,

	IDColumn:     "receive_note_id",
	AutoColumn:   "receive_note_auto",
	ItemFKColumn: "ref_receive_note_auto",

	PostProc:       "USP_POST_RECEIVE",
	PostParamName:  "@ReceiveNoteID",
	PostedStatus:   "POSTED",
	PostedStatuses: []string{"POSTED"},

	// สินค้าเข้าคลังที่ระบุไว้ในหัวเอกสาร
	LockPairsSQL: `
SELECT DISTINCT i.ref_sku_auto, h.ref_warehouse_auto
  FROM dbo.tb_receive_item i
  INNER JOIN dbo.tb_receive_note h ON h.autoID = i.ref_receive_note_auto
 WHERE i.ref_receive_note_auto = @p1 AND i.is_delete = 0
 ORDER BY i.ref_sku_auto ASC, h.ref_warehouse_auto ASC`,

	TotalsSQL: `
UPDATE h
   SET h.total_qty    = ISNULL(t.qty, 0),
       h.total_amount = ISNULL(t.amt, 0),
       h.update_by    = @p2
  FROM dbo.tb_receive_note h
  OUTER APPLY (SELECT SUM(i.qty) AS qty, SUM(i.amount) AS amt
                 FROM dbo.tb_receive_item i
                WHERE i.ref_receive_note_auto = h.autoID AND i.is_delete = 0) t
 WHERE h.autoID = @p1`,

	HeaderRefs: []schema.Ref{
		{Field: "vendor_id", Table: "tb_vendor", Column: "ref_vendor_auto",
			Label: "คู่ค้า", Required: true},
		{Field: "warehouse_id", Table: "tb_warehouse", Column: "ref_warehouse_auto",
			Label: "คลังปลายทาง", Required: true},
		{Field: "company_id", Table: "tb_company", Column: "ref_company_auto", Label: "บริษัท"},
	},
	HeaderFields: docFields(
		schema.Field{Name: "trade_type", Kind: schema.KindString, Label: "รูปแบบการค้า",
			EnumValues: []string{"BUY", "CONSIGN"}},
		schema.Field{Name: "vendor_doc_no", Kind: schema.KindString, MaxLen: 50,
			Label: "เลขที่เอกสารของคู่ค้า"},
	),

	ItemRefs: []schema.Ref{
		{Field: "sku_id", Table: "tb_product_sku", Column: "ref_sku_auto",
			Label: "SKU", Required: true},
	},
	ItemFields: []schema.Field{
		{Name: "qty", Kind: schema.KindDecimal, Required: true,
			Label: "จำนวน", Min: schema.Float64(0)},
		{Name: "unit_cost", Kind: schema.KindDecimal, Required: true,
			Label: "ราคาทุนต่อหน่วย", Min: schema.Float64(0)},
		{Name: "cover_price", Kind: schema.KindDecimal, Label: "ราคาปก", Min: schema.Float64(0)},
		{Name: "remark", Kind: schema.KindString, MaxLen: 255, Label: "หมายเหตุ"},
	},

	Filters: []schema.Filter{
		{Param: "vendor_id", Column: "vendor_id", Kind: schema.KindString},
		{Param: "warehouse_code", Column: "warehouse_code", Kind: schema.KindString},
		{Param: "trade_type", Column: "trade_type", Kind: schema.KindString},
	},

	// trade_type ไม่ได้บังคับให้ส่ง ถ้าไม่ระบุจะตกไปใช้ DEFAULT ของคอลัมน์
	// ซึ่งตั้งไว้ตรงกับรูปแบบการค้าที่พบบ่อยที่สุด
	//
	// การกรอกผิดตรงนี้ทำให้ยอดที่ต้องจ่ายคืนเจ้าของสินค้าผิดทั้งงวด จึงควรให้
	// หน้าจอเติมค่าจาก tb_vendor.trade_type ของคู่ค้าที่เลือกไว้ให้ผู้ใช้เห็นก่อนบันทึก
}

// ─────────────────────────────────────────────────────────────────────────────
// ใบส่งสินค้าให้ลูกค้า
// ─────────────────────────────────────────────────────────────────────────────

var Order = &Spec{
	Name:  "order",
	Label: "ใบส่งหนังสือ",

	HeaderTable:  "tb_order",
	ItemTable:    "tb_order_item",
	HeaderSource: "dbo.vw_order_header",
	ItemSource:   "dbo.vw_order_item",

	IDColumn:     "order_id",
	AutoColumn:   "order_auto",
	ItemFKColumn: "ref_order_auto",

	PostProc:      "USP_POST_ORDER",
	PostParamName: "@OrderID",

	// สถานะหลังโพสต์ใบส่งคือ DELIVERED ไม่ใช่ POSTED เหมือนเอกสารอีกสามชนิด
	// เพราะใบส่งยังมีขั้นตอนวางบิลต่อ ฝั่ง frontend ต้องรู้ข้อนี้
	PostedStatus:   "DELIVERED",
	PostedStatuses: []string{"DELIVERED", "INVOICED"},

	LockPairsSQL: `
SELECT DISTINCT i.ref_sku_auto, h.ref_warehouse_auto
  FROM dbo.tb_order_item i
  INNER JOIN dbo.tb_order h ON h.autoID = i.ref_order_auto
 WHERE i.ref_order_auto = @p1 AND i.is_delete = 0
 ORDER BY i.ref_sku_auto ASC, h.ref_warehouse_auto ASC`,

	TotalsSQL: `
UPDATE h
   SET h.total_qty    = ISNULL(t.qty, 0),
       h.total_amount = ISNULL(t.amt, 0),
       h.update_by    = @p2
  FROM dbo.tb_order h
  OUTER APPLY (SELECT SUM(i.qty_delivered) AS qty, SUM(i.amount) AS amt
                 FROM dbo.tb_order_item i
                WHERE i.ref_order_auto = h.autoID AND i.is_delete = 0) t
 WHERE h.autoID = @p1`,

	HeaderRefs: []schema.Ref{
		{Field: "customer_id", Table: "tb_customer", Column: "ref_customer_auto",
			Label: "ลูกค้า", Required: true},
		{Field: "warehouse_id", Table: "tb_warehouse", Column: "ref_warehouse_auto",
			Label: "คลังต้นทาง", Required: true},
		{Field: "route_id", Table: "tb_route", Column: "ref_route_auto", Label: "สายจัดจำหน่าย"},
		{Field: "company_id", Table: "tb_company", Column: "ref_company_auto", Label: "บริษัท"},
	},
	HeaderFields: docFields(
		schema.Field{Name: "order_type", Kind: schema.KindString, Required: true,
			Label: "ประเภทใบส่ง", EnumValues: []string{"CONSIGN", "SALE", "TRANSFER"}},
		schema.Field{Name: "period_key", Kind: schema.KindString, MaxLen: 20, Label: "งวด"},
	),

	ItemRefs: []schema.Ref{
		{Field: "sku_id", Table: "tb_product_sku", Column: "ref_sku_auto",
			Label: "SKU", Required: true},
	},
	ItemFields: []schema.Field{
		{Name: "qty_ordered", Kind: schema.KindDecimal, Label: "จำนวนสั่ง", Min: schema.Float64(0)},
		{Name: "qty_delivered", Kind: schema.KindDecimal, Required: true,
			Label: "จำนวนส่ง", Min: schema.Float64(0)},
		{Name: "cover_price", Kind: schema.KindDecimal, Label: "ราคาปก", Min: schema.Float64(0)},
		{Name: "unit_price", Kind: schema.KindDecimal, Required: true,
			Label: "ราคาต่อหน่วย", Min: schema.Float64(0)},
		{Name: "discount_percent", Kind: schema.KindDecimal, Label: "ส่วนลด (%)", Min: schema.Float64(0)},
		{Name: "remark", Kind: schema.KindString, MaxLen: 255, Label: "หมายเหตุ"},
	},

	Filters: []schema.Filter{
		{Param: "customer_id", Column: "customer_id", Kind: schema.KindString},
		{Param: "order_type", Column: "order_type", Kind: schema.KindString},
		{Param: "period_key", Column: "period_key", Kind: schema.KindString},
		{Param: "route_code", Column: "route_code", Kind: schema.KindString},
	},

	ValidateHeader: func(vals map[string]any, items []Item) error {
		// ใบส่งแบบฝากขายที่ไม่ระบุงวด จะไม่ถูกบันทึกลงประวัติการจัดส่ง
		// ทำให้ฟังก์ชันเสนอยอดจากประวัติในงวดถัดไปมองไม่เห็นรอบนี้
		//
		// ปล่อยผ่านเงียบ ๆ ไม่ได้ เพราะกว่าจะรู้ตัวคือเดือนถัดไปและแก้ย้อนหลังไม่ได้
		if vals["order_type"] == "CONSIGN" {
			if v, ok := vals["period_key"].(string); !ok || v == "" {
				return httpx.Validation(
					"ใบส่งแบบฝากขายต้องระบุ period_key มิฉะนั้นจะไม่ถูกบันทึกลงประวัติการจัดส่ง").
					WithField("period_key", httpx.CodeFieldRequired, "")
			}
		}
		return nil
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// ใบรับคืนจากลูกค้า
// ─────────────────────────────────────────────────────────────────────────────

var ReturnNote = &Spec{
	Name:  "return-note",
	Label: "ใบรับคืน",

	HeaderTable: "tb_return_note",
	ItemTable:   "tb_return_item",

	// TEMP: ยังไม่มี View รองรับเอกสารชุดนี้
	HeaderSource: `(SELECT h.autoID AS return_note_auto, h.return_note_id, h.doc_no, h.doc_date,
	                       h.doc_status, h.period_key, h.total_qty, h.total_amount, h.remark,
	                       c.customer_id, c.customer_name,
	                       o.order_id AS ref_order_id,
	                       h.is_active, h.id_status, h.update_by, h.update_date
	                  FROM dbo.tb_return_note h
	                  INNER JOIN dbo.tb_customer c ON c.autoID = h.ref_customer_auto
	                  LEFT  JOIN dbo.tb_order    o ON o.autoID = h.ref_order_auto
	                 WHERE h.is_delete = 0) AS src`,

	ItemSource: `(SELECT i.autoID AS return_item_auto, h.return_note_id, i.line_no,
	                     s.sku_id, s.sku_code, s.issue_no,
	                     p.product_id, p.product_name,
	                     i.qty_returned, i.unit_price, i.cover_price, i.amount,
	                     i.condition_status, i.remark, i.update_by, i.update_date
	                FROM dbo.tb_return_item i
	                INNER JOIN dbo.tb_return_note h ON h.autoID = i.ref_return_note_auto
	                INNER JOIN dbo.tb_product_sku s ON s.autoID = i.ref_sku_auto
	                INNER JOIN dbo.tb_product     p ON p.autoID = s.ref_product_auto
	               WHERE i.is_delete = 0) AS src`,

	IDColumn:     "return_note_id",
	AutoColumn:   "return_note_auto",
	ItemFKColumn: "ref_return_note_auto",

	PostProc:       "USP_POST_RETURN",
	PostParamName:  "@ReturnNoteID",
	PostedStatus:   "POSTED",
	PostedStatuses: []string{"POSTED", "CREDITED"},

	// คลังปลายทางของสินค้าคืนขึ้นกับสภาพสินค้าเป็นรายบรรทัด
	// ของสภาพดีเข้าคลังรับคืน ของชำรุดเข้าคลังของเสีย และไม่มีอะไรกลับเข้าคลังหลัก
	// จึงต้องจองล็อกคลังคนละใบตามสภาพ ไม่ใช่คลังเดียวจากหัวเอกสาร
	LockPairsSQL: `
SELECT DISTINCT i.ref_sku_auto,
       w.autoID AS warehouse_auto
  FROM dbo.tb_return_item i
  CROSS APPLY (SELECT TOP 1 autoID
                 FROM dbo.tb_warehouse
                WHERE warehouse_code = CASE i.condition_status
                                            WHEN 'DAMAGED' THEN 'DMG'
                                            ELSE 'RET'
                                       END
                  AND is_delete = 0) w
 WHERE i.ref_return_note_auto = @p1 AND i.is_delete = 0
 ORDER BY i.ref_sku_auto ASC, warehouse_auto ASC`,

	TotalsSQL: `
UPDATE h
   SET h.total_qty    = ISNULL(t.qty, 0),
       h.total_amount = ISNULL(t.amt, 0),
       h.update_by    = @p2
  FROM dbo.tb_return_note h
  OUTER APPLY (SELECT SUM(i.qty_returned) AS qty, SUM(i.amount) AS amt
                 FROM dbo.tb_return_item i
                WHERE i.ref_return_note_auto = h.autoID AND i.is_delete = 0) t
 WHERE h.autoID = @p1`,

	HeaderRefs: []schema.Ref{
		{Field: "customer_id", Table: "tb_customer", Column: "ref_customer_auto",
			Label: "ลูกค้า", Required: true},
		{Field: "ref_order_id", Table: "tb_order", Column: "ref_order_auto",
			Label: "ใบส่งต้นทาง"},
		{Field: "company_id", Table: "tb_company", Column: "ref_company_auto", Label: "บริษัท"},
	},
	HeaderFields: docFields(
		schema.Field{Name: "period_key", Kind: schema.KindString, MaxLen: 20, Label: "งวด"},
	),

	ItemRefs: []schema.Ref{
		{Field: "sku_id", Table: "tb_product_sku", Column: "ref_sku_auto",
			Label: "SKU", Required: true},
	},
	ItemFields: []schema.Field{
		{Name: "qty_returned", Kind: schema.KindDecimal, Required: true,
			Label: "จำนวนคืน", Min: schema.Float64(0)},
		{Name: "unit_price", Kind: schema.KindDecimal, Label: "ราคาต่อหน่วย", Min: schema.Float64(0)},
		{Name: "cover_price", Kind: schema.KindDecimal, Label: "ราคาปก", Min: schema.Float64(0)},
		// สภาพสินค้าเป็นตัวกำหนดคลังปลายทาง จึงบังคับให้ระบุเสมอ
		// ค่าเริ่มต้นเงียบ ๆ จะทำให้ของชำรุดปนกลับเข้าคลังขายได้
		{Name: "condition_status", Kind: schema.KindString, Required: true,
			Label: "สภาพสินค้า", EnumValues: []string{"GOOD", "DAMAGED"}},
		{Name: "remark", Kind: schema.KindString, MaxLen: 255, Label: "หมายเหตุ"},
	},

	Filters: []schema.Filter{
		{Param: "customer_id", Column: "customer_id", Kind: schema.KindString},
		{Param: "period_key", Column: "period_key", Kind: schema.KindString},
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// ใบส่งคืนเจ้าของสินค้า
// ─────────────────────────────────────────────────────────────────────────────

var VendorReturnNote = &Spec{
	Name:  "vendor-return-note",
	Label: "ใบส่งคืนคู่ค้า",

	HeaderTable: "tb_vendor_return_note",
	ItemTable:   "tb_vendor_return_item",

	// TEMP: ยังไม่มี View รองรับเอกสารชุดนี้
	HeaderSource: `(SELECT h.autoID AS vendor_return_note_auto, h.vendor_return_note_id,
	                       h.doc_no, h.doc_date, h.doc_status,
	                       h.total_qty, h.total_amount, h.remark,
	                       v.vendor_id, v.vendor_name,
	                       w.warehouse_id, w.warehouse_code, w.warehouse_name,
	                       h.is_active, h.id_status, h.update_by, h.update_date
	                  FROM dbo.tb_vendor_return_note h
	                  INNER JOIN dbo.tb_vendor    v ON v.autoID = h.ref_vendor_auto
	                  INNER JOIN dbo.tb_warehouse w ON w.autoID = h.ref_warehouse_auto
	                 WHERE h.is_delete = 0) AS src`,

	ItemSource: `(SELECT i.autoID AS vendor_return_item_auto, h.vendor_return_note_id, i.line_no,
	                     s.sku_id, s.sku_code, s.issue_no,
	                     p.product_id, p.product_name,
	                     i.qty_returned, i.unit_cost, i.amount, i.remark,
	                     i.update_by, i.update_date
	                FROM dbo.tb_vendor_return_item i
	                INNER JOIN dbo.tb_vendor_return_note h ON h.autoID = i.ref_vendor_return_note_auto
	                INNER JOIN dbo.tb_product_sku       s ON s.autoID = i.ref_sku_auto
	                INNER JOIN dbo.tb_product           p ON p.autoID = s.ref_product_auto
	               WHERE i.is_delete = 0) AS src`,

	IDColumn:     "vendor_return_note_id",
	AutoColumn:   "vendor_return_note_auto",
	ItemFKColumn: "ref_vendor_return_note_auto",

	PostProc:       "USP_POST_VENDOR_RETURN",
	PostParamName:  "@VendorReturnNoteID",
	PostedStatus:   "POSTED",
	PostedStatuses: []string{"POSTED", "SETTLED"},

	// ตัดออกจากคลังที่ระบุในหัวเอกสาร ซึ่งปกติคือคลังรับคืน
	LockPairsSQL: `
SELECT DISTINCT i.ref_sku_auto, h.ref_warehouse_auto
  FROM dbo.tb_vendor_return_item i
  INNER JOIN dbo.tb_vendor_return_note h ON h.autoID = i.ref_vendor_return_note_auto
 WHERE i.ref_vendor_return_note_auto = @p1 AND i.is_delete = 0
 ORDER BY i.ref_sku_auto ASC, h.ref_warehouse_auto ASC`,

	TotalsSQL: `
UPDATE h
   SET h.total_qty    = ISNULL(t.qty, 0),
       h.total_amount = ISNULL(t.amt, 0),
       h.update_by    = @p2
  FROM dbo.tb_vendor_return_note h
  OUTER APPLY (SELECT SUM(i.qty_returned) AS qty, SUM(i.amount) AS amt
                 FROM dbo.tb_vendor_return_item i
                WHERE i.ref_vendor_return_note_auto = h.autoID AND i.is_delete = 0) t
 WHERE h.autoID = @p1`,

	HeaderRefs: []schema.Ref{
		{Field: "vendor_id", Table: "tb_vendor", Column: "ref_vendor_auto",
			Label: "คู่ค้า", Required: true},
		{Field: "warehouse_id", Table: "tb_warehouse", Column: "ref_warehouse_auto",
			Label: "คลังต้นทาง", Required: true},
		{Field: "company_id", Table: "tb_company", Column: "ref_company_auto", Label: "บริษัท"},
	},
	HeaderFields: docFields(),

	ItemRefs: []schema.Ref{
		{Field: "sku_id", Table: "tb_product_sku", Column: "ref_sku_auto",
			Label: "SKU", Required: true},
	},
	ItemFields: []schema.Field{
		{Name: "qty_returned", Kind: schema.KindDecimal, Required: true,
			Label: "จำนวนคืน", Min: schema.Float64(0)},
		{Name: "unit_cost", Kind: schema.KindDecimal, Required: true,
			Label: "ราคาทุนต่อหน่วย", Min: schema.Float64(0)},
		{Name: "remark", Kind: schema.KindString, MaxLen: 255, Label: "หมายเหตุ"},
	},

	Filters: []schema.Filter{
		{Param: "vendor_id", Column: "vendor_id", Kind: schema.KindString},
		{Param: "warehouse_code", Column: "warehouse_code", Kind: schema.KindString},
	},
}
