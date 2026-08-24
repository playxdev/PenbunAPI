// Path: internal/domain/document/specs_test.go
package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllSpecsAreValid(t *testing.T) {
	require.NoError(t, Validate())
}

// ทุก spec ต้องบอกได้ว่าจะจองล็อกคู่ไหนก่อนโพสต์
// ไม่มีข้อนี้เท่ากับเปิดช่องให้เอกสารสองใบตัดสต็อกซ้อนกันจนติดลบ
func TestEverySpecLocksBeforePosting(t *testing.T) {
	for _, s := range All() {
		t.Run(s.Name, func(t *testing.T) {
			require.NotEmpty(t, s.LockPairsSQL)

			// ต้องเรียงผลลัพธ์เสมอ ถ้าสองใบจองล็อกสลับลำดับกันจะเกิด deadlock
			assert.Contains(t, s.LockPairsSQL, "ORDER BY",
				"LockPairsSQL ต้อง ORDER BY เพื่อให้ลำดับการจองล็อกเหมือนกันทุกครั้ง")

			assert.Contains(t, s.LockPairsSQL, "ref_sku_auto")
			assert.Contains(t, s.LockPairsSQL, "@p1")
			assert.Contains(t, s.LockPairsSQL, "is_delete = 0")
		})
	}
}

func TestEverySpecRecalculatesTotals(t *testing.T) {
	for _, s := range All() {
		t.Run(s.Name, func(t *testing.T) {
			// ยอดรวมต้องมาจากรายการจริงเสมอ ไม่ใช่จากค่าที่ client ส่งมา
			assert.Contains(t, s.TotalsSQL, "SUM(")
			assert.Contains(t, s.TotalsSQL, "@p1")
			assert.Contains(t, s.TotalsSQL, "@p2")
			assert.Contains(t, s.TotalsSQL, "is_delete = 0")
		})
	}
}

func TestPostedStatusIsListedInPostedStatuses(t *testing.T) {
	for _, s := range All() {
		t.Run(s.Name, func(t *testing.T) {
			assert.True(t, s.isPosted(s.PostedStatus),
				"PostedStatus %q ต้องอยู่ใน PostedStatuses ด้วย มิฉะนั้นการโพสต์ซ้ำจะไม่ถูกตรวจจับ",
				s.PostedStatus)
			assert.False(t, s.isPosted(StatusDraft))
			assert.False(t, s.isPosted(StatusConfirmed))
		})
	}
}

// ใบส่งจบที่ DELIVERED ไม่ใช่ POSTED เหมือนเอกสารอีกสามชนิด
func TestOrderPostsToDelivered(t *testing.T) {
	assert.Equal(t, "DELIVERED", Order.PostedStatus)
	assert.True(t, Order.isPosted("INVOICED"), "ใบส่งที่วางบิลแล้วก็ถือว่าโพสต์ไปแล้ว")

	for _, s := range []*Spec{ReceiveNote, ReturnNote, VendorReturnNote} {
		assert.Equal(t, "POSTED", s.PostedStatus, s.Name)
	}
}

// คลังปลายทางของสินค้าคืนขึ้นกับสภาพสินค้าเป็นรายบรรทัด
// ของชำรุดต้องไม่ปนกลับเข้าคลังที่ใช้ขาย
func TestReturnNoteRoutesByCondition(t *testing.T) {
	sql := ReturnNote.LockPairsSQL
	assert.Contains(t, sql, "condition_status")
	assert.Contains(t, sql, "'DMG'")
	assert.Contains(t, sql, "'RET'")
	assert.NotContains(t, sql, "h.ref_warehouse_auto",
		"ใบรับคืนต้องไม่ใช้คลังจากหัวเอกสาร")

	var found bool
	for _, f := range ReturnNote.ItemFields {
		if f.Name == "condition_status" {
			found = true
			assert.True(t, f.Required, "สภาพสินค้าต้องบังคับระบุ")
			assert.ElementsMatch(t, []string{"GOOD", "DAMAGED"}, f.EnumValues)
		}
	}
	assert.True(t, found, "ใบรับคืนต้องมีฟิลด์ condition_status")
}

// ใบส่งแบบฝากขายที่ไม่ระบุงวดจะไม่ถูกบันทึกลงประวัติ
// ทำให้การเสนอยอดของงวดถัดไปมองไม่เห็นรอบนี้ และแก้ย้อนหลังไม่ได้
func TestOrderRequiresPeriodKeyForConsignment(t *testing.T) {
	require.NotNil(t, Order.ValidateHeader)

	err := Order.ValidateHeader(map[string]any{"order_type": "CONSIGN"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "period_key")

	assert.NoError(t, Order.ValidateHeader(
		map[string]any{"order_type": "CONSIGN", "period_key": "2026-08"}, nil))

	// การขายขาดไม่ผูกกับงวด จึงไม่ต้องระบุ
	assert.NoError(t, Order.ValidateHeader(map[string]any{"order_type": "SALE"}, nil))
}

func TestSpecValidateCatchesMissingPieces(t *testing.T) {
	t.Run("ไม่มี LockPairsSQL", func(t *testing.T) {
		s := *Order
		s.LockPairsSQL = ""
		assert.ErrorContains(t, s.Validate(), "LockPairsSQL")
	})

	t.Run("รายการต้องอ้างถึง SKU", func(t *testing.T) {
		s := *Order
		s.ItemRefs = nil
		assert.ErrorContains(t, s.Validate(), "ref_sku_auto")
	})
}

// เอกสารทุกชนิดต้องอ่านผ่าน View หรือ derived table ที่กรองแถวที่ถูกลบออกแล้ว
func TestSourcesExcludeDeletedRows(t *testing.T) {
	for _, s := range All() {
		t.Run(s.Name, func(t *testing.T) {
			for label, src := range map[string]string{
				"HeaderSource": s.HeaderSource,
				"ItemSource":   s.ItemSource,
			} {
				if strings.HasPrefix(src, "dbo.vw_") {
					continue // View กรองให้แล้วในนิยามของตัวมันเอง
				}
				assert.Contains(t, src, "is_delete = 0",
					"%s ของ %s ต้องกรองแถวที่ถูกลบออก", label, s.Name)
			}
		})
	}
}
