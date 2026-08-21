package controllers

import (
	"github.com/gofiber/fiber/v2"

	"PenbunAPI/config"
	"PenbunAPI/models"
	"PenbunAPI/utils"
)

func SelectAllCustomer(c *fiber.Ctx) error {
	rows, err := config.DB.Query("SELECT autoID, customer_id, customer_name, address, phone1, phone2, tax_id, credit_limit, is_active, id_status, update_by, update_date, is_delete FROM tb_customer WHERE is_delete = 0")
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Customer
	for rows.Next() {
		var item models.Customer
		if err := rows.Scan(&item.AutoID, &item.CustomerID, &item.CustomerName, &item.Address, &item.Phone1, &item.Phone2, &item.TaxID, &item.CreditLimit, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer list retrieved", items)
}

func SelectPageCustomer(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	offset := (page - 1) * limit

	rows, err := config.DB.Query("SELECT autoID, customer_id, customer_name, address, phone1, phone2, tax_id, credit_limit, is_active, id_status, update_by, update_date, is_delete FROM tb_customer WHERE is_delete = 0 ORDER BY update_date DESC OFFSET ? ROWS FETCH NEXT ? ROWS ONLY", offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Customer
	for rows.Next() {
		var item models.Customer
		if err := rows.Scan(&item.AutoID, &item.CustomerID, &item.CustomerName, &item.Address, &item.Phone1, &item.Phone2, &item.TaxID, &item.CreditLimit, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer page retrieved", items)
}

func SelectCustomerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Customer
	err := config.DB.QueryRow("SELECT autoID, customer_id, customer_name, address, phone1, phone2, tax_id, credit_limit, is_active, id_status, update_by, update_date, is_delete FROM tb_customer WHERE customer_id = ? AND is_delete = 0", id).
		Scan(&item.AutoID, &item.CustomerID, &item.CustomerName, &item.Address, &item.Phone1, &item.Phone2, &item.TaxID, &item.CreditLimit, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete)
	if err != nil {
		return utils.FailResponse(c, "Customer not found")
	}
	return utils.SuccessResponse(c, "Customer found", item)
}

func SelectCustomerByName(c *fiber.Ctx) error {
	name := c.Params("name")
	rows, err := config.DB.Query("SELECT autoID, customer_id, customer_name, address, phone1, phone2, tax_id, credit_limit, is_active, id_status, update_by, update_date, is_delete FROM tb_customer WHERE customer_name LIKE '%' + ? + '%' AND is_delete = 0", name)
	if err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	defer rows.Close()

	var items []models.Customer
	for rows.Next() {
		var item models.Customer
		if err := rows.Scan(&item.AutoID, &item.CustomerID, &item.CustomerName, &item.Address, &item.Phone1, &item.Phone2, &item.TaxID, &item.CreditLimit, &item.IsActive, &item.IDStatus, &item.UpdateBy, &item.UpdateDate, &item.IsDelete); err != nil {
			return utils.ErrorResponse(c, err.Error())
		}
		items = append(items, item)
	}
	return utils.SuccessResponse(c, "Customer search results", items)
}

func InsertCustomer(c *fiber.Ctx) error {
	var item models.Customer
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}
	if item.CustomerName == "" {
		return utils.FailResponse(c, "Customer name is required")
	}

	steps := []utils.TransactionStep{
		{Name: "InsertCustomer", Query: "INSERT INTO tb_customer (customer_name, address, phone1, phone2, tax_id, credit_limit, id_status, update_by) VALUES (?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'ACTIVE'), ?)",
			Args: []interface{}{item.CustomerName, item.Address, item.Phone1, item.Phone2, item.TaxID, item.CreditLimit, item.IDStatus, item.UpdateBy}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer added successfully", fiber.Map{"customer_name": item.CustomerName})
}

func UpdateCustomerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.Customer
	if err := c.BodyParser(&item); err != nil {
		return utils.FailResponse(c, "Invalid request body")
	}

	steps := []utils.TransactionStep{
		{Name: "UpdateCustomer", Query: "UPDATE tb_customer SET customer_name = COALESCE(NULLIF(?, ''), customer_name), address = COALESCE(?, address), phone1 = COALESCE(?, phone1), phone2 = COALESCE(?, phone2), tax_id = COALESCE(?, tax_id), credit_limit = COALESCE(?, credit_limit), id_status = COALESCE(NULLIF(?, ''), id_status), update_by = ? WHERE customer_id = ? AND is_delete = 0",
			Args: []interface{}{item.CustomerName, item.Address, item.Phone1, item.Phone2, item.TaxID, item.CreditLimit, item.IDStatus, item.UpdateBy, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer updated successfully", fiber.Map{"customer_id": id})
}

func DeleteCustomerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	username := c.Query("user", "UNKNOWN")
	steps := []utils.TransactionStep{
		{Name: "DeleteCustomer", Query: "UPDATE tb_customer SET is_delete = 1, update_by = ? WHERE customer_id = ?", Args: []interface{}{username, id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer deleted successfully", nil)
}

func RemoveCustomerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	steps := []utils.TransactionStep{
		{Name: "RemoveCustomer", Query: "DELETE FROM tb_customer WHERE customer_id = ?", Args: []interface{}{id}},
	}
	if err := utils.ExecuteTransaction(steps); err != nil {
		return utils.ErrorResponse(c, err.Error())
	}
	return utils.SuccessResponse(c, "Customer removed permanently", nil)
}
