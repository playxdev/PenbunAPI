package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllWarehouse(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, warehouse_id, warehouse_name, location, is_active, update_by, update_date, is_delete FROM tb_warehouse WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Warehouse
	for rows.Next() {
		var item models.Warehouse
		if err := rows.Scan(&item.AutoID, &item.WarehouseID, &item.WarehouseName, &item.Location, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Warehouse list retrieved", items)
}

func SelectPageWarehouse(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, warehouse_id, warehouse_name, location, is_active, update_by, update_date, is_delete FROM tb_warehouse WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Warehouse
	for rows.Next() {
		var item models.Warehouse
		if err := rows.Scan(&item.AutoID, &item.WarehouseID, &item.WarehouseName, &item.Location, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Warehouse page retrieved", items)
}

func SelectWarehouseByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Warehouse
	err := config.DB.QueryRow("SELECT autoID, warehouse_id, warehouse_name, location, is_active, update_by, update_date, is_delete FROM tb_warehouse WHERE warehouse_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.WarehouseID, &item.WarehouseName, &item.Location, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Warehouse not found")
	}
	return utils.SuccessResponse(c, "Warehouse found", item)
}

func SelectWarehouseByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, warehouse_id, warehouse_name, location, is_active, update_by, update_date, is_delete FROM tb_warehouse WHERE warehouse_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Warehouse
	for rows.Next() {
		var item models.Warehouse
		if err := rows.Scan(&item.AutoID, &item.WarehouseID, &item.WarehouseName, &item.Location, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Warehouse search results", items)
}

func InsertWarehouse(c *fiber.Ctx) error {
	var item models.Warehouse
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.WarehouseName == "" {
		return utils.FailResponse(c, "Warehouse name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertWarehouse", Query: "INSERT INTO tb_warehouse (warehouse_name, location, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.WarehouseName, item.Location, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Warehouse added successfully", fiber.Map{"warehouse_name": item.WarehouseName})
}

func UpdateWarehouseByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Warehouse
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateWarehouse", Query: "UPDATE tb_warehouse SET warehouse_name = COALESCE(NULLIF(?, ''), warehouse_name), location = COALESCE(?, location), update_by = ? WHERE warehouse_id = ? AND is_delete = 0",
			Args: []interface{}{item.WarehouseName, item.Location, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Warehouse updated successfully", fiber.Map{"warehouse_id": id})
}

func DeleteWarehouseByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteWarehouse", Query: "UPDATE tb_warehouse SET is_delete = 1, update_by = ? WHERE warehouse_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Warehouse deleted successfully", nil)
}

func RemoveWarehouseByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveWarehouse", Query: "DELETE FROM tb_warehouse WHERE warehouse_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Warehouse removed permanently", nil)
}
