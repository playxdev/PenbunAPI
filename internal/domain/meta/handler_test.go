// Path: internal/domain/meta/handler_test.go
package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"penbun/api/internal/crud"
	"penbun/api/internal/platform/authz"
)

// คำตอบของ /meta/permissions ต้องมาจากสิทธิ์ในฐาน ไม่ใช่จาก descriptor
//
// รูปแบบ read/write ของ endpoint นี้หยาบกว่าสี่การกระทำที่ฐานเก็บ write จึงเป็นจริง
// เมื่อทำได้อย่างน้อยหนึ่งใน insert / update / delete
func TestBuildPermissionsReadsFromTheDatabase(t *testing.T) {
	res := []*crud.Resource{{Name: "customer"}, {Name: "users"}, {Name: "book"}}

	admin := buildPermissions(res, "ADMIN", map[string]authz.Access{
		"customer": {View: true, Insert: true, Update: true, Delete: true},
		"users":    {View: true, Insert: true, Update: true, Delete: true},
		"book":     {View: true, Insert: true, Update: true, Delete: true},
	})
	assert.Equal(t, "ADMIN", admin.Level)
	assert.Equal(t, Access{Read: true, Write: true}, admin.Resources["customer"])
	assert.Equal(t, Access{Read: true, Write: true}, admin.Resources["users"])

	user := buildPermissions(res, "USER", map[string]authz.Access{
		"customer": {View: true},
		"book":     {View: true},
	})
	assert.Equal(t, "USER", user.Level)
	assert.Equal(t, Access{Read: true, Write: false}, user.Resources["customer"],
		"USER ต้องอ่านข้อมูลหลักได้ หน้าจอเอกสารเลือกลูกค้าจากตารางนี้")
	assert.Equal(t, Access{Read: true, Write: false}, user.Resources["book"])
}

// resource ที่ไม่มีแถวสิทธิ์เลยต้องยังปรากฏในคำตอบ เป็น false ทั้งคู่
//
// ถ้าปล่อยให้คีย์หายไป หน้าจอจะแยกไม่ออกระหว่าง "เข้าไม่ได้" กับ "ไม่มี resource นี้"
// แล้วจะไปเดาเอาเองว่าอย่างไหน ซึ่งเดาผิดได้ทั้งสองทาง
func TestBuildPermissionsKeepsResourcesWithNoGrant(t *testing.T) {
	got := buildPermissions([]*crud.Resource{{Name: "users"}}, "USER", map[string]authz.Access{})
	assert.Contains(t, got.Resources, "users")
	assert.Equal(t, Access{Read: false, Write: false}, got.Resources["users"])
}
