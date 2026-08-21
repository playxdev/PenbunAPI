package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllVendorType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, vendor_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_vendor_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.VendorType
	for rows.Next() {
		var item models.VendorType
		if err := rows.Scan(&item.AutoID, &item.VendorTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor type list retrieved", items)
}

func SelectPageVendorType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, vendor_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_vendor_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.VendorType
	for rows.Next() {
		var item models.VendorType
		if err := rows.Scan(&item.AutoID, &item.VendorTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor type page retrieved", items)
}

func SelectVendorTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.VendorType
	err := config.DB.QueryRow("SELECT autoID, vendor_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_vendor_type WHERE vendor_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.VendorTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Vendor type not found")
	}
	return utils.SuccessResponse(c, "Vendor type found", item)
}

func SelectVendorTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, vendor_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_vendor_type WHERE type_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.VendorType
	for rows.Next() {
		var item models.VendorType
		if err := rows.Scan(&item.AutoID, &item.VendorTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor type search results", items)
}

func InsertVendorType(c *fiber.Ctx) error {
	var item models.VendorType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.TypeName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertVendorType", Query: "INSERT INTO tb_vendor_type (type_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.TypeName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor type added successfully", fiber.Map{"type_name": item.TypeName})
}

func UpdateVendorTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.VendorType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateVendorType", Query: "UPDATE tb_vendor_type SET type_name = COALESCE(NULLIF(?, ''), type_name), description = COALESCE(?, description), update_by = ? WHERE vendor_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.TypeName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor type updated successfully", fiber.Map{"vendor_type_id": id})
}

func DeleteVendorTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteVendorType", Query: "UPDATE tb_vendor_type SET is_delete = 1, update_by = ? WHERE vendor_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor type deleted successfully", nil)
}

func RemoveVendorTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveVendorType", Query: "DELETE FROM tb_vendor_type WHERE vendor_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor type removed permanently", nil)
}
