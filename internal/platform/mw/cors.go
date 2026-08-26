// Path: internal/platform/mw/cors.go
package mw

import (
	"log/slog"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// CORSAudit บันทึก origin ที่นโยบายไม่อนุญาต
//
// การถูก CORS ปฏิเสธเป็นความล้มเหลวที่ "เงียบที่สุด" ในระบบ เพราะฝั่ง API
// ทุกอย่างดูปกติ preflight ตอบ 204 เหมือนกันทั้ง origin ที่อนุญาตและไม่อนุญาต
// ต่างกันแค่มี header Access-Control-Allow-Origin หรือไม่ ส่วนเบราว์เซอร์จะ
// บล็อกเองโดยไม่ส่งคำขอจริงตามมา บรรทัดใน access log จึงมีแค่ OPTIONS
// และไม่มีอะไรบอกว่าเกิดอะไรขึ้น
//
// ผลคือคนแก้ปัญหาต้องเดาระหว่าง "ตั้งค่าไม่ติด" กับ "หน้าเว็บเรียกผิด" ซึ่ง
// แยกจากภายนอกไม่ได้เลย middleware ตัวนี้เปลี่ยนมันเป็นบรรทัดเดียวใน log
//
// พิมพ์ origin ละครั้งต่อการรันหนึ่งรอบ ไม่ใช่ทุกคำขอ เพราะแท็บเดียวที่เปิดค้าง
// ยิงซ้ำได้เป็นร้อยครั้งต่อนาที และบรรทัดที่ 2 เป็นต้นไปไม่ได้บอกอะไรเพิ่ม
func CORSAudit(allowed []string, lg *slog.Logger) fiber.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}

	var seen sync.Map

	return func(c fiber.Ctx) error {
		origin := c.Get(fiber.HeaderOrigin)
		if origin == "" {
			// คำขอจากเครื่องมืออย่าง curl หรือ Postman ไม่มี Origin
			// และไม่อยู่ใต้กฎ CORS ตั้งแต่ต้น
			return c.Next()
		}
		if _, ok := set[origin]; ok {
			return c.Next()
		}
		if _, dup := seen.LoadOrStore(origin, struct{}{}); !dup {
			lg.Warn("cors origin rejected",
				"origin", origin,
				"method", c.Method(),
				"path", c.Path(),
				"allowed", allowed)
		}
		return c.Next()
	}
}
