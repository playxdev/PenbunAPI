// Path: internal/resources/discount.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// กลุ่มส่วนลดและกฎส่วนลด — สองตารางที่ PenbunSQL v9 เพิ่มเข้ามา
//
// tb_discount ที่มีอยู่เดิมคือแคมเปญแบน ๆ ไม่มีมิติลูกค้าและสินค้า จึงตอบไม่ได้ว่า
// ร้านหนึ่งจ่ายเท่าไหร่สำหรับหนังสือเล่มหนึ่ง (PENBUN-TODO §8.2) tb_price_rule
// ตอบคำถามนั้นด้วยกฎที่แยกตาม rule_scope และ UFN_RESOLVE_DISCOUNT เป็นตัวรวมชั้น
//
// descriptor สองตัวนี้ให้แค่ CRUD ของกฎ การคิดส่วนลดตอนออกเอกสารอยู่ที่ฐานข้อมูล
// ไม่ใช่ที่นี่ — ตัวเลขที่ทุกใบใช้ต้องมาจากที่เดียว

var DiscountGroup = &crud.Resource{
	Name:          "discount-group",
	Label:         "กลุ่มส่วนลด",
	Source:        "dbo.vw_discount_group",
	Table:         "tb_discount_group",
	IDColumn:      "discount_group_id",
	AutoColumn:    "discount_group_auto",
	SearchColumns: []string{"group_name", "group_code"},
	SortColumns: map[string]string{
		"code": "group_code", "name": "group_name", "updated": "update_date",
	},
	DefaultSort: "code",
	Fields: []schema.Field{
		{Name: "group_code", Kind: schema.KindString, Required: true, MaxLen: 20,
			Label: "รหัสกลุ่ม", NoUpdate: true},
		{Name: "group_name", Kind: schema.KindString, Required: true, MaxLen: 100, Label: "ชื่อกลุ่ม"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}

// ruleScopes ต้องตรงกับ CK_tb_price_rule_scope ใน PenbunSQL v9
// ตรวจที่นี่ด้วยเพื่อให้ error อ่านรู้เรื่องกว่าข้อความ CHECK constraint
var ruleScopes = []string{"CUSTOMER_SKU", "CUSTOMER", "ROUTE", "GROUP_SKU", "GROUP", "SKU"}

var PriceRule = &crud.Resource{
	Name:          "price-rule",
	Label:         "กฎส่วนลด",
	Source:        "dbo.vw_price_rule",
	Table:         "tb_price_rule",
	IDColumn:      "price_rule_id",
	AutoColumn:    "price_rule_auto",
	SearchColumns: []string{"rule_name", "rule_code"},
	SortColumns: map[string]string{
		"code": "rule_code", "name": "rule_name", "scope": "rule_scope",
		"start": "start_date", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "rule_scope", Column: "rule_scope", Kind: schema.KindString},
		{Param: "discount_group_id", Column: "discount_group_id", Kind: schema.KindString},
		{Param: "customer_id", Column: "customer_id", Kind: schema.KindString},
		{Param: "route_id", Column: "route_id", Kind: schema.KindString},
		{Param: "sku_id", Column: "sku_id", Kind: schema.KindString},
		{Param: "is_on_top", Column: "is_on_top", Kind: schema.KindBool},
	},
	// ปลายทางทั้งสี่เป็น optional ทุกตัว : CK_tb_price_rule_target เป็นตัวบังคับว่า
	// scope ไหนต้องมีปลายทางอะไร กฎเดียวกันเขียนสองที่แล้วมีวันไม่ตรงกัน
	Refs: []schema.Ref{
		{Field: "discount_group_id", Table: "tb_discount_group",
			Column: "ref_discount_group_auto", Label: "กลุ่มส่วนลด"},
		{Field: "customer_id", Table: "tb_customer",
			Column: "ref_customer_auto", Label: "ลูกค้า"},
		{Field: "route_id", Table: "tb_route",
			Column: "ref_route_auto", Label: "สาย"},
		{Field: "sku_id", Table: "tb_product_sku",
			Column: "ref_sku_auto", Label: "SKU"},
	},
	Fields: []schema.Field{
		{Name: "rule_code", Kind: schema.KindString, MaxLen: 20, Label: "รหัสกฎ", NoUpdate: true},
		{Name: "rule_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อกฎ"},
		{Name: "rule_scope", Kind: schema.KindString, Required: true, MaxLen: 20,
			Label: "ขอบเขต", EnumValues: ruleScopes},
		{Name: "discount_percent", Kind: schema.KindDecimal, Label: "ส่วนลด (%)",
			Min: schema.Float64(0)},
		{Name: "net_price", Kind: schema.KindDecimal, Label: "ราคาสุทธิต่อหน่วย",
			Min: schema.Float64(0)},
		{Name: "min_qty", Kind: schema.KindInt, Label: "จำนวนขั้นต่ำ", Min: schema.Float64(0)},
		{Name: "max_qty", Kind: schema.KindInt, Label: "จำนวนสูงสุด", Min: schema.Float64(0)},
		{Name: "is_on_top", Kind: schema.KindBool, Label: "บวกทับส่วนลดชั้นอื่น"},
		{Name: "priority", Kind: schema.KindInt, Label: "ลำดับซ้อน", Min: schema.Float64(0)},
		{Name: "start_date", Kind: schema.KindDate, Label: "วันที่เริ่มใช้"},
		{Name: "end_date", Kind: schema.KindDate, Label: "วันที่สิ้นสุด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}
