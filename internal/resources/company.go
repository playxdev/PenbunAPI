// Path: internal/resources/company.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// TEMP: ยังไม่มี View รองรับ ใช้ derived table ไปก่อน
var Company = &crud.Resource{
	Name:  "company",
	Label: "บริษัท",
	Source: `(SELECT autoID AS company_auto, company_id, company_code, name_th, name_en,
	                 tax_id, branch_code, branch_name, address, province, zip_code,
	                 phone, email, website, ` + audit + `
	            FROM dbo.tb_company WHERE is_delete = 0) AS src`,
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
		{Name: "branch_name", Kind: schema.KindString, MaxLen: 100, Label: "ชื่อสาขา"},
		{Name: "address", Kind: schema.KindString, MaxLen: 255, Label: "ที่อยู่"},
		{Name: "province", Kind: schema.KindString, MaxLen: 100, Label: "จังหวัด"},
		{Name: "zip_code", Kind: schema.KindString, MaxLen: 10, Label: "รหัสไปรษณีย์"},
		{Name: "phone", Kind: schema.KindString, MaxLen: 30, Label: "โทรศัพท์"},
		{Name: "email", Kind: schema.KindString, MaxLen: 150, Label: "อีเมล"},
		{Name: "website", Kind: schema.KindString, MaxLen: 150, Label: "เว็บไซต์"},
	},
}

// TEMP: ยังไม่มี View รองรับ ใช้ derived table ไปก่อน
var Discount = &crud.Resource{
	Name:  "discount",
	Label: "ส่วนลด",
	Source: `(SELECT d.autoID AS discount_auto, d.discount_id, d.discount_code, d.discount_name,
	                 d.discount_value, d.is_percent, d.min_order_amount,
	                 d.start_date, d.end_date, d.description,
	                 dt.discount_type_id, dt.discount_type_name,
	                 d.is_active, d.id_status, d.update_by, d.update_date
	            FROM dbo.tb_discount d
	            INNER JOIN dbo.tb_discount_type dt ON dt.autoID = d.ref_discount_type_auto
	           WHERE d.is_delete = 0) AS src`,
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
		{Name: "description", Kind: schema.KindString, MaxLen: 255, Label: "คำอธิบาย"},
	},
}
