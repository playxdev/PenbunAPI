// Path: internal/domain/book/handler.go
package book

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/crud"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

// หนังสือหนึ่งเล่มผูกกับสินค้าหนึ่งแถวแบบหนึ่งต่อหนึ่ง บังคับด้วย unique constraint
// ฝั่งฐานข้อมูล การสร้างจึงต้องเขียนสองตารางในทรานแซกชันเดียวเสมอ
//
// การอ่าน (GET) ใช้ generic CRUD engine ผ่าน descriptor ที่ตั้ง ReadOnly ไว้
// package นี้รับผิดชอบเฉพาะการเขียนซึ่งเกินขอบเขตของ engine กลาง
type Handler struct {
	db       *repository.DB
	resolver *repository.Resolver
	crud     *crud.Engine
}

func NewHandler(db *repository.DB, res *repository.Resolver, ce *crud.Engine) *Handler {
	return &Handler{db: db, resolver: res, crud: ce}
}

func (h *Handler) Register(api fiber.Router) {
	g := api.Group("/book")
	g.Post("", h.create)
	g.Put("/:id", h.update)
	g.Delete("/:id", h.softDelete)
}

// productFields คือคอลัมน์ฝั่ง tb_product ที่ endpoint นี้รับ
var productFields = []schema.Field{
	{Name: "product_code", Kind: schema.KindString, Required: true, MaxLen: 50, Label: "รหัสสินค้า"},
	{Name: "product_name", Kind: schema.KindString, MaxLen: 250, Label: "ชื่อสินค้า"},
	{Name: "cost_price", Kind: schema.KindDecimal, Label: "ราคาทุน", Min: schema.Float64(0)},
	{Name: "sell_price", Kind: schema.KindDecimal, Label: "ราคาขาย", Min: schema.Float64(0)},
	{Name: "barcode", Kind: schema.KindString, MaxLen: 50, Label: "บาร์โค้ด"},
	{Name: "weight_kg", Kind: schema.KindDecimal, Label: "น้ำหนัก (กก.)", Min: schema.Float64(0)},
	{Name: "pack_qty", Kind: schema.KindDecimal, Label: "จำนวนต่อแพ็ก", Min: schema.Float64(0)},
	{Name: "description", Kind: schema.KindString, MaxLen: 500, Label: "รายละเอียด"},
}

var productRefs = []schema.Ref{
	{Field: "product_group_id", Table: "tb_product_group", Column: "ref_product_group_auto",
		Label: "กลุ่มสินค้า", Required: true},
	{Field: "product_format_type_id", Table: "tb_product_format_type",
		Column: "ref_product_format_type_auto", Label: "รูปแบบสินค้า"},
	{Field: "unit_type_id", Table: "tb_unit_type", Column: "ref_unit_type_auto", Label: "หน่วยนับ"},
	{Field: "vendor_id", Table: "tb_vendor", Column: "ref_vendor_auto", Label: "คู่ค้า"},
}

// bookFields คือคอลัมน์ฝั่ง tb_book
var bookFields = []schema.Field{
	{Name: "book_name", Kind: schema.KindString, Required: true, MaxLen: 250, Label: "ชื่อหนังสือ"},
	{Name: "author", Kind: schema.KindString, MaxLen: 200, Label: "ผู้แต่ง"},
	{Name: "translator", Kind: schema.KindString, MaxLen: 200, Label: "ผู้แปล"},
	{Name: "isbn", Kind: schema.KindString, MaxLen: 20, Label: "ISBN"},
	{Name: "publisher_name", Kind: schema.KindString, MaxLen: 200, Label: "สำนักพิมพ์"},
	{Name: "page_count", Kind: schema.KindInt, Label: "จำนวนหน้า", Min: schema.Float64(0)},
	{Name: "cover_price", Kind: schema.KindDecimal, Label: "ราคาปก", Min: schema.Float64(0)},
	{Name: "net_price", Kind: schema.KindDecimal, Label: "ราคาสุทธิ", Min: schema.Float64(0)},
	{Name: "vendor_discount_percent", Kind: schema.KindDecimal,
		Label: "ส่วนลดจากคู่ค้า (%)", Min: schema.Float64(0)},
	{Name: "customer_discount_percent", Kind: schema.KindDecimal,
		Label: "ส่วนลดให้ลูกค้า (%)", Min: schema.Float64(0)},
	// อภินันท์ — จำนวนเล่มที่ให้เปล่า ทั้งใบชำระเจ้าของหนังสือและใบรับเงิน
	// ล่วงหน้าอ่านค่านี้จากแฟ้มหนังสือ ไม่ได้กรอกซ้ำในเอกสาร
	{Name: "complimentary_qty", Kind: schema.KindInt, Label: "อภินันท์ (เล่ม)", Min: schema.Float64(0)},
	{Name: "effective_date", Kind: schema.KindDate, Label: "วันที่มีผล"},
	{Name: "book_description", Column: "description", Kind: schema.KindString,
		MaxLen: 500, Label: "รายละเอียดหนังสือ"},
}

