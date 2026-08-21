package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllProductFormatType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, product_format_type_id, format_name, description, is_active, update_by, update_date, is_delete FROM tb_product_format_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductFormatType
	for rows.Next() {
		var item models.ProductFormatType
		if err := rows.Scan(&item.AutoID, &item.ProductFormatTypeID, &item.FormatName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product format type list retrieved", items)
}

func SelectPageProductFormatType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, product_format_type_id, format_name, description, is_active, update_by, update_date, is_delete FROM tb_product_format_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductFormatType
	for rows.Next() {
		var item models.ProductFormatType
		if err := rows.Scan(&item.AutoID, &item.ProductFormatTypeID, &item.FormatName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product format type page retrieved", items)
}

func SelectProductFormatTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductFormatType
	err := config.DB.QueryRow("SELECT autoID, product_format_type_id, format_name, description, is_active, update_by, update_date, is_delete FROM tb_product_format_type WHERE product_format_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.ProductFormatTypeID, &item.FormatName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Product format type not found")
	}
	return utils.SuccessResponse(c, "Product format type found", item)
}

func SelectProductFormatTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, product_format_type_id, format_name, description, is_active, update_by, update_date, is_delete FROM tb_product_format_type WHERE format_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductFormatType
	for rows.Next() {
		var item models.ProductFormatType
		if err := rows.Scan(&item.AutoID, &item.ProductFormatTypeID, &item.FormatName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product format type search results", items)
}

func InsertProductFormatType(c *fiber.Ctx) error {
	var item models.ProductFormatType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.FormatName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertProductFormatType", Query: "INSERT INTO tb_product_format_type (format_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.FormatName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product format type added successfully", fiber.Map{"format_name": item.FormatName})
}

func UpdateProductFormatTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductFormatType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateProductFormatType", Query: "UPDATE tb_product_format_type SET format_name = COALESCE(NULLIF(?, ''), format_name), description = COALESCE(?, description), update_by = ? WHERE product_format_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.FormatName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product format type updated successfully", fiber.Map{"product_format_type_id": id})
}

func DeleteProductFormatTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteProductFormatType", Query: "UPDATE tb_product_format_type SET is_delete = 1, update_by = ? WHERE product_format_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product format type deleted successfully", nil)
}

func RemoveProductFormatTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveProductFormatType", Query: "DELETE FROM tb_product_format_type WHERE product_format_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product format type removed permanently", nil)
}
