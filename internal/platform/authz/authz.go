// Path: internal/platform/authz/authz.go
//
// authz คือชั้นที่ตัดสินว่าคำขอหนึ่งผ่านหรือไม่ผ่าน โดยถามฐานข้อมูล
//
// ก่อนหน้านี้คำตอบอยู่ใน tb_users.user_level สองค่า และเขียนไว้บนเส้นทางแต่ละเส้น
// เป็น mw.RequireLevel("ADMIN") ซึ่งพูดได้แค่ "ทั้งตารางนี้ ADMIN เท่านั้น"
// PenbunSQL v12 ย้ายคำตอบลงฐานเป็น tb_role · tb_user_role · tb_privilege
// ที่นี่คือฝั่งที่อ่านมันมาบังคับใช้จริง
//
// สิทธิ์หนึ่งแถวคือ resource หนึ่งตัว x สี่การกระทำ ชื่อ resource เป็นชื่อเดียวกับ
// crud.Resource.Name เส้นทางที่ mount กับแถวสิทธิ์จึงอ้างชื่อเดียวกันเสมอ
package authz

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
)

// Action คือสี่การกระทำที่ tb_privilege เก็บ ตรงกับช่องติ๊กบนหน้าจอ M002_P0004
type Action string

const (
	View   Action = "view"
	Insert Action = "insert"
	Update Action = "update"
	Delete Action = "delete"
)

// ActionFor แปลง HTTP method เป็นการกระทำ
//
// method ที่ไม่รู้จักคืน view ไม่ได้ แต่ต้องเป็นค่าที่ไม่มีใครมีสิทธิ์ — ถ้าเดาเป็น view
// method แปลก ๆ จะกลายเป็นช่องอ่านข้อมูลฟรี จึงคืน false ให้ผู้เรียกปฏิเสธไปเลย
func ActionFor(method string) (Action, bool) {
	switch method {
	case http.MethodGet, http.MethodHead:
		return View, true
	case http.MethodPost:
		return Insert, true
	case http.MethodPut, http.MethodPatch:
		return Update, true
	case http.MethodDelete:
		return Delete, true
	}
	return "", false
}

// Access คือสี่การกระทำต่อ resource หนึ่งตัว
type Access struct {
	View   bool
	Insert bool
	Update bool
	Delete bool
}

// Allows ตอบว่าการกระทำนี้ทำได้ไหม
//
// Action ที่ไม่รู้จักคืน false เสมอ ไม่ใช่ true — ค่าเริ่มต้นของการตัดสินสิทธิ์
// ต้องเป็นปฏิเสธ ไม่ใช่อนุญาต
func (a Access) Allows(act Action) bool {
	switch act {
	case View:
		return a.View
	case Insert:
		return a.Insert
	case Update:
		return a.Update
	case Delete:
		return a.Delete
	}
	return false
}

// Repo อ่านสิทธิ์จาก vw_user_privilege
type Repo struct{ db *repository.DB }

func NewRepo(db *repository.DB) *Repo { return &Repo{db: db} }

// For คืนสิทธิ์ทั้งชุดของผู้ใช้หนึ่งคน
//
// vw_user_privilege รวมสิทธิ์ข้ามบทบาทแบบ union ให้แล้ว ผู้ใช้ถือได้หลายบทบาท
// บทบาทใดให้ผ่านก็ผ่าน
//
// อ่านทั้งชุดครั้งเดียวแทนที่จะถามทีละ resource เพราะคำขอหนึ่งถามแค่ครั้งเดียว
// และ /auth/me ต้องการทั้งชุดอยู่แล้ว โค้ดอ่านสิทธิ์จึงมีทางเดียว
func (r *Repo) For(ctx context.Context, userID string) (map[string]Access, error) {
	const q = `
SELECT resource_code, can_view, can_insert, can_update, can_delete
  FROM dbo.vw_user_privilege
 WHERE user_id = @p1`

	rows, err := r.db.Exec().QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]Access, 32)
	for rows.Next() {
		var name string
		var v, i, u, d int
		if err := rows.Scan(&name, &v, &i, &u, &d); err != nil {
			return nil, err
		}
		out[name] = Access{View: v == 1, Insert: i == 1, Update: u == 1, Delete: d == 1}
	}
	return out, rows.Err()
}

