package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBook_JSON(t *testing.T) {
	author := "Test Author"
	price := 29.99
	book := Book{
		AutoID:   1,
		BookID:   "B001",
		BookName: "Test Book",
		Author:   &author,
		Price:    &price,
		IsActive: true,
		UpdateBy: "admin",
	}

	data, err := json.Marshal(book)
	assert.NoError(t, err)

	var decoded Book
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "Test Book", decoded.BookName)
	assert.Equal(t, "Test Author", *decoded.Author)
	assert.Equal(t, 29.99, *decoded.Price)
}

func TestBook_OMitempty(t *testing.T) {
	book := Book{
		AutoID:   1,
		BookName: "Test",
	}

	data, err := json.Marshal(book)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)
	_, hasAuthor := decoded["author"]
	assert.False(t, hasAuthor)
	_, hasPrice := decoded["price"]
	assert.False(t, hasPrice)
}

func TestCustomer_JSON(t *testing.T) {
	addr := "123 Main St"
	phone := "555-1234"
	credit := 10000.0
	customer := Customer{
		AutoID:       1,
		CustomerID:   "C001",
		CustomerName: "Test Customer",
		Address:      &addr,
		Phone1:       &phone,
		CreditLimit:  &credit,
		IsActive:     true,
		IDStatus:     "ACTIVE",
	}

	data, err := json.Marshal(customer)
	assert.NoError(t, err)

	var decoded Customer
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Test Customer", decoded.CustomerName)
	assert.Equal(t, "123 Main St", *decoded.Address)
}

func TestVendor_JSON(t *testing.T) {
	vendor := Vendor{
		AutoID:     1,
		VendorID:   "V001",
		VendorName: "Test Vendor",
		IsActive:   true,
		IDStatus:   "ACTIVE",
	}

	data, err := json.Marshal(vendor)
	assert.NoError(t, err)

	var decoded Vendor
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Test Vendor", decoded.VendorName)
}

func TestDiscount_JSON(t *testing.T) {
	isPercent := true
	value := 10.0
	discount := Discount{
		AutoID:        1,
		DiscountID:    "D001",
		DiscountName:  "10% Off",
		IsPercent:     &isPercent,
		DiscountValue: &value,
		IsActive:      true,
		IDStatus:      "ACTIVE",
	}

	data, err := json.Marshal(discount)
	assert.NoError(t, err)

	var decoded Discount
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "10% Off", decoded.DiscountName)
	assert.True(t, *decoded.IsPercent)
}

func TestWarehouse_JSON(t *testing.T) {
	loc := "Building A"
	warehouse := Warehouse{
		AutoID:        1,
		WarehouseID:   "W001",
		WarehouseName: "Main Warehouse",
		Location:      &loc,
		IsActive:      true,
	}

	data, err := json.Marshal(warehouse)
	assert.NoError(t, err)

	var decoded Warehouse
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Main Warehouse", decoded.WarehouseName)
	assert.Equal(t, "Building A", *decoded.Location)
}

func TestProduct_JSON(t *testing.T) {
	product := Product{
		AutoID:         1,
		ProductCode:    "PC001",
		ProductName:    "Test Product",
		ProductGroupID: "PG001",
		CountStock:     true,
		IsActive:       true,
		IDStatus:       "ACTIVE",
	}

	data, err := json.Marshal(product)
	assert.NoError(t, err)

	var decoded Product
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Test Product", decoded.ProductName)
	assert.Equal(t, "PC001", decoded.ProductCode)
}

func TestProduct_OMitempty(t *testing.T) {
	product := Product{
		AutoID:         1,
		ProductName:    "Test",
		ProductCode:    "PC001",
		ProductGroupID: "PG001",
	}

	data, err := json.Marshal(product)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)
	_, hasProductID := decoded["product_id"]
	assert.False(t, hasProductID)
	_, hasBarcode := decoded["barcode"]
	assert.False(t, hasBarcode)
}

