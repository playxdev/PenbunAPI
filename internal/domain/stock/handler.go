// Path: internal/domain/stock/handler.go
package stock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/config"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
)

type Handler struct {
	repo     *Repo
	db       *repository.DB
	resolver *repository.Resolver
	cfg      *config.Config
}

func NewHandler(repo *Repo, db *repository.DB, res *repository.Resolver, cfg *config.Config) *Handler {
	return &Handler{repo: repo, db: db, resolver: res, cfg: cfg}
}

func (h *Handler) Register(api fiber.Router) {
	g := api.Group("/stock")
	g.Get("/onhand", h.onHand)
	g.Get("/movements", h.movements)

	// การปรับและย้ายสต็อกด้วยมือจำกัดที่ ADMIN
	// รายการเหล่านี้ไม่มีเอกสารรองรับ จึงตรวจสอบย้อนหลังได้ยากกว่าการโพสต์เอกสาร
	g.Post("/adjust", mw.RequireLevel("ADMIN"), h.adjust)
	g.Post("/transfer", mw.RequireLevel("ADMIN"), h.transfer)
	g.Post("/rebuild", mw.RequireLevel("ADMIN"), h.rebuild)

	cg := api.Group("/consign")
	cg.Get("/outstanding", h.consignOutstanding)
	cg.Post("/rebuild", mw.RequireLevel("ADMIN"), h.rebuildConsign)
}

func (h *Handler) onHand(c fiber.Ctx) error {
	page, limit := paging(c)
	f := OnHandFilter{
		WarehouseCode: c.Query("warehouse_code"),
		SkuID:         c.Query("sku_id"),
		ProductID:     c.Query("product_id"),
		Search:        c.Query("q"),
		OnlyPositive:  c.Query("only_positive") == "true",
		Page:          page,
		Limit:         limit,
	}
	if v := c.Query("below"); v != "" {
		below := fiber.Query[float64](c, "below", 0)
		f.Below = &below
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	items, total, err := h.repo.OnHand(ctx, f)
	if err != nil {
		return err
	}
	return httpx.OK(c, fmt.Sprintf("พบยอดคงเหลือ %d รายการ", total),
		httpx.NewPage(items, page, limit, total))
}

func (h *Handler) movements(c fiber.Ctx) error {
	page, limit := paging(c)
	f := MovementFilter{
		SkuID:         c.Query("sku_id"),
		WarehouseCode: c.Query("warehouse_code"),
		CustomerID:    c.Query("customer_id"),
		VendorID:      c.Query("vendor_id"),
		MovementType:  c.Query("movement_type"),
		DocNo:         c.Query("doc_no"),
		Page:          page,
		Limit:         limit,
	}
	if v := c.Query("date_from"); v != "" {
		f.DateFrom = &v
	}
	if v := c.Query("date_to"); v != "" {
		f.DateTo = &v
	}

	// tb_stock_movement เป็น append-only และจะโตเรื่อย ๆ ตลอดอายุระบบ
	// การ query ที่ไม่ระบุขอบเขตเลยจะกลายเป็น full scan ที่กิน DB ทั้งเครื่อง
	if f.SkuID == "" && f.DocNo == "" && f.DateFrom == nil && f.DateTo == nil {
		return httpx.Validation("ต้องระบุอย่างน้อยหนึ่งอย่าง: sku_id, doc_no, date_from หรือ date_to")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	items, total, err := h.repo.Movements(ctx, f)
	if err != nil {
		return err
	}
	return httpx.OK(c, fmt.Sprintf("พบรายการเคลื่อนไหว %d รายการ", total),
		httpx.NewPage(items, page, limit, total))
}

type adjustRequest struct {
	SkuID         string   `json:"sku_id"`
	WarehouseCode string   `json:"warehouse_code"`
	QtyChange     *float64 `json:"qty_change"`
	Remark        string   `json:"remark"`
}

func (h *Handler) adjust(c fiber.Ctx) error {
	var req adjustRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}
	switch {
	case req.SkuID == "":
		return httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุ sku_id")
	case req.WarehouseCode == "":
		return httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุ warehouse_code")
	case req.QtyChange == nil || *req.QtyChange == 0:
		return httpx.Validation("qty_change ต้องไม่เป็น 0")
	case req.Remark == "":
		// การปรับสต็อกที่ไม่มีเหตุผลกำกับทำให้ audit trail ไร้ประโยชน์
		// ตอนไล่ย้อนว่ายอดเพี้ยนตรงไหน
		return httpx.BadRequest(httpx.CodeFieldRequired, "ต้องระบุเหตุผลใน remark")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	skuAuto, whAuto, err := h.resolveSkuWarehouse(ctx, req.SkuID, req.WarehouseCode)
	if err != nil {
		return err
	}

	err = h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// ล็อกระดับ SKU x คลัง — USP_APPLY_STOCK_MOVEMENT เช็คสต็อกด้วย SELECT
		// ที่ไม่มี UPDLOCK/HOLDLOCK จึงต้องกันการเข้าพร้อมกันจากฝั่ง API
		if err := repository.AppLock(ctx, tx, lockKey(skuAuto, whAuto), h.cfg.AppLockWaitMS); err != nil {
			return err
		}
		return ApplyMovement(ctx, tx, Movement{
			SkuAuto: skuAuto, WarehouseAuto: whAuto,
			MovementType: "ADJUST", QtyChange: *req.QtyChange,
			Remark: &req.Remark,
		}, mw.Username(c))
	})
	if err != nil {
		return err
	}

	return httpx.Created(c, "ปรับสต็อกเรียบร้อย", fiber.Map{
		"sku_id":         req.SkuID,
		"warehouse_code": req.WarehouseCode,
		"qty_change":     *req.QtyChange,
	})
}

