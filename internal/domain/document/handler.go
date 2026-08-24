// Path: internal/domain/document/handler.go
package document

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/config"
	"penbun/api/internal/crud"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

type Engine struct {
	repo     *Repo
	db       *repository.DB
	resolver *repository.Resolver
	crud     *crud.Engine
	cfg      *config.Config
}

func NewEngine(db *repository.DB, res *repository.Resolver, ce *crud.Engine, cfg *config.Config) *Engine {
	return &Engine{repo: NewRepo(db), db: db, resolver: res, crud: ce, cfg: cfg}
}

// Mount ติดตั้ง 9 endpoint ของเอกสารหนึ่งชนิด
//
//	GET    /{name}                 list
//	GET    /{name}/{id}            header + items
//	POST   /{name}                 สร้าง (DRAFT) พร้อม items ในครั้งเดียว
//	PUT    /{name}/{id}            แก้ header       เฉพาะ DRAFT
//	PUT    /{name}/{id}/items      แทนที่ items      เฉพาะ DRAFT
//	PUT    /{name}/{id}/confirm    DRAFT → CONFIRMED
//	PUT    /{name}/{id}/post       เรียก Stored Procedure  เฉพาะ CONFIRMED
//	PUT    /{name}/{id}/cancel     → CANCELLED       เฉพาะ DRAFT / CONFIRMED
//	DELETE /{name}/{id}            soft delete       เฉพาะ DRAFT
func (e *Engine) Mount(router fiber.Router, s *Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	g := router.Group("/" + s.Name)

	g.Get("", e.list(s))
	g.Get("/:id", e.get(s))
	g.Post("", e.create(s))
	g.Put("/:id", e.updateHeader(s))
	g.Put("/:id/items", e.replaceItems(s))
	g.Put("/:id/confirm", e.confirm(s))
	g.Put("/:id/post", e.post(s))
	g.Put("/:id/cancel", e.cancel(s))
	g.Delete("/:id", e.softDelete(s))
	return nil
}

func (e *Engine) MountAll(router fiber.Router, specs []*Spec) error {
	for _, s := range specs {
		if err := e.Mount(router, s); err != nil {
			return fmt.Errorf("mount document %s: %w", s.Name, err)
		}
	}
	return nil
}

// ───────────────────────────── read ─────────────────────────────

type ListParams struct {
	Page, Limit int
	DocStatus   string
	DocNo       string
	DateFrom    string
	DateTo      string
	Filters     map[string]any
}

func (e *Engine) list(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		p := ListParams{
			Page:      fiber.Query[int](c, "page", 1),
			Limit:     fiber.Query[int](c, "limit", 50),
			DocStatus: c.Query("doc_status"),
			DocNo:     c.Query("doc_no"),
			DateFrom:  c.Query("date_from"),
			DateTo:    c.Query("date_to"),
			Filters:   map[string]any{},
		}
		if p.Page < 1 {
			p.Page = 1
		}
		if p.Limit < 1 || p.Limit > 200 {
			p.Limit = 50
		}
		for _, f := range s.Filters {
			if v := c.Query(f.Param); v != "" {
				p.Filters[f.Param] = v
			}
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
		defer cancel()

		items, total, err := e.repo.List(ctx, e.db.Exec(), s, p)
		if err != nil {
			return err
		}
		return httpx.OK(c, fmt.Sprintf("พบ%s %d ใบ", s.Label, total),
			httpx.NewPage(items, p.Page, p.Limit, total))
	}
}

func (e *Engine) get(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
		defer cancel()

		return e.respond(c, ctx, s, id, fiber.StatusOK, "พบ"+s.Label)
	}
}

// ───────────────────────────── create ─────────────────────────────

