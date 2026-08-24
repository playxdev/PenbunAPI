// Path: internal/platform/httpx/sqlerr.go
package httpx

import (
	"database/sql"
	"errors"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
)

// รหัส error ของ SQL Server ที่เราแปลความหมายได้
const (
	sqlErrNullNotAllowed  = 515   // insert NULL ลงคอลัมน์ NOT NULL
	sqlErrConstraint      = 547   // FK / CHECK violation
	sqlErrUniqueKey       = 2627  // PRIMARY KEY / UNIQUE constraint
	sqlErrUniqueIndex     = 2601  // UNIQUE index
	sqlErrDeadlock        = 1205  // deadlock victim
	sqlErrArithOverflow   = 8115  // แปลงค่าแล้ว overflow
	sqlErrStringTruncated = 2628  // string หรือ binary จะถูกตัด
	sqlErrRaiserror       = 50000 // RAISERROR ที่เราเขียนเองใน USP_*
)

// MapSQLError แปล error จาก driver ให้เป็น AppError
// คืน nil ถ้า err ไม่ใช่ error ที่มาจากฐานข้อมูล เพื่อให้ผู้เรียกลองทางอื่นต่อ
func MapSQLError(err error) *AppError {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NotFound("ไม่พบข้อมูลที่ต้องการ").WithInternal(err)
	}

	var me mssql.Error
	if !errors.As(err, &me) {
		return nil
	}

	switch me.Number {
	case sqlErrUniqueKey, sqlErrUniqueIndex:
		return Conflict(CodeDuplicate, "ข้อมูลซ้ำกับที่มีอยู่แล้วในระบบ").WithInternal(err)

	case sqlErrConstraint:
		return mapConstraintViolation(me, err)

	case sqlErrNullNotAllowed:
		return BadRequest(CodeFieldRequired, "มีฟิลด์ที่จำเป็นไม่ได้ส่งมา").WithInternal(err)

	case sqlErrArithOverflow, sqlErrStringTruncated:
		return BadRequest(CodeValueOutOfRange, "ค่าที่ส่งมายาวหรือใหญ่เกินกว่าที่ระบบรองรับ").WithInternal(err)

	case sqlErrDeadlock:
		return &AppError{
			HTTPStatus: 503, Code: CodeDBUnavailable,
			Message:  "ระบบกำลังประมวลผลรายการอื่นอยู่ กรุณาลองใหม่อีกครั้ง",
			Internal: err,
		}

	case sqlErrRaiserror:
		return mapRaiserror(me, err)
	}

	return Internal("ระบบขัดข้อง กรุณาลองใหม่อีกครั้ง").WithInternal(err)
}

// 547 ใช้ได้ทั้ง FOREIGN KEY และ CHECK จึงต้องอ่านข้อความประกอบ
func mapConstraintViolation(me mssql.Error, err error) *AppError {
	msg := me.Message

	// CHECK constraint ของ v7 ตั้งชื่อขึ้นต้นด้วย CK_ ทุกตัว
	if strings.Contains(msg, "CHECK constraint") || strings.Contains(msg, "CK_") {
		return BadRequest(CodeInvalidEnum, "ค่าที่ส่งมาไม่อยู่ในรายการที่ระบบยอมรับ").WithInternal(err)
	}

	// FK NO ACTION ตอนลบ/แก้ตารางแม่ ข้อความจะขึ้นต้นด้วย
	// "The DELETE statement conflicted..." หรือ "The UPDATE statement conflicted..."
	if strings.Contains(msg, "DELETE statement conflicted") ||
		strings.Contains(msg, "UPDATE statement conflicted with the REFERENCE") {
		return Conflict(CodeRefInUse, "ลบหรือแก้ไขไม่ได้ เพราะยังมีข้อมูลอื่นอ้างอิงอยู่").WithInternal(err)
	}

	// ที่เหลือคือ INSERT/UPDATE ที่ ref_*_auto ชี้ไปแถวที่ไม่มีจริง
	// ปกติ resolver ควรดักไปแล้ว ถ้ามาถึงตรงนี้แปลว่ามีเส้นทางที่ข้าม resolver
	return BadRequest(CodeRefNotFound, "ข้อมูลอ้างอิงที่ระบุไม่มีอยู่ในระบบ").WithInternal(err)
}

// RAISERROR ใน PenbunSQL v7 เขียนข้อความเป็นภาษาไทยและตั้งใจให้ผู้ใช้อ่าน
// จึงส่งข้อความผ่านไปตรง ๆ ได้ ต่างจาก error อื่นที่ต้องกลบ
func mapRaiserror(me mssql.Error, err error) *AppError {
	msg := strings.TrimSpace(me.Message)

	switch {
	case strings.HasPrefix(msg, "STOCK:"):
		return Conflict(CodeInsufficientStock, msg).WithInternal(err)

	// USP_POST_* ปฏิเสธเอกสารที่สถานะไม่ใช่ CONFIRMED
	// เอกสารที่โพสต์ไปแล้วจะเปลี่ยนสถานะเป็น POSTED/DELIVERED จึงเข้าเงื่อนไขนี้
	case strings.Contains(msg, "ต้องอยู่สถานะ CONFIRMED"):
		return Conflict(CodeAlreadyPosted, msg).WithInternal(err)

	case strings.Contains(msg, "ไม่พบ"):
		return NotFound(msg).WithInternal(err)

	case strings.HasPrefix(msg, "APPLOCK:"):
		return &AppError{
			HTTPStatus: 503, Code: CodeDBUnavailable,
			Message:  "มีการโพสต์เอกสารในคลังนี้อยู่ กรุณารอสักครู่แล้วลองใหม่",
			Internal: err,
		}
	}

	return Unprocessable(CodeBusinessRule, msg).WithInternal(err)
}