// FieldSets คืนคอลัมน์ที่ endpoint นี้เขียนจริง แยกตามตาราง
//
// มีไว้ให้เทสต์ drift เทียบกับ INFORMATION_SCHEMA ได้ — /book เป็น handler ที่
// เขียนเอง ไม่ได้อยู่ใน resources.All() ฟิลด์ของมันจึงไม่เคยถูกตรวจกับฐานจริง
// และนั่นคือที่มาของ translator ที่รับมาตั้งแต่ v4.0.0 โดยไม่มีคอลัมน์รองรับ
func FieldSets() map[string][]schema.Field {
	return map[string][]schema.Field{
		"tb_product": productFields,
		"tb_book":    bookFields,
	}
}

var bookRefs = []schema.Ref{
	{Field: "book_type_id", Table: "tb_book_type", Column: "ref_book_type_auto",
		Label: "ประเภทหนังสือ", Required: true},
}

func (h *Handler) create(c fiber.Ctx) error {
	body, err := crud.DecodeBody(c)
	if err != nil {
		return err
	}
	if err := rejectReserved(body); err != nil {
		return err
	}

	prodVals, err := collect(productFields, body, true)
	if err != nil {
		return err
	}
	bookVals, err := collect(bookFields, body, true)
	if err != nil {
		return err
	}

	// ชื่อสินค้ายึดตามชื่อหนังสือถ้าไม่ได้ระบุแยก
	// การบังคับให้พิมพ์สองครั้งเปิดช่องให้ทั้งสองชื่อเพี้ยนจากกันโดยไม่ตั้งใจ
	if _, ok := prodVals["product_name"]; !ok {
		if v, ok := bookVals["book_name"]; ok {
			prodVals["product_name"] = v
		}
	}
	if err := required(productFields, prodVals); err != nil {
		return err
	}
	if err := required(bookFields, bookVals); err != nil {
		return err
	}

	// หนังสือเป็นสินค้าที่นับสต็อกเสมอ ไม่เปิดให้ตั้งค่านี้เอง
	prodVals["count_stock"] = true

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	user := mw.Username(c)
	var bookID string

	err = h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		prodRefs, err := h.crud.ResolveRefs(ctx, tx, productRefs, body, true)
		if err != nil {
			return err
		}
		bkRefs, err := h.crud.ResolveRefs(ctx, tx, bookRefs, body, true)
		if err != nil {
			return err
		}

		productAuto, err := insertRow(ctx, tx, "tb_product", prodVals, prodRefs, user)
		if err != nil {
			return err
		}

		bkRefs["ref_product_auto"] = productAuto
		bookAuto, err := insertRow(ctx, tx, "tb_book", bookVals, bkRefs, user)
		if err != nil {
			return err
		}

		return tx.QueryRowContext(ctx,
			"SELECT book_id FROM dbo.tb_book WHERE autoID = @p1", bookAuto).Scan(&bookID)
	})
	if err != nil {
		return err
	}

	h.resolver.Invalidate("tb_product")
	return h.respond(c, bookID, fiber.StatusCreated, "เพิ่มหนังสือเรียบร้อย")
}

func (h *Handler) update(c fiber.Ctx) error {
	id := c.Params("id")
	body, err := crud.DecodeBody(c)
	if err != nil {
		return err
	}
	if err := rejectReserved(body); err != nil {
		return err
	}

	prodVals, err := collect(productFields, body, false)
	if err != nil {
		return err
	}
	bookVals, err := collect(bookFields, body, false)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	user := mw.Username(c)

	err = h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		bookAuto, productAuto, err := h.locate(ctx, tx, id)
		if err != nil {
			return err
		}

		prodRefs, err := h.crud.ResolveRefs(ctx, tx, productRefs, body, false)
		if err != nil {
			return err
		}
		bkRefs, err := h.crud.ResolveRefs(ctx, tx, bookRefs, body, false)
		if err != nil {
			return err
		}

		if len(prodVals)+len(prodRefs)+len(bookVals)+len(bkRefs) == 0 {
			return httpx.Validation("ไม่มีข้อมูลที่ต้องแก้ไข")
		}
		if len(prodVals) > 0 || len(prodRefs) > 0 {
			if err := updateRow(ctx, tx, "tb_product", productAuto, prodVals, prodRefs, user); err != nil {
				return err
			}
		}
		if len(bookVals) > 0 || len(bkRefs) > 0 {
			if err := updateRow(ctx, tx, "tb_book", bookAuto, bookVals, bkRefs, user); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return h.respond(c, id, fiber.StatusOK, "แก้ไขหนังสือเรียบร้อย")
}

// softDelete ลบทั้งสองแถวพร้อมกัน
//
// ถ้าลบเฉพาะ tb_book จะเหลือแถวสินค้ากำพร้าที่ยังโผล่ในรายการสินค้าและยัง
// ถูกอ้างในเอกสารได้ ทั้งที่ผู้ใช้เชื่อว่าลบหนังสือไปแล้ว
func (h *Handler) softDelete(c fiber.Ctx) error {
	id := c.Params("id")

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	user := mw.Username(c)

	err := h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		bookAuto, productAuto, err := h.locate(ctx, tx, id)
		if err != nil {
			return err
		}
		const q = "UPDATE dbo.%s SET is_delete = 1, update_by = @p2 WHERE autoID = @p1 AND is_delete = 0"
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(q, "tb_book"), bookAuto, user); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf(q, "tb_product"), productAuto, user)
		return err
	})
	if err != nil {
		return err
	}

	h.resolver.Invalidate("tb_product")
	return httpx.NoContent(c)
}

