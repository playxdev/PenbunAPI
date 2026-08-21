package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllProductCategory(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, product_category_id, category_name, description, is_active, update_by, update_date, is_delete FROM tb_product_category WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductCategory
	for rows.Next() {
		var item models.ProductCategory
		if err := rows.Scan(&item.AutoID, &item.ProductCategoryID, &item.CategoryName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product category list retrieved", items)
}

func SelectPageProductCategory(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, product_category_id, category_name, description, is_active, update_by, update_date, is_delete FROM tb_product_category WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductCategory
	for rows.Next() {
		var item models.ProductCategory
		if err := rows.Scan(&item.AutoID, &item.ProductCategoryID, &item.CategoryName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product category page retrieved", items)
}

func SelectProductCategoryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductCategory
	err := config.DB.QueryRow("SELECT autoID, product_category_id, category_name, description, is_active, update_by, update_date, is_delete FROM tb_product_category WHERE product_category_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.ProductCategoryID, &item.CategoryName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Product category not found")
	}
	return utils.SuccessResponse(c, "Product category found", item)
}

func SelectProductCategoryByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, product_category_id, category_name, description, is_active, update_by, update_date, is_delete FROM tb_product_category WHERE category_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.ProductCategory
	for rows.Next() {
		var item models.ProductCategory
		if err := rows.Scan(&item.AutoID, &item.ProductCategoryID, &item.CategoryName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Product category search results", items)
}

func InsertProductCategory(c *fiber.Ctx) error {
	var item models.ProductCategory
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.CategoryName == "" {
		return utils.FailResponse(c, "Category name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertProductCategory", Query: "INSERT INTO tb_product_category (category_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.CategoryName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product category added successfully", fiber.Map{"category_name": item.CategoryName})
}

func UpdateProductCategoryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ProductCategory
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateProductCategory", Query: "UPDATE tb_product_category SET category_name = COALESCE(NULLIF(?, ''), category_name), description = COALESCE(?, description), update_by = ? WHERE product_category_id = ? AND is_delete = 0",
			Args: []interface{}{item.CategoryName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product category updated successfully", fiber.Map{"product_category_id": id})
}

func DeleteProductCategoryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteProductCategory", Query: "UPDATE tb_product_category SET is_delete = 1, update_by = ? WHERE product_category_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product category deleted successfully", nil)
}

func RemoveProductCategoryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveProductCategory", Query: "DELETE FROM tb_product_category WHERE product_category_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Product category removed permanently", nil)
}
