// Path: internal/platform/mw/auth.go
package mw

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"penbun/api/internal/platform/httpx"
)

// key ของ Locals — ใช้ชนิดของตัวเองกัน key ชนกับ middleware อื่น
type localKey string

const (
	keyUserID   localKey = "penbun.user_id"
	keyUsername localKey = "penbun.username"
	keyLevel    localKey = "penbun.user_level"
	keyMustPW   localKey = "penbun.must_change_pw"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	jwt.RegisteredClaims
	Username     string `json:"username"`
	UserLevel    string `json:"user_level"`
	MustChangePW bool   `json:"must_change_pw"`
	TokenType    string `json:"typ"`
}

// TokenStore เก็บ jti ที่ถูกเพิกถอน
//
// 4.0 ใช้ implementation ใน memory ซึ่งหายเมื่อ restart และไม่ทำงานข้าม instance
// การประกาศเป็น interface ตั้งแต่ต้นทำให้เปลี่ยนไปใช้ Redis หรือตารางใน DB
// ได้โดยไม่ต้องแตะ handler เลย
type TokenStore interface {
	Revoke(jti string, expiresAt time.Time)
	IsRevoked(jti string) bool
}

type MemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

func NewMemoryTokenStore() *MemoryTokenStore {
	s := &MemoryTokenStore{tokens: make(map[string]time.Time)}
	go s.prune()
	return s
}

func (s *MemoryTokenStore) Revoke(jti string, expiresAt time.Time) {
	if jti == "" {
		return
	}
	s.mu.Lock()
	s.tokens[jti] = expiresAt
	s.mu.Unlock()
}

func (s *MemoryTokenStore) IsRevoked(jti string) bool {
	s.mu.RLock()
	exp, ok := s.tokens[jti]
	s.mu.RUnlock()
	return ok && time.Now().Before(exp)
}

// prune ทิ้ง jti ที่หมดอายุแล้ว ไม่งั้น map จะโตไปเรื่อย ๆ
func (s *MemoryTokenStore) prune() {
	for range time.Tick(10 * time.Minute) {
		now := time.Now()
		s.mu.Lock()
		for jti, exp := range s.tokens {
			if now.After(exp) {
				delete(s.tokens, jti)
			}
		}
		s.mu.Unlock()
	}
}

type Authenticator struct {
	Secret []byte
	Store  TokenStore
}

// publicPaths คือเส้นทางที่เข้าได้โดยไม่ต้องมี token
//
// ประกาศไว้ตรงนี้แทนการพึ่งลำดับการ register route เพราะ Fiber จับ middleware
// จาก prefix — group /api/v2 ที่มี Protect() จะครอบ /api/v2/auth/login ด้วย
// ทำให้ login ต้องมี token ก่อนถึงจะ login ได้ ซึ่งเป็นไปไม่ได้
//
// การเขียนเป็นรายการชัดเจนแบบนี้ทำให้เพิ่ม endpoint ใหม่แล้วไม่หลุดเป็น public
// โดยบังเอิญจากการวางลำดับผิด
var publicPaths = map[string]bool{
	"/api/v2/auth/login":   true,
	"/api/v2/auth/refresh": true,
}

// passwordChangeExempt คือเส้นทางที่ผู้ใช้ซึ่งถูกบังคับเปลี่ยนรหัสผ่านยังเข้าได้
// ถ้าไม่ยกเว้นให้ ผู้ใช้จะติดวงวน คือเข้าไปเปลี่ยนรหัสผ่านก็ไม่ได้
var passwordChangeExempt = map[string]bool{
	"/api/v2/auth/me":              true,
	"/api/v2/auth/change-password": true,
	"/api/v2/auth/logout":          true,
}

// Protect ตรวจ access token และใส่ตัวตนลง Locals
func (a *Authenticator) Protect() fiber.Handler {
	return func(c fiber.Ctx) error {
		if publicPaths[c.Path()] {
			return c.Next()
		}
		claims, err := a.parseFromHeader(c)
		if err != nil {
			return err
		}
		if claims.TokenType != TokenTypeAccess {
			return httpx.Unauthorized("ต้องใช้ access token")
		}

		fiber.Locals[string](c, keyUserID, claims.Subject)
		fiber.Locals[string](c, keyUsername, claims.Username)
		fiber.Locals[string](c, keyLevel, claims.UserLevel)
		fiber.Locals[bool](c, keyMustPW, claims.MustChangePW)
		return c.Next()
	}
}

