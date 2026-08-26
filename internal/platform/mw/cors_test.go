// Path: internal/platform/mw/cors_test.go
package mw

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func newAuditApp(t *testing.T, allowed []string) (*fiber.App, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	lg := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	app := fiber.New()
	app.Use(CORSAudit(allowed, lg))
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("ok") })
	return app, &buf
}

func get(t *testing.T, app *fiber.App, origin string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	if origin != "" {
		req.Header.Set(fiber.HeaderOrigin, origin)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 — the audit must never block a request", res.StatusCode)
	}
}

func TestCORSAudit_AllowedOriginIsSilent(t *testing.T) {
	app, buf := newAuditApp(t, []string{"https://www.phenbun.com"})
	get(t, app, "https://www.phenbun.com")

	if buf.Len() != 0 {
		t.Fatalf("allowed origin logged something: %s", buf.String())
	}
}

func TestCORSAudit_NoOriginIsSilent(t *testing.T) {
	// curl และ Postman ไม่ส่ง Origin และไม่อยู่ใต้กฎ CORS
	app, buf := newAuditApp(t, []string{"https://www.phenbun.com"})
	get(t, app, "")

	if buf.Len() != 0 {
		t.Fatalf("origin-less request logged something: %s", buf.String())
	}
}

func TestCORSAudit_RejectedOriginIsLoggedOnce(t *testing.T) {
	app, buf := newAuditApp(t, []string{"http://localhost:5173"})

	get(t, app, "https://www.phenbun.com")
	first := buf.String()

	if !strings.Contains(first, "cors origin rejected") {
		t.Fatalf("rejection was not logged: %q", first)
	}
	if !strings.Contains(first, "https://www.phenbun.com") {
		t.Fatalf("log does not name the origin that was refused: %q", first)
	}
	// รายการที่อนุญาตต้องอยู่ในบรรทัดเดียวกัน ไม่งั้นคนอ่าน log ยังต้องเดาว่า
	// ค่าที่ process ถืออยู่จริงคืออะไร ซึ่งเป็นคำถามเดียวที่ต้องการคำตอบ
	if !strings.Contains(first, "localhost:5173") {
		t.Fatalf("log does not name the allowed list: %q", first)
	}

	// แท็บที่เปิดค้างยิงซ้ำได้เป็นร้อยครั้ง บรรทัดที่ 2 ไม่ได้บอกอะไรเพิ่ม
	get(t, app, "https://www.phenbun.com")
	if buf.String() != first {
		t.Fatalf("same origin logged twice:\n%s", buf.String())
	}

	// origin ใหม่ยังต้องได้บรรทัดของตัวเอง
	get(t, app, "https://phenbun.com")
	if !strings.Contains(strings.TrimPrefix(buf.String(), first), "https://phenbun.com") {
		t.Fatalf("a second distinct origin was swallowed:\n%s", buf.String())
	}
}
