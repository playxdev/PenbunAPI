// Path: internal/domain/meta/handler_test.go
package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"penbun/api/internal/crud"
)

// รายการว่างของ mw.RequireLevel แปลว่าไม่จำกัด ที่นี่ต้องตีความเหมือนกัน
// ถ้าต่างกัน หน้าจอจะซ่อนสิ่งที่ API ยอมให้ทำ หรือโชว์สิ่งที่ API ปฏิเสธ
func TestAllowsMatchesRequireLevel(t *testing.T) {
	assert.True(t, allows(nil, "USER"))
	assert.True(t, allows([]string{}, "USER"))
	assert.True(t, allows([]string{"ADMIN"}, "ADMIN"))
	assert.False(t, allows([]string{"ADMIN"}, "USER"))
	assert.True(t, allows([]string{"ADMIN", "USER"}, "USER"))
	assert.False(t, allows([]string{"ADMIN"}, ""))
}

func TestBuildPermissions(t *testing.T) {
	res := []*crud.Resource{
		// ข้อมูลหลักทั่วไป — ทุกคนอ่านได้ ADMIN เขียนได้
		{Name: "customer", RequireLevelWrite: []string{"ADMIN"}},
		// ผู้ใช้งาน — ADMIN เท่านั้นที่อ่านได้ และเขียนผ่าน engine กลางไม่ได้เลย
		{Name: "users", ReadOnly: true, RequireLevel: []string{"ADMIN"}},
		// หนังสือ — engine กลางอ่านอย่างเดียว แต่การเขียนมีจริงที่ domain/book
		{Name: "book", ReadOnly: true, RequireLevelWrite: []string{"ADMIN"}},
	}

	admin := buildPermissions(res, "ADMIN")
	assert.Equal(t, "ADMIN", admin.Level)
	assert.Equal(t, Access{Read: true, Write: true}, admin.Resources["customer"])
	assert.Equal(t, Access{Read: true, Write: false}, admin.Resources["users"])
	assert.Equal(t, Access{Read: true, Write: true}, admin.Resources["book"])

	user := buildPermissions(res, "USER")
	assert.Equal(t, "USER", user.Level)
	assert.Equal(t, Access{Read: true, Write: false}, user.Resources["customer"],
		"USER ต้องอ่านข้อมูลหลักได้ หน้าจอเอกสารเลือกลูกค้าจากตารางนี้")
	assert.Equal(t, Access{Read: false, Write: false}, user.Resources["users"])
	assert.Equal(t, Access{Read: true, Write: false}, user.Resources["book"])
}