func TestCustomerType_JSON(t *testing.T) {
	creditDay := 30
	desc := "Standard customer"
	ct := CustomerType{
		AutoID:        1,
		TypeName:      "Standard",
		BaseCreditDay: &creditDay,
		Description:   &desc,
		IsActive:      true,
	}

	data, err := json.Marshal(ct)
	assert.NoError(t, err)

	var decoded CustomerType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Standard", decoded.TypeName)
	assert.Equal(t, 30, *decoded.BaseCreditDay)
}

func TestVendorType_JSON(t *testing.T) {
	desc := "Local vendor"
	vt := VendorType{
		AutoID:      1,
		TypeName:    "Local",
		Description: &desc,
		IsActive:    true,
	}

	data, err := json.Marshal(vt)
	assert.NoError(t, err)

	var decoded VendorType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Local", decoded.TypeName)
}

func TestBookType_JSON(t *testing.T) {
	desc := "Fiction books"
	bt := BookType{
		AutoID:      1,
		TypeName:    "Fiction",
		Description: &desc,
		IsActive:    true,
	}

	data, err := json.Marshal(bt)
	assert.NoError(t, err)

	var decoded BookType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Fiction", decoded.TypeName)
}

func TestDiscountType_JSON(t *testing.T) {
	desc := "Seasonal discount"
	dt := DiscountType{
		AutoID:           1,
		DiscountTypeName: "Seasonal",
		Description:      &desc,
		IsActive:         true,
	}

	data, err := json.Marshal(dt)
	assert.NoError(t, err)

	var decoded DiscountType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Seasonal", decoded.DiscountTypeName)
}

func TestUnitType_JSON(t *testing.T) {
	desc := "Pieces"
	ut := UnitType{
		AutoID:       1,
		UnitTypeName: "PCS",
		Description:  &desc,
		IsActive:     true,
	}

	data, err := json.Marshal(ut)
	assert.NoError(t, err)

	var decoded UnitType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "PCS", decoded.UnitTypeName)
}

func TestProductCategory_JSON(t *testing.T) {
	desc := "Electronics"
	pc := ProductCategory{
		AutoID:       1,
		CategoryName: "Electronics",
		Description:  &desc,
		IsActive:     true,
	}

	data, err := json.Marshal(pc)
	assert.NoError(t, err)

	var decoded ProductCategory
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Electronics", decoded.CategoryName)
}

func TestProductFormatType_JSON(t *testing.T) {
	desc := "Hardcover"
	pft := ProductFormatType{
		AutoID:      1,
		FormatName:  "Hardcover",
		Description: &desc,
		IsActive:    true,
	}

	data, err := json.Marshal(pft)
	assert.NoError(t, err)

	var decoded ProductFormatType
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Hardcover", decoded.FormatName)
}

func TestProductGroup_JSON(t *testing.T) {
	desc := "Books group"
	pg := ProductGroup{
		AutoID:           1,
		ProductGroupName: "Books",
		Description:      &desc,
		IsActive:         true,
	}

	data, err := json.Marshal(pg)
	assert.NoError(t, err)

	var decoded ProductGroup
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "Books", decoded.ProductGroupName)
}

func TestApiResponse_OmitsEmptyFields(t *testing.T) {
	resp := ApiResponse{
		Status:  "success",
		Message: "ok",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)
	_, hasData := decoded["data"]
	assert.False(t, hasData)
	_, hasToken := decoded["token"]
	assert.False(t, hasToken)
	_, hasError := decoded["error"]
	assert.False(t, hasError)
}

func TestLoginRequest_JSON(t *testing.T) {
	req := LoginRequest{
		Username: "admin",
		Password: "secret",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded LoginRequest
	json.Unmarshal(data, &decoded)
	assert.Equal(t, "admin", decoded.Username)
	assert.Equal(t, "secret", decoded.Password)
}