// ───────────────────────────── helpers ─────────────────────────────

func (h *Handler) locate(ctx context.Context, tx *sql.Tx, bookID string) (bookAuto, productAuto int, err error) {
	const q = `SELECT TOP 1 autoID, ref_product_auto FROM dbo.tb_book
	            WHERE book_id = @p1 AND is_delete = 0`
	err = tx.QueryRowContext(ctx, q, bookID).Scan(&bookAuto, &productAuto)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, httpx.NotFound(fmt.Sprintf("ไม่พบหนังสือ '%s'", bookID))
	}
	return bookAuto, productAuto, err
}

func (h *Handler) respond(c fiber.Ctx, bookID string, status int, msg string) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel()

	row, err := repository.QueryOne(ctx, h.db.Exec(),
		"SELECT TOP 1 * FROM dbo.vw_book WHERE book_id = @p1", []any{bookID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.NotFound(fmt.Sprintf("ไม่พบหนังสือ '%s'", bookID))
		}
		return err
	}
	if status == fiber.StatusCreated {
		return httpx.Created(c, msg, row)
	}
	return httpx.OK(c, msg, row)
}

func insertRow(
	ctx context.Context, tx *sql.Tx, table string,
	vals map[string]any, refs map[string]int, updateBy string,
) (int, error) {
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
	phs = append(phs, a.Add(updateBy))

	q := repository.InsertReturningAuto(table, cols, phs)

	var autoID int
	err := tx.QueryRowContext(ctx, q, a.Values()...).Scan(&autoID)
	return autoID, err
}

func updateRow(
	ctx context.Context, tx *sql.Tx, table string, autoID int,
	vals map[string]any, refs map[string]int, updateBy string,
) error {
	a := &schema.Args{}
	sets := make([]string, 0, len(vals)+len(refs)+1)

	for _, n := range schema.SortedKeys(vals) {
		sets = append(sets, n+" = "+a.Add(vals[n]))
	}
	for _, n := range schema.SortedKeys(refs) {
		sets = append(sets, n+" = "+a.Add(refs[n]))
	}
	sets = append(sets, "update_by = "+a.Add(updateBy))

	q := fmt.Sprintf("UPDATE dbo.%s SET %s WHERE autoID = %s AND is_delete = 0",
		table, strings.Join(sets, ", "), a.Add(autoID))
	_, err := tx.ExecContext(ctx, q, a.Values()...)
	return err
}

func rejectReserved(body map[string]json.RawMessage) error {
	for k := range body {
		switch k {
		case "autoID", "auto_id", "prefix", "update_by", "update_date",
			"is_delete", "id_status", "book_id", "product_id", "book_auto", "product_auto":
			return httpx.Validation(fmt.Sprintf("ห้ามส่งฟิลด์ '%s' — ระบบเป็นผู้กำหนด", k)).
				WithField(k, httpx.CodeValidationFailed, "")
		case "count_stock":
			return httpx.Validation("ห้ามส่งฟิลด์ 'count_stock' — หนังสือนับสต็อกเสมอ").
				WithField(k, httpx.CodeValidationFailed, "")
		}
	}
	return nil
}

func collect(fields []schema.Field, body map[string]json.RawMessage, isInsert bool) (map[string]any, error) {
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		raw, ok := body[f.Name]
		if !ok {
			continue
		}
		if isInsert && f.NoInsert {
			continue
		}
		if !isInsert && f.NoUpdate {
			return nil, httpx.Validation(fmt.Sprintf("ฟิลด์ '%s' แก้ไขภายหลังไม่ได้", f.Name)).
				WithField(f.Name, httpx.CodeValidationFailed, "")
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

func required(fields []schema.Field, vals map[string]any) error {
	for _, f := range fields {
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
