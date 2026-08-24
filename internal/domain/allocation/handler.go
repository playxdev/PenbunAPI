// Path: internal/domain/allocation/handler.go
package allocation

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/repository"
	"penbun/api/internal/schema"
)

// โหมดการเสนอยอดจัดส่งงวดใหม่
const (
	ModeLast = "LAST" // ยึดยอดส่งงวดล่าสุด
	ModeAvg  = "AVG"  // เฉลี่ยยอดส่งย้อนหลังหลายงวด
	ModeSold = "SOLD" // ยึดยอดขายสุทธิ คือส่งหักคืน
)

var Modes = []string{ModeLast, ModeAvg, ModeSold}

type Handler struct {
	db       *repository.DB
	resolver *repository.Resolver
}

func NewHandler(db *repository.DB, res *repository.Resolver) *Handler {
	return &Handler{db: db, resolver: res}
}

func (h *Handler) Register(api fiber.Router) {
	g := api.Group("/allocation")
	g.Get("/history", h.history)
	g.Post("/pull", h.pull)
}

func (h *Handler) history(c fiber.Ctx) error {
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 50)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	a := &schema.Args{}
	where := " WHERE 1 = 1"
	add := func(col, val string) {
		if val != "" {
			where += fmt.Sprintf(" AND %s = %s", col, a.Add(val))
		}
	}
	add("customer_id", c.Query("customer_id"))
	add("sku_id", c.Query("sku_id"))
	add("route_code", c.Query("route_code"))
	add("period_key", c.Query("period_key"))
	if v := c.Query("is_locked"); v != "" {
		where += fmt.Sprintf(" AND is_locked = %s", a.Add(v == "true" || v == "1"))
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	var total int64
	if err := h.db.Exec().QueryRowContext(ctx,
		"SELECT COUNT_BIG(1) FROM dbo.vw_allocation_history"+where, a.Snapshot()...).
		Scan(&total); err != nil {
		return err
	}

	off := a.Add((page - 1) * limit)
	lim := a.Add(limit)
	q := fmt.Sprintf(
		"SELECT * FROM dbo.vw_allocation_history%s"+
			" ORDER BY period_key DESC, customer_id ASC, sku_code ASC"+
			" OFFSET %s ROWS FETCH NEXT %s ROWS ONLY", where, off, lim)

	items, err := repository.QueryMany(ctx, h.db.Exec(), q, a.Values())
	if err != nil {
		return err
	}
	return httpx.OK(c, fmt.Sprintf("พบประวัติการจัดส่ง %d รายการ", total),
		httpx.NewPage(items, page, limit, total))
}

type pullRequest struct {
	SkuID         string `json:"sku_id"`
	RouteID       string `json:"route_id"`
	CustomerID    string `json:"customer_id"`
	Mode          string `json:"mode"`
	LookbackCount int    `json:"lookback_count"`
}

// pull เสนอยอดจัดส่งของงวดใหม่จากประวัติงวดก่อน
//
// endpoint นี้ไม่เขียนอะไรลงฐานข้อมูลเลย เป็นการคำนวณข้อเสนออย่างเดียว
// ผู้ใช้ต้องตรวจและปรับก่อนนำไปสร้างใบส่งจริง ซึ่งตรงกับวิธีทำงานเดิมที่
// พนักงานดูยอดงวดก่อนแล้วตัดสินใจเองเป็นราย ๆ
func (h *Handler) pull(c fiber.Ctx) error {
	var req pullRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}

	if req.Mode == "" {
		req.Mode = ModeSold
	}
	if !contains(Modes, req.Mode) {
		return httpx.BadRequest(httpx.CodeInvalidEnum,
			"mode ต้องเป็น LAST, AVG หรือ SOLD").
			WithField("mode", httpx.CodeInvalidEnum, req.Mode)
	}
	if req.LookbackCount <= 0 {
		req.LookbackCount = 3
	}
	if req.LookbackCount > 12 {
		return httpx.Validation("lookback_count ต้องไม่เกิน 12 งวด").
			WithField("lookback_count", httpx.CodeValueOutOfRange, "")
	}
	if req.SkuID == "" && req.CustomerID == "" {
		// ไม่มีขอบเขตเลยแปลว่าไล่ทั้งประวัติทั้งระบบ ซึ่งกินทรัพยากรมากโดยไม่ได้ผลที่ใช้ได้
		return httpx.Validation("ต้องระบุอย่างน้อย sku_id หรือ customer_id")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	skuAuto, err := h.optionalRef(ctx, "tb_product_sku", req.SkuID, "sku_id", "SKU")
	if err != nil {
		return err
	}
	routeAuto, err := h.optionalRef(ctx, "tb_route", req.RouteID, "route_id", "สายจัดจำหน่าย")
	if err != nil {
		return err
	}
	customerAuto, err := h.optionalRef(ctx, "tb_customer", req.CustomerID, "customer_id", "ลูกค้า")
	if err != nil {
		return err
	}

	const q = `
EXEC dbo.USP_PULL_ALLOCATION_FROM_HISTORY
     @SkuAuto       = @p1,
     @RouteAuto     = @p2,
     @CustomerAuto  = @p3,
     @Mode          = @p4,
     @LookbackCount = @p5`

	items, err := repository.QueryMany(ctx, h.db.Exec(), q,
		[]any{skuAuto, routeAuto, customerAuto, req.Mode, req.LookbackCount})
	if err != nil {
		return err
	}

	return httpx.OK(c,
		fmt.Sprintf("คำนวณข้อเสนอ %d รายการ (โหมด %s)", len(items), req.Mode),
		fiber.Map{
			"mode":           req.Mode,
			"lookback_count": req.LookbackCount,
			"items":          items,
		})
}

// optionalRef คืน nil เมื่อไม่ได้ระบุ เพื่อให้ Stored Procedure ตีความว่าไม่กรอง
func (h *Handler) optionalRef(ctx context.Context, table, businessID, field, label string) (any, error) {
	if businessID == "" {
		return nil, nil
	}
	autoID, err := h.resolver.Resolve(ctx, table, businessID)
	if err != nil {
		if err == repository.ErrRefNotFound {
			return nil, httpx.RefNotFound(field, businessID, label)
		}
		return nil, err
	}
	return autoID, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
