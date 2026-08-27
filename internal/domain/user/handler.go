// Path: internal/domain/user/handler.go
package user

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"penbun/api/internal/config"
	"penbun/api/internal/crud"
	"penbun/api/internal/domain/auth"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

// การอ่าน (GET /users) ใช้ generic CRUD engine ผ่าน descriptor ที่ตั้ง ReadOnly ไว้
// package นี้รับผิดชอบการสร้างผู้ใช้ ซึ่งเกินขอบเขตของ engine กลางอยู่สองเรื่อง
//
//	รหัสผ่าน  ต้องผ่าน bcrypt ก่อนลงตาราง engine เขียนค่าที่รับมาลงคอลัมน์ตรง ๆ
//	user_level  ตัดสินสิทธิ์ของทั้ง API การเปิดให้เขียนผ่าน descriptor แปลว่า
//	            ใครก็ตามที่แก้ผู้ใช้ได้ ยกสิทธิ์ตัวเองเป็น ADMIN ได้ด้วย
type Handler struct {
	db       *repository.DB
	resolver *repository.Resolver
	crud     *crud.Engine
	cfg      *config.Config
}

func NewHandler(db *repository.DB, res *repository.Resolver, ce *crud.Engine, cfg *config.Config) *Handler {
	return &Handler{db: db, resolver: res, crud: ce, cfg: cfg}
}

// Register ต่อท้ายกลุ่ม /users ที่ crud engine ติดตั้งไว้แล้ว
// ตัวกรอง RequireLevel("ADMIN") ของกลุ่มนั้นครอบเส้นทางนี้ด้วย และยังใส่ซ้ำตรงนี้
// เพื่อให้อ่านโค้ดแล้วเห็นสิทธิ์ที่ต้องใช้โดยไม่ต้องไปเปิด descriptor
func (h *Handler) Register(api fiber.Router) {
	api.Post("/users", mw.RequireLevel("ADMIN"), h.create)
}

// levels คือค่าที่ tb_users.user_level รับได้จริงในรุ่นนี้
// รอ tb_role ของ PenbunSQL แล้วรายการนี้จะย้ายไปอยู่ในฐานข้อมูล
var levels = map[string]bool{"ADMIN": true, "USER": true}

// ชื่อผู้ใช้ไปอยู่ในคอลัมน์ update_by ของทุกตารางในระบบ จึงจำกัดให้เป็นอักษรละติน
// ตัวเลข จุด ขีดกลางและขีดล่าง เพื่อไม่ให้บันทึกการแก้ไขอ่านยากหรือคัดลอกผิด
var userNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

var warehouseRef = []schema.Ref{
	{Field: "warehouse_id", Table: "tb_warehouse", Column: "ref_warehouse_auto",
		Label: "คลังประจำตัว"},
}

type createRequest struct {
	UserName  string  `json:"user_name"`
	Password  string  `json:"password"`
	FullName  *string `json:"full_name"`
	Email     *string `json:"email"`
	UserLevel string  `json:"user_level"`
	Remark    *string `json:"remark"`
}

func (h *Handler) create(c fiber.Ctx) error {
	var req createRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}

	// crud.DecodeBody อ่าน body ซ้ำอีกครั้งเพื่อส่งให้ ResolveRefs
	// ซึ่งรับ map ของ raw JSON ไม่ใช่ struct
	body, err := crud.DecodeBody(c)
	if err != nil {
		return err
	}

	vals, err := h.validate(&req)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	actor := mw.Username(c)
	var userID string

	err = h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		refs, err := h.crud.ResolveRefs(ctx, tx, warehouseRef, body, true)
		if err != nil {
			return err
		}

		a := &schema.Args{}
		cols := make([]string, 0, len(vals)+len(refs)+1)
		phs := make([]string, 0, len(vals)+len(refs)+1)
		for _, n := range schema.SortedKeys(vals) {
			cols = append(cols, n)
			phs = append(phs, a.Add(vals[n]))
		}
		for _, n := range schema.SortedKeys(refs) {
			cols = append(cols, n)
			phs = append(phs, a.Add(refs[n]))
		}
		cols = append(cols, "update_by")
		phs = append(phs, a.Add(actor))

		var autoID int
		q := repository.InsertReturningAuto("tb_users", cols, phs)
		if err := tx.QueryRowContext(ctx, q, a.Values()...).Scan(&autoID); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx,
			"SELECT user_id FROM dbo.tb_users WHERE autoID = @p1", autoID).Scan(&userID)
	})
	if err != nil {
		return err
	}

	ctx2, cancel2 := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel2()

	row, err := repository.QueryOne(ctx2, h.db.Exec(),
		"SELECT TOP 1 * FROM dbo.vw_users WHERE user_id = @p1", []any{userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound("ไม่พบผู้ใช้ '" + userID + "'")
		}
		return err
	}
	return httpx.Created(c, "เพิ่มผู้ใช้เรียบร้อย", row)
}

