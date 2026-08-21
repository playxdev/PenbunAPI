package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllBookType(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, book_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_book_type WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.BookType
	for rows.Next() {
		var item models.BookType
		if err := rows.Scan(&item.AutoID, &item.BookTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book type list retrieved", items)
}

func SelectPageBookType(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, book_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_book_type WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.BookType
	for rows.Next() {
		var item models.BookType
		if err := rows.Scan(&item.AutoID, &item.BookTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book type page retrieved", items)
}

func SelectBookTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.BookType
	err := config.DB.QueryRow("SELECT autoID, book_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_book_type WHERE book_type_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.BookTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Book type not found")
	}
	return utils.SuccessResponse(c, "Book type found", item)
}

func SelectBookTypeByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, book_type_id, type_name, description, is_active, update_by, update_date, is_delete FROM tb_book_type WHERE type_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.BookType
	for rows.Next() {
		var item models.BookType
		if err := rows.Scan(&item.AutoID, &item.BookTypeID, &item.TypeName, &item.Description, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book type search results", items)
}

func InsertBookType(c *fiber.Ctx) error {
	var item models.BookType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.TypeName == "" {
		return utils.FailResponse(c, "Type name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertBookType", Query: "INSERT INTO tb_book_type (type_name, description, update_by) VALUES (?, ?, ?)",
			Args: []interface{}{item.TypeName, item.Description, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book type added successfully", fiber.Map{"type_name": item.TypeName})
}

func UpdateBookTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.BookType
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateBookType", Query: "UPDATE tb_book_type SET type_name = COALESCE(NULLIF(?, ''), type_name), description = COALESCE(?, description), update_by = ? WHERE book_type_id = ? AND is_delete = 0",
			Args: []interface{}{item.TypeName, item.Description, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book type updated successfully", fiber.Map{"book_type_id": id})
}

func DeleteBookTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteBookType", Query: "UPDATE tb_book_type SET is_delete = 1, update_by = ? WHERE book_type_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book type deleted successfully", nil)
}

func RemoveBookTypeByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveBookType", Query: "DELETE FROM tb_book_type WHERE book_type_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book type removed permanently", nil)
}
