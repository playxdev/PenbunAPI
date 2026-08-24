// Path: internal/resources/product.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// ─────────────────────────────────────────────────────────────────────────────
// Layer 4–5 : Product (Hybrid Core)
//
// tb_product ตารางเดียวรองรับทั้งสินค้ามีสต็อกและบริการ ควบคุมด้วย count_stock
//   count_stock = 1  หนังสือ / นิตยสาร / กล่องพัสดุ  → ตัดสต็อก
//   count_stock = 0  ค่าส่ง EMS / ค่าเช่า Cloud      → ไม่ตัดสต็อก
// ─────────────────────────────────────────────────────────────────────────────

var Product = &crud.Resource{
	Name:          "product",
	Label:         "สินค้า",
	Source:        "dbo.vw_product",
	Table:         "tb_product",
	IDColumn:      "product_id",
	AutoColumn:    "product_auto",
	SearchColumns: []string{"product_name", "product_code", "barcode"},
	SortColumns: map[string]string{
		"name": "product_name", "code": "product_code", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "product_category_id", Column: "product_category_id", Kind: schema.KindString},
		{Param: "product_group_id", Column: "product_group_id", Kind: schema.KindString},
		{Param: "vendor_id", Column: "vendor_id", Kind: schema.KindString},
		{Param: "count_stock", Column: "count_stock", Kind: schema.KindBool},
	},
	Refs: []schema.Ref{
		{Field: "product_group_id", Table: "tb_product_group",
			Column: "ref_product_group_auto", Label: "กลุ่มสินค้า", Required: true},
		{Field: "product_format_type_id", Table: "tb_product_format_type",
			Column: "ref_product_format_type_auto", Label: "รูปแบบสินค้า"},
		{Field: "unit_type_id", Table: "tb_unit_type",
			Column: "ref_unit_type_auto", Label: "หน่วยนับ"},
		{Field: "vendor_id", Table: "tb_vendor",
			Column: "ref_vendor_auto", Label: "คู่ค้า"},
	},
	Fields: []schema.Field{
		{Name: "product_code", Kind: schema.KindString, Required: true, MaxLen: 50, Label: "รหัสสินค้า"},
		{Name: "product_name", Kind: schema.KindString, Required: true, MaxLen: 250, Label: "ชื่อสินค้า"},
		{Name: "count_stock", Kind: schema.KindBool, Label: "นับสต็อก"},
		{Name: "cost_price", Kind: schema.KindDecimal, Label: "ราคาทุน"},
		{Name: "sell_price", Kind: schema.KindDecimal, Label: "ราคาขาย"},
		{Name: "barcode", Kind: schema.KindString, MaxLen: 50, Label: "บาร์โค้ด"},
		{Name: "weight_kg", Kind: schema.KindDecimal, Label: "น้ำหนัก (กก.)"},
		{Name: "pack_qty", Kind: schema.KindDecimal, Label: "จำนวนต่อแพ็ก"},
		{Name: "description", Kind: schema.KindString, MaxLen: 500, Label: "รายละเอียด"},
	},
}

// ProductSKU คือ "ฉบับ" ในภาษาของโปรแกรมเดิม
// สต็อกทั้งระบบนับที่ระดับ SKU ไม่ใช่ระดับ product
var ProductSKU = &crud.Resource{
	Name:          "product-sku",
	Label:         "SKU",
	Source:        "dbo.vw_product_sku",
	Table:         "tb_product_sku",
	IDColumn:      "sku_id",
	AutoColumn:    "sku_auto",
	SearchColumns: []string{"sku_code", "barcode", "variation_name"},
	SortColumns: map[string]string{
		"code": "sku_code", "issue": "issue_no", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "product_id", Column: "product_id", Kind: schema.KindString},
		{Param: "issue_no", Column: "issue_no", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "product_id", Table: "tb_product", Column: "ref_product_auto",
			Label: "สินค้า", Required: true, NoUpdate: true},
	},
	Fields: []schema.Field{
		{Name: "sku_code", Kind: schema.KindString, Required: true, MaxLen: 50, Label: "รหัส SKU"},
		{Name: "barcode", Kind: schema.KindString, MaxLen: 50, Label: "บาร์โค้ด"},
		{Name: "vendor_part_no", Kind: schema.KindString, MaxLen: 50, Label: "รหัสของคู่ค้า"},
		{Name: "variation_name", Kind: schema.KindString, MaxLen: 150, Label: "ชื่อรุ่น/แบบ"},
		{Name: "issue_no", Kind: schema.KindString, MaxLen: 30, Label: "ฉบับที่"},
		{Name: "volume_no", Kind: schema.KindString, MaxLen: 30, Label: "เล่มที่"},
		{Name: "edition_label", Kind: schema.KindString, MaxLen: 50, Label: "ครั้งที่พิมพ์"},
		{Name: "cost_price", Kind: schema.KindDecimal, Label: "ราคาทุน"},
		{Name: "sell_price", Kind: schema.KindDecimal, Label: "ราคาขาย"},
		{Name: "cover_price", Kind: schema.KindDecimal, Label: "ราคาปก"},
		{Name: "pack_qty", Kind: schema.KindDecimal, Label: "จำนวนต่อแพ็ก"},
		{Name: "publication_date", Kind: schema.KindDate, Label: "วันวางแผง"},
		{Name: "return_deadline", Kind: schema.KindDate, Label: "กำหนดรับคืน"},
		{Name: "description", Kind: schema.KindString, MaxLen: 500, Label: "รายละเอียด"},
	},
}

// Book เป็น read-only ใน engine กลาง
//
// v7 บังคับ UQ_tb_book_product คือหนังสือหนึ่งเล่มผูก tb_product หนึ่งแถวเสมอ
// การสร้างจึงต้องเขียนสองตารางในทรานแซกชันเดียว ซึ่งเกินขอบเขตของ generic CRUD
// → POST/PUT/DELETE อยู่ที่ internal/domain/book แทน
var Book = &crud.Resource{
	Name:          "book",
	Label:         "หนังสือ",
	Source:        "dbo.vw_book",
	Table:         "tb_book",
	IDColumn:      "book_id",
	AutoColumn:    "book_auto",
	SearchColumns: []string{"book_name", "author", "isbn"},
	SortColumns: map[string]string{
		"name": "book_name", "author": "author", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "book_type_id", Column: "book_type_id", Kind: schema.KindString},
		{Param: "vendor_id", Column: "vendor_id", Kind: schema.KindString},
	},
	ReadOnly: true,
}
