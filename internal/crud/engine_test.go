// Path: internal/crud/engine_test.go
package crud

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routeSet(app *fiber.App) map[string]bool {
	out := map[string]bool{}
	for _, r := range app.GetRoutes() {
		out[r.Method+" "+r.Path] = true
	}
	return out
}

// handlerCount นับ handler ต่อเส้นทาง ตัวกรองสิทธิ์เป็น handler ตัวหนึ่ง
// เส้นทางที่ถือตัวกรองไว้จึงมีสองตัว เส้นทางที่ไม่มีเหลือตัวเดียว
func handlerCount(app *fiber.App) map[string]int {
	out := map[string]int{}
	for _, r := range app.GetRoutes() {
		out[r.Method+" "+r.Path] = len(r.Handlers)
	}
	return out
}

var readOnly = &Resource{
	Name: "users", Label: "ผู้ใช้งาน",
	Source: "dbo.vw_users", Table: "tb_users",
	IDColumn: "user_id", AutoColumn: "user_auto",
	SortColumns: map[string]string{"username": "user_name"},
	DefaultSort: "username",
	ReadOnly:    true,
}

// ReadOnly ต้องปิดเส้นทางเขียนของ engine จริง ๆ ไม่ใช่แค่ซ่อนจากเอกสาร
// ถ้าหลุดข้อนี้ POST /users จะเขียน user_password จาก body ตรงลงตาราง
func TestReadOnlyMountsOnlyReads(t *testing.T) {
	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil).Mount(app, readOnly))

	routes := routeSet(app)
	assert.True(t, routes["GET /users"])
	assert.True(t, routes["GET /users/:id"])
	assert.False(t, routes["POST /users"])
	assert.False(t, routes["PUT /users/:id"])
	assert.False(t, routes["DELETE /users/:id"])
}

func writable() *Resource {
	r := *readOnly
	r.Name = "widget"
	r.ReadOnly = false
	r.RequireLevelWrite = []string{"ADMIN"}
	return &r
}

func TestWritableMountsFiveEndpoints(t *testing.T) {
	r := writable()

	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil).Mount(app, r))

	routes := routeSet(app)
	for _, want := range []string{
		"GET /widget", "GET /widget/:id",
		"POST /widget", "PUT /widget/:id", "DELETE /widget/:id",
	} {
		assert.True(t, routes[want], "ขาดเส้นทาง %s", want)
	}
}

// ตัวกรองสิทธิ์ต้องอยู่บนเส้นทางเขียนทุกเส้น และต้องไม่อยู่บนเส้นทางอ่าน
//
// ตัวกรองถูกใส่ทีละเส้นทาง ไม่ใช่บนกลุ่ม เพราะกลุ่มครอบทุก method แล้วจะกัน
// การอ่านไปด้วย ราคาของทางเลือกนั้นคือ "ใส่ไม่ครบ" ซึ่งเป็นสิ่งที่เทสต์นี้เฝ้าอยู่
// ถ้าหลุด ผู้ใช้ระดับ USER จะแก้และลบข้อมูลหลักของทั้งระบบได้
func TestWriteRoutesCarryTheLevelGuard(t *testing.T) {
	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil).Mount(app, writable()))

	n := handlerCount(app)
	for _, w := range []string{"POST /widget", "PUT /widget/:id", "DELETE /widget/:id"} {
		assert.Equal(t, 2, n[w], "%s ต้องถือตัวกรอง user_level ไว้", w)
	}
	for _, ro := range []string{"GET /widget", "GET /widget/:id"} {
		assert.Equal(t, 1, n[ro], "%s เป็นการอ่าน ต้องไม่ติดตัวกรอง user_level", ro)
	}
}
