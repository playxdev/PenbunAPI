package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllDiscountType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, discount_type_id, discount_type_name, description, is_active, update_by, update_date, is_delete FROM tb_discount_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.DiscountType
	for rows.Next() {
		var item models.DiscountType
		if err := rows.Scan(&item.AutoID, &item.DiscountTypeID, &item.DiscountTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount type list retrieved", items)
}

func SelectPageDiscountType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, discount_type_id, discount_type_name, description, is_active, update_by, update_date, is_delete FROM tb_discount_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.DiscountType
	for rows.Next() {
		var item models.DiscountType
		if err := rows.Scan(&item.AutoID, &item.DiscountTypeID, &item.DiscountTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount type page retrieved", items)
}

func SelectDiscountTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.DiscountType
	err := config.DB.QueryRow("SELECT autoID, discount_type_id, discount_type_name, description, is_active, update_by, update_date, is_delete FROM tb_discount_type WHERE discount_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.DiscountTypeID, &item.DiscountTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Discount type not found")
	}
	return utils.SuccessResponse(c, "Discount type found", item)
}

func SelectDiscountTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, discount_type_id, discount_type_name, description, is_active, update_by, update_date, is_delete FROM tb_discount_type WHERE discount_type_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.DiscountType
	for rows.Next() {
		var item models.DiscountType
		if err := rows.Scan(&item.AutoID, &item.DiscountTypeID, &item.DiscountTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount type search results", items)
}

func InsertDiscountType(c *fiber.Ctx) error {
	var item models.DiscountType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.DiscountTypeName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertDiscountType", Query: "INSERT INTO tb_discount_type (discount_type_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.DiscountTypeName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount type added successfully", fiber.Map{"discount_type_name": item.DiscountTypeName})
}

func UpdateDiscountTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.DiscountType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateDiscountType", Query: "UPDATE tb_discount_type SET discount_type_name = COALESCE(NULLIF(?, ''), discount_type_name), description = COALESCE(?, description), update_by = ? WHERE discount_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.DiscountTypeName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount type updated successfully", fiber.Map{"discount_type_id": id})
}

func DeleteDiscountTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteDiscountType", Query: "UPDATE tb_discount_type SET is_delete = 1, update_by = ? WHERE discount_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount type deleted successfully", nil)
}

func RemoveDiscountTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveDiscountType", Query: "DELETE FROM tb_discount_type WHERE discount_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount type removed permanently", nil)
}
