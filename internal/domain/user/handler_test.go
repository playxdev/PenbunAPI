// Path: internal/domain/user/handler_test.go
package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"penbun/api/internal/config"
	"penbun/api/internal/platform/httpx"
)

func handler() *Handler {
	// cost ต่ำสุดที่ bcrypt ยอมรับ เทสต์ไม่ได้วัดความแข็งของ hash
	return &Handler{cfg: &config.Config{BcryptCost: bcrypt.MinCost}}
}

func str(s string) *string { return &s }

func ok() *createRequest {
	return &createRequest{UserName: "somchai", Password: "Penbun2026", UserLevel: "USER"}
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	ae, is := err.(*httpx.AppError)
	require.True(t, is, "ต้องเป็น AppError ไม่ใช่ %T", err)
	require.Len(t, ae.Fields, 1, "ข้อความต้องชี้ช่องที่ผิดเสมอ")
	return ae.Fields[0].Field
}

// รหัสผ่านต้องออกจาก validate เป็น hash เท่านั้น
// ถ้าหลุดข้อนี้ plaintext จะไปนอนอยู่ใน tb_users ให้ทุกคนที่อ่านตารางได้เห็น
func TestPasswordIsHashed(t *testing.T) {
	vals, err := handler().validate(ok())
	require.NoError(t, err)

	stored, is := vals["user_password"].(string)
	require.True(t, is)
	assert.NotEqual(t, "Penbun2026", stored)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored), []byte("Penbun2026")))
}

// status_change_pw ต้องไม่ถูกเขียน DEFAULT 1 ของตารางคือสิ่งเดียวที่บังคับให้
// ผู้ใช้ใหม่เปลี่ยนรหัสผ่านที่ผู้ดูแลตั้งให้
func TestStatusChangePwIsLeftToTheTable(t *testing.T) {
	vals, err := handler().validate(ok())
	require.NoError(t, err)

	assert.NotContains(t, vals, "status_change_pw")
	assert.NotContains(t, vals, "status_user_locked")
	assert.NotContains(t, vals, "counting_password_fail")
	assert.NotContains(t, vals, "user_id")
}

func TestLevelIsUppercasedAndChecked(t *testing.T) {
	req := ok()
	req.UserLevel = "admin"
	vals, err := handler().validate(req)
	require.NoError(t, err)
	assert.Equal(t, "ADMIN", vals["user_level"])

	req.UserLevel = "MANAGER"
	_, err = handler().validate(req)
	assert.Equal(t, "user_level", fieldOf(t, err))
}

func TestRefusals(t *testing.T) {
	long := ""
	for range 101 {
		long += "a"
	}

	cases := map[string]struct {
		mutate func(*createRequest)
		field  string
	}{
		"ชื่อผู้ใช้ว่าง":        {func(r *createRequest) { r.UserName = "  " }, "user_name"},
		"ชื่อผู้ใช้ไม่ใช่ละติน": {func(r *createRequest) { r.UserName = "อารีย์" }, "user_name"},
		"ชื่อผู้ใช้สั้นไป":      {func(r *createRequest) { r.UserName = "ab" }, "user_name"},
		"ชื่อผู้ใช้มีช่องว่าง":  {func(r *createRequest) { r.UserName = "som chai" }, "user_name"},
		"ไม่ได้ส่งรหัสผ่าน":     {func(r *createRequest) { r.Password = "" }, "password"},
		"รหัสผ่านสั้นไป":        {func(r *createRequest) { r.Password = "Pb2026" }, "password"},
		"รหัสผ่านไม่มีตัวเลข":   {func(r *createRequest) { r.Password = "Penbunpassword" }, "password"},
		"ไม่ได้ส่งสิทธิ์":       {func(r *createRequest) { r.UserLevel = "" }, "user_level"},
		"อีเมลยาวเกินคอลัมน์":   {func(r *createRequest) { r.Email = str(long) }, "email"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := ok()
			tc.mutate(req)
			_, err := handler().validate(req)
			assert.Equal(t, tc.field, fieldOf(t, err))
		})
	}
}

// ค่าที่ว่างต้องไม่ถูกเขียนลงคอลัมน์ ปล่อยให้เป็น NULL ตาม DEFAULT
// สตริงว่างในฐานดูเหมือนมีค่าแต่ไม่มี และหน้าจอจะพิมพ์ช่องว่างแทน "—"
func TestBlankOptionalsAreOmittedAndValuesTrimmed(t *testing.T) {
	req := ok()
	req.FullName = str("   ")
	req.Email = str("  somchai@penbun.local  ")

	vals, err := handler().validate(req)
	require.NoError(t, err)

	assert.NotContains(t, vals, "full_name")
	assert.Equal(t, "somchai@penbun.local", vals["email"])
	assert.Equal(t, "somchai", vals["user_name"])
}

// เทสต์ drift อ่านรายการนี้ไปเทียบกับ INFORMATION_SCHEMA
// ถ้าคอลัมน์หายไปจากรายการ การเขียนคอลัมน์นั้นจะไม่มีใครตรวจอีกเลย
func TestFieldSetsCoversEveryColumnWritten(t *testing.T) {
	req := ok()
	req.FullName = str("สมชาย เกษมสุข")
	req.Email = str("somchai@penbun.local")
	req.Remark = str("หัวหน้าคลัง")

	vals, err := handler().validate(req)
	require.NoError(t, err)

	declared := map[string]bool{}
	for _, f := range FieldSets()["tb_users"] {
		declared[f.Name] = true
	}
	for col := range vals {
		assert.True(t, declared[col], "คอลัมน์ %s ถูกเขียนแต่ไม่ได้ประกาศใน FieldSets", col)
	}
}
