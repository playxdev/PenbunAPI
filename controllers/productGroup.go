package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllProductGroup(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, product_group_id, product_group_name, description, is_active, update_by, update_date, is_delete FROM tb_product_group WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductGroup
	for rows.Next() {
		var item models.ProductGroup
		if err := rows.Scan(&item.AutoID, &item.ProductGroupID, &item.ProductGroupName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product group list retrieved", items)
}

func SelectPageProductGroup(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, product_group_id, product_group_name, description, is_active, update_by, update_date, is_delete FROM tb_product_group WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductGroup
	for rows.Next() {
		var item models.ProductGroup
		if err := rows.Scan(&item.AutoID, &item.ProductGroupID, &item.ProductGroupName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product group page retrieved", items)
}

func SelectProductGroupByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductGroup
	err := config.DB.QueryRow("SELECT autoID, product_group_id, product_group_name, description, is_active, update_by, update_date, is_delete FROM tb_product_group WHERE product_group_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.ProductGroupID, &item.ProductGroupName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Product group not found")
	}
	return utils.SuccessResponse(c, "Product group found", item)
}

func SelectProductGroupByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, product_group_id, product_group_name, description, is_active, update_by, update_date, is_delete FROM tb_product_group WHERE product_group_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductGroup
	for rows.Next() {
		var item models.ProductGroup
		if err := rows.Scan(&item.AutoID, &item.ProductGroupID, &item.ProductGroupName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product group search results", items)
}

func InsertProductGroup(c *fiber.Ctx) error {
	var item models.ProductGroup
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.ProductGroupName == "" {
		return utils.FailResponse(c, "Group name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertProductGroup", Query: "INSERT INTO tb_product_group (product_group_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.ProductGroupName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product group added successfully", fiber.Map{"product_group_name": item.ProductGroupName})
}

func UpdateProductGroupByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductGroup
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateProductGroup", Query: "UPDATE tb_product_group SET product_group_name = COALESCE(NULLIF(?, ''), product_group_name), description = COALESCE(?, description), update_by = ? WHERE product_group_id = ? AND is_delete = 0",
			Args: []interface{}{item.ProductGroupName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product group updated successfully", fiber.Map{"product_group_id": id})
}

func DeleteProductGroupByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteProductGroup", Query: "UPDATE tb_product_group SET is_delete = 1, update_by = ? WHERE product_group_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product group deleted successfully", nil)
}

func RemoveProductGroupByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveProductGroup", Query: "DELETE FROM tb_product_group WHERE product_group_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product group removed permanently", nil)
}
