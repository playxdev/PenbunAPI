// Path: internal/resources/lookup.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// ─────────────────────────────────────────────────────────────────────────────
// Layer 1 : Lookup
//
// TEMP: PenbunSQL v7 ยังไม่มี View ให้ตารางกลุ่มนี้ จึงใช้ derived table แทน
// เมื่อ DB v7.1 เพิ่ม vw_* ครบแล้ว ให้เปลี่ยน Source เป็นชื่อ View ตรง ๆ
// แล้วลบ subselect ทิ้ง — ไม่ต้องแตะส่วนอื่นของ descriptor เลย
// ─────────────────────────────────────────────────────────────────────────────

// audit คือคอลัมน์มาตรฐานที่ทุกตารางมีเหมือนกัน ใส่ท้าย SELECT ของ derived table
const audit = "is_active, id_status, update_by, update_date"

var CustomerType = &crud.Resource{
	Name:  "customer-type",
	Label: "ประเภทลูกค้า",
	Source: `(SELECT autoID AS customer_type_auto, customer_type_id, type_name, description,
	                 base_credit_day, ` + audit + `
	            FROM dbo.tb_customer_type WHERE is_delete = 0) AS src`,
	Table:         "tb_customer_type",
	IDColumn:      "customer_type_id",
	AutoColumn:    "customer_type_auto",
	SearchColumns: []string{"type_name"},
	SortColumns: map[string]string{
		"name":    "type_name",
		"updated": "update_date",
	},
	DefaultSort: "updated",
	Fields: []schema.Field{
		{Name: "type_name", Kind: schema.KindString, Required: true, MaxLen: 255, Label: "ชื่อประเภท"},
		{Name: "description", Kind: schema.KindString, MaxLen: 1000, Label: "คำอธิบาย"},
		{Name: "base_credit_day", Kind: schema.KindInt, Label: "เครดิตพื้นฐาน (วัน)"},
	},
}

var VendorType = &crud.Resource{
	Name:  "vendor-type",
	Label: "ประเภทคู่ค้า",
	Source: `(SELECT autoID AS vendor_type_auto, vendor_type_id, type_name, description, ` + audit + `
	            FROM dbo.tb_vendor_type WHERE is_delete = 0) AS src`,
	Table:         "tb_vendor_type",
	IDColumn:      "vendor_type_id",
	AutoColumn:    "vendor_type_auto",
	SearchColumns: []string{"type_name"},
	SortColumns:   map[string]string{"name": "type_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "type_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อประเภท"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var UnitType = &crud.Resource{
	Name:  "unit-type",
	Label: "หน่วยนับ",
	Source: `(SELECT autoID AS unit_type_auto, unit_type_id, unit_type_name, description, ` + audit + `
	            FROM dbo.tb_unit_type WHERE is_delete = 0) AS src`,
	Table:         "tb_unit_type",
	IDColumn:      "unit_type_id",
	AutoColumn:    "unit_type_auto",
	SearchColumns: []string{"unit_type_name"},
	SortColumns:   map[string]string{"name": "unit_type_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "unit_type_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อหน่วยนับ"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var BookType = &crud.Resource{
	Name:  "book-type",
	Label: "ประเภทหนังสือ",
	Source: `(SELECT autoID AS book_type_auto, book_type_id, type_name, description, ` + audit + `
	            FROM dbo.tb_book_type WHERE is_delete = 0) AS src`,
	Table:         "tb_book_type",
	IDColumn:      "book_type_id",
	AutoColumn:    "book_type_auto",
	SearchColumns: []string{"type_name"},
	SortColumns:   map[string]string{"name": "type_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "type_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อประเภท"},
		{Name: "description", Kind: schema.KindString, MaxLen: 250, Label: "คำอธิบาย"},
	},
}

var ProductCategory = &crud.Resource{
	Name:  "product-category",
	Label: "หมวดสินค้า",
	Source: `(SELECT autoID AS product_category_auto, product_category_id, category_code,
	                 category_name, description, ` + audit + `
	            FROM dbo.tb_product_category WHERE is_delete = 0) AS src`,
	Table:         "tb_product_category",
	IDColumn:      "product_category_id",
	AutoColumn:    "product_category_auto",
	SearchColumns: []string{"category_name", "category_code"},
	SortColumns:   map[string]string{"name": "category_name", "code": "category_code", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "category_code", Kind: schema.KindString, Required: true, MaxLen: 20, Label: "รหัสหมวด"},
		{Name: "category_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อหมวด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var ProductFormatType = &crud.Resource{
	Name:  "product-format-type",
	Label: "รูปแบบสินค้า",
	Source: `(SELECT autoID AS product_format_type_auto, product_format_type_id, format_name,
	                 description, ` + audit + `
	            FROM dbo.tb_product_format_type WHERE is_delete = 0) AS src`,
	Table:         "tb_product_format_type",
	IDColumn:      "product_format_type_id",
	AutoColumn:    "product_format_type_auto",
	SearchColumns: []string{"format_name"},
	SortColumns:   map[string]string{"name": "format_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "format_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อรูปแบบ"},
		{Name: "description", Kind: schema.KindString, MaxLen: 250, Label: "คำอธิบาย"},
	},
}

var DiscountType = &crud.Resource{
	Name:  "discount-type",
	Label: "ประเภทส่วนลด",
	Source: `(SELECT autoID AS discount_type_auto, discount_type_id, discount_type_name,
	                 description, ` + audit + `
	            FROM dbo.tb_discount_type WHERE is_delete = 0) AS src`,
	Table:         "tb_discount_type",
	IDColumn:      "discount_type_id",
	AutoColumn:    "discount_type_auto",
	SearchColumns: []string{"discount_type_name"},
	SortColumns:   map[string]string{"name": "discount_type_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Fields: []schema.Field{
		{Name: "discount_type_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อประเภทส่วนลด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 250, Label: "คำอธิบาย"},
	},
}
