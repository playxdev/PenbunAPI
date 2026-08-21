package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllVendor(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, vendor_id, vendor_name, address, phone1, phone2, is_active, id_status, update_by, update_date, is_delete FROM tb_vendor WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Vendor
	for rows.Next() {
		var item models.Vendor
		if err := rows.Scan(&item.AutoID, &item.VendorID, &item.VendorName, &item.Address, &item.Phone1, &item.Phone2, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor list retrieved", items)
}

func SelectPageVendor(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, vendor_id, vendor_name, address, phone1, phone2, is_active, id_status, update_by, update_date, is_delete FROM tb_vendor WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Vendor
	for rows.Next() {
		var item models.Vendor
		if err := rows.Scan(&item.AutoID, &item.VendorID, &item.VendorName, &item.Address, &item.Phone1, &item.Phone2, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor page retrieved", items)
}

func SelectVendorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Vendor
	err := config.DB.QueryRow("SELECT autoID, vendor_id, vendor_name, address, phone1, phone2, is_active, id_status, update_by, update_date, is_delete FROM tb_vendor WHERE vendor_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.VendorID, &item.VendorName, &item.Address, &item.Phone1, &item.Phone2, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Vendor not found")
	}
	return utils.SuccessResponse(c, "Vendor found", item)
}

func SelectVendorByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, vendor_id, vendor_name, address, phone1, phone2, is_active, id_status, update_by, update_date, is_delete FROM tb_vendor WHERE vendor_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Vendor
	for rows.Next() {
		var item models.Vendor
		if err := rows.Scan(&item.AutoID, &item.VendorID, &item.VendorName, &item.Address, &item.Phone1, &item.Phone2, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Vendor search results", items)
}

func InsertVendor(c *fiber.Ctx) error {
	var item models.Vendor
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.VendorName == "" {
		return utils.FailResponse(c, "Vendor name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertVendor", Query: "INSERT INTO tb_vendor (vendor_name, address, phone1, phone2, id_status, update_by) VALUES (?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'ACTIVE'), ?)",
			Args: []interface{}{item.VendorName, item.Address, item.Phone1, item.Phone2, item.IDStatus, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor added successfully", fiber.Map{"vendor_name": item.VendorName})
}

func UpdateVendorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Vendor
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateVendor", Query: "UPDATE tb_vendor SET vendor_name = COALESCE(NULLIF(?, ''), vendor_name), address = COALESCE(?, address), phone1 = COALESCE(?, phone1), phone2 = COALESCE(?, phone2), id_status = COALESCE(NULLIF(?, ''), id_status), update_by = ? WHERE vendor_id = ? AND is_delete = 0",
			Args: []interface{}{item.VendorName, item.Address, item.Phone1, item.Phone2, item.IDStatus, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor updated successfully", fiber.Map{"vendor_id": id})
}

func DeleteVendorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteVendor", Query: "UPDATE tb_vendor SET is_delete = 1, update_by = ? WHERE vendor_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor deleted successfully", nil)
}

func RemoveVendorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveVendor", Query: "DELETE FROM tb_vendor WHERE vendor_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Vendor removed permanently", nil)
}
