// Path: internal/domain/auth/handler.go
package auth

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
)

type Handler struct {
	svc  *Service
	auth *mw.Authenticator
}

func NewHandler(svc *Service, a *mw.Authenticator) *Handler {
	return &Handler{svc: svc, auth: a}
}

// Register ติดตั้งเส้นทางทั้งหมดของ auth ลงบนกลุ่ม /api/v2
//
// ทุกเส้นทางอยู่ใต้ middleware ชุดเดียวกัน ความแตกต่างเรื่องสิทธิ์ถูกกำหนดโดย
// publicPaths และ passwordChangeExempt ใน package mw ไม่ใช่โดยลำดับการ register
//
//	/auth/login, /auth/refresh          ไม่ต้องมี token
//	/auth/me, /auth/change-password     ต้องมี token แต่เข้าได้แม้ต้องเปลี่ยนรหัส
//	/auth/logout                        ต้องมี token
func (h *Handler) Register(api fiber.Router) {
	g := api.Group("/auth")
	g.Post("/login", h.login)
	g.Post("/refresh", h.refresh)
	g.Get("/me", h.me)
	g.Post("/change-password", h.changePassword)
	g.Post("/logout", h.logout)

	api.Put("/users/:id/unlock", mw.RequireLevel("ADMIN"), h.unlock)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	pair, err := h.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return err
	}
	return httpx.OK(c, "เข้าสู่ระบบสำเร็จ", pair)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(c fiber.Ctx) error {
	var req refreshRequest
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		return httpx.Validation("ต้องระบุ refresh_token")
	}

	claims, err := h.auth.ParseRefresh(req.RefreshToken)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel()

	pair, err := h.svc.Refresh(ctx, claims)
	if err != nil {
		return err
	}
	return httpx.OK(c, "ต่ออายุเซสชันเรียบร้อย", pair)
}

func (h *Handler) me(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutLookup)
	defer cancel()

	info, err := h.svc.Me(ctx, mw.UserID(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, "ข้อมูลผู้ใช้ปัจจุบัน", info)
}

func (h *Handler) logout(c fiber.Ctx) error {
	// token ที่ผ่าน Protect() มาแล้วย่อม parse ได้ จึงอ่านซ้ำเพื่อเอา jti
	claims, err := h.auth.ParseHeader(c)
	if err != nil {
		return err
	}
	h.svc.Logout(claims)
	return httpx.NoContent(c)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) changePassword(c fiber.Ctx) error {
	var req changePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Validation("request body ไม่ถูกต้อง")
	}

	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	pair, err := h.svc.ChangePassword(ctx, mw.UserID(c), req.CurrentPassword, req.NewPassword)
	if err != nil {
		return err
	}
	return httpx.OK(c, "เปลี่ยนรหัสผ่านเรียบร้อย", pair)
}

func (h *Handler) unlock(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c, repository.TimeoutWrite)
	defer cancel()

	if err := h.svc.Unlock(ctx, c.Params("id"), mw.Username(c)); err != nil {
		return err
	}
	return httpx.OK(c, "ปลดล็อกบัญชีเรียบร้อย", nil)
}
