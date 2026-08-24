// Path: internal/platform/httpx/sqlerr_test.go
package httpx

import (
	"database/sql"
	"fmt"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sqlError(number int32, message string) error {
	return mssql.Error{Number: number, Message: message}
}

func TestMapSQLError_NotDatabaseError(t *testing.T) {
	// error ที่ไม่ได้มาจากฐานข้อมูลต้องคืน nil เพื่อให้ผู้เรียกลองทางอื่นต่อ
	assert.Nil(t, MapSQLError(nil))
	assert.Nil(t, MapSQLError(fmt.Errorf("some transport failure")))
}

func TestMapSQLError_NoRows(t *testing.T) {
	got := MapSQLError(sql.ErrNoRows)
	require.NotNil(t, got)
	assert.Equal(t, 404, got.HTTPStatus)
	assert.Equal(t, CodeNotFound, got.Code)
}

func TestMapSQLError_Duplicate(t *testing.T) {
	for _, number := range []int32{2627, 2601} {
		got := MapSQLError(sqlError(number, "Violation of UNIQUE KEY constraint 'UQ_tb_customer_code'."))
		require.NotNil(t, got)
		assert.Equal(t, 409, got.HTTPStatus)
		assert.Equal(t, CodeDuplicate, got.Code)

		// ชื่อ constraint ต้องไม่หลุดออกไปหา client
		assert.NotContains(t, got.Message, "UQ_tb_customer_code")
	}
}

// 547 ใช้ได้ทั้ง FOREIGN KEY และ CHECK จึงต้องแยกด้วยข้อความประกอบ
func TestMapSQLError_ConstraintVariants(t *testing.T) {
	cases := []struct {
		name       string
		message    string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "ค่าไม่อยู่ในรายการที่ CHECK กำหนด",
			message:    "The INSERT statement conflicted with the CHECK constraint \"CK_tb_order_type\".",
			wantStatus: 400,
			wantCode:   CodeInvalidEnum,
		},
		{
			name:       "ลบตารางแม่ที่ยังมีลูกอ้างอยู่",
			message:    "The DELETE statement conflicted with the REFERENCE constraint \"FK_tb_customer_type\".",
			wantStatus: 409,
			wantCode:   CodeRefInUse,
		},
		{
			name:       "ปลายทางของ FK ไม่มีอยู่จริง",
			message:    "The INSERT statement conflicted with the FOREIGN KEY constraint \"FK_tb_customer_customer_type\".",
			wantStatus: 400,
			wantCode:   CodeRefNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapSQLError(sqlError(547, tc.message))
			require.NotNil(t, got)
			assert.Equal(t, tc.wantStatus, got.HTTPStatus)
			assert.Equal(t, tc.wantCode, got.Code)
		})
	}
}

func TestMapSQLError_Deadlock(t *testing.T) {
	got := MapSQLError(sqlError(1205, "Transaction was deadlocked on lock resources."))
	require.NotNil(t, got)
	assert.Equal(t, 503, got.HTTPStatus)
	assert.Equal(t, CodeDBUnavailable, got.Code)
}

// ข้อความจาก RAISERROR ในกระบวนงานเขียนไว้ให้ผู้ใช้อ่านโดยตรง
// จึงส่งผ่านไปได้ ต่างจาก error อื่นที่ต้องกลบ
func TestMapSQLError_Raiserror(t *testing.T) {
	cases := []struct {
		name       string
		message    string
		wantStatus int
		wantCode   string
		passThru   bool
	}{
		{
			name:       "สต็อกไม่พอ",
			message:    "STOCK: คลัง DC มีคงเหลือ 10 ไม่พอตัด 50",
			wantStatus: 409,
			wantCode:   CodeInsufficientStock,
			passThru:   true,
		},
		{
			name:       "โพสต์ซ้ำ",
			message:    "POST_ORDER: ใบส่ง ORDA000010 ต้องอยู่สถานะ CONFIRMED (ปัจจุบัน DELIVERED)",
			wantStatus: 409,
			wantCode:   CodeAlreadyPosted,
			passThru:   true,
		},
		{
			name:       "จองล็อกไม่ได้",
			message:    "APPLOCK: ไม่สามารถจองล็อก PENBUN:STOCK:1:2 ได้ (rc=-1)",
			wantStatus: 503,
			wantCode:   CodeDBUnavailable,
			passThru:   false,
		},
		{
			name:       "กฎธุรกิจอื่น",
			message:    "POST_RETURN: ใบรับคืนไม่มีรายการ",
			wantStatus: 422,
			wantCode:   CodeBusinessRule,
			passThru:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapSQLError(sqlError(50000, tc.message))
			require.NotNil(t, got)
			assert.Equal(t, tc.wantStatus, got.HTTPStatus)
			assert.Equal(t, tc.wantCode, got.Code)
			if tc.passThru {
				assert.Equal(t, tc.message, got.Message)
			} else {
				assert.NotEqual(t, tc.message, got.Message)
			}
		})
	}
}

func TestMapSQLError_UnknownFallsBackToInternal(t *testing.T) {
	got := MapSQLError(sqlError(4060, "Cannot open database \"PENBUN\" requested by the login."))
	require.NotNil(t, got)
	assert.Equal(t, 500, got.HTTPStatus)
	assert.Equal(t, CodeInternal, got.Code)

	// รายละเอียดต้นทางเก็บไว้ให้ log เท่านั้น ไม่ส่งออกไปหา client
	assert.NotContains(t, got.Message, "PENBUN")
	assert.ErrorContains(t, got.Internal, "PENBUN")
}
