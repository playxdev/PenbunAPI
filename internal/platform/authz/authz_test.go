// Path: internal/platform/authz/authz_test.go
package authz

import "testing"

// ค่าเริ่มต้นของการตัดสินสิทธิ์ต้องเป็นปฏิเสธ
//
// Access ที่เป็น zero value คือสิ่งที่ได้จากการอ่านแผนที่ด้วยคีย์ที่ไม่มีอยู่
// ซึ่งเกิดทุกครั้งที่ resource ไม่มีแถวสิทธิ์ หรือมีคนพิมพ์ชื่อ resource ผิดตอน mount
// ถ้า zero value แปลว่า "ผ่าน" ความผิดพลาดเล็ก ๆ พวกนี้จะเปิดเส้นทางให้ทุกคน
func TestZeroAccessAllowsNothing(t *testing.T) {
	var a Access
	for _, act := range []Action{View, Insert, Update, Delete} {
		if a.Allows(act) {
			t.Errorf("Access ว่างต้องไม่อนุญาต %s", act)
		}
	}
}

// Action ที่ไม่รู้จักต้องไม่ผ่าน แม้ผู้ใช้จะมีสิทธิ์ครบทุกช่อง
func TestUnknownActionIsDenied(t *testing.T) {
	full := Access{View: true, Insert: true, Update: true, Delete: true}
	if full.Allows(Action("execute")) {
		t.Error("การกระทำที่ไม่รู้จักต้องถูกปฏิเสธเสมอ")
	}
}

func TestAccessAllowsOnlyWhatIsGranted(t *testing.T) {
	readOnly := Access{View: true}
	if !readOnly.Allows(View) {
		t.Error("view ที่ให้ไว้ต้องผ่าน")
	}
	for _, act := range []Action{Insert, Update, Delete} {
		if readOnly.Allows(act) {
			t.Errorf("%s ไม่ได้ให้ไว้ ต้องไม่ผ่าน", act)
		}
	}
}

// การแปลง method เป็นการกระทำคือจุดที่เส้นทางกับแถวสิทธิ์มาเจอกัน
// ผิดตรงนี้แปลว่าคำขอเขียนถูกตรวจด้วยสิทธิ์อ่าน
func TestActionForMapsMethods(t *testing.T) {
	cases := map[string]Action{
		"GET": View, "HEAD": View,
		"POST":   Insert,
		"PUT":    Update,
		"PATCH":  Update,
		"DELETE": Delete,
	}
	for method, want := range cases {
		got, ok := ActionFor(method)
		if !ok || got != want {
			t.Errorf("%s ต้องได้ %s ได้ %s (ok=%v)", method, want, got, ok)
		}
	}
}

// method ที่ไม่รู้จักต้องคืน false ไม่ใช่เดาเป็น view
// ถ้าเดาเป็น view ทุก method แปลก ๆ จะกลายเป็นช่องอ่านข้อมูลที่ไม่มีใครตั้งใจเปิด
func TestActionForRefusesUnknownMethod(t *testing.T) {
	if _, ok := ActionFor("OPTIONS"); ok {
		t.Error("method ที่ไม่รู้จักต้องไม่ถูกแปลงเป็นการกระทำใด")
	}
}
