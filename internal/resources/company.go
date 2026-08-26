// Path: internal/resources/company.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

var Company = &crud.Resource{
	Name:          "company",
	Label:         "บริษัท",
	Source:        "dbo.vw_company",
	Table:         "tb_company",
	IDColumn:      "company_id",
	AutoColumn:    "company_auto",
	SearchColumns: []string{"name_th", "name_en", "company_code", "tax_id"},
	SortColumns: map[string]string{
		"code": "company_code", "name": "name_th", "updated": "update_date",
	},
	DefaultSort: "code",
	Fields: []schema.Field{
		{Name: "company_code", Kind: schema.KindString, Required: true, MaxLen: 20,
			Label: "รหัสบริษัท", NoUpdate: true},
		{Name: "name_th", Kind: schema.KindString, Required: true, MaxLen: 200, Label: "ชื่อภาษาไทย"},
		{Name: "name_en", Kind: schema.KindString, MaxLen: 200, Label: "ชื่อภาษาอังกฤษ"},
		{Name: "tax_id", Kind: schema.KindString, MaxLen: 20, Label: "เลขประจำตัวผู้เสียภาษี"},
		{Name: "branch_code", Kind: schema.KindString, MaxLen: 10, Label: "รหัสสาขา"},
		{Name: "address", Kind: schema.KindString, Label: "ที่อยู่"},
		{Name: "province", Kind: schema.KindString, MaxLen: 100, Label: "จังหวัด"},
		{Name: "zip_code", Kind: schema.KindString, MaxLen: 20, Label: "รหัสไปรษณีย์"},
		{Name: "phone", Kind: schema.KindString, MaxLen: 50, Label: "โทรศัพท์"},
		{Name: "email", Kind: schema.KindString, MaxLen: 100, Label: "อีเมล"},
		{Name: "website", Kind: schema.KindString, MaxLen: 100, Label: "เว็บไซต์"},
	},
}

var Discount = &crud.Resource{
	Name:          "discount",
	Label:         "ส่วนลด",
	Source:        "dbo.vw_discount",
	Table:         "tb_discount",
	IDColumn:      "discount_id",
	AutoColumn:    "discount_auto",
	SearchColumns: []string{"discount_name", "discount_code"},
	SortColumns: map[string]string{
		"code": "discount_code", "name": "discount_name", "updated": "update_date",
	},
	DefaultSort: "updated",
	Filters: []schema.Filter{
		{Param: "discount_type_id", Column: "discount_type_id", Kind: schema.KindString},
		{Param: "is_percent", Column: "is_percent", Kind: schema.KindBool},
	},
	Refs: []schema.Ref{
		{Field: "discount_type_id", Table: "tb_discount_type",
			Column: "ref_discount_type_auto", Label: "ประเภทส่วนลด", Required: true},
	},
	Fields: []schema.Field{
		{Name: "discount_code", Kind: schema.KindString, Required: true, MaxLen: 20,
			Label: "รหัสส่วนลด", NoUpdate: true},
		{Name: "discount_name", Kind: schema.KindString, Required: true, MaxLen: 150, Label: "ชื่อส่วนลด"},
		{Name: "discount_value", Kind: schema.KindDecimal, Label: "มูลค่าส่วนลด",
			Min: schema.Float64(0)},
		{Name: "is_percent", Kind: schema.KindBool, Label: "คิดเป็นเปอร์เซ็นต์"},
		{Name: "min_order_amount", Kind: schema.KindDecimal, Label: "ยอดสั่งซื้อขั้นต่ำ",
			Min: schema.Float64(0)},
		{Name: "start_date", Kind: schema.KindDate, Label: "วันที่เริ่มใช้"},
		{Name: "end_date", Kind: schema.KindDate, Label: "วันที่สิ้นสุด"},
		{Name: "description", Kind: schema.KindString, MaxLen: 500, Label: "คำอธิบาย"},
	},
}
