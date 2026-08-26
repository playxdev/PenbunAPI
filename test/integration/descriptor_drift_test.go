//go:build integration

package integration

import (
	"testing"

	"penbun/api/internal/domain/book"
	"penbun/api/internal/domain/document"
	"penbun/api/internal/resources"
	"penbun/api/internal/schema"
)

// TestDescriptorMatchesSchema เทียบ descriptor ทุกตัวกับคอลัมน์จริงในฐานข้อมูล
//
// descriptor ที่หลุดจาก schema ไม่แสดงอาการตอน start และไม่แสดงตอนอ่านด้วย
// มันโผล่ตอนมีคนกดบันทึกจริงแล้วได้ 500 กลับไป เทสต์นี้จึงต้องรันกับฐานจริง
// ไม่ใช่ mock เพราะสิ่งที่ต้องการจับคือความต่างระหว่างโค้ดกับฐาน ไม่ใช่ตรรกะในโค้ด
//
// MaxLen ที่กว้างกว่าคอลัมน์อันตรายที่สุด เพราะ validation ปล่อยผ่านแล้วไปตาย
// ตอน INSERT ส่วนที่แคบกว่าไม่ทำให้พัง แต่ปฏิเสธค่าที่ฐานรับได้ ซึ่งก็ยังผิดอยู่ดี
func TestDescriptorMatchesSchema(t *testing.T) {
	db := openTestDB(t)

	type column struct {
		dataType string
		maxLen   int
	}

	tables := map[string]map[string]column{}
	rows, err := db.QueryContext(t.Context(),
		`SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, ISNULL(CHARACTER_MAXIMUM_LENGTH, -1)
		   FROM INFORMATION_SCHEMA.COLUMNS`)
	if err != nil {
		t.Fatalf("read INFORMATION_SCHEMA.COLUMNS: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, name, dataType string
		var maxLen int
		if err := rows.Scan(&table, &name, &dataType, &maxLen); err != nil {
			t.Fatalf("scan column row: %v", err)
		}
		if tables[table] == nil {
			tables[table] = map[string]column{}
		}
		tables[table][name] = column{dataType, maxLen}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate column rows: %v", err)
	}

	// ตรวจ field ชุดหนึ่งกับตารางหนึ่ง
	//
	// ค้นคอลัมน์ด้วย ColumnName() ไม่ใช่ Name เพราะ descriptor ตั้งชื่อ JSON ให้ต่าง
	// จากชื่อคอลัมน์ได้ เช่น book_description ที่เขียนลง description
	check := func(t *testing.T, table string, fields []schema.Field) {
		t.Helper()
		cols, ok := tables[table]
		if !ok {
			t.Fatalf("table %s does not exist", table)
		}
		for _, f := range fields {
			name := f.ColumnName()
			col, ok := cols[name]
			if !ok {
				t.Errorf("%s: no column %s in %s", f.Name, name, table)
				continue
			}
			// -1 คือชนิดที่ไม่มีความยาว เช่น int, bit, datetime หรือ nvarchar(MAX)
			if f.Kind != schema.KindString || col.maxLen <= 0 {
				continue
			}
			// EnumValues คุมค่าไว้แน่นกว่าความยาวอยู่แล้ว ไม่ต้องมี MaxLen ซ้ำ
			if len(f.EnumValues) > 0 {
				continue
			}
			switch {
			case f.MaxLen == 0:
				t.Errorf("%s: no MaxLen but column is %s(%d)", f.Name, col.dataType, col.maxLen)
			case f.MaxLen > col.maxLen:
				t.Errorf("%s: MaxLen %d exceeds column %s(%d) — writes that pass validation will fail on INSERT",
					f.Name, f.MaxLen, col.dataType, col.maxLen)
			case f.MaxLen < col.maxLen:
				t.Errorf("%s: MaxLen %d is narrower than column %s(%d) — rejects values the database accepts",
					f.Name, f.MaxLen, col.dataType, col.maxLen)
			}
		}
	}

	for _, r := range resources.All() {
		t.Run(r.Name, func(t *testing.T) { check(t, r.Table, r.Fields) })
	}

	// เอกสารมีสอง descriptor ต่อหนึ่งชนิด หัวและรายการคนละตาราง
	//
	// ก่อน v8 ชุดนี้ไม่เคยถูกตรวจเลย ทั้งที่ doc_no ประกาศ MaxLen 50 ไว้ตั้งแต่ต้น
	// ขณะที่คอลัมน์กว้าง 30 — ค่ายาว 31-50 ตัวจึงผ่านทุกด่านแล้วไปตายที่ INSERT
	for _, d := range document.All() {
		t.Run(d.Name+"/header", func(t *testing.T) { check(t, d.HeaderTable, d.HeaderFields) })
		t.Run(d.Name+"/items", func(t *testing.T) { check(t, d.ItemTable, d.ItemFields) })
	}

	// /book เขียนสองตารางผ่าน handler ที่เขียนเอง ไม่ได้อยู่ใน resources.All()
	for table, fields := range book.FieldSets() {
		t.Run("book/"+table, func(t *testing.T) { check(t, table, fields) })
	}
}
