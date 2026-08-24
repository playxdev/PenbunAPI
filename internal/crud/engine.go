// Path: internal/crud/engine.go
package crud

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

type Engine struct {
	DB       *repository.DB
	Resolver *repository.Resolver
}

func NewEngine(db *repository.DB, res *repository.Resolver) *Engine {
	return &Engine{DB: db, Resolver: res}
}

// Mount ติดตั้ง 5 endpoint มาตรฐานของ resource หนึ่งตัว
//
//	GET    /{name}          list + search + page
//	GET    /{name}/{id}     เดี่ยว
//	POST   /{name}          สร้าง
//	PUT    /{name}/{id}     partial update
//	DELETE /{name}/{id}     soft delete
func (e *Engine) Mount(router fiber.Router, r *Resource) error {
	if err := r.Validate(); err != nil {
		return err
	}
	g := router.Group("/" + r.Name)

	g.Get("", e.list(r))
	g.Get("/:id", e.getByID(r))

	if !r.ReadOnly {
		g.Post("", e.create(r))
		g.Put("/:id", e.update(r))
		g.Delete("/:id", e.softDelete(r))
	}
	return nil
}

func (e *Engine) MountAll(router fiber.Router, resources []*Resource) error {
	for _, r := range resources {
		if err := e.Mount(router, r); err != nil {
			return fmt.Errorf("mount %s: %w", r.Name, err)
		}
	}
	return nil
}

// ───────────────────────────── read ─────────────────────────────

func (e *Engine) list(r *Resource) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, err := r.parseListParams(c)
		if err != nil {
			return err
		}

		listSQL, countSQL, listArgs, countArgs := r.buildList(p)

		ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
		defer cancel()

		var total int64
		if err := e.DB.Exec().QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return err
		}

		rows, err := e.DB.Exec().QueryContext(ctx, listSQL, listArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		items, err := repository.ScanRows(rows)
		if err != nil {
			return err
		}

		return httpx.OK(c, fmt.Sprintf("พบข้อมูล%s %d รายการ", r.Label, total),
			httpx.NewPage(items, p.Page, p.Limit, total))
	}
}

func (e *Engine) getByID(r *Resource) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		q, qa := r.buildGetByID(id)

		ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
		defer cancel()

		row, err := repository.QueryOne(ctx, e.DB.Exec(), q, qa)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", r.Label, id))
			}
			return err
		}
		return httpx.OK(c, "พบ"+r.Label, row)
	}
}

// ───────────────────────────── write ─────────────────────────────

func (e *Engine) create(r *Resource) fiber.Handler {
	return func(c fiber.Ctx) error {
		body, err := DecodeBody(c)
		if err != nil {
			return err
		}
		if err := r.rejectReservedKeys(body); err != nil {
			return err
		}

		vals, err := r.collectFields(body, true)
		if err != nil {
			return err
		}
		if err := r.checkRequired(vals); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		var created repository.Row
		err = e.DB.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			refs, err := e.ResolveRefs(ctx, tx, r.Refs, body, true)
			if err != nil {
				return err
			}

			insSQL, insArgs := r.buildInsert(vals, refs, mw.Username(c))
			var autoID int
			if err := tx.QueryRowContext(ctx, insSQL, insArgs...).Scan(&autoID); err != nil {
				return err
			}

			// อ่านกลับจาก Source ในทรานแซกชันเดียวกันเพื่อให้ได้ Business ID
			// ที่ Trigger เพิ่งเติมให้ (ดูเหตุผลใน buildInsert)
			selSQL, selArgs := r.buildGetByAuto(autoID)
			row, err := repository.QueryOne(ctx, tx, selSQL, selArgs)
			if err != nil {
				return err
			}
			created = row
			return nil
		})
		if err != nil {
			return err
		}

		e.Resolver.Invalidate(r.Table)
		return httpx.Created(c, fmt.Sprintf("เพิ่ม%sเรียบร้อย", r.Label), created)
	}
}