func (e *Engine) create(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		body, err := crud.DecodeBody(c)
		if err != nil {
			return err
		}
		if err := rejectReserved(s, body); err != nil {
			return err
		}

		vals, err := collectFields(s.HeaderFields, body, true)
		if err != nil {
			return err
		}
		rawItems, err := decodeItems(body)
		if err != nil {
			return err
		}
		if len(rawItems) == 0 {
			return httpx.Validation(s.Label + "ต้องมีรายการอย่างน้อย 1 บรรทัด")
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutPostDoc)
		defer cancel()

		user := mw.Username(c)
		var businessID string

		err = e.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			refs, err := e.crud.ResolveRefs(ctx, tx, s.HeaderRefs, body, true)
			if err != nil {
				return err
			}
			if s.DefaultsFrom != nil {
				if err := s.DefaultsFrom(vals, refs); err != nil {
					return err
				}
			}
			if err := checkRequired(s.HeaderFields, vals); err != nil {
				return err
			}

			items, err := e.resolveItems(ctx, tx, s, rawItems)
			if err != nil {
				return err
			}
			if s.ValidateHeader != nil {
				if err := s.ValidateHeader(vals, items); err != nil {
					return err
				}
			}

			headerAuto, err := e.repo.InsertHeader(ctx, tx, s, vals, refs, user)
			if err != nil {
				return err
			}
			if err := e.repo.InsertItems(ctx, tx, s, headerAuto, items, user); err != nil {
				return err
			}
			if err := e.repo.RecalcTotals(ctx, tx, s, headerAuto, user); err != nil {
				return err
			}

			businessID, err = e.repo.BusinessIDByAuto(ctx, tx, s, headerAuto)
			return err
		})
		if err != nil {
			return err
		}

		return e.respond(c, ctx, s, businessID, fiber.StatusCreated, "สร้าง"+s.Label+"เรียบร้อย")
	}
}

// ───────────────────────────── update ─────────────────────────────

func (e *Engine) updateHeader(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		body, err := crud.DecodeBody(c)
		if err != nil {
			return err
		}
		if err := rejectReserved(s, body); err != nil {
			return err
		}
		vals, err := collectFields(s.HeaderFields, body, false)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		user := mw.Username(c)

		err = e.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			meta, err := e.mustMeta(ctx, tx, s, id)
			if err != nil {
				return err
			}
			if err := requireDraft(s, meta, "แก้ไขหัวเอกสาร"); err != nil {
				return err
			}

			refs, err := e.crud.ResolveRefs(ctx, tx, s.HeaderRefs, body, false)
			if err != nil {
				return err
			}
			if len(vals) == 0 && len(refs) == 0 {
				return httpx.Validation("ไม่มีข้อมูลที่ต้องแก้ไข")
			}

			n, err := e.repo.UpdateHeader(ctx, tx, s, meta.AutoID, vals, refs, user)
			if err != nil {
				return err
			}
			if n == 0 {
				return httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", s.Label, id))
			}
			return nil
		})
		if err != nil {
			return err
		}

		return e.respond(c, ctx, s, id, fiber.StatusOK, "แก้ไข"+s.Label+"เรียบร้อย")
	}
}

// replaceItems แทนที่รายการทั้งใบ
//
// ไม่มี endpoint แก้รายการทีละบรรทัด เพราะยอดรวมของ header ต้องคำนวณใหม่ทุกครั้ง
// การแทนที่ทั้งชุดทำให้สถานะสุดท้ายชัดเจนและ retry ได้โดยไม่มีผลข้างเคียง
func (e *Engine) replaceItems(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		body, err := crud.DecodeBody(c)
		if err != nil {
			return err
		}
		rawItems, err := decodeItems(body)
		if err != nil {
			return err
		}
		if len(rawItems) == 0 {
			return httpx.Validation("ต้องมีรายการอย่างน้อย 1 บรรทัด")
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutPostDoc)
		defer cancel()

		user := mw.Username(c)

		err = e.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			meta, err := e.mustMeta(ctx, tx, s, id)
			if err != nil {
				return err
			}
			if err := requireDraft(s, meta, "แก้ไขรายการ"); err != nil {
				return err
			}

			items, err := e.resolveItems(ctx, tx, s, rawItems)
			if err != nil {
				return err
			}
			if err := e.repo.DeleteItems(ctx, tx, s, meta.AutoID, user); err != nil {
				return err
			}
			if err := e.repo.InsertItems(ctx, tx, s, meta.AutoID, items, user); err != nil {
				return err
			}
			return e.repo.RecalcTotals(ctx, tx, s, meta.AutoID, user)
		})
		if err != nil {
			return err
		}

		return e.respond(c, ctx, s, id, fiber.StatusOK, "แก้ไขรายการเรียบร้อย")
	}
}

// ─────────────────────── state transitions ───────────────────────

