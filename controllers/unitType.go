package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllUnitType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, unit_type_id, unit_type_name, description, is_active, update_by, update_date, is_delete FROM tb_unit_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.UnitType
	for rows.Next() {
		var item models.UnitType
		if err := rows.Scan(&item.AutoID, &item.UnitTypeID, &item.UnitTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Unit type list retrieved", items)
}

func SelectPageUnitType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, unit_type_id, unit_type_name, description, is_active, update_by, update_date, is_delete FROM tb_unit_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.UnitType
	for rows.Next() {
		var item models.UnitType
		if err := rows.Scan(&item.AutoID, &item.UnitTypeID, &item.UnitTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Unit type page retrieved", items)
}

func SelectUnitTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.UnitType
	err := config.DB.QueryRow("SELECT autoID, unit_type_id, unit_type_name, description, is_active, update_by, update_date, is_delete FROM tb_unit_type WHERE unit_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.UnitTypeID, &item.UnitTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Unit type not found")
	}
	return utils.SuccessResponse(c, "Unit type found", item)
}

func SelectUnitTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, unit_type_id, unit_type_name, description, is_active, update_by, update_date, is_delete FROM tb_unit_type WHERE unit_type_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.UnitType
	for rows.Next() {
		var item models.UnitType
		if err := rows.Scan(&item.AutoID, &item.UnitTypeID, &item.UnitTypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Unit type search results", items)
}

func InsertUnitType(c *fiber.Ctx) error {
	var item models.UnitType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.UnitTypeName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertUnitType", Query: "INSERT INTO tb_unit_type (unit_type_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.UnitTypeName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Unit type added successfully", fiber.Map{"unit_type_name": item.UnitTypeName})
}

func UpdateUnitTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.UnitType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateUnitType", Query: "UPDATE tb_unit_type SET unit_type_name = COALESCE(NULLIF(?, ''), unit_type_name), description = COALESCE(?, description), update_by = ? WHERE unit_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.UnitTypeName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Unit type updated successfully", fiber.Map{"unit_type_id": id})
}

func DeleteUnitTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteUnitType", Query: "UPDATE tb_unit_type SET is_delete = 1, update_by = ? WHERE unit_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Unit type deleted successfully", nil)
}

func RemoveUnitTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveUnitType", Query: "DELETE FROM tb_unit_type WHERE unit_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Unit type removed permanently", nil)
}
