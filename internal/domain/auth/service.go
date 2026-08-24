// Path: internal/domain/auth/service.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"penbun/api/internal/config"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/mw"
)

type Service struct {
	repo  *Repo
	cfg   *config.Config
	store mw.TokenStore
}

func NewService(repo *Repo, cfg *config.Config, store mw.TokenStore) *Service {
	return &Service{repo: repo, cfg: cfg, store: store}
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

type UserInfo struct {
	UserID             string  `json:"user_id"`
	UserName           string  `json:"user_name"`
	FullName           *string `json:"full_name"`
	Email              *string `json:"email"`
	UserLevel          string  `json:"user_level"`
	MustChangePassword bool    `json:"must_change_password"`
	LastLoginDate      *string `json:"last_login_date"`
}

// invalidCredentials — ข้อความเดียวสำหรับทุกกรณีที่เข้าสู่ระบบไม่ผ่าน
// ถ้าแยกข้อความว่า "ไม่มีผู้ใช้นี้" กับ "รหัสผ่านผิด" API จะกลายเป็นเครื่องมือ
// ไล่เดาชื่อผู้ใช้ให้ผู้โจมตีฟรี ๆ
func invalidCredentials() *httpx.AppError {
	return httpx.Unauthorized("ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง")
}

func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, httpx.Validation("กรุณากรอกชื่อผู้ใช้และรหัสผ่าน")
	}

	user, err := s.repo.ByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// เทียบ hash หลอกเพื่อให้เวลาตอบเท่ากับกรณีที่มีผู้ใช้จริง
			// ไม่งั้นผู้โจมตีวัดเวลาตอบแล้วรู้ได้ว่าชื่อผู้ใช้ไหนมีอยู่
			_ = bcrypt.CompareHashAndPassword(
				[]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"), []byte(password))
			return nil, invalidCredentials()
		}
		return nil, err
	}

	if user.Locked {
		return nil, httpx.Locked("บัญชีถูกระงับชั่วคราวเนื่องจากใส่รหัสผ่านผิดหลายครั้ง กรุณาติดต่อผู้ดูแลระบบ")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if regErr := s.repo.RegisterFailure(ctx, user.AutoID, s.cfg.AuthMaxFail); regErr != nil {
			return nil, regErr
		}
		if user.FailCount+1 >= s.cfg.AuthMaxFail {
			return nil, httpx.Locked("ใส่รหัสผ่านผิดครบจำนวนที่กำหนด บัญชีถูกระงับ กรุณาติดต่อผู้ดูแลระบบ")
		}
		return nil, invalidCredentials()
	}

	if err := s.repo.RegisterSuccess(ctx, user.AutoID); err != nil {
		return nil, err
	}

	return s.issue(user)
}

// Refresh หมุน token — refresh token เดิมถูกเพิกถอนทันทีที่ใช้
// ถ้าใครขโมย refresh token ไป การใช้งานของเจ้าของตัวจริงจะทำให้ token ที่ขโมยไปใช้ไม่ได้
func (s *Service) Refresh(ctx context.Context, claims *mw.Claims) (*TokenPair, error) {
	user, err := s.repo.ByUserID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, httpx.Unauthorized("บัญชีนี้ใช้งานไม่ได้แล้ว")
		}
		return nil, err
	}
	if user.Locked {
		return nil, httpx.Locked("บัญชีถูกระงับ")
	}

	s.store.Revoke(claims.ID, claims.ExpiresAt.Time)
	return s.issue(user)
}

func (s *Service) Logout(claims *mw.Claims) {
	if claims != nil && claims.ExpiresAt != nil {
		s.store.Revoke(claims.ID, claims.ExpiresAt.Time)
	}
}

func (s *Service) Me(ctx context.Context, userID string) (*UserInfo, error) {
	user, err := s.repo.ByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, httpx.NotFound("ไม่พบข้อมูลผู้ใช้")
		}
		return nil, err
	}
	return toUserInfo(user), nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) (*TokenPair, error) {
	if err := validatePasswordPolicy(next); err != nil {
		return nil, err
	}

	user, err := s.repo.ByUserID(ctx, userID)
	if err != nil {
		return nil, httpx.Unauthorized("ไม่พบข้อมูลผู้ใช้")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		return nil, httpx.Unauthorized("รหัสผ่านปัจจุบันไม่ถูกต้อง")
	}
	if current == next {
		return nil, httpx.Validation("รหัสผ่านใหม่ต้องไม่ซ้ำกับรหัสผ่านเดิม")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), s.cfg.BcryptCost)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetPassword(ctx, user.AutoID, string(hash), user.UserName); err != nil {
		return nil, err
	}

	user.MustChangePW = false
	return s.issue(user)
}

func (s *Service) Unlock(ctx context.Context, userID, actor string) error {
	n, err := s.repo.Unlock(ctx, userID, actor)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.NotFound("ไม่พบผู้ใช้ '" + userID + "'")
	}
	return nil
}

// ---------- token ----------

func (s *Service) issue(u *User) (*TokenPair, error) {
	access, err := s.sign(u, mw.TokenTypeAccess, s.cfg.AccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.sign(u, mw.TokenTypeRefresh, s.cfg.RefreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.AccessTTL.Seconds()),
		User:         toUserInfo(u),
	}, nil
}

func (s *Service) sign(u *User, typ string, ttl time.Duration) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := mw.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.UserID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "PenbunAPI",
		},
		Username:     u.UserName,
		UserLevel:    u.UserLevel,
		MustChangePW: u.MustChangePW,
		TokenType:    typ,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
}

func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func toUserInfo(u *User) *UserInfo {
	info := &UserInfo{
		UserID:             u.UserID,
		UserName:           u.UserName,
		UserLevel:          u.UserLevel,
		MustChangePassword: u.MustChangePW,
	}
	if u.FullName.Valid {
		info.FullName = &u.FullName.String
	}
	if u.Email.Valid {
		info.Email = &u.Email.String
	}
	if u.LastLogin.Valid {
		s := u.LastLogin.Time.Format(time.RFC3339)
		info.LastLoginDate = &s
	}
	return info
}

func validatePasswordPolicy(pw string) error {
	if len([]rune(pw)) < 8 {
		return httpx.Validation("รหัสผ่านต้องยาวอย่างน้อย 8 ตัวอักษร")
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return httpx.Validation("รหัสผ่านต้องมีทั้งตัวอักษรและตัวเลข")
	}
	return nil
}