func (e *Engine) confirm(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		meta, err := e.mustMeta(ctx, e.db.Exec(), s, id)
		if err != nil {
			return err
		}
		if meta.DocStatus != StatusDraft {
			return httpx.Unprocessable(httpx.CodeBusinessRule, fmt.Sprintf(
				"ยืนยันได้เฉพาะเอกสารสถานะ DRAFT (ปัจจุบัน %s)", meta.DocStatus))
		}
		if meta.ItemCount == 0 {
			return httpx.Unprocessable(httpx.CodeBusinessRule, s.Label+"ไม่มีรายการ")
		}

		n, err := e.repo.SetStatus(ctx, e.db.Exec(), s, meta.AutoID,
			StatusDraft, StatusConfirmed, mw.Username(c))
		if err != nil {
			return err
		}
		if n == 0 {
			return httpx.Conflict(httpx.CodeBusinessRule, "สถานะเอกสารเปลี่ยนไปแล้ว กรุณาโหลดใหม่")
		}
		return e.respond(c, ctx, s, id, fiber.StatusOK, "ยืนยัน"+s.Label+"เรียบร้อย")
	}
}

// post เรียก Stored Procedure ที่ทำให้เอกสารมีผลต่อสต็อกจริง
//
// ก่อนเรียก proc ต้องจอง application lock ของทุกคู่ (SKU, คลัง) ที่เอกสารนี้แตะ
//
// เหตุผล: การตรวจว่าสต็อกพอหรือไม่ กับการหักสต็อกจริง เป็นคนละคำสั่งกัน
// ถ้าเอกสารสองใบที่กินสินค้าตัวเดียวกันจากคลังเดียวกันเข้ามาพร้อมกัน ทั้งคู่จะ
// อ่านยอดคงเหลือค่าเดิม ผ่านการตรวจทั้งคู่ แล้วหักซ้อนกันจนสต็อกติดลบทั้งที่
// คลังไม่ได้เปิด allow_negative_stock
//
// จองล็อกเรียงตาม autoID จากน้อยไปมากเสมอ ถ้าสองใบจองสลับลำดับกันจะ deadlock
func (e *Engine) post(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c, repository.TimeoutPostDoc)
		defer cancel()

		user := mw.Username(c)

		err := e.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			meta, err := e.mustMeta(ctx, tx, s, id)
			if err != nil {
				return err
			}
			if s.isPosted(meta.DocStatus) {
				return httpx.Conflict(httpx.CodeAlreadyPosted, fmt.Sprintf(
					"%s %s ถูกโพสต์ไปแล้ว (สถานะ %s)", s.Label, id, meta.DocStatus))
			}
			if meta.DocStatus != StatusConfirmed {
				return httpx.Unprocessable(httpx.CodeBusinessRule, fmt.Sprintf(
					"%s %s ต้องอยู่สถานะ CONFIRMED ก่อนโพสต์ (ปัจจุบัน %s)",
					s.Label, id, meta.DocStatus))
			}
			if meta.ItemCount == 0 {
				return httpx.Unprocessable(httpx.CodeBusinessRule, s.Label+"ไม่มีรายการ")
			}

			pairs, err := e.repo.LockPairs(ctx, tx, s, meta.AutoID)
			if err != nil {
				return err
			}
			for _, p := range pairs {
				key := fmt.Sprintf("PENBUN:STOCK:%d:%d", p.SkuAuto, p.WarehouseAuto)
				if err := repository.AppLock(ctx, tx, key, e.cfg.AppLockWaitMS); err != nil {
					return err
				}
			}

			return e.repo.Post(ctx, tx, s, id, user)
		})
		if err != nil {
			return err
		}

		return e.respond(c, ctx, s, id, fiber.StatusOK,
			fmt.Sprintf("โพสต์%sเรียบร้อย (สถานะ %s)", s.Label, s.PostedStatus))
	}
}

// cancel ยกเลิกเอกสารที่ยังไม่ได้โพสต์
//
// เอกสารที่โพสต์แล้วยกเลิกไม่ได้ เพราะบัญชีการเคลื่อนไหวสต็อกเป็นแบบเพิ่มอย่างเดียว
// การลบผลย้อนหลังจะทำให้ยอดคงเหลือกับประวัติไม่ตรงกันถาวร ต้องออกเอกสาร
// กลับรายการแทน
func (e *Engine) cancel(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		meta, err := e.mustMeta(ctx, e.db.Exec(), s, id)
		if err != nil {
			return err
		}
		if meta.DocStatus != StatusDraft && meta.DocStatus != StatusConfirmed {
			return httpx.Unprocessable(httpx.CodeBusinessRule, fmt.Sprintf(
				"ยกเลิกได้เฉพาะเอกสารสถานะ DRAFT หรือ CONFIRMED (ปัจจุบัน %s) "+
					"เอกสารที่โพสต์แล้วต้องออกเอกสารกลับรายการแทน", meta.DocStatus))
		}

		n, err := e.repo.SetStatus(ctx, e.db.Exec(), s, meta.AutoID,
			meta.DocStatus, StatusCancelled, mw.Username(c))
		if err != nil {
			return err
		}
		if n == 0 {
			return httpx.Conflict(httpx.CodeBusinessRule, "สถานะเอกสารเปลี่ยนไปแล้ว กรุณาโหลดใหม่")
		}
		return e.respond(c, ctx, s, id, fiber.StatusOK, "ยกเลิก"+s.Label+"เรียบร้อย")
	}
}

