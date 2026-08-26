// Path: internal/resources/master.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// warehouseTypes ตรงกับ CK_tb_warehouse_type ของ v7
// ตรวจฝั่ง API ด้วยเพื่อให้ผู้ใช้ได้ข้อความที่อ่านรู้เรื่องกว่าข้อความ CHECK constraint
var warehouseTypes = []string{"DC", "BRANCH", "RETURN", "DAMAGED", "PROVINCE", "INTERNATIONAL"}

var Warehouse = &crud.Resource{
	Name:          "warehouse",
	Label:         "คลังสินค้า",
	Source:        "dbo.vw_warehouse",
	Table:         "tb_warehouse",
	IDColumn:      "warehouse_id",
	AutoColumn:    "warehouse_auto",
	SearchColumns: []string{"warehouse_name", "warehouse_code"},
	SortColumns: map[string]string{
		"code": "warehouse_code", "name": "warehouse_name", "updated": "update_date",
	},
	DefaultSort: "code",
	Filters: []schema.Filter{
		{Param: "warehouse_type", Column: "warehouse_type", Kind: schema.KindString},
		{Param: "province", Column: "province", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "company_id", Table: "tb_company", Column: "ref_company_auto", Label: "บริษัท"},
	},
	Fields: []schema.Field{
		{Name: "warehouse_code", Kind: schema.KindString, Required: true, MaxLen: 20, Label: "รหัสคลัง", NoUpdate: true},
		{Name: "warehouse_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อคลัง"},
		{Name: "warehouse_type", Kind: schema.KindString, Required: true, Label: "ประเภทคลัง", EnumValues: warehouseTypes},
		{Name: "is_main_dc", Kind: schema.KindBool, Label: "เป็นคลังหลัก"},
		{Name: "allow_negative_stock", Kind: schema.KindBool, Label: "อนุญาตให้สต็อกติดลบ"},
		{Name: "address", Kind: schema.KindString, MaxLen: 255, Label: "ที่อยู่"},
		{Name: "province", Kind: schema.KindString, MaxLen: 100, Label: "จังหวัด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var ProductGroup = &crud.Resource{
	Name:          "product-group",
	Label:         "กลุ่มสินค้า",
	Source:        "dbo.vw_product_group",
	Table:         "tb_product_group",
	IDColumn:      "product_group_id",
	AutoColumn:    "product_group_auto",
	SearchColumns: []string{"product_group_name"},
	SortColumns:   map[string]string{"name": "product_group_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Filters: []schema.Filter{
		{Param: "product_category_id", Column: "product_category_id", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "product_category_id", Table: "tb_product_category",
			Column: "ref_product_category_auto", Label: "หมวดสินค้า", Required: true},
	},
	Fields: []schema.Field{
		{Name: "product_group_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อกลุ่มสินค้า"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var Customer = &crud.Resource{
	Name:          "customer",
	Label:         "ลูกค้า",
	Source:        "dbo.vw_customer",
	Table:         "tb_customer",
	IDColumn:      "customer_id",
	AutoColumn:    "customer_auto",
	SearchColumns: []string{"customer_name", "customer_code", "tax_id"},
	SortColumns: map[string]string{
		"name": "customer_name", "code": "customer_code", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "customer_type_id", Column: "customer_type_id", Kind: schema.KindString},
		{Param: "province", Column: "province", Kind: schema.KindString},
		{Param: "discount_group", Column: "discount_group", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "customer_type_id", Table: "tb_customer_type",
			Column: "ref_customer_type_auto", Label: "ประเภทลูกค้า", Required: true},
	},
	Fields: []schema.Field{
		{Name: "customer_code", Kind: schema.KindString, MaxLen: 20, Label: "รหัสลูกค้า"},
		{Name: "customer_name", Kind: schema.KindString, Required: true, MaxLen: 200, Label: "ชื่อลูกค้า"},
		{Name: "report_name", Kind: schema.KindString, MaxLen: 200, Label: "ชื่อในรายงาน"},
		{Name: "tax_id", Kind: schema.KindString, MaxLen: 20, Label: "เลขประจำตัวผู้เสียภาษี"},
		{Name: "branch_code", Kind: schema.KindString, MaxLen: 10, Label: "รหัสสาขา"},
		{Name: "branch_name", Kind: schema.KindString, MaxLen: 50, Label: "ชื่อสาขา"},
		{Name: "contact_person", Kind: schema.KindString, MaxLen: 100, Label: "ผู้ติดต่อ"},
		{Name: "phone1", Kind: schema.KindString, MaxLen: 50, Label: "โทรศัพท์"},
		{Name: "phone2", Kind: schema.KindString, MaxLen: 50, Label: "โทรศัพท์สำรอง"},
		{Name: "email", Kind: schema.KindString, MaxLen: 100, Label: "อีเมล"},
		{Name: "line_id", Kind: schema.KindString, MaxLen: 50, Label: "LINE ID"},
		{Name: "address", Kind: schema.KindString, MaxLen: 255, Label: "ที่อยู่"},
		{Name: "sub_district", Kind: schema.KindString, MaxLen: 100, Label: "ตำบล/แขวง"},
		{Name: "district", Kind: schema.KindString, MaxLen: 100, Label: "อำเภอ/เขต"},
		{Name: "province", Kind: schema.KindString, MaxLen: 100, Label: "จังหวัด"},
		{Name: "zip_code", Kind: schema.KindString, MaxLen: 20, Label: "รหัสไปรษณีย์"},
		{Name: "credit_limit", Kind: schema.KindDecimal, Label: "วงเงินเครดิต"},
		{Name: "credit_term_day", Kind: schema.KindInt, Label: "เครดิต (วัน)"},
		{Name: "is_vat", Kind: schema.KindBool, Label: "จด VAT"},
		{Name: "invoice_format", Kind: schema.KindString, MaxLen: 20, Label: "รูปแบบใบกำกับ"},
		{Name: "discount_group", Kind: schema.KindString, MaxLen: 20, Label: "กลุ่มส่วนลด"},
		{Name: "note", Kind: schema.KindString, MaxLen: 500, Label: "หมายเหตุ"},
	},
}

var Vendor = &crud.Resource{
	Name:          "vendor",
	Label:         "คู่ค้า",
	Source:        "dbo.vw_vendor",
	Table:         "tb_vendor",
	IDColumn:      "vendor_id",
	AutoColumn:    "vendor_auto",
	SearchColumns: []string{"vendor_name", "tax_id"},
	SortColumns:   map[string]string{"name": "vendor_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Filters: []schema.Filter{
		{Param: "vendor_type_id", Column: "vendor_type_id", Kind: schema.KindString},
		{Param: "trade_type", Column: "trade_type", Kind: schema.KindString},
		{Param: "province", Column: "province", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "vendor_type_id", Table: "tb_vendor_type",
			Column: "ref_vendor_type_auto", Label: "ประเภทคู่ค้า", Required: true},
	},
	Fields: []schema.Field{
		{Name: "vendor_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อคู่ค้า"},
		{Name: "tax_id", Kind: schema.KindString, MaxLen: 20, Label: "เลขประจำตัวผู้เสียภาษี"},
		{Name: "branch_code", Kind: schema.KindString, MaxLen: 10, Label: "รหัสสาขา"},
		{Name: "branch_name", Kind: schema.KindString, MaxLen: 50, Label: "ชื่อสาขา"},
		{Name: "contact_person", Kind: schema.KindString, MaxLen: 100, Label: "ผู้ติดต่อ"},
		{Name: "phone1", Kind: schema.KindString, MaxLen: 50, Label: "โทรศัพท์"},
		{Name: "phone2", Kind: schema.KindString, MaxLen: 50, Label: "โทรศัพท์สำรอง"},
		{Name: "email", Kind: schema.KindString, MaxLen: 100, Label: "อีเมล"},
		{Name: "website", Kind: schema.KindString, MaxLen: 100, Label: "เว็บไซต์"},
		{Name: "address", Kind: schema.KindString, MaxLen: 255, Label: "ที่อยู่"},
		{Name: "sub_district", Kind: schema.KindString, MaxLen: 100, Label: "ตำบล/แขวง"},
		{Name: "district", Kind: schema.KindString, MaxLen: 100, Label: "อำเภอ/เขต"},
		{Name: "province", Kind: schema.KindString, MaxLen: 100, Label: "จังหวัด"},
		{Name: "zip_code", Kind: schema.KindString, MaxLen: 20, Label: "รหัสไปรษณีย์"},
		{Name: "credit_term_day", Kind: schema.KindInt, Label: "เครดิต (วัน)"},
		{Name: "currency", Kind: schema.KindString, MaxLen: 10, Label: "สกุลเงิน"},
		// เงื่อนไขฝากขาย — ใช้เป็นฐานคำนวณเงินจ่ายเจ้าของหนังสือ
		{Name: "trade_type", Kind: schema.KindString, Required: true, Label: "รูปแบบการค้า",
			EnumValues: []string{"BUY", "CONSIGN"}},
		{Name: "consign_share_percent", Kind: schema.KindDecimal, Label: "ส่วนแบ่งฝากขาย (%)"},
		{Name: "settlement_cycle", Kind: schema.KindString, MaxLen: 20, Label: "รอบจ่ายเงิน"},
		{Name: "settlement_day", Kind: schema.KindInt, Label: "วันที่จ่ายเงิน"},
		{Name: "return_window_day", Kind: schema.KindInt, Label: "ระยะเวลารับคืน (วัน)"},
		{Name: "withholding_tax_percent", Kind: schema.KindDecimal, Label: "หัก ณ ที่จ่าย (%)"},
		{Name: "bank_name", Kind: schema.KindString, MaxLen: 100, Label: "ธนาคาร"},
		{Name: "bank_branch", Kind: schema.KindString, MaxLen: 100, Label: "สาขาธนาคาร"},
		{Name: "bank_account_no", Kind: schema.KindString, MaxLen: 30, Label: "เลขที่บัญชี"},
		{Name: "bank_account_name", Kind: schema.KindString, MaxLen: 150, Label: "ชื่อบัญชี"},
		{Name: "note", Kind: schema.KindString, MaxLen: 500, Label: "หมายเหตุ"},
	},
}

var Route = &crud.Resource{
	Name:          "route",
	Label:         "สายจัดจำหน่าย",
	Source:        "dbo.vw_route",
	Table:         "tb_route",
	IDColumn:      "route_id",
	AutoColumn:    "route_auto",
	SearchColumns: []string{"route_name", "route_code"},
	SortColumns: map[string]string{
		"code": "route_code", "name": "route_name", "order": "sort_order", "updated": "update_date",
	},
	DefaultSort: "code",
	Filters: []schema.Filter{
		{Param: "route_type", Column: "route_type", Kind: schema.KindString},
		{Param: "warehouse_id", Column: "warehouse_id", Kind: schema.KindString},
	},
	Refs: []schema.Ref{
		{Field: "warehouse_id", Table: "tb_warehouse", Column: "ref_warehouse_auto", Label: "คลังต้นทาง"},
	},
	Fields: []schema.Field{
		{Name: "route_code", Kind: schema.KindString, Required: true, MaxLen: 20, Label: "รหัสสาย", NoUpdate: true},
		{Name: "route_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อสาย"},
		{Name: "route_type", Kind: schema.KindString, Required: true, Label: "ชนิดสาย",
			EnumValues: []string{"LEGACY_LINE", "REGION", "DAILY"}},
		{Name: "region_name", Kind: schema.KindString, MaxLen: 50, Label: "ภาค"},
		{Name: "sort_order", Kind: schema.KindInt, Label: "ลำดับ"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

var CustomerRoute = &crud.Resource{
	Name:       "customer-route",
	Label:      "การผูกลูกค้ากับสาย",
	Source:     "dbo.vw_customer_route",
	Table:      "tb_customer_route",
	IDColumn:   "customer_route_id",
	AutoColumn: "customer_route_auto",
	// v8 ทำให้ค้นหาได้จริง — v7 ไม่มี SearchColumns เลย ?q= จึงถูกละเลยเงียบ ๆ
	SearchColumns: []string{"customer_name", "route_code", "route_name"},
	SortColumns:   map[string]string{"seq": "delivery_seq", "route": "route_code"},
	DefaultSort:   "route",
	Filters: []schema.Filter{
		{Param: "customer_id", Column: "customer_id", Kind: schema.KindString},
		{Param: "route_code", Column: "route_code", Kind: schema.KindString},
		{Param: "is_primary", Column: "is_primary", Kind: schema.KindBool},
	},
	Refs: []schema.Ref{
		{Field: "customer_id", Table: "tb_customer", Column: "ref_customer_auto",
			Label: "ลูกค้า", Required: true, NoUpdate: true},
		{Field: "route_id", Table: "tb_route", Column: "ref_route_auto",
			Label: "สายจัดจำหน่าย", Required: true, NoUpdate: true},
	},
	Fields: []schema.Field{
		// ลูกค้าหนึ่งรายมีสายหลักได้สายเดียว v7 บังคับด้วย filtered unique index
		// ถ้าตั้งซ้ำจะได้ 409 DUPLICATE จาก error 2601
		{Name: "is_primary", Kind: schema.KindBool, Label: "เป็นสายหลัก"},
		{Name: "delivery_seq", Kind: schema.KindInt, Label: "ลำดับจุดจอด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}
