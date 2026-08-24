// Path: internal/platform/httpx/envelope.go
package httpx

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Envelope คือรูปแบบเดียวของทุก response ไม่มีข้อยกเว้น
type Envelope struct {
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Code    string       `json:"code"`
	Data    any          `json:"data"`
	Errors  []FieldError `json:"errors,omitempty"`
	TraceID string       `json:"trace_id"`
}

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
	Value string `json:"value,omitempty"`
}

// Page คือ shape ของ data สำหรับ endpoint แบบ list ทุกตัว
type Page struct {
	Items      any   `json:"items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func NewPage(items any, page, limit int, total int64) Page {
	if items == nil {
		items = []any{}
	}
	tp := int64(0)
	if limit > 0 {
		tp = (total + int64(limit) - 1) / int64(limit)
	}
	return Page{Items: items, Page: page, Limit: limit, Total: total, TotalPages: tp}
}

func OK(c fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(Envelope{
		Status: "success", Code: CodeOK, Message: message,
		Data: data, TraceID: requestid.FromContext(c),
	})
}

func Created(c fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{
		Status: "success", Code: CodeOK, Message: message,
		Data: data, TraceID: requestid.FromContext(c),
	})
}

func NoContent(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Accepted ใช้กับงาน rebuild cache ที่อาจกินเวลา
func Accepted(c fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusAccepted).JSON(Envelope{
		Status: "success", Code: CodeOK, Message: message,
		Data: data, TraceID: requestid.FromContext(c),
	})
}