func (e *Engine) softDelete(s *Spec) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		meta, err := e.mustMeta(ctx, e.db.Exec(), s, id)
		if err != nil {
			return err
		}
		if err := requireDraft(s, meta, "ลบ"); err != nil {
			return err
		}

		if _, err := e.repo.SoftDeleteHeader(ctx, e.db.Exec(), s, meta.AutoID, mw.Username(c)); err != nil {
			return err
		}
		return httpx.NoContent(c)
	}
}

// ───────────────────────────── helpers ─────────────────────────────

func (e *Engine) respond(c fiber.Ctx, ctx context.Context, s *Spec, id string, status int, msg string) error {
	header, err := e.repo.HeaderView(ctx, e.db.Exec(), s, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", s.Label, id))
		}
		return err
	}
	items, err := e.repo.ItemsView(ctx, e.db.Exec(), s, id)
	if err != nil {
		return err
	}

	data := fiber.Map{"header": header, "items": items}
	if status == fiber.StatusCreated {
		return httpx.Created(c, msg, data)
	}
	return httpx.OK(c, msg, data)
}

func (e *Engine) mustMeta(ctx context.Context, ex repository.Executor, s *Spec, id string) (*Meta, error) {
	meta, err := e.repo.MetaByID(ctx, ex, s, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", s.Label, id))
		}
		return nil, err
	}
	return meta, nil
}

func requireDraft(s *Spec, meta *Meta, action string) error {
	if meta.DocStatus != StatusDraft {
		return httpx.Unprocessable(httpx.CodeBusinessRule, fmt.Sprintf(
			"%sได้เฉพาะเอกสารสถานะ DRAFT (ปัจจุบัน %s)", action, meta.DocStatus))
	}
	return nil
}

func decodeItems(body map[string]json.RawMessage) ([]map[string]json.RawMessage, error) {
	raw, ok := body["items"]
	if !ok {
		return nil, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, httpx.Validation("items ต้องเป็น array ของ object").
			WithField("items", httpx.CodeValidationFailed, "")
	}
	return items, nil
}

// resolveItems แปลงรายการดิบเป็น Item ที่พร้อมเขียนลงฐาน
func (e *Engine) resolveItems(
	ctx context.Context, tx *sql.Tx, s *Spec, raw []map[string]json.RawMessage,
) ([]Item, error) {
	out := make([]Item, 0, len(raw))
	seenLine := map[int]bool{}

	// สินค้าตัวเดียวกันโผล่ได้หลายบรรทัด จำผลไว้ในรอบนี้เพื่อไม่ให้ query ซ้ำ
	cache := map[string]int{}

	for i, item := range raw {
		lineNo := i + 1
		if v, ok := item["line_no"]; ok {
			var n int
			if err := json.Unmarshal(v, &n); err != nil {
				return nil, httpx.Validation(fmt.Sprintf("บรรทัดที่ %d: line_no ต้องเป็นจำนวนเต็ม", i+1))
			}
			if n > 0 {
				lineNo = n
			}
		}
		if seenLine[lineNo] {
			return nil, httpx.Validation(fmt.Sprintf("line_no %d ซ้ำกัน", lineNo))
		}
		seenLine[lineNo] = true

		vals, err := collectFieldsPrefixed(s.ItemFields, item, true, fmt.Sprintf("items[%d].", i))
		if err != nil {
			return nil, err
		}
		if err := checkRequiredPrefixed(s.ItemFields, vals, fmt.Sprintf("บรรทัดที่ %d: ", lineNo)); err != nil {
			return nil, err
		}

		refs := make(map[string]int, len(s.ItemRefs))
		for _, ref := range s.ItemRefs {
			rawVal, present := item[ref.Field]
			if !present || string(rawVal) == "null" {
				if ref.Required {
					return nil, httpx.BadRequest(httpx.CodeFieldRequired,
						fmt.Sprintf("บรรทัดที่ %d: ต้องระบุ%s", lineNo, ref.Label)).
						WithField(fmt.Sprintf("items[%d].%s", i, ref.Field), httpx.CodeFieldRequired, "")
				}
				continue
			}

			var businessID string
			if err := json.Unmarshal(rawVal, &businessID); err != nil {
				return nil, httpx.Validation(fmt.Sprintf(
					"บรรทัดที่ %d: %s ต้องเป็นรหัสอ้างอิงในรูปแบบข้อความ", lineNo, ref.Label))
			}

			cacheKey := ref.Table + "\x00" + businessID
			autoID, hit := cache[cacheKey]
			if !hit {
				resolved, err := e.resolver.ResolveTx(ctx, tx, ref.Table, businessID)
				if err != nil {
					if errors.Is(err, repository.ErrRefNotFound) {
						return nil, httpx.RefNotFound(
							fmt.Sprintf("items[%d].%s", i, ref.Field), businessID, ref.Label)
					}
					return nil, err
				}
				autoID = resolved
				cache[cacheKey] = autoID
			}
			refs[ref.Column] = autoID
		}

		out = append(out, Item{LineNo: lineNo, Refs: refs, Values: vals})
	}
	return out, nil
}

