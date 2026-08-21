package controllers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func generateBusinessID(prefix string, autoID int64) string {
	seriesSize := int64(999999)
	seriesIndex := ((autoID - 1) / seriesSize) % 26
	seriesChar := string(rune('A' + seriesIndex))
	runningNum := ((autoID - 1) % seriesSize) + 1
	return fmt.Sprintf("%s%s%06d", prefix, seriesChar, runningNum)
}

func SelectAllProduct(c *fiber.Ctx) error {
	rows, err := config.DB.Query(`SELECT autoID, product_id, product_code, product_name, product_group_id, product_format_type_id, unit_type_id, vendor_id, count_stock, cost_price, sell_price, barcode, weight_kg, description, update_by, update_date, is_active, id_status, is_delete FROM tb_product WHERE is_delete = 0`)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Product
	for rows.Next() {
		var item models.Product
		if err := rows.Scan(&item.AutoID, &item.ProductID, &item.ProductCode, &item.ProductName, &item.ProductGroupID, &item.ProductFormatTypeID, &item.UnitTypeID, &item.VendorID, &item.CountStock, &item.CostPrice, &item.SellPrice, &item.Barcode, &item.WeightKg, &item.Description, &item.UpdateBy, &item.UpdateDate, &item.IsActive, &item.IDStatus, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product list retrieved", items)
}

func SelectPageProduct(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	rows, err := config.DB.Query(`SELECT autoID, product_id, product_code, product_name, product_group_id, product_format_type_id, unit_type_id, vendor_id, count_stock, cost_price, sell_price, barcode, weight_kg, description, update_by, update_date, is_active, id_status, is_delete FROM tb_product WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`, offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Product
	for rows.Next() {
		var item models.Product
		if err := rows.Scan(&item.AutoID, &item.ProductID, &item.ProductCode, &item.ProductName, &item.ProductGroupID, &item.ProductFormatTypeID, &item.UnitTypeID, &item.VendorID, &item.CountStock, &item.CostPrice, &item.SellPrice, &item.Barcode, &item.WeightKg, &item.Description, &item.UpdateBy, &item.UpdateDate, &item.IsActive, &item.IDStatus, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product page retrieved", items)
}

func SelectProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Product
	err := config.DB.QueryRow(`SELECT autoID, product_id, product_code, product_name, product_group_id, product_format_type_id, unit_type_id, vendor_id, count_stock, cost_price, sell_price, barcode, weight_kg, description, update_by, update_date, is_active, id_status, is_delete FROM tb_product WHERE product_id = ? AND is_delete = 0`, id).
		Scan(&item.AutoID, &item.ProductID, &item.ProductCode, &item.ProductName, &item.ProductGroupID, &item.ProductFormatTypeID, &item.UnitTypeID, &item.VendorID, &item.CountStock, &item.CostPrice, &item.SellPrice, &item.Barcode, &item.WeightKg, &item.Description, &item.UpdateBy, &item.UpdateDate, &item.IsActive, &item.IDStatus, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Product not found")
	}
	return utils.SuccessResponse(c, "Product found", item)
}

func SelectProductByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query(`SELECT autoID, product_id, product_code, product_name, product_group_id, product_format_type_id, unit_type_id, vendor_id, count_stock, cost_price, sell_price, barcode, weight_kg, description, update_by, update_date, is_active, id_status, is_delete FROM tb_product WHERE product_name LIKE '%' + ? + '%' AND is_delete = 0`, name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Product
	for rows.Next() {
		var item models.Product
		if err := rows.Scan(&item.AutoID, &item.ProductID, &item.ProductCode, &item.ProductName, &item.ProductGroupID, &item.ProductFormatTypeID, &item.UnitTypeID, &item.VendorID, &item.CountStock, &item.CostPrice, &item.SellPrice, &item.Barcode, &item.WeightKg, &item.Description, &item.UpdateBy, &item.UpdateDate, &item.IsActive, &item.IDStatus, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product search results", items)
}

func InsertProduct(c *fiber.Ctx) error {
	var item models.Product
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.ProductName == "" {
		return utils.FailResponse(c, "Product name is required")
	}
	if item.ProductCode == "" {
		return utils.FailResponse(c, "Product code is required")
	}
	if item.ProductGroupID == "" {
		return utils.FailResponse(c, "Product group ID is required")
	}

	tx, err := config.DB.Begin()
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	username := getUsername(c)

	var autoID int64
	err = tx.QueryRow(`INSERT INTO tb_product (product_code, product_name, product_group_id, product_format_type_id, unit_type_id, vendor_id, count_stock, cost_price, sell_price, barcode, weight_kg, description, id_status, update_by) OUTPUT INSERTED.autoID VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'ACTIVE'), ?)`,
		item.ProductCode, item.ProductName, item.ProductGroupID, item.ProductFormatTypeID, item.UnitTypeID, item.VendorID, item.CountStock, item.CostPrice, item.SellPrice, item.Barcode, item.WeightKg, item.Description, item.IDStatus, username).Scan(&autoID)
	if err != nil {
		tx.Rollback()
		return utils.ErrorResponse(c, err.Error())
	}

	productID := generateBusinessID("PDT", autoID)

	_, err = tx.Exec(`UPDATE tb_product SET product_id = ? WHERE autoID = ?`, productID, autoID)
	if err != nil {
		tx.Rollback()
		return utils.ErrorResponse(c, err.Error())
	}

	if err := tx.Commit(); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}

	return utils.SuccessResponse(c, "Product added successfully", fiber.Map{"product_id": productID, "product_name": item.ProductName})
}

func UpdateProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Product
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	username := getUsername(c)

	steps := []utils.TransactionStep{
		{Name: "UpdateProduct", Query: `UPDATE tb_product SET product_code = COALESCE(NULLIF(?, ''), product_code), product_name = COALESCE(NULLIF(?, ''), product_name), product_group_id = COALESCE(NULLIF(?, ''), product_group_id), product_format_type_id = COALESCE(?, product_format_type_id), unit_type_id = COALESCE(?, unit_type_id), vendor_id = COALESCE(?, vendor_id), count_stock = COALESCE(?, count_stock), cost_price = COALESCE(?, cost_price), sell_price = COALESCE(?, sell_price), barcode = COALESCE(?, barcode), weight_kg = COALESCE(?, weight_kg), description = COALESCE(?, description), id_status = COALESCE(NULLIF(?, ''), id_status), update_by = ? WHERE product_id = ? AND is_delete = 0`,
			Args: []interface{}{item.ProductCode, item.ProductName, item.ProductGroupID, item.ProductFormatTypeID, item.UnitTypeID, item.VendorID, item.CountStock, item.CostPrice, item.SellPrice, item.Barcode, item.WeightKg, item.Description, item.IDStatus, username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product updated successfully", fiber.Map{"product_id": id})
}

func DeleteProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := getUsername(c)
	steps := []utils.TransactionStep{
		{Name: "DeleteProduct", Query: "UPDATE tb_product SET is_delete = 1, update_by = ? WHERE product_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product deleted successfully", nil)
}

func RemoveProductByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveProduct", Query: "DELETE FROM tb_product WHERE product_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product removed permanently", nil)
}
