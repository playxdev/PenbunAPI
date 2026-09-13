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
	require.NoError(t, NewEngine(nil, nil, nil).Mount(app, readOnly))

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
	return &r
}

func TestWritableMountsFiveEndpoints(t *testing.T) {
	r := writable()

	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil, nil).Mount(app, r))

	routes := routeSet(app)
	for _, want := range []string{
		"GET /widget", "GET /widget/:id",
		"POST /widget", "PUT /widget/:id", "DELETE /widget/:id",
	} {
		assert.True(t, routes[want], "ขาดเส้นทาง %s", want)
	}
}

// ตัวกรองสิทธิ์ต้องครอบทุกเส้นทางของ resource ทั้งอ่านและเขียน
//
// ตัวกรองอยู่บนกลุ่มได้แล้วเพราะ authz.GuardResource อ่านการกระทำจาก HTTP method
// เอง ไม่ต้องแยกอ่านกับเขียนเหมือน mw.RequireLevel ที่รู้แค่ระดับผู้ใช้
// ราคาของกลุ่มคือถ้าลืมใส่ ทั้ง resource หลุดพร้อมกัน ไม่ใช่หลุดทีละเส้น
// เทสต์นี้จึงนับ handler ของทุกเส้นทาง ไม่ใช่เฉพาะเส้นทางเขียน
func TestEveryRouteCarriesThePrivilegeGuard(t *testing.T) {
	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil, nil).Mount(app, writable()))

	n := handlerCount(app)
	for _, want := range []string{
		"GET /widget", "GET /widget/:id",
		"POST /widget", "PUT /widget/:id", "DELETE /widget/:id",
	} {
		assert.Equal(t, 2, n[want], "%s ต้องถือตัวกรองสิทธิ์ไว้", want)
	}
}

// ReadOnly ก็ต้องถูกกรอง — การอ่านของ users คือสิ่งที่ต้องกันเป็นอันดับแรก
func TestReadOnlyRoutesAreGuardedToo(t *testing.T) {
	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil, nil).Mount(app, readOnly))

	n := handlerCount(app)
	assert.Equal(t, 2, n["GET /users"])
	assert.Equal(t, 2, n["GET /users/:id"])
}
