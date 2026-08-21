package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllDiscount(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, discount_id, discount_name, is_percent, discount_value, min_order_amount, start_date, end_date, is_active, id_status, update_by, update_date, is_delete FROM tb_discount WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Discount
	for rows.Next() {
		var item models.Discount
		if err := rows.Scan(&item.AutoID, &item.DiscountID, &item.DiscountName, &item.IsPercent, &item.DiscountValue, &item.MinOrderAmount, &item.StartDate, &item.EndDate, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount list retrieved", items)
}

func SelectPageDiscount(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, discount_id, discount_name, is_percent, discount_value, min_order_amount, start_date, end_date, is_active, id_status, update_by, update_date, is_delete FROM tb_discount WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Discount
	for rows.Next() {
		var item models.Discount
		if err := rows.Scan(&item.AutoID, &item.DiscountID, &item.DiscountName, &item.IsPercent, &item.DiscountValue, &item.MinOrderAmount, &item.StartDate, &item.EndDate, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount page retrieved", items)
}

func SelectDiscountByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Discount
	err := config.DB.QueryRow("SELECT autoID, discount_id, discount_name, is_percent, discount_value, min_order_amount, start_date, end_date, is_active, id_status, update_by, update_date, is_delete FROM tb_discount WHERE discount_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.DiscountID, &item.DiscountName, &item.IsPercent, &item.DiscountValue, &item.MinOrderAmount, &item.StartDate, &item.EndDate, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Discount not found")
	}
	return utils.SuccessResponse(c, "Discount found", item)
}

func SelectDiscountByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, discount_id, discount_name, is_percent, discount_value, min_order_amount, start_date, end_date, is_active, id_status, update_by, update_date, is_delete FROM tb_discount WHERE discount_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Discount
	for rows.Next() {
		var item models.Discount
		if err := rows.Scan(&item.AutoID, &item.DiscountID, &item.DiscountName, &item.IsPercent, &item.DiscountValue, &item.MinOrderAmount, &item.StartDate, &item.EndDate, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Discount search results", items)
}

func InsertDiscount(c *fiber.Ctx) error {
	var item models.Discount
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.DiscountName == "" {
		return utils.FailResponse(c, "Discount name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertDiscount", Query: "INSERT INTO tb_discount (discount_name, is_percent, discount_value, min_order_amount, start_date, end_date, id_status, update_by) VALUES (?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'ACTIVE'), ?)",
			Args: []interface{}{item.DiscountName, item.IsPercent, item.DiscountValue, item.MinOrderAmount, item.StartDate, item.EndDate, item.IDStatus, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount added successfully", fiber.Map{"discount_name": item.DiscountName})
}

func UpdateDiscountByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Discount
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateDiscount", Query: "UPDATE tb_discount SET discount_name = COALESCE(NULLIF(?, ''), discount_name), is_percent = COALESCE(?, is_percent), discount_value = COALESCE(?, discount_value), min_order_amount = COALESCE(?, min_order_amount), start_date = COALESCE(?, start_date), end_date = COALESCE(?, end_date), id_status = COALESCE(NULLIF(?, ''), id_status), update_by = ? WHERE discount_id = ? AND is_delete = 0",
			Args: []interface{}{item.DiscountName, item.IsPercent, item.DiscountValue, item.MinOrderAmount, item.StartDate, item.EndDate, item.IDStatus, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount updated successfully", fiber.Map{"discount_id": id})
}

func DeleteDiscountByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteDiscount", Query: "UPDATE tb_discount SET is_delete = 1, update_by = ? WHERE discount_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount deleted successfully", nil)
}

func RemoveDiscountByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveDiscount", Query: "DELETE FROM tb_discount WHERE discount_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Discount removed permanently", nil)
}