// Guard คืน middleware ที่ตรวจสิทธิ์ของ resource หนึ่งตัวด้วยการกระทำที่ระบุตายตัว
//
// ใช้กับเส้นทางเดี่ยวที่ method ไม่ได้บอกความหมายตรงตัว เช่น PUT /users/{id}/unlock
// ซึ่งเป็นการแก้ผู้ใช้ ไม่ใช่การแก้ "unlock"
//
// resource ต้องเป็นชื่อที่มีแถวอยู่จริงใน tb_privilege — ชื่อที่พิมพ์ผิดจะไม่มีใคร
// มีสิทธิ์ ซึ่งเป็นผลที่ปลอดภัยแต่ตรวจไม่ได้ตอนคอมไพล์
// TestEveryGuardedResourceIsSeeded จึงตรวจให้แทน
func Guard(repo *Repo, resource string, action Action) fiber.Handler {
	return func(c fiber.Ctx) error {
		return check(c, repo, resource, action)
	}
}

// GuardResource คืน middleware ที่ตรวจสิทธิ์ของ resource หนึ่งตัว โดยอ่านการกระทำ
// จาก HTTP method ของคำขอ
//
// ใช้กับ Group ได้เพราะไม่ต้องรู้ล่วงหน้าว่าเส้นทางไหนเป็นอ่านหรือเขียน เส้นทางที่
// เพิ่มเข้ากลุ่มทีหลังจึงถูกครอบอัตโนมัติ ไม่หลุดเพราะมีคนลืมใส่ตัวกรอง
// ซึ่งเป็นสิ่งที่ mw.RequireLevel ทำไม่ได้ตอนที่การอ่านและการเขียนต้องใช้สิทธิ์ต่างกัน
func GuardResource(repo *Repo, resource string) fiber.Handler {
	return func(c fiber.Ctx) error {
		action, ok := ActionFor(c.Method())
		if !ok {
			return forbidden()
		}
		return check(c, repo, resource, action)
	}
}

// check คือการตัดสินจริง ทางเดียว ทั้ง Guard และ GuardResource เรียกตัวนี้
//
// อ่านฐานทุกคำขอโดยตั้งใจ ไม่แคช : สิทธิ์ที่ถูกเพิกถอนต้องมีผลทันที ไม่ใช่รอ TTL
// หมดอายุ และ query นี้อ่าน View ที่รวมแถวไม่ถึงร้อยแถว ถ้าวันหนึ่งวัดแล้วพบว่าแพง
// ค่อยใส่แคชอายุสั้นพร้อมทางล้างแคชตอนแก้สิทธิ์ ไม่ใช่ใส่ไว้ก่อนโดยไม่ได้วัด
func check(c fiber.Ctx, repo *Repo, resource string, action Action) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel()

	perms, err := repo.For(ctx, mw.UserID(c))
	if err != nil {
		// อ่านสิทธิ์ไม่ได้ ไม่ใช่ "ไม่มีสิทธิ์" และไม่ใช่ "มีสิทธิ์"
		// ปฏิเสธคำขอไว้ก่อน แต่บอกตามจริงว่าระบบมีปัญหา ไม่ใช่ 403
		// ที่จะทำให้คนไปไล่แก้สิทธิ์ทั้งที่ต้นเหตุอยู่ที่ฐานข้อมูล
		return err
	}
	if !perms[resource].Allows(action) {
		return forbidden()
	}
	return c.Next()
}

func forbidden() error {
	return httpx.Forbidden(httpx.CodeForbidden, "สิทธิ์ของคุณไม่เพียงพอสำหรับการดำเนินการนี้")
}
