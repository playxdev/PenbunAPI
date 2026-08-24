// Path: internal/repository/insert_test.go
package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ทุกตารางใน PenbunSQL v7 มี Trigger AFTER INSERT สำหรับเติม Business ID
// SQL Server จึงปฏิเสธ OUTPUT ที่ไม่มี INTO ด้วย Msg 334 ซึ่งทำให้การสร้าง
// ข้อมูลทุกชนิดตอบ 500 ไม่ใช่แค่ตารางใดตารางหนึ่ง
func TestInsertReturningAuto_OutputsIntoTableVariable(t *testing.T) {
	q := InsertReturningAuto("tb_vendor_type", []string{"type_name", "update_by"}, []string{"@p1", "@p2"})

	assert.Contains(t, q, "DECLARE @pb_inserted TABLE (autoID INT)")
	assert.Contains(t, q, "OUTPUT INSERTED.autoID INTO @pb_inserted")
	assert.Contains(t, q, "SELECT autoID FROM @pb_inserted")

	// OUTPUT ที่ไม่มี INTO คือรูปแบบที่ฐานข้อมูลปฏิเสธ ห้ามหลุดกลับมาอีก
	assert.False(t, strings.Contains(q, "OUTPUT INSERTED.autoID VALUES"), q)

	assert.Contains(t, q, "INSERT INTO dbo.tb_vendor_type (type_name, update_by)")
	assert.Contains(t, q, "VALUES (@p1, @p2)")
}

// คำสั่งสุดท้ายของ batch ต้องเป็น SELECT เพื่อให้ QueryRow อ่าน autoID ได้
func TestInsertReturningAuto_EndsWithTheSelect(t *testing.T) {
	q := strings.TrimSpace(InsertReturningAuto("tb_company", []string{"name_th"}, []string{"@p1"}))
	assert.True(t, strings.HasSuffix(q, "SELECT autoID FROM @pb_inserted;"), q)
}

// ค่าจาก request ต้องมาเป็น parameter เสมอ ตัวสร้างคำสั่งรับแต่ชื่อคอลัมน์
// กับ placeholder ที่ descriptor เป็นคนกำหนด
func TestInsertReturningAuto_TakesNoLiteralValues(t *testing.T) {
	q := InsertReturningAuto("tb_route", []string{"route_code", "route_name"}, []string{"@p1", "@p2"})
	assert.NotContains(t, q, "'")
}
