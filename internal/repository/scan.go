// Path: internal/repository/scan.go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Row คือผลลัพธ์หนึ่งแถวในรูปที่ marshal เป็น JSON ได้ตรง ๆ
type Row map[string]any

// bangkok — ทุก DATETIME ใน PenbunSQL เก็บเป็นเวลาไทยอยู่แล้ว (ไม่มี offset)
// จึงต้องติดโซนกลับตอนอ่าน ไม่งั้น client จะตีความเป็น UTC แล้วเพี้ยนไป 7 ชั่วโมง
var bangkok = time.FixedZone("ICT", 7*3600)

// ScanRows แปลง *sql.Rows เป็น []Row พร้อมทำให้ชนิดข้อมูลเป็นมิตรกับ JSON
//
// จุดสำคัญคือ DECIMAL/MONEY: driver คืนมาเป็น []byte ถ้าปล่อยไปจะกลายเป็น
// base64 ใน JSON ส่วนการแปลงเป็น float64 ก็ทำให้ยอดเงินคลาดเคลื่อน
// จึงแปลงเป็น json.Number ซึ่งฝั่ง JSON เห็นเป็นตัวเลขแต่ไม่เสียความละเอียด
func ScanRows(rows *sql.Rows) ([]Row, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	out := make([]Row, 0, 32)
	holders := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range holders {
		holders[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(holders...); err != nil {
			return nil, err
		}
		row := make(Row, len(cols))
		for i, name := range cols {
			row[name] = normalize(values[i], types[i].DatabaseTypeName())
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ScanOne คืนแถวเดียว หรือ sql.ErrNoRows ถ้าไม่เจอ
func ScanOne(rows *sql.Rows) (Row, error) {
	list, err := ScanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return list[0], nil
}

// QueryOne ยิง query แล้วคืนแถวแรก ใช้ได้ทั้งบน *sql.DB และ *sql.Tx
func QueryOne(ctx context.Context, ex Executor, q string, args []any) (Row, error) {
	rows, err := ex.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return ScanOne(rows)
}

// QueryMany ยิง query แล้วคืนทุกแถว
func QueryMany(ctx context.Context, ex Executor, q string, args []any) ([]Row, error) {
	rows, err := ex.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return ScanRows(rows)
}

func normalize(v any, dbType string) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		switch dbType {
		case "DECIMAL", "NUMERIC", "MONEY", "SMALLMONEY":
			return json.Number(string(val))
		case "UNIQUEIDENTIFIER", "VARBINARY", "BINARY", "IMAGE":
			return fmt.Sprintf("%x", val)
		default:
			return string(val)
		}
	case time.Time:
		// DATETIME ที่ไม่มีโซน driver จะให้มาเป็น UTC — ย้ายหน้าปัดมาเป็น ICT
		if val.Location() == time.UTC {
			val = time.Date(val.Year(), val.Month(), val.Day(),
				val.Hour(), val.Minute(), val.Second(), val.Nanosecond(), bangkok)
		}
		return val.Format(time.RFC3339)
	default:
		return v
	}
}
