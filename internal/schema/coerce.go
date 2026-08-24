// Path: internal/schema/coerce.go
package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"penbun/api/internal/platform/httpx"
)

var dateLayouts = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}

// Coerce แปลงค่าดิบจาก JSON เป็นชนิดที่ driver ส่งลง SQL Server ได้
// คืน AppError ที่ระบุชื่อฟิลด์เสมอเมื่อแปลงไม่ได้ เพื่อให้ frontend ชี้จุดผิดถูก
func Coerce(raw json.RawMessage, kind Kind, name, label string) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	bad := func(want string) error {
		return httpx.Validation(fmt.Sprintf("%s ต้องเป็น%s", label, want)).
			WithField(name, httpx.CodeValidationFailed, strings.Trim(string(raw), `"`))
	}

	switch kind {
	case KindString:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, bad("ข้อความ")
		}
		return s, nil

	case KindInt:
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, bad("จำนวนเต็ม")
		}
		v, err := strconv.ParseInt(n.String(), 10, 64)
		if err != nil {
			return nil, bad("จำนวนเต็ม")
		}
		return v, nil

	case KindDecimal:
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, bad("ตัวเลข")
		}
		v, err := strconv.ParseFloat(n.String(), 64)
		if err != nil {
			return nil, bad("ตัวเลข")
		}
		return v, nil

	case KindBool:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, bad("true หรือ false")
		}
		return b, nil

	case KindDate:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, bad("วันที่")
		}
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
		return nil, bad("วันที่ในรูปแบบ YYYY-MM-DD หรือ RFC3339")
	}
	return nil, fmt.Errorf("schema: unknown kind %d", kind)
}

// ParseDate ใช้กับฟิลด์วันที่ที่ handler อ่านเองจาก struct ไม่ได้ผ่าน Coerce
func ParseDate(s, field string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, httpx.Validation(field+" ต้องอยู่ในรูปแบบ YYYY-MM-DD หรือ RFC3339").
		WithField(field, httpx.CodeValidationFailed, s)
}

// Validate ตรวจข้อจำกัดที่ประกาศไว้ใน Field หลังจาก Coerce แล้ว
func Validate(f Field, v any) error {
	if v == nil {
		return nil
	}
	label := f.DisplayLabel()

	if s, ok := v.(string); ok {
		if f.MaxLen > 0 && len([]rune(s)) > f.MaxLen {
			return httpx.BadRequest(httpx.CodeValueOutOfRange,
				fmt.Sprintf("%s ยาวเกิน %d ตัวอักษร", label, f.MaxLen)).
				WithField(f.Name, httpx.CodeValueOutOfRange, "")
		}
		if len(f.EnumValues) > 0 && !contains(f.EnumValues, s) {
			return httpx.BadRequest(httpx.CodeInvalidEnum,
				fmt.Sprintf("%s ต้องเป็นหนึ่งใน: %s", label, strings.Join(f.EnumValues, ", "))).
				WithField(f.Name, httpx.CodeInvalidEnum, s)
		}
	}

	if f.Min != nil {
		var got float64
		switch n := v.(type) {
		case int64:
			got = float64(n)
		case float64:
			got = n
		default:
			return nil
		}
		if got < *f.Min {
			return httpx.BadRequest(httpx.CodeValueOutOfRange,
				fmt.Sprintf("%s ต้องไม่น้อยกว่า %g", label, *f.Min)).
				WithField(f.Name, httpx.CodeValueOutOfRange, "")
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
