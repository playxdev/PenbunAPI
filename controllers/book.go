package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllBook(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, book_id, book_name, author, price, is_active, update_by, update_date, is_delete FROM tb_book WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Book
	for rows.Next() {
		var item models.Book
		if err := rows.Scan(&item.AutoID, &item.BookID, &item.BookName, &item.Author, &item.Price, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book list retrieved", items)
}

func SelectPageBook(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, book_id, book_name, author, price, is_active, update_by, update_date, is_delete FROM tb_book WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Book
	for rows.Next() {
		var item models.Book
		if err := rows.Scan(&item.AutoID, &item.BookID, &item.BookName, &item.Author, &item.Price, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book page retrieved", items)
}

func SelectBookByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Book
	err := config.DB.QueryRow("SELECT autoID, book_id, book_name, author, price, is_active, update_by, update_date, is_delete FROM tb_book WHERE book_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.BookID, &item.BookName, &item.Author, &item.Price, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Book not found")
	}
	return utils.SuccessResponse(c, "Book found", item)
}

func SelectBookByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, book_id, book_name, author, price, is_active, update_by, update_date, is_delete FROM tb_book WHERE book_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Book
	for rows.Next() {
		var item models.Book
		if err := rows.Scan(&item.AutoID, &item.BookID, &item.BookName, &item.Author, &item.Price, &item.IsActive, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Book search results", items)
}

func InsertBook(c *fiber.Ctx) error {
	var item models.Book
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.BookName == "" {
		return utils.FailResponse(c, "Book name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertBook", Query: "INSERT INTO tb_book (book_name, author, price, update_by) VALUES (?, ?, ?, ?)",
			Args: []interface{}{item.BookName, item.Author, item.Price, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book added successfully", fiber.Map{"book_name": item.BookName})
}

func UpdateBookByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Book
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateBook", Query: "UPDATE tb_book SET book_name = COALESCE(NULLIF(?, ''), book_name), author = COALESCE(?, author), price = COALESCE(?, price), update_by = ? WHERE book_id = ? AND is_delete = 0",
			Args: []interface{}{item.BookName, item.Author, item.Price, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book updated successfully", fiber.Map{"book_id": id})
}

func DeleteBookByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteBook", Query: "UPDATE tb_book SET is_delete = 1, update_by = ? WHERE book_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book deleted successfully", nil)
}

func RemoveBookByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveBook", Query: "DELETE FROM tb_book WHERE book_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Book removed permanently", nil)
}
