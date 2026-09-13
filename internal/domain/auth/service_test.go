// Path: internal/domain/auth/service_test.go
package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"penbun/api/internal/platform/authz"
)

// สิทธิ์ที่ออกไปหน้าจอต้องตรงกับแถวที่ฐานคืนมาทุกช่อง ไม่ใช่แค่ "มีสิทธิ์/ไม่มี"
// ถ้าสี่ช่องนี้เพี้ยน หน้าจอจะโชว์ปุ่มที่กดแล้วได้ 403 ซึ่งเป็นปัญหาเดิมที่
// /meta/permissions ถูกสร้างขึ้นมาแก้
func TestToPermissionsKeepsEveryAction(t *testing.T) {
	got := toPermissions(map[string]authz.Access{
		"customer": {View: true},
		"order":    {View: true, Insert: true, Update: true, Delete: true},
	})

	if len(got) != 2 {
		t.Fatalf("ต้องได้ 2 resource ได้ %d", len(got))
	}
	if c := got["customer"]; !c.View || c.Insert || c.Update || c.Delete {
		t.Errorf("customer ต้องอ่านได้อย่างเดียว ได้ %+v", c)
	}
	if o := got["order"]; !o.View || !o.Insert || !o.Update || !o.Delete {
		t.Errorf("order ต้องได้ครบสี่ ได้ %+v", o)
	}
}

// resource ที่ไม่มีแถวสิทธิ์คือ "ไม่มีสิทธิ์" ไม่ใช่ "มีสิทธิ์ทุกอย่าง"
// ฝั่งเว็บอ่านแผนที่นี้ด้วยการถามชื่อ resource ตรง ๆ ค่าที่หายไปจึงต้องเป็นศูนย์ทั้งสี่ช่อง
func TestToPermissionsMissingResourceIsNoAccess(t *testing.T) {
	got := toPermissions(map[string]authz.Access{"customer": {View: true}})
	if a := got["users"]; a.View || a.Insert || a.Update || a.Delete {
		t.Errorf("resource ที่ไม่มีแถวต้องไม่มีสิทธิ์เลย ได้ %+v", a)
	}
}

// ไม่มีสิทธิ์เลยต้องออกเป็น {} ไม่ใช่ null — หน้าจอจะได้ไม่ต้องเช็ค null ก่อนวนลูป
func TestToPermissionsEmptyMarshalsAsObject(t *testing.T) {
	b, err := json.Marshal(map[string]any{"permissions": toPermissions(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"permissions":{}}` {
		t.Errorf("ต้องได้ {} ได้ %s", b)
	}
}

// การแก้โปรไฟล์ต้องปฏิเสธก่อนแตะฐาน ไม่ใช่ปล่อยไปตายตอน UPDATE
// ความยาวที่เกินคอลัมน์เป็น 400 ไม่ใช่ 500 เพราะเป็นคำขอที่ผิด ไม่ใช่ระบบพัง
func TestUpdateProfileRefusesBadInputBeforeTouchingTheDatabase(t *testing.T) {
	// repo เป็น nil โดยตั้งใจ — เคสที่ผ่าน validation จะ panic ซึ่งแปลว่าเทสต์ผิด
	s := &Service{}

	cases := map[string]ProfileUpdate{
		"ชื่อยาวเกินคอลัมน์":  {FullName: strings.Repeat("ก", 151)},
		"อีเมลยาวเกินคอลัมน์": {Email: strings.Repeat("a", 95) + "@x.com"},
		"อีเมลไม่มี @":        {Email: "not-an-email"},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := s.UpdateProfile(t.Context(), "USRA000001", "tester", in); err == nil {
				t.Fatal("ต้องถูกปฏิเสธ")
			}
		})
	}
}