func (e *Engine) update(r *Resource) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		body, err := DecodeBody(c)
		if err != nil {
			return err
		}
		if err := r.rejectReservedKeys(body); err != nil {
			return err
		}

		vals, err := r.collectFields(body, false)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		var updated repository.Row
		err = e.DB.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
			refs, err := e.ResolveRefs(ctx, tx, r.Refs, body, false)
			if err != nil {
				return err
			}
			if len(vals) == 0 && len(refs) == 0 {
				return httpx.Validation("ไม่มีข้อมูลที่ต้องแก้ไข")
			}

			updSQL, updArgs := r.buildUpdate(id, vals, refs, mw.Username(c))
			res, err := tx.ExecContext(ctx, updSQL, updArgs...)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", r.Label, id))
			}

			selSQL, selArgs := r.buildGetByID(id)
			row, err := repository.QueryOne(ctx, tx, selSQL, selArgs)
			if err != nil {
				return err
			}
			updated = row
			return nil
		})
		if err != nil {
			return err
		}

		e.Resolver.Invalidate(r.Table)
		return httpx.OK(c, fmt.Sprintf("แก้ไข%sเรียบร้อย", r.Label), updated)
	}
}

func (e *Engine) softDelete(r *Resource) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		q, qa := r.buildSoftDelete(id, mw.Username(c))

		ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
		defer cancel()

		res, err := e.DB.Exec().ExecContext(ctx, q, qa...)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return httpx.NotFound(fmt.Sprintf("ไม่พบ%s '%s'", r.Label, id))
		}

		e.Resolver.Invalidate(r.Table)
		return httpx.NoContent(c)
	}
}

// ───────────────────────── request binding ─────────────────────────

// DecodeBody อ่าน body เป็น map เพื่อให้แยกได้ว่าฟิลด์ไหน "ไม่ได้ส่งมา"
// ต่างจาก "ส่งมาเป็น null" ซึ่ง struct binding ธรรมดาแยกไม่ออก
func DecodeBody(c fiber.Ctx) (map[string]json.RawMessage, error) {
	raw := c.Body()
	if len(raw) == 0 {
		return nil, httpx.Validation("ไม่พบข้อมูลใน request body")
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, httpx.Validation("request body ไม่ใช่ JSON object ที่ถูกต้อง")
	}
	return body, nil
}

// reservedKeys คือคอลัมน์ที่ฐานข้อมูลเป็นเจ้าของ client ส่งมาไม่ได้
var reservedKeys = map[string]string{
	"auto_id":     "ระบบเป็นผู้กำหนด",
	"autoID":      "ระบบเป็นผู้กำหนด",
	"prefix":      "ระบบเป็นผู้กำหนด",
	"update_by":   "ระบบดึงจากผู้ใช้ที่เข้าสู่ระบบ",
	"update_date": "ระบบเป็นผู้กำหนด",
	"is_delete":   "ใช้ DELETE endpoint แทน",
	"id_status":   "ระบบเป็นผู้กำหนด",
}

// rejectReservedKeys ปฏิเสธแทนที่จะเมินเฉย
//
// การเมินเฉยทำให้ client เชื่อว่าค่าที่ส่งไปมีผล แล้วไปเจอความจริงตอนขึ้น production
func (r *Resource) rejectReservedKeys(body map[string]json.RawMessage) error {
	for k := range body {
		if reason, bad := reservedKeys[k]; bad {
			return httpx.Validation(fmt.Sprintf("ห้ามส่งฟิลด์ '%s' — %s", k, reason)).
				WithField(k, httpx.CodeValidationFailed, "")
		}
		if k == r.IDColumn || k == r.AutoColumn {
			return httpx.Validation(fmt.Sprintf("ห้ามส่งฟิลด์ '%s' — ระบบสร้างรหัสให้อัตโนมัติ", k)).
				WithField(k, httpx.CodeValidationFailed, "")
		}
	}
	return nil
}

