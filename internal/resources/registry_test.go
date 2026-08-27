// Path: internal/resources/registry_test.go
package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDescriptorsAreValid(t *testing.T) {
	require.NoError(t, Validate())
}

func TestResourceNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range All() {
		assert.False(t, seen[r.Name], "ชื่อ resource ซ้ำ: %s", r.Name)
		seen[r.Name] = true
	}
}

// ทุก Source ต้องกรองแถวที่ถูกลบออกไปแล้ว
// ถ้าหลุดข้อนี้ ข้อมูลที่ผู้ใช้ลบไปแล้วจะกลับมาโผล่ในหน้าจอ
func TestSourcesExcludeDeletedRows(t *testing.T) {
	for _, r := range All() {
		t.Run(r.Name, func(t *testing.T) {
			if strings.HasPrefix(r.Source, "dbo.vw_") {
				return // View กรองให้แล้วในนิยามของตัวมันเอง
			}
			assert.Contains(t, r.Source, "is_delete = 0",
				"derived table ของ %s ต้องกรองแถวที่ถูกลบออก", r.Name)
		})
	}
}

// AutoColumn ต้องเป็นคอลัมน์ที่ Source คืนออกมาจริง
// ถ้าตั้งชื่อไม่ตรง การอ่านกลับหลังสร้างจะล้มทันทีที่มีคนกดบันทึก
func TestAutoColumnAppearsInSource(t *testing.T) {
	for _, r := range All() {
		t.Run(r.Name, func(t *testing.T) {
			if strings.HasPrefix(r.Source, "dbo.vw_") {
				return // ตรวจได้เฉพาะกับฐานข้อมูลจริง
			}
			assert.Contains(t, r.Source, r.AutoColumn,
				"Source ของ %s ต้องคืนคอลัมน์ %s", r.Name, r.AutoColumn)
		})
	}
}

func TestWritableResourcesDeclareRequiredFields(t *testing.T) {
	for _, r := range All() {
		if r.ReadOnly {
			continue
		}
		t.Run(r.Name, func(t *testing.T) {
			hasRequired := false
			for _, f := range r.Fields {
				if f.Required {
					hasRequired = true
					break
				}
			}
			for _, ref := range r.Refs {
				if ref.Required {
					hasRequired = true
					break
				}
			}
			assert.True(t, hasRequired,
				"%s เขียนได้แต่ไม่มีฟิลด์บังคับเลย ตรวจว่าตั้ง descriptor ครบหรือยัง", r.Name)
		})
	}
}

// หนังสือผูกกับสินค้าแบบหนึ่งต่อหนึ่ง การสร้างต้องเขียนสองตาราง
// จึงต้องปิดการเขียนผ่าน engine กลาง ไม่งั้นจะได้แถวหนังสือที่ไม่มีสินค้ารองรับ
func TestBookIsReadOnlyInGenericEngine(t *testing.T) {
	assert.True(t, Book.ReadOnly)
}

// ผู้ใช้งานเป็น resource เดียวที่คืนข้อมูลของคนอื่น
// ถ้าวันหนึ่งมีคนถอด RequireLevel ออก รายชื่อผู้ใช้ทั้งระบบจะเปิดให้ทุกคนที่ login
// อ่านได้ทันที เทสต์นี้จึงล็อกทั้งสองอย่างไว้พร้อมกัน
func TestUserResourceIsAdminOnlyAndReadOnly(t *testing.T) {
	assert.Equal(t, []string{"ADMIN"}, User.RequireLevel)
	assert.True(t, User.ReadOnly, "การสร้างและแก้ผู้ใช้ต้องไม่ผ่าน generic engine")
	assert.Equal(t, "dbo.vw_users", User.Source,
		"ต้องอ่านผ่าน View ที่ไม่คืน user_password")
}
