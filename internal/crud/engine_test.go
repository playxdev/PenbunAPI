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

func TestWritableMountsFiveEndpoints(t *testing.T) {
	r := *readOnly
	r.Name = "widget"
	r.ReadOnly = false

	app := fiber.New()
	require.NoError(t, NewEngine(nil, nil).Mount(app, &r))

	routes := routeSet(app)
	for _, want := range []string{
		"GET /widget", "GET /widget/:id",
		"POST /widget", "PUT /widget/:id", "DELETE /widget/:id",
	} {
		assert.True(t, routes[want], "ขาดเส้นทาง %s", want)
	}
}
