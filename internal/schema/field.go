// Path: internal/schema/field.go
package schema

// Kind บอกวิธีแปลงค่าจาก JSON ก่อนส่งลง SQL Server
type Kind int

const (
	KindString Kind = iota
	KindInt
	KindDecimal
	KindBool
	KindDate // YYYY-MM-DD หรือ RFC3339
)

// Field คือคอลัมน์ที่ client เขียนได้
//
// คอลัมน์ที่ไม่อยู่ในรายการของ resource หรือ document = client แตะไม่ได้
// ไม่ว่าจะส่งอะไรมาก็ตาม ชื่อคอลัมน์ทุกตัวที่ถูกต่อเข้า SQL string มาจากที่นี่
// เท่านั้น ค่าจาก request ลง parameter เสมอ
type Field struct {
	Name string // key ใน JSON
	// Column คือชื่อคอลัมน์จริง ปล่อยว่างได้ถ้าตรงกับ Name
	//
	// จำเป็นเมื่อ endpoint หนึ่งเขียนสองตารางที่มีชื่อคอลัมน์ซ้ำกัน เช่น
	// tb_product กับ tb_book ต่างก็มี description ถ้าใช้ key เดียวกันใน JSON
	// ค่าจะถูกเขียนลงทั้งสองตารางโดยที่ผู้ใช้ตั้งใจแก้แค่ตารางเดียว
	Column     string
	Kind       Kind
	Required   bool // บังคับตอนสร้าง
	NoInsert   bool // อ่านอย่างเดียว สร้างไม่ได้
	NoUpdate   bool // ตั้งได้ตอนสร้าง แต่แก้ทีหลังไม่ได้
	MaxLen     int  // 0 = ไม่จำกัด ใช้กับ KindString
	Label      string
	EnumValues []string // ถ้าไม่ว่าง ตรวจก่อนถึง DB เพื่อให้ error อ่านง่ายกว่าข้อความ CHECK constraint
	Min        *float64 // ใช้กับ KindInt / KindDecimal
}

// ColumnName คืนชื่อคอลัมน์ที่จะใช้ประกอบ SQL
func (f Field) ColumnName() string {
	if f.Column != "" {
		return f.Column
	}
	return f.Name
}

// DisplayLabel คืนชื่อที่เอาไปใส่ในข้อความ error ได้
func (f Field) DisplayLabel() string {
	if f.Label != "" {
		return f.Label
	}
	return f.Name
}

// Ref คือ reference ที่ client ส่งมาเป็น Business ID
// engine จะแปลงเป็น autoID แล้วเขียนลง Column
type Ref struct {
	Field    string // key ใน JSON เช่น "customer_type_id"
	Table    string // ตารางแม่ เช่น "tb_customer_type"
	Column   string // คอลัมน์ FK เช่น "ref_customer_type_auto"
	Label    string // ชื่อไทยสำหรับข้อความ error
	Required bool
	NoUpdate bool
}

// Filter คือ query parameter ที่กรองได้
type Filter struct {
	Param  string
	Column string
	Kind   Kind
}

func Float64(v float64) *float64 { return &v }
