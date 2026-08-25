//go:build integration

package integration

import (
	"testing"

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

	for _, r := range resources.All() {
		t.Run(r.Name, func(t *testing.T) {
			cols, ok := tables[r.Table]
			if !ok {
				t.Fatalf("table %s does not exist", r.Table)
			}
			for _, f := range r.Fields {
				col, ok := cols[f.Name]
				if !ok {
					t.Errorf("%s: no column %s in %s", f.Name, f.Name, r.Table)
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
		})
	}
}