// RequirePasswordChanged ปิดทุกเส้นทางสำหรับผู้ใช้ที่ยังไม่ได้เปลี่ยนรหัสผ่านครั้งแรก
// ตาม Authentication Spec M001 (tb_users.status_change_pw)
func RequirePasswordChanged() fiber.Handler {
	return func(c fiber.Ctx) error {
		if publicPaths[c.Path()] || passwordChangeExempt[c.Path()] {
			return c.Next()
		}
		if MustChangePassword(c) {
			return httpx.Forbidden(httpx.CodeMustChangePassword,
				"กรุณาเปลี่ยนรหัสผ่านก่อนใช้งานระบบ")
		}
		return c.Next()
	}
}

// RequireLevel จำกัดสิทธิ์ตาม tb_users.user_level
// 4.0 รองรับแค่ ADMIN / USER เพราะ DB v7 ยังไม่มี tb_role
func RequireLevel(levels ...string) fiber.Handler {
	allowed := make(map[string]bool, len(levels))
	for _, l := range levels {
		allowed[l] = true
	}
	return func(c fiber.Ctx) error {
		if !allowed[UserLevel(c)] {
			return httpx.Forbidden(httpx.CodeForbidden, "สิทธิ์ของคุณไม่เพียงพอสำหรับการดำเนินการนี้")
		}
		return c.Next()
	}
}

// ParseRefresh ใช้ตอนขอ token ใหม่ — token มาใน body ไม่ใช่ header
func (a *Authenticator) ParseRefresh(token string) (*Claims, error) {
	claims, err := a.parse(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, httpx.Unauthorized("ต้องใช้ refresh token")
	}
	return claims, nil
}

// ParseHeader อ่านและตรวจ token จาก Authorization header
// เปิดเป็น public เพราะ /auth/logout ต้องการ jti ของ token ปัจจุบันไปเพิกถอน
func (a *Authenticator) ParseHeader(c fiber.Ctx) (*Claims, error) {
	return a.parseFromHeader(c)
}

func (a *Authenticator) parseFromHeader(c fiber.Ctx) (*Claims, error) {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return nil, httpx.Unauthorized("ไม่พบข้อมูลการเข้าสู่ระบบ")
	}
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return nil, httpx.Unauthorized("รูปแบบ Authorization header ไม่ถูกต้อง")
	}
	return a.parse(token)
}

func (a *Authenticator) parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		// แยก token หมดอายุออกจาก token เสีย เพื่อให้ client รู้ว่าควรเรียก
		// /auth/refresh แทนที่จะเด้งผู้ใช้ออกไปหน้า login
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, &httpx.AppError{
				HTTPStatus: fiber.StatusUnauthorized,
				Code:       httpx.CodeTokenExpired,
				Message:    "เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่",
			}
		}
		return nil, httpx.Unauthorized("ข้อมูลการเข้าสู่ระบบไม่ถูกต้อง")
	}
	if a.Store.IsRevoked(claims.ID) {
		return nil, httpx.Unauthorized("เซสชันนี้ถูกยกเลิกแล้ว")
	}
	return claims, nil
}

func UserID(c fiber.Ctx) string    { return fiber.Locals[string](c, keyUserID) }
func UserLevel(c fiber.Ctx) string { return fiber.Locals[string](c, keyLevel) }

func MustChangePassword(c fiber.Ctx) bool { return fiber.Locals[bool](c, keyMustPW) }

// Username คือค่าที่จะถูกเขียนลง update_by ทุกจุดในระบบ
//
// ห้ามรับจาก query string หรือ body เด็ดขาด เพราะจะทำให้ปลอมเป็นใครก็ได้
func Username(c fiber.Ctx) string {
	if u := fiber.Locals[string](c, keyUsername); u != "" {
		return u
	}
	// ไม่ fallback เป็น 'System' — ถ้ามาถึงตรงนี้แปลว่า route ลืมใส่ Protect()
	// ซึ่งเป็น bug ของเรา ไม่ใช่ input ของผู้ใช้
	panic("mw.Username called on a route without Protect() middleware")
}