func rejectReserved(s *Spec, body map[string]json.RawMessage) error {
	for k := range body {
		switch k {
		case "autoID", "auto_id", "prefix", "update_by", "update_date", "is_delete", "id_status":
			return httpx.Validation(fmt.Sprintf("ห้ามส่งฟิลด์ '%s' — ระบบเป็นผู้กำหนด", k)).
				WithField(k, httpx.CodeValidationFailed, "")
		case "doc_status":
			// สถานะเปลี่ยนได้ผ่าน confirm / post / cancel เท่านั้น
			return httpx.Validation(
				"ห้ามส่งฟิลด์ 'doc_status' — ใช้ /confirm, /post หรือ /cancel แทน").
				WithField(k, httpx.CodeValidationFailed, "")
		case s.IDColumn, s.AutoColumn:
			return httpx.Validation(fmt.Sprintf("ห้ามส่งฟิลด์ '%s' — ระบบสร้างรหัสให้อัตโนมัติ", k)).
				WithField(k, httpx.CodeValidationFailed, "")
		}
		if strings.HasPrefix(k, "total_") || k == "net_amount" {
			return httpx.Validation(fmt.Sprintf(
				"ห้ามส่งฟิลด์ '%s' — ระบบคำนวณยอดรวมจากรายการจริง", k)).
				WithField(k, httpx.CodeValidationFailed, "")
		}
	}
	return nil
}

func collectFields(fields []schema.Field, body map[string]json.RawMessage, isInsert bool) (map[string]any, error) {
	return collectFieldsPrefixed(fields, body, isInsert, "")
}

func collectFieldsPrefixed(
	fields []schema.Field, body map[string]json.RawMessage, isInsert bool, prefix string,
) (map[string]any, error) {
	out := make(map[string]any, len(body))
	for _, f := range fields {
		raw, present := body[f.Name]
		if !present {
			continue
		}
		if isInsert && f.NoInsert {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' กำหนดตอนสร้างไม่ได้", prefix+f.Name)).
				WithField(prefix+f.Name, httpx.CodeValidationFailed, "")
		}
		if !isInsert && f.NoUpdate {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' แก้ไขภายหลังไม่ได้", prefix+f.Name)).
				WithField(prefix+f.Name, httpx.CodeValidationFailed, "")
		}

		v, err := schema.Coerce(raw, f.Kind, prefix+f.Name, f.DisplayLabel())
		if err != nil {
			return nil, err
		}
		if err := schema.Validate(f, v); err != nil {
			return nil, err
		}
		out[f.ColumnName()] = v
	}
	return out, nil
}

func checkRequired(fields []schema.Field, vals map[string]any) error {
	return checkRequiredPrefixed(fields, vals, "")
}

func checkRequiredPrefixed(fields []schema.Field, vals map[string]any, msgPrefix string) error {
	for _, f := range fields {
		if !f.Required {
			continue
		}
		v, ok := vals[f.ColumnName()]
		if !ok || v == nil || v == "" {
			return httpx.BadRequest(httpx.CodeFieldRequired, msgPrefix+"ต้องระบุ"+f.DisplayLabel()).
				WithField(f.Name, httpx.CodeFieldRequired, "")
		}
	}
	return nil
}
