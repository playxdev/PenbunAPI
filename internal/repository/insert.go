// Path: internal/repository/insert.go
package repository

import (
	"fmt"
	"strings"
)

// InsertReturningAuto สร้างคำสั่ง INSERT ที่อ่าน autoID กลับมาได้
// แม้ตารางเป้าหมายจะมี Trigger เปิดอยู่
//
// SQL Server ปฏิเสธ OUTPUT ที่ไม่มี INTO เมื่อตารางมี Trigger ทำงานอยู่:
//
//	The target table '...' of the DML statement cannot have any enabled
//	triggers if the statement contains an OUTPUT clause without INTO clause.
//
// ทุกตารางใน PenbunSQL v7 มี Trigger AFTER INSERT สำหรับเติม Business ID
// คำสั่ง INSERT ทุกคำสั่งของระบบจึงต้องส่ง OUTPUT ลงตัวแปรตารางก่อน
// แล้วค่อย SELECT ออกมาเป็นผลลัพธ์ของ batch
//
// ห้าม OUTPUT คอลัมน์ Business ID เด็ดขาด
//
// Trigger เป็นชนิด AFTER INSERT ซึ่ง UPDATE แถวตามหลังคำสั่ง INSERT
// ค่าที่ OUTPUT เห็นจึงเป็นค่า ณ ตอน INSERT ซึ่งยังเป็น NULL อยู่
// ต้องเอา autoID ที่ได้ไป SELECT จาก Source ต่อในทรานแซกชันเดียวกันเสมอ
//
// cols และ placeholders ต้องยาวเท่ากันและมาจาก descriptor เท่านั้น
// ค่าจาก request ต้องลง parameter ผ่าน schema.Args ไม่ใช่ต่อเข้ากับสตริงนี้
func InsertReturningAuto(table string, cols, placeholders []string) string {
	return fmt.Sprintf(
		"DECLARE @pb_inserted TABLE (autoID INT); "+
			"INSERT INTO dbo.%s (%s) OUTPUT INSERTED.autoID INTO @pb_inserted VALUES (%s); "+
			"SELECT autoID FROM @pb_inserted;",
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
}
