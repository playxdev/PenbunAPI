//go:build integration

package integration

import (
	"testing"

	"penbun/api/internal/domain/document"
	"penbun/api/internal/resources"
)

// domainOnly คือ resource ที่ไม่มี descriptor แต่มีเส้นทางของตัวเอง
// ชื่อพวกนี้ถูกส่งให้ authz.GuardResource ตรง ๆ ใน handler ของแต่ละ domain
var domainOnly = []string{"stock", "consign", "allocation"}

// TestEveryMountedResourceHasPrivilegeRows เทียบชื่อ resource ที่ API ติดตั้งเส้นทางให้
// กับชื่อที่มีแถวอยู่จริงใน tb_privilege
//
// ชื่อทั้งสองฝั่งเป็นข้อความ ไม่มีอะไรผูกให้ตรงกันตอนคอมไพล์ เพิ่ม descriptor ใหม่
// แล้วลืม seed สิทธิ์ = เส้นทางนั้นตอบ 403 ให้ทุกคนรวมทั้ง ADMIN โดยไม่มีอะไรเตือน
// ส่วนแถวสิทธิ์ที่ไม่มีเส้นทางรองรับคือขยะที่ทำให้หน้าจอสิทธิ์โชว์ของที่กดไม่ได้
//
// ต้องรันกับฐานจริงเพราะสิ่งที่จับคือความต่างระหว่างโค้ดกับฐาน ไม่ใช่ตรรกะในโค้ด
func TestEveryMountedResourceHasPrivilegeRows(t *testing.T) {
	db := openTestDB(t)

	seeded := map[string]bool{}
	rows, err := db.QueryContext(t.Context(),
		`SELECT DISTINCT resource_code FROM dbo.tb_privilege WHERE is_delete = 0`)
	if err != nil {
		t.Fatalf("read tb_privilege: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan resource_code: %v", err)
		}
		seeded[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tb_privilege: %v", err)
	}

	mounted := map[string]bool{}
	for _, r := range resources.All() {
		mounted[r.Name] = true
	}
	for _, s := range document.All() {
		mounted[s.Name] = true
	}
	for _, n := range domainOnly {
		mounted[n] = true
	}

	for name := range mounted {
		if !seeded[name] {
			t.Errorf("resource %q มีเส้นทางแต่ไม่มีแถวใน tb_privilege — ทุกคนจะได้ 403", name)
		}
	}
	for name := range seeded {
		if !mounted[name] {
			t.Errorf("tb_privilege มีแถวของ %q แต่ไม่มีเส้นทางไหนใช้ชื่อนี้", name)
		}
	}
}
