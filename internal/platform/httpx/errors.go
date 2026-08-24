// Path: internal/platform/httpx/errors.go
package httpx

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

const (
	CodeOK                 = "OK"
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeFieldRequired      = "FIELD_REQUIRED"
	CodeRefNotFound        = "REF_NOT_FOUND"
	CodeInvalidEnum        = "INVALID_ENUM"
	CodeValueOutOfRange    = "VALUE_OUT_OF_RANGE"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeForbidden          = "FORBIDDEN"
	CodeMustChangePassword = "MUST_CHANGE_PASSWORD"
	CodeNotFound           = "NOT_FOUND"
	CodeEndpointRemoved    = "ENDPOINT_REMOVED"
	CodeDuplicate          = "DUPLICATE"
	CodeRefInUse           = "REF_IN_USE"
	CodeInsufficientStock  = "INSUFFICIENT_STOCK"
	CodeAlreadyPosted      = "ALREADY_POSTED"
	CodeAccountLocked      = "ACCOUNT_LOCKED"
	CodeBusinessRule       = "BUSINESS_RULE"
	CodeInternal           = "INTERNAL"
	CodeDBUnavailable      = "DB_UNAVAILABLE"
)

// AppError คือ error ชนิดเดียวที่ handler ควรคืนออกมา
// error อื่นที่หลุดขึ้นมาถึง GlobalErrorHandler จะถูกกลบเป็น INTERNAL
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Fields     []FieldError
	// Internal เก็บ error ต้นทางไว้เข้า log อย่างเดียว ไม่เคยส่งออกไปหา client
	Internal error
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Internal }

func (e *AppError) WithInternal(err error) *AppError {
	c := *e
	c.Internal = err
	return &c
}

func (e *AppError) WithField(field, code, value string) *AppError {
	c := *e
	c.Fields = append(append([]FieldError{}, e.Fields...), FieldError{Field: field, Code: code, Value: value})
	return &c
}

func newErr(status int, code, msg string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: msg}
}

func BadRequest(code, msg string) *AppError { return newErr(fiber.StatusBadRequest, code, msg) }
func Validation(msg string) *AppError {
	return newErr(fiber.StatusBadRequest, CodeValidationFailed, msg)
}
func Unauthorized(msg string) *AppError {
	return newErr(fiber.StatusUnauthorized, CodeUnauthorized, msg)
}
func Forbidden(code, msg string) *AppError {
	return newErr(fiber.StatusForbidden, code, msg)
}
func NotFound(msg string) *AppError { return newErr(fiber.StatusNotFound, CodeNotFound, msg) }
func Conflict(code, msg string) *AppError {
	return newErr(fiber.StatusConflict, code, msg)
}
func Locked(msg string) *AppError { return newErr(fiber.StatusLocked, CodeAccountLocked, msg) }
func Unprocessable(code, msg string) *AppError {
	return newErr(fiber.StatusUnprocessableEntity, code, msg)
}
func Internal(msg string) *AppError {
	return newErr(fiber.StatusInternalServerError, CodeInternal, msg)
}
func Gone(msg string) *AppError { return newErr(fiber.StatusGone, CodeEndpointRemoved, msg) }

// RefNotFound เป็นกรณีที่เจอบ่อยที่สุดตอนแปลง Business ID -> autoID
func RefNotFound(field, value, label string) *AppError {
	return (&AppError{
		HTTPStatus: fiber.StatusBadRequest,
		Code:       CodeRefNotFound,
		Message:    fmt.Sprintf("ไม่พบ%s '%s'", label, value),
	}).WithField(field, CodeRefNotFound, value)
}

// GlobalErrorHandler เป็นทางออกเดียวของ error ทุกตัวในระบบ
func GlobalErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		trace := requestid.FromContext(c)

		var appErr *AppError
		if !errors.As(err, &appErr) {
			// ลองแปลจาก SQL Server error ก่อน ถ้าไม่ใช่ค่อยดู fiber.Error
			if mapped := MapSQLError(err); mapped != nil {
				appErr = mapped
			} else {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					appErr = newErr(fe.Code, statusToCode(fe.Code), fe.Message)
				} else {
					appErr = Internal("ระบบขัดข้อง กรุณาลองใหม่อีกครั้ง").WithInternal(err)
				}
			}
		}

		// 4xx ไม่ log ที่นี่ เพราะ access log พิมพ์ เวลา/method/path/status/message
		// ครบอยู่แล้ว การ log ซ้ำทำให้หนึ่งคำขอกินสองบรรทัดโดยไม่ได้ข้อมูลใหม่
		//
		// 5xx ยังต้อง log เพิ่ม เพราะ error ต้นทางไม่เคยถูกส่งออกไปหา client
		// จึงไม่มีทางโผล่ใน access log
		if appErr.HTTPStatus >= 500 {
			// log เฉพาะ error ต้นทาง ไม่เอา AppError.Error() ทั้งก้อน
			// เพราะข้อความที่ตอบ client อยู่ใน access log แล้ว
			cause := appErr.Error()
			if appErr.Internal != nil {
				cause = appErr.Internal.Error()
			}
			log.Error("request failed",
				"method", c.Method(),
				"path", c.Path(),
				"code", appErr.Code,
				"error", cause,
				"trace_id", trace,
			)
		}

		status := "fail"
		if appErr.HTTPStatus >= 500 {
			status = "error"
		}

		return c.Status(appErr.HTTPStatus).JSON(Envelope{
			Status:  status,
			Code:    appErr.Code,
			Message: appErr.Message,
			Errors:  appErr.Fields,
			Data:    nil,
			TraceID: trace,
		})
	}
}

func statusToCode(status int) string {
	switch status {
	case fiber.StatusNotFound:
		return CodeNotFound
	case fiber.StatusUnauthorized:
		return CodeUnauthorized
	case fiber.StatusForbidden:
		return CodeForbidden
	case fiber.StatusMethodNotAllowed, fiber.StatusBadRequest:
		return CodeValidationFailed
	default:
		return CodeInternal
	}
}
