// Path: internal/crud/query_test.go
package crud

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"penbun/api/internal/schema"
)

func sample() *Resource {
	return &Resource{
		Name:          "customer",
		Label:         "ลูกค้า",
		Source:        "dbo.vw_customer",
		Table:         "tb_customer",
		IDColumn:      "customer_id",
		AutoColumn:    "customer_auto",
		SearchColumns: []string{"customer_name", "customer_code"},
		SortColumns:   map[string]string{"name": "customer_name", "updated": "update_date"},
		DefaultSort:   "updated",
		Filters: []schema.Filter{
			{Param: "province", Column: "province", Kind: schema.KindString},
		},
		Refs: []schema.Ref{
			{Field: "customer_type_id", Table: "tb_customer_type",
				Column: "ref_customer_type_auto", Label: "ประเภทลูกค้า", Required: true},
		},
		Fields: []schema.Field{
			{Name: "customer_name", Kind: schema.KindString, Required: true, MaxLen: 200},
			{Name: "credit_limit", Kind: schema.KindDecimal},
		},
	}
}

func TestValidate_RejectsBadDescriptor(t *testing.T) {
	t.Run("DefaultSort ต้องมีอยู่ใน SortColumns", func(t *testing.T) {
		r := sample()
		r.DefaultSort = "nope"
		assert.ErrorContains(t, r.Validate(), "DefaultSort")
	})

	t.Run("ref ห้ามชื่อชนกับ field", func(t *testing.T) {
		r := sample()
		r.Fields = append(r.Fields, schema.Field{Name: "customer_type_id", Kind: schema.KindString})
		assert.ErrorContains(t, r.Validate(), "collides")
	})

	t.Run("descriptor ที่ถูกต้องต้องผ่าน", func(t *testing.T) {
		assert.NoError(t, sample().Validate())
	})
}

// ลำดับของหน้าต้องนิ่ง ไม่งั้นแถวเดิมจะโผล่สองหน้าและบางแถวจะหายไปเงียบ ๆ
func TestBuildList_AlwaysHasTiebreaker(t *testing.T) {
	listSQL, _, _, _ := sample().buildList(ListParams{Page: 1, Limit: 50})
	assert.Contains(t, listSQL, "ORDER BY update_date DESC, customer_auto DESC")
	assert.Contains(t, listSQL, "OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY")
}

func TestBuildList_SearchAndFilterUseParameters(t *testing.T) {
	listSQL, countSQL, listArgs, countArgs := sample().buildList(ListParams{
		Page:    2,
		Limit:   20,
		Search:  "O'Brien", // อัญประกาศเดี่ยวต้องไม่ทำให้ query พัง
		Filters: map[string]any{"province": "นนทบุรี"},
	})

	// ค่าจากผู้ใช้ต้องไม่ปรากฏใน SQL string เลย
	assert.NotContains(t, listSQL, "O'Brien")
	assert.NotContains(t, listSQL, "นนทบุรี")
	assert.Contains(t, listArgs, "O'Brien")
	assert.Contains(t, listArgs, "นนทบุรี")

	// query นับใช้ WHERE ชุดเดียวกันแต่ไม่มีพารามิเตอร์ของการแบ่งหน้า
	assert.Len(t, countArgs, 2)
	assert.Len(t, listArgs, 4)
	assert.NotContains(t, countSQL, "OFFSET")

	// offset คำนวณจากหน้าที่ 2
	assert.Equal(t, 20, listArgs[2])
	assert.Equal(t, 20, listArgs[3])
}

func TestBuildList_SortDirection(t *testing.T) {
	asc, _, _, _ := sample().buildList(ListParams{Page: 1, Limit: 10, Sort: "name", SortAsc: true})
	assert.Contains(t, asc, "ORDER BY customer_name ASC")

	desc, _, _, _ := sample().buildList(ListParams{Page: 1, Limit: 10, Sort: "name"})
	assert.Contains(t, desc, "ORDER BY customer_name DESC")
}

// การสร้างรหัสธุรกิจเกิดหลังคำสั่ง INSERT จบ ค่าที่ OUTPUT คืนจึงยังว่างอยู่
// จึงต้องคืนเฉพาะ autoID แล้วไปอ่านรหัสธุรกิจจาก Source อีกครั้ง
func TestBuildInsert_OutputsAutoIDOnly(t *testing.T) {
	q, args := sample().buildInsert(
		map[string]any{"customer_name": "ร้านทดสอบ"},
		map[string]int{"ref_customer_type_auto": 3},
		"somchai",
	)

	assert.Contains(t, q, "OUTPUT INSERTED.autoID")
	assert.NotContains(t, q, "INSERTED.customer_id")

	// ทุกตารางของ v7 มี Trigger AFTER INSERT ซึ่งทำให้ SQL Server ปฏิเสธ
	// OUTPUT ที่ไม่มี INTO — เคยทำให้การสร้างข้อมูลทุกชนิดล้มด้วย 500
	assert.Contains(t, q, "OUTPUT INSERTED.autoID INTO @pb_inserted")
	assert.Contains(t, q, "DECLARE @pb_inserted TABLE (autoID INT)")
	assert.Contains(t, q, "SELECT autoID FROM @pb_inserted")

	// update_by ต่อท้ายเสมอและมาจาก parameter ไม่ใช่จากการต่อ string
	assert.Contains(t, q, "update_by")
	assert.Equal(t, "somchai", args[len(args)-1])

	// คอลัมน์ที่ฐานข้อมูลเป็นเจ้าของต้องไม่ถูกส่งไป
	for _, forbidden := range []string{"prefix", "customer_id", "update_date", "is_delete", "id_status"} {
		assert.NotContains(t, q, forbidden+",")
	}
}

func TestBuildInsert_ColumnOrderIsStable(t *testing.T) {
	// SQL ที่หน้าตาเหมือนเดิมทุกครั้งทำให้ฐานข้อมูลใช้แผนการทำงานเดิมซ้ำได้
	vals := map[string]any{"credit_limit": 1000.0, "customer_name": "ก"}
	first, _ := sample().buildInsert(vals, nil, "u")
	for i := 0; i < 20; i++ {
		again, _ := sample().buildInsert(vals, nil, "u")
		require.Equal(t, first, again)
	}
}

func TestBuildUpdate_OnlyTouchesProvidedColumns(t *testing.T) {
	q, args := sample().buildUpdate("CUSA000041",
		map[string]any{"credit_limit": 5000.0}, nil, "somchai")

	assert.Contains(t, q, "SET credit_limit = @p1")
	assert.NotContains(t, q, "customer_name")
	assert.Contains(t, q, "WHERE customer_id = @p3 AND is_delete = 0")
	assert.Equal(t, "CUSA000041", args[2])
}

// การลบตั้งเฉพาะ is_delete แล้วปล่อยให้ทริกเกอร์ฝั่งฐานข้อมูลจัดการฟิลด์ที่เหลือ
func TestBuildSoftDelete(t *testing.T) {
	q, args := sample().buildSoftDelete("CUSA000041", "somchai")

	assert.Contains(t, q, "SET is_delete = 1")
	assert.NotContains(t, q, "id_status")
	assert.NotContains(t, q, "is_active")
	assert.True(t, strings.HasSuffix(q, "AND is_delete = 0"))
	assert.Equal(t, []any{"somchai", "CUSA000041"}, args)
}