type transferRequest struct {
	SkuID             string   `json:"sku_id"`
	FromWarehouseCode string   `json:"from_warehouse_code"`
	ToWarehouseCode   string   `json:"to_warehouse_code"`
	Qty               *float64 `json:"qty"`
	Remark            string   `json:"remark"`
}

// transfer ย้ายของระหว่างคลัง
//
// v7 ไม่มี USP_TRANSFER_STOCK ให้ API จึงต้องเรียก USP_APPLY_STOCK_MOVEMENT
// สองครั้งเอง ความ atomic จึงเป็นความรับผิดชอบของ Go ในกรณีนี้ ถ้าขาที่สอง
// ล้มแล้วขาแรกไม่ rollback ของจะหายไปจากระบบทั้งที่ยังอยู่จริงในคลัง
// → ทั้งสองขาต้องอยู่ใน WithTx เดียวกันเสมอ
func (h *Handler) transfer(c fiber.Ctx) error {
	var req transferRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}
	switch {
	case req.SkuID == "" || req.FromWarehouseCode == "" || req.ToWarehouseCode == "":
		return httpx.BadRequest(httpx.CodeFieldRequired,
			"ต้องระบุ sku_id, from_warehouse_code และ to_warehouse_code")
	case req.FromWarehouseCode == req.ToWarehouseCode:
		return httpx.Validation("คลังต้นทางและปลายทางต้องไม่ใช่คลังเดียวกัน")
	case req.Qty == nil || *req.Qty <= 0:
		return httpx.Validation("qty ต้องมากกว่า 0")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	skuAuto, fromAuto, err := h.resolveSkuWarehouse(ctx, req.SkuID, req.FromWarehouseCode)
	if err != nil {
		return err
	}
	toAuto, err := h.warehouseAuto(ctx, req.ToWarehouseCode)
	if err != nil {
		return err
	}

	var remark *string
	if req.Remark != "" {
		remark = &req.Remark
	}
	user := mw.Username(c)

	err = h.db.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// จองล็อกตามลำดับ autoID เสมอ เพื่อไม่ให้การย้ายสวนทางกัน
		// (DC→PRO พร้อมกับ PRO→DC) จองล็อกสลับลำดับแล้วเกิด deadlock
		first, second := fromAuto, toAuto
		if first > second {
			first, second = second, first
		}
		if err := repository.AppLock(ctx, tx, lockKey(skuAuto, first), h.cfg.AppLockWaitMS); err != nil {
			return err
		}
		if err := repository.AppLock(ctx, tx, lockKey(skuAuto, second), h.cfg.AppLockWaitMS); err != nil {
			return err
		}

		if err := ApplyMovement(ctx, tx, Movement{
			SkuAuto: skuAuto, WarehouseAuto: fromAuto,
			MovementType: "TRANSFER_OUT", QtyChange: -*req.Qty, Remark: remark,
		}, user); err != nil {
			return err
		}
		return ApplyMovement(ctx, tx, Movement{
			SkuAuto: skuAuto, WarehouseAuto: toAuto,
			MovementType: "TRANSFER_IN", QtyChange: *req.Qty, Remark: remark,
		}, user)
	})
	if err != nil {
		return err
	}

	return httpx.Created(c, "โอนสต็อกเรียบร้อย", fiber.Map{
		"sku_id":              req.SkuID,
		"from_warehouse_code": req.FromWarehouseCode,
		"to_warehouse_code":   req.ToWarehouseCode,
		"qty":                 *req.Qty,
	})
}

