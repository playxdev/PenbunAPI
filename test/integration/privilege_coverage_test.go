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

// TestEveryRoleGrantsSomethingAndEveryLevelHasARole ผูกสามอย่างเข้าด้วยกัน:
// บทบาทในฐาน · สิทธิ์ของบทบาทนั้น · และ user_level ที่บัญชีจริงถืออยู่
//
// ตั้งแต่ POST /users เลิกเก็บรายการ user_level ไว้ใน Go แล้วไปอ่าน tb_role แทน
// บทบาทที่ไม่มีสิทธิ์สักแถวจะกลายเป็นบัญชีที่สร้างได้แต่เปิดอะไรไม่ได้เลย
// และ user_level ที่ไม่มีบทบาทรองรับจะทำให้บัญชีเดิมล็อกอินได้แต่ไม่มีสิทธิ์
func TestEveryRoleGrantsSomethingAndEveryLevelHasARole(t *testing.T) {
	db := openTestDB(t)

	rows, err := db.QueryContext(t.Context(), `
SELECT r.role_code, COUNT(p.autoID)
  FROM dbo.tb_role r
  LEFT JOIN dbo.tb_privilege p ON p.ref_role_auto = r.autoID AND p.is_delete = 0
 WHERE r.is_delete = 0 AND r.is_active = 1
 GROUP BY r.role_code`)
	if err != nil {
		t.Fatalf("read roles: %v", err)
	}
	defer rows.Close()

	roles := map[string]int{}
	for rows.Next() {
		var code string
		var n int
		if err := rows.Scan(&code, &n); err != nil {
			t.Fatalf("scan role: %v", err)
		}
		roles[code] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate roles: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("tb_role ว่าง — ฐานยังไม่ได้ seed")
	}

	for code, n := range roles {
		if n == 0 {
			t.Errorf("บทบาท %q ไม่มีสิทธิ์สักแถว — สร้างบัญชีได้แต่เปิดอะไรไม่ได้เลย", code)
		}
	}

	levels, err := db.QueryContext(t.Context(),
		`SELECT DISTINCT user_level FROM dbo.tb_users WHERE is_delete = 0`)
	if err != nil {
		t.Fatalf("read user levels: %v", err)
	}
	defer levels.Close()
	for levels.Next() {
		var lvl string
		if err := levels.Scan(&lvl); err != nil {
			t.Fatalf("scan user_level: %v", err)
		}
		if _, ok := roles[lvl]; !ok {
			t.Errorf("มีบัญชีที่ user_level = %q แต่ไม่มีบทบาทชื่อนี้ใน tb_role", lvl)
		}
	}
	if err := levels.Err(); err != nil {
		t.Fatalf("iterate user levels: %v", err)
	}
}