func (r *Resource) collectFields(body map[string]json.RawMessage, isInsert bool) (map[string]any, error) {
	out := make(map[string]any, len(body))
	for key, raw := range body {
		f, ok := r.field(key)
		if !ok {
			if _, isRef := r.ref(key); isRef {
				continue // ref จัดการแยกใน ResolveRefs
			}
			continue // ฟิลด์ที่ไม่รู้จักถูกเมิน ไม่ทำให้ request ล้ม
		}
		if isInsert && f.NoInsert {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' กำหนดตอนสร้างไม่ได้", key)).
				WithField(key, httpx.CodeValidationFailed, "")
		}
		if !isInsert && f.NoUpdate {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' แก้ไขภายหลังไม่ได้", key)).
				WithField(key, httpx.CodeValidationFailed, "")
		}

		v, err := schema.Coerce(raw, f.Kind, f.Name, f.DisplayLabel())
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

func (r *Resource) checkRequired(vals map[string]any) error {
	for _, f := range r.Fields {
		if !f.Required {
			continue
		}
		v, ok := vals[f.ColumnName()]
		if !ok || v == nil || v == "" {
			return httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุ"+f.DisplayLabel()).
				WithField(f.Name, httpx.CodeFieldRequired, "")
		}
	}
	return nil
}

// ResolveRefs แปลง Business ID ทุกตัวใน body เป็น autoID
// เปิดเป็น method สาธารณะเพราะ document engine ใช้ตัวเดียวกัน
func (e *Engine) ResolveRefs(
	ctx context.Context, tx *sql.Tx, refs []schema.Ref,
	body map[string]json.RawMessage, isInsert bool,
) (map[string]int, error) {
	out := make(map[string]int, len(refs))

	for _, ref := range refs {
		raw, present := body[ref.Field]
		if !present || string(raw) == "null" {
			if isInsert && ref.Required {
				return nil, httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุ"+ref.Label).
					WithField(ref.Field, httpx.CodeFieldRequired, "")
			}
			continue
		}
		if !isInsert && ref.NoUpdate {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' แก้ไขภายหลังไม่ได้", ref.Field)).
				WithField(ref.Field, httpx.CodeValidationFailed, "")
		}

		var businessID string
		if err := json.Unmarshal(raw, &businessID); err != nil {
			return nil, httpx.Validation(ref.Label+" ต้องเป็นรหัสอ้างอิงในรูปแบบข้อความ").
				WithField(ref.Field, httpx.CodeValidationFailed, "")
		}

		autoID, err := e.Resolver.ResolveTx(ctx, tx, ref.Table, businessID)
		if err != nil {
			if errors.Is(err, repository.ErrRefNotFound) {
				return nil, httpx.RefNotFound(ref.Field, businessID, ref.Label)
			}
			return nil, err
		}
		out[ref.Column] = autoID
	}
	return out, nil
}

func (r *Resource) parseListParams(c fiber.Ctx) (ListParams, error) {
	p := ListParams{
		Page:    fiber.Query[int](c, "page", 1),
		Limit:   fiber.Query[int](c, "limit", defaultLimit),
		Search:  strings.TrimSpace(c.Query("q")),
		Filters: map[string]any{},
	}
	if p.Page < 1 {
		p.Page = 1
	}
	// clamp แทนการ error — การขอเกินโควตาไม่ใช่ความผิดที่ควรทำให้ request ล้ม
	if p.Limit < 1 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}

	sort := c.Query("sort")
	if strings.HasPrefix(sort, "-") {
		sort = sort[1:]
	} else if sort != "" {
		p.SortAsc = true
	}
	if sort != "" {
		if _, ok := r.SortColumns[sort]; !ok {
			allowed := schema.SortedKeys(r.SortColumns)
			return p, httpx.Validation(fmt.Sprintf(
				"เรียงตาม '%s' ไม่ได้ ใช้ได้เฉพาะ: %s", sort, strings.Join(allowed, ", ")))
		}
		p.Sort = sort
	}

	if v := c.Query("is_active"); v != "" {
		active := v == "true" || v == "1"
		p.IsActive = &active
	}

	for _, f := range r.Filters {
		v := c.Query(f.Param)
		if v == "" {
			continue
		}
		raw := json.RawMessage(v)
		if f.Kind == schema.KindString || f.Kind == schema.KindDate {
			encoded, err := json.Marshal(v)
			if err != nil {
				return p, httpx.Validation("query parameter ไม่ถูกต้อง")
			}
			raw = encoded
		}
		val, err := schema.Coerce(raw, f.Kind, f.Param, f.Param)
		if err != nil {
			return p, err
		}
		p.Filters[f.Param] = val
	}
	return p, nil
}
