package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllCustomerType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, customer_type_id, type_name, base_credit_day, description, is_active, update_by, update_date, is_delete FROM tb_customer_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.CustomerType
	for rows.Next() {
		var item models.CustomerType
		if err := rows.Scan(&item.AutoID, &item.CustomerTypeID, &item.TypeName, &item.BaseCreditDay, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer type list retrieved", items)
}

func SelectPageCustomerType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, customer_type_id, type_name, base_credit_day, description, is_active, update_by, update_date, is_delete FROM tb_customer_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.CustomerType
	for rows.Next() {
		var item models.CustomerType
		if err := rows.Scan(&item.AutoID, &item.CustomerTypeID, &item.TypeName, &item.BaseCreditDay, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer type page retrieved", items)
}

func SelectCustomerTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.CustomerType
	err := config.DB.QueryRow("SELECT autoID, customer_type_id, type_name, base_credit_day, description, is_active, update_by, update_date, is_delete FROM tb_customer_type WHERE customer_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.CustomerTypeID, &item.TypeName, &item.BaseCreditDay, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Customer type not found")
	}
	return utils.SuccessResponse(c, "Customer type found", item)
}

func SelectCustomerTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, customer_type_id, type_name, base_credit_day, description, is_active, update_by, update_date, is_delete FROM tb_customer_type WHERE type_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.CustomerType
	for rows.Next() {
		var item models.CustomerType
		if err := rows.Scan(&item.AutoID, &item.CustomerTypeID, &item.TypeName, &item.BaseCreditDay, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer type search results", items)
}

func InsertCustomerType(c *fiber.Ctx) error {
	var item models.CustomerType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.TypeName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertCustomerType", Query: "INSERT INTO tb_customer_type (type_name, base_credit_day, description, update_by) VALUES (?, ?, ?, ?)",
			Args: []interface{}{item.TypeName, item.BaseCreditDay, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer type added successfully", fiber.Map{"type_name": item.TypeName})
}

func UpdateCustomerTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.CustomerType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateCustomerType", Query: "UPDATE tb_customer_type SET type_name = COALESCE(NULLIF(?, ''), type_name), base_credit_day = COALESCE(?, base_credit_day), description = COALESCE(?, description), update_by = ? WHERE customer_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.TypeName, item.BaseCreditDay, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer type updated successfully", fiber.Map{"customer_type_id": id})
}

func DeleteCustomerTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteCustomerType", Query: "UPDATE tb_customer_type SET is_delete = 1, update_by = ? WHERE customer_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer type deleted successfully", nil)
}

func RemoveCustomerTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveCustomerType", Query: "DELETE FROM tb_customer_type WHERE customer_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer type removed permanently", nil)
}
