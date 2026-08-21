package routes

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestV2Route(t *testing.T) {
	app := fiber.New()
	SetupV2Routes(app)

	req := httptest.NewRequest("GET", "/api/v2/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, "PenbunAPI v2 endpoints coming soon", result["message"])
}

func TestPublicRoutes_Exist(t *testing.T) {
	app := fiber.New()
	SetupPublicRoutes(app, "test-secret")

	req := httptest.NewRequest("POST", "/api/v1/public/login", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 404, resp.StatusCode)
}

func TestProtectedRoutes_RequireAuth(t *testing.T) {
	app := fiber.New()
	SetupV1Routes(app, "test-secret")

	routes := []string{
		"/api/v1/protected/book/all",
		"/api/v1/protected/book/page",
		"/api/v1/protected/book/select/id/B001",
		"/api/v1/protected/customer/all",
		"/api/v1/protected/customer/page",
		"/api/v1/protected/customer/select/id/C001",
		"/api/v1/protected/vendor/all",
		"/api/v1/protected/vendor/page",
		"/api/v1/protected/vendor/select/id/V001",
		"/api/v1/protected/discount/all",
		"/api/v1/protected/discount/page",
		"/api/v1/protected/product-group/all",
		"/api/v1/protected/product-group/page",
		"/api/v1/protected/warehouse/all",
		"/api/v1/protected/warehouse/page",
		"/api/v1/protected/unit-type/all",
		"/api/v1/protected/unit-type/page",
		"/api/v1/protected/product-category/all",
		"/api/v1/protected/product-category/page",
		"/api/v1/protected/customer-type/all",
		"/api/v1/protected/customer-type/page",
		"/api/v1/protected/vendor-type/all",
		"/api/v1/protected/vendor-type/page",
		"/api/v1/protected/discount-type/all",
		"/api/v1/protected/discount-type/page",
		"/api/v1/protected/book-type/all",
		"/api/v1/protected/book-type/page",
		"/api/v1/protected/product-format-type/all",
		"/api/v1/protected/product-format-type/page",
		"/api/v1/protected/product/all",
		"/api/v1/protected/product/page",
		"/api/v1/protected/product/select/id/PDTA000001",
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 401, resp.StatusCode, "route %s should require auth", route)
		})
	}
}

func TestProtectedWriteRoutes_RequireAuth(t *testing.T) {
	app := fiber.New()
	SetupV1Routes(app, "test-secret")

	type writeRoute struct {
		method string
		path   string
	}

	routes := []writeRoute{
		{"POST", "/api/v1/protected/book/insert"},
		{"PUT", "/api/v1/protected/book/update/B001"},
		{"PUT", "/api/v1/protected/book/delete/B001"},
		{"DELETE", "/api/v1/protected/book/remove/B001"},
		{"POST", "/api/v1/protected/customer/insert"},
		{"PUT", "/api/v1/protected/customer/update/C001"},
		{"PUT", "/api/v1/protected/customer/delete/C001"},
		{"DELETE", "/api/v1/protected/customer/remove/C001"},
		{"POST", "/api/v1/protected/vendor/insert"},
		{"PUT", "/api/v1/protected/vendor/update/V001"},
		{"PUT", "/api/v1/protected/vendor/delete/V001"},
		{"DELETE", "/api/v1/protected/vendor/remove/V001"},
		{"POST", "/api/v1/protected/discount/insert"},
		{"PUT", "/api/v1/protected/discount/update/D001"},
		{"PUT", "/api/v1/protected/discount/delete/D001"},
		{"DELETE", "/api/v1/protected/discount/remove/D001"},
		{"POST", "/api/v1/protected/product/insert"},
		{"PUT", "/api/v1/protected/product/update/PDTA000001"},
		{"PUT", "/api/v1/protected/product/delete/PDTA000001"},
		{"DELETE", "/api/v1/protected/product/remove/PDTA000001"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			req := httptest.NewRequest(r.method, r.path, nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, 401, resp.StatusCode, "route %s %s should require auth", r.method, r.path)
		})
	}
}

func TestJWTMiddlewareOnRoutes(t *testing.T) {
	app := fiber.New()
	SetupV1Routes(app, "test-secret")

	req := httptest.NewRequest("GET", "/api/v1/protected/book/all", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}
