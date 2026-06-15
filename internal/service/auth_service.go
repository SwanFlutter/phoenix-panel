package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/security"
)

// ErrInvalidCredentials is returned for any failed login (kept generic to avoid
// username enumeration).
var ErrInvalidCredentials = errors.New("invalid credentials")

// AuthService handles admin authentication.
type AuthService struct {
	db     *gorm.DB
	jwtMgr *security.JWTManager
}

// NewAuthService constructs an AuthService.
func NewAuthService(db *gorm.DB, jwtMgr *security.JWTManager) *AuthService {
	return &AuthService{db: db, jwtMgr: jwtMgr}
}

// LoginResult bundles the issued token and its expiry.
type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	Admin     models.Admin
}

// Login verifies credentials and issues a JWT. On success it records the login
// timestamp/IP. The same generic error is returned for unknown user, wrong
// password, and disabled account.
func (s *AuthService) Login(ctx context.Context, username, password, ip string) (*LoginResult, error) {
	var admin models.Admin
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Still run a hash verification against a dummy to equalize timing.
		_, _ = security.VerifyPassword(password, dummyHash)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !admin.IsActive {
		return nil, ErrInvalidCredentials
	}

	ok, err := security.VerifyPassword(password, admin.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	token, expiry, err := s.jwtMgr.Generate(admin.ID, admin.Username, string(admin.Role))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.db.WithContext(ctx).Model(&admin).Updates(map[string]any{
		"last_login_at": &now,
		"last_login_ip": ip,
	})

	return &LoginResult{Token: token, ExpiresAt: expiry, Admin: admin}, nil
}

// ChangePassword updates an admin's password after verifying the current one.
func (s *AuthService) ChangePassword(ctx context.Context, adminID uint, current, next string) error {
	if len(next) < 8 {
		return errors.New("new password must be at least 8 characters")
	}
	var admin models.Admin
	if err := s.db.WithContext(ctx).First(&admin, adminID).Error; err != nil {
		return err
	}
	ok, err := security.VerifyPassword(current, admin.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}
	hash, err := security.HashPassword(next)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&admin).Update("password_hash", hash).Error
}

// dummyHash is a precomputed argon2id hash used to equalize timing when the
// username does not exist (mitigates user-enumeration via response time).
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$RdescudvJCsgt3ub+b+dWRWJTmaQVgSJMOdMtJ7XPmo"