func (h *Handler) rebuild(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutPostDoc)
	defer cancel()

	if err := h.repo.RebuildStockCache(ctx, mw.Username(c)); err != nil {
		return err
	}
	return httpx.Accepted(c, "สร้างยอดคงเหลือใหม่จาก Ledger เรียบร้อย", nil)
}

func (h *Handler) rebuildConsign(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutPostDoc)
	defer cancel()

	if err := h.repo.RebuildConsignBalance(ctx, mw.Username(c)); err != nil {
		return err
	}
	return httpx.Accepted(c, "สร้างยอดฝากขายใหม่จากเอกสารที่โพสต์แล้วเรียบร้อย", nil)
}

func (h *Handler) consignOutstanding(c fiber.Ctx) error {
	page, limit := paging(c)

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	where := " WHERE 1 = 1"
	var args []any
	if v := c.Query("customer_id"); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND customer_id = @p%d", len(args))
	}
	if v := c.Query("sku_id"); v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND sku_id = @p%d", len(args))
	}
	if c.Query("only_outstanding", "true") == "true" {
		where += " AND qty_outstanding <> 0"
	}

	var total int64
	if err := h.db.Exec().QueryRowContext(ctx,
		"SELECT COUNT_BIG(1) FROM dbo.vw_consign_outstanding"+where, args...).Scan(&total); err != nil {
		return err
	}

	args = append(args, (page-1)*limit, limit)
	q := fmt.Sprintf(
		"SELECT * FROM dbo.vw_consign_outstanding%s ORDER BY customer_id ASC, sku_code ASC"+
			" OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY", where, len(args)-1, len(args))

	rows, err := h.db.Exec().QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	items, err := repository.ScanRows(rows)
	if err != nil {
		return err
	}
	return httpx.OK(c, fmt.Sprintf("พบยอดฝากขายคงค้าง %d รายการ", total),
		httpx.NewPage(items, page, limit, total))
}

// ---------- helpers ----------

// lockKey ล็อกที่ระดับ SKU x คลัง ไม่ใช่ทั้งระบบ
// การโพสต์เอกสารคนละสินค้าคนละคลังจึงยังทำพร้อมกันได้
func lockKey(skuAuto, warehouseAuto int) string {
	return fmt.Sprintf("PENBUN:STOCK:%d:%d", skuAuto, warehouseAuto)
}

func (h *Handler) resolveSkuWarehouse(ctx context.Context, skuID, whCode string) (int, int, error) {
	skuAuto, err := h.resolver.Resolve(ctx, "tb_product_sku", skuID)
	if err != nil {
		if errors.Is(err, repository.ErrRefNotFound) {
			return 0, 0, httpx.RefNotFound("sku_id", skuID, "SKU")
		}
		return 0, 0, err
	}
	whAuto, err := h.warehouseAuto(ctx, whCode)
	if err != nil {
		return 0, 0, err
	}
	return skuAuto, whAuto, nil
}

// warehouseAuto ค้นด้วย warehouse_code ไม่ใช่ warehouse_id
// เพราะรหัสคลัง (DC / BKK / PRO / RET / DMG) คือสิ่งที่ผู้ใช้และเอกสารจริงใช้กัน
func (h *Handler) warehouseAuto(ctx context.Context, code string) (int, error) {
	const q = `SELECT TOP 1 autoID FROM dbo.tb_warehouse
	            WHERE warehouse_code = @p1 AND is_delete = 0`
	var autoID int
	if err := h.db.Exec().QueryRowContext(ctx, q, code).Scan(&autoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, httpx.RefNotFound("warehouse_code", code, "คลังสินค้า")
		}
		return 0, err
	}
	return autoID, nil
}

func paging(c fiber.Ctx) (int, int) {
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 50)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}
