package controllers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"PenbunAPI/middleware"
)

func generateTestToken() string {
	claims := jwt.MapClaims{
		"username":   "tester",
		"user_level": "ADMIN",
		"exp":        time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("test-secret"))
	return tokenStr
}

func setupProtectedRoute(handler fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Use(middleware.JWTMiddleware("test-secret"))
	app.Get("/api/v1/protected/test/all", handler)
	return app
}

func TestProtectedRoute_WithAuth(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.JWTMiddleware("test-secret"))
	app.Get("/api/v1/protected/test/all", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "data": "ok"})
	})

	t.Run("missing auth returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected/test/all", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("valid auth returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/protected/test/all", nil)
		req.Header.Set("Authorization", "Bearer "+generateTestToken())
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestInsertWithInvalidBody(t *testing.T) {
	tests := []struct {
		name    string
		route   string
		handler fiber.Handler
	}{
		{"Book", "/api/v1/protected/book/insert", InsertBook},
		{"Customer", "/api/v1/protected/customer/insert", InsertCustomer},
		{"Vendor", "/api/v1/protected/vendor/insert", InsertVendor},
		{"Discount", "/api/v1/protected/discount/insert", InsertDiscount},
		{"Warehouse", "/api/v1/protected/warehouse/insert", InsertWarehouse},
		{"CustomerType", "/api/v1/protected/customer-type/insert", InsertCustomerType},
		{"VendorType", "/api/v1/protected/vendor-type/insert", InsertVendorType},
		{"BookType", "/api/v1/protected/book-type/insert", InsertBookType},
		{"DiscountType", "/api/v1/protected/discount-type/insert", InsertDiscountType},
		{"UnitType", "/api/v1/protected/unit-type/insert", InsertUnitType},
		{"ProductFormatType", "/api/v1/protected/product-format-type/insert", InsertProductFormatType},
		{"ProductCategory", "/api/v1/protected/product-category/insert", InsertProductCategory},
		{"ProductGroup", "/api/v1/protected/product-group/insert", InsertProductGroup},
		{"Product", "/api/v1/protected/product/insert", InsertProduct},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post(tt.route, middleware.JWTMiddleware("test-secret"), tt.handler)

			body := strings.NewReader(`invalid json`)
			req := httptest.NewRequest("POST", tt.route, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestToken())

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode, "route %s should return 400 for invalid body", tt.route)
		})
	}
}

func TestInsertWithEmptyName(t *testing.T) {
	tests := []struct {
		name    string
		route   string
		handler fiber.Handler
		body    string
	}{
		{"Book", "/api/v1/protected/book/insert", InsertBook, `{"book_name": ""}`},
		{"Customer", "/api/v1/protected/customer/insert", InsertCustomer, `{"customer_name": ""}`},
		{"Vendor", "/api/v1/protected/vendor/insert", InsertVendor, `{"vendor_name": ""}`},
		{"Warehouse", "/api/v1/protected/warehouse/insert", InsertWarehouse, `{"warehouse_name": ""}`},
		{"ProductGroup", "/api/v1/protected/product-group/insert", InsertProductGroup, `{"group_name": ""}`},
		{"ProductCategory", "/api/v1/protected/product-category/insert", InsertProductCategory, `{"category_name": ""}`},
		{"Discount", "/api/v1/protected/discount/insert", InsertDiscount, `{"discount_name": ""}`},
		{"CustomerType", "/api/v1/protected/customer-type/insert", InsertCustomerType, `{"type_name": ""}`},
		{"VendorType", "/api/v1/protected/vendor-type/insert", InsertVendorType, `{"type_name": ""}`},
		{"BookType", "/api/v1/protected/book-type/insert", InsertBookType, `{"type_name": ""}`},
		{"DiscountType", "/api/v1/protected/discount-type/insert", InsertDiscountType, `{"type_name": ""}`},
		{"UnitType", "/api/v1/protected/unit-type/insert", InsertUnitType, `{"type_name": ""}`},
		{"ProductFormatType", "/api/v1/protected/product-format-type/insert", InsertProductFormatType, `{"type_name": ""}`},
		{"Product", "/api/v1/protected/product/insert", InsertProduct, `{"product_name": ""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post(tt.route, middleware.JWTMiddleware("test-secret"), tt.handler)

			body := strings.NewReader(tt.body)
			req := httptest.NewRequest("POST", tt.route, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestToken())

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode, "route %s should return 400 for empty name", tt.route)
		})
	}
}

func TestUpdateEndpoint_WithInvalidBody(t *testing.T) {
	tests := []struct {
		method  string
		route   string
		handler fiber.Handler
	}{
		{"PUT", "/api/v1/protected/book/update/B001", UpdateBookByID},
		{"PUT", "/api/v1/protected/customer/update/C001", UpdateCustomerByID},
		{"PUT", "/api/v1/protected/vendor/update/V001", UpdateVendorByID},
		{"PUT", "/api/v1/protected/discount/update/D001", UpdateDiscountByID},
		{"PUT", "/api/v1/protected/warehouse/update/W001", UpdateWarehouseByID},
		{"PUT", "/api/v1/protected/customer-type/update/CT001", UpdateCustomerTypeByID},
		{"PUT", "/api/v1/protected/vendor-type/update/VT001", UpdateVendorTypeByID},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.route, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.route, middleware.JWTMiddleware("test-secret"), tt.handler)

			body := strings.NewReader(`invalid`)
			req := httptest.NewRequest(tt.method, tt.route, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestToken())

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)
		})
	}
}


