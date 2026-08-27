// Path: internal/resources/user.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// User — ผู้ใช้งานของระบบ อ่านอย่างเดียว และเฉพาะ ADMIN
//
// ReadOnly ไม่ได้แปลว่าผู้ใช้แก้ไม่ได้ตลอดไป แต่การสร้างและแก้ผู้ใช้ผ่าน generic
// engine จะเปิดทางให้เขียน user_password กับ user_level ตรง ๆ ซึ่งต้องผ่าน bcrypt
// และต้องมีกติกากันคนลบสิทธิ์ ADMIN คนสุดท้ายทิ้ง งานนั้นเป็นของ domain แยกต่างหาก
// ไม่ใช่ descriptor
//
// vw_users (PenbunSQL v11) ไม่คืน user_password และ counting_password_fail
// SELECT ของ engine เป็น SELECT * จึงไม่มีทางที่ hash จะหลุดออกไปทาง endpoint นี้
var User = &crud.Resource{
	Name:          "user",
	Label:         "ผู้ใช้งาน",
	Source:        "dbo.vw_users",
	Table:         "tb_users",
	IDColumn:      "user_id",
	AutoColumn:    "user_auto",
	SearchColumns: []string{"user_name", "full_name", "email"},
	SortColumns: map[string]string{
		"username": "user_name", "name": "full_name",
		"level": "user_level", "login": "last_login_date", "updated": "update_date",
	},
	DefaultSort: "username",
	Filters: []schema.Filter{
		{Param: "user_level", Column: "user_level", Kind: schema.KindString},
		{Param: "warehouse_id", Column: "warehouse_id", Kind: schema.KindString},
		{Param: "status_user_locked", Column: "status_user_locked", Kind: schema.KindBool},
	},
	ReadOnly:     true,
	RequireLevel: []string{"ADMIN"},
}
