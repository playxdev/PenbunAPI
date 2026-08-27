// Path: internal/resources/user.go
package resources

import (
	"penbun/api/internal/crud"
	"penbun/api/internal/schema"
)

// User — ผู้ใช้งานของระบบ อ่านอย่างเดียว และเฉพาะ ADMIN
//
// ReadOnly ปิดเฉพาะเส้นทางเขียนของ engine กลาง การสร้างผู้ใช้อยู่ที่
// domain/user เพราะรหัสผ่านต้องผ่าน bcrypt และ user_level ตัดสินสิทธิ์ของทั้ง API
// สองเรื่องนี้เขียนผ่าน descriptor ไม่ได้
//
// vw_users (PenbunSQL v11) ไม่คืน user_password และ counting_password_fail
// SELECT ของ engine เป็น SELECT * จึงไม่มีทางที่ hash จะหลุดออกไปทาง endpoint นี้
//
// Name เป็นพหูพจน์ ต่างจาก resource อื่นทั้งหมด เพราะ PUT /users/{id}/unlock
// มีอยู่ก่อนแล้วตั้งแต่ v4.0.0 การตั้งเป็น "user" จะได้ระบบที่มีทั้ง /user และ
// /users อยู่พร้อมกัน ซึ่งไม่มีใครจำถูก
var User = &crud.Resource{
	Name:          "users",
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
