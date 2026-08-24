// Path: internal/domain/meta/handler.go
package meta

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/repository"
)

// Handler ให้ค่า enum ทั้งหมดที่ฐานข้อมูลยอมรับ
//
// อ่านจาก CHECK constraint จริงแทนการเขียนรายการซ้ำไว้ในโค้ด เพราะรายการที่
// เขียนซ้ำจะค่อย ๆ ต่างจากของจริงเมื่อฝั่งฐานข้อมูลเพิ่มค่าใหม่ แล้วหน้าจอจะ
// เสนอตัวเลือกที่บันทึกไม่ได้ หรือซ่อนตัวเลือกที่ใช้ได้จริง
type Handler struct {
	db *repository.DB

	mu       sync.RWMutex
	cached   map[string][]string
	cachedAt time.Time
	ttl      time.Duration
}

func NewHandler(db *repository.DB) *Handler {
	return &Handler{db: db, ttl: 10 * time.Minute}
}

func (h *Handler) Register(api fiber.Router) {
	api.Get("/meta/enums", h.enums)
}

func (h *Handler) enums(c fiber.Ctx) error {
	h.mu.RLock()
	cached, at := h.cached, h.cachedAt
	h.mu.RUnlock()

	if cached != nil && time.Since(at) < h.ttl {
		return httpx.OK(c, "ค่าที่ระบบยอมรับ", cached)
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutRead)
	defer cancel()

	// อ่านนิยามของ CHECK constraint ทุกตัวที่ตั้งชื่อขึ้นต้นด้วย CK_
	const q = `
SELECT c.name, c.definition, OBJECT_NAME(c.parent_object_id) AS table_name,
       COL_NAME(c.parent_object_id, c.parent_column_id)      AS column_name
  FROM sys.check_constraints c
 WHERE c.name LIKE 'CK[_]%' AND c.parent_column_id > 0
 ORDER BY c.name`

	rows, err := h.db.Exec().QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	out := make(map[string][]string)
	for rows.Next() {
		var name, definition, table, column string
		if err := rows.Scan(&name, &definition, &table, &column); err != nil {
			return err
		}
		values := parseInList(definition)
		if len(values) == 0 {
			continue // CHECK ที่เป็นช่วงตัวเลขหรือเงื่อนไขอื่น ไม่ใช่ enum
		}
		key := strings.TrimPrefix(table, "tb_") + "_" + column
		out[key] = values
	}
	if err := rows.Err(); err != nil {
		return err
	}

	h.mu.Lock()
	h.cached, h.cachedAt = out, time.Now()
	h.mu.Unlock()

	return httpx.OK(c, "ค่าที่ระบบยอมรับ", out)
}

// literalPattern จับค่าสตริงในนิยามของ CHECK constraint
// SQL Server เก็บนิยามในรูป ([col]='A' OR [col]='B') หรือ ([col] IN ('A','B'))
var literalPattern = regexp.MustCompile(`N?'([^']*)'`)

func parseInList(definition string) []string {
	matches := literalPattern.FindAllStringSubmatch(definition, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		v := m[1]
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