// validate คืนคอลัมน์ที่พร้อมเขียนลง tb_users
//
// status_change_pw ไม่อยู่ในนี้โดยตั้งใจ — DEFAULT ของตารางคือ 1 ผู้ใช้ใหม่จึงถูก
// บังคับให้เปลี่ยนรหัสผ่านที่ผู้ดูแลตั้งให้ตั้งแต่การเข้าสู่ระบบครั้งแรกเสมอ
func (h *Handler) validate(req *createRequest) (map[string]any, error) {
	name := strings.TrimSpace(req.UserName)
	if name == "" {
		return nil, httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุชื่อผู้ใช้").
			WithField("user_name", httpx.CodeFieldRequired, "")
	}
	if !userNamePattern.MatchString(name) {
		return nil, httpx.Validation(
			"ชื่อผู้ใช้ใช้ได้เฉพาะ a-z A-Z 0-9 . _ - ยาว 3-50 ตัวอักษร").
			WithField("user_name", httpx.CodeValidationFailed, "")
	}

	level := strings.ToUpper(strings.TrimSpace(req.UserLevel))
	if level == "" {
		return nil, httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุสิทธิ์การใช้งาน").
			WithField("user_level", httpx.CodeFieldRequired, "")
	}
	if !levels[level] {
		return nil, httpx.Validation("สิทธิ์การใช้งานต้องเป็น ADMIN หรือ USER").
			WithField("user_level", httpx.CodeValidationFailed, "")
	}

	if req.Password == "" {
		return nil, httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุรหัสผ่านเริ่มต้น").
			WithField("password", httpx.CodeFieldRequired, "")
	}
	if err := auth.ValidatePasswordPolicy(req.Password); err != nil {
		if ae, ok := err.(*httpx.AppError); ok {
			return nil, ae.WithField("password", httpx.CodeValidationFailed, "")
		}
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}

	vals := map[string]any{
		"user_name":     name,
		"user_password": string(hash),
		"user_level":    level,
	}
	if err := optional(vals, "full_name", req.FullName, 150, "ชื่อ–สกุล"); err != nil {
		return nil, err
	}
	if err := optional(vals, "email", req.Email, 100, "อีเมล"); err != nil {
		return nil, err
	}
	if err := optional(vals, "remark", req.Remark, 255, "หมายเหตุ"); err != nil {
		return nil, err
	}
	return vals, nil
}

// optional เก็บค่าที่ส่งมาและไม่ว่าง ค่าที่ว่างถูกข้ามไปเพื่อให้คอลัมน์เป็น NULL
// ตาม DEFAULT ของตาราง แทนที่จะเป็นสตริงว่างที่ดูเหมือนมีค่าแต่ไม่มี
func optional(vals map[string]any, col string, v *string, maxLen int, label string) error {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	if len([]rune(s)) > maxLen {
		return httpx.Validation(label+" ยาวเกิน "+strconv.Itoa(maxLen)+" ตัวอักษร").
			WithField(col, httpx.CodeValidationFailed, "")
	}
	vals[col] = s
	return nil
}
