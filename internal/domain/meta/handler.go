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

	"penbun/api/internal/crud"
	"penbun/api/internal/platform/authz"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
)

// Handler ให้ค่า enum ทั้งหมดที่ฐานข้อมูลยอมรับ
//
// อ่านจาก CHECK constraint จริงแทนการเขียนรายการซ้ำไว้ในโค้ด เพราะรายการที่
// เขียนซ้ำจะค่อย ๆ ต่างจากของจริงเมื่อฝั่งฐานข้อมูลเพิ่มค่าใหม่ แล้วหน้าจอจะ
// เสนอตัวเลือกที่บันทึกไม่ได้ หรือซ่อนตัวเลือกที่ใช้ได้จริง
type Handler struct {
	db *repository.DB

	// resources คือ descriptor ชุดเดียวกับที่ crud.Engine ติดตั้งเส้นทางให้
	// /meta/permissions อ่านสิทธิ์จากตรงนี้ ไม่ได้เขียนรายการซ้ำไว้ต่างหาก
	resources []*crud.Resource

	// authz คือแหล่งเดียวกับที่ตัวกรองบนเส้นทางใช้ตัดสิน
	authz *authz.Repo

	mu       sync.RWMutex
	cached   map[string][]string
	cachedAt time.Time
	ttl      time.Duration
}

func NewHandler(db *repository.DB, res []*crud.Resource, az *authz.Repo) *Handler {
	return &Handler{db: db, resources: res, authz: az, ttl: 10 * time.Minute}
}

func (h *Handler) Register(api fiber.Router) {
	api.Get("/meta/enums", h.enums)
	api.Get("/meta/permissions", h.permissions)
}

// Access คือสิ่งที่ผู้ใช้คนที่เรียกทำได้กับ resource หนึ่งตัว
type Access struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// Permissions คือคำตอบของ /meta/permissions
type Permissions struct {
	Level     string            `json:"level"`
	Resources map[string]Access `json:"resources"`
}

// permissions บอกหน้าจอว่าผู้ใช้คนนี้อ่านและเขียน resource ไหนได้บ้าง
//
// อ่านจาก vw_user_privilege ซึ่งเป็นแหล่งเดียวกับที่ authz.GuardResource ใช้ตัดสินจริง
// คำตอบของ endpoint นี้จึงตรงกับสิ่งที่เส้นทางจะทำเสมอ ไม่ใช่สำเนาที่ค่อย ๆ เพี้ยน
//
// รายชื่อ resource มาจาก descriptor ชุดที่ mount จริง ไม่ใช่จากแถวที่บังเอิญมีในฐาน
// resource ที่ไม่มีแถวสิทธิ์เลยจึงยังปรากฏในคำตอบ เป็น read:false write:false
// หน้าจอจะได้รู้ว่ามันมีอยู่แต่เข้าไม่ได้ แทนที่จะหาคีย์ไม่เจอแล้วเดาเอง
//
// นี่ไม่ใช่การตรวจสิทธิ์ ตัวที่ตรวจจริงคือ authz.GuardResource บนกลุ่มของแต่ละ resource
// endpoint นี้แค่ทำให้หน้าจอพูดตรงกับเส้นทางเหล่านั้น
func (h *Handler) permissions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel()

	perms, err := h.authz.For(ctx, mw.UserID(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, "สิทธิ์ของผู้ใช้", buildPermissions(h.resources, mw.UserLevel(c), perms))
}

// buildPermissions แยกออกมาจาก handler เพื่อให้ทดสอบได้โดยไม่ต้องมี request
//
// write เป็นจริงเมื่อทำอย่างใดอย่างหนึ่งใน insert / update / delete ได้
// เพราะรูปแบบ read/write ของ endpoint นี้หยาบกว่าสี่การกระทำที่ฐานเก็บ
// หน้าจอที่ต้องการความละเอียดเต็มให้อ่าน permissions จาก GET /auth/me แทน
func buildPermissions(res []*crud.Resource, level string, perms map[string]authz.Access) Permissions {
	out := make(map[string]Access, len(res))
	for _, r := range res {
		a := perms[r.Name]
		out[r.Name] = Access{
			Read:  a.View,
			Write: a.Insert || a.Update || a.Delete,
		}
	}
	return Permissions{Level: level, Resources: out}
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
