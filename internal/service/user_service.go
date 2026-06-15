// Package service holds the business logic layer, sitting between HTTP handlers
// and the database. Handlers stay thin; services own validation and invariants.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/security"
)

// Sentinel errors the HTTP layer maps to status codes.
var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("already exists")
	ErrValidation    = errors.New("validation failed")
)

// UserService encapsulates user CRUD and provisioning.
type UserService struct {
	db *gorm.DB
}

// NewUserService constructs a UserService.
func NewUserService(db *gorm.DB) *UserService { return &UserService{db: db} }

// CreateUserInput is the validated payload for creating a user.
type CreateUserInput struct {
	Username     string
	DataLimit    int64
	ExpireAt     *time.Time
	DataStrategy models.DataLimitStrategy
	SSMethod     string
	Note         string
	InboundIDs   []uint
}

// Create provisions a new user with freshly generated credentials.
func (s *UserService) Create(ctx context.Context, in CreateUserInput) (*models.User, error) {
	if in.Username == "" {
		return nil, fmt.Errorf("%w: username is required", ErrValidation)
	}
	if in.DataLimit < 0 {
		return nil, fmt.Errorf("%w: data_limit must be >= 0", ErrValidation)
	}

	uuid := security.NewUUID()
	trojanPw, err := security.NewPassword(16)
	if err != nil {
		return nil, err
	}
	ssPw, err := security.NewPassword(16)
	if err != nil {
		return nil, err
	}
	subToken, err := security.NewToken(24)
	if err != nil {
		return nil, err
	}

	strategy := in.DataStrategy
	if strategy == "" {
		strategy = models.StrategyNoReset
	}
	ssMethod := in.SSMethod
	if ssMethod == "" {
		ssMethod = "chacha20-ietf-poly1305"
	}

	user := &models.User{
		Username:       in.Username,
		Status:         models.UserActive,
		UUID:           uuid,
		TrojanPassword: trojanPw,
		SSPassword:     ssPw,
		SSMethod:       ssMethod,
		SubToken:       subToken,
		DataLimit:      in.DataLimit,
		DataStrategy:   strategy,
		ExpireAt:       in.ExpireAt,
		Note:           in.Note,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: username already taken", ErrConflict)
			}
			return err
		}
		return associateInbounds(tx, user.ID, in.InboundIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, user.ID)
}

// GetByID loads a user with their inbounds.
func (s *UserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).Preload("Inbounds").First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetBySubToken loads a user by their public subscription token.
func (s *UserService) GetBySubToken(ctx context.Context, token string) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).Preload("Inbounds").Where("sub_token = ?", token).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListParams controls pagination and filtering for List.
type ListParams struct {
	Offset int
	Limit  int
	Status models.UserStatus
	Search string
}

// List returns a page of users and the total count.
func (s *UserService) List(ctx context.Context, p ListParams) ([]models.User, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.User{})
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.Search != "" {
		like := "%" + p.Search + "%"
		q = q.Where("username LIKE ? OR note LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	var users []models.User
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&users).Error
	return users, total, err
}

// UpdateUserInput carries optional fields; nil pointers mean "leave unchanged".
type UpdateUserInput struct {
	Status       *models.UserStatus
	DataLimit    *int64
	ExpireAt     **time.Time // double pointer: distinguish "set null" from "unchanged"
	DataStrategy *models.DataLimitStrategy
	Note         *string
	InboundIDs   *[]uint
}

// Update mutates an existing user.
func (s *UserService) Update(ctx context.Context, id uint, in UpdateUserInput) (*models.User, error) {
	return s.txReturnUser(ctx, id, func(tx *gorm.DB, u *models.User) error {
		if in.Status != nil {
			u.Status = *in.Status
		}
		if in.DataLimit != nil {
			if *in.DataLimit < 0 {
				return fmt.Errorf("%w: data_limit must be >= 0", ErrValidation)
			}
			u.DataLimit = *in.DataLimit
		}
		if in.ExpireAt != nil {
			u.ExpireAt = *in.ExpireAt
		}
		if in.DataStrategy != nil {
			u.DataStrategy = *in.DataStrategy
		}
		if in.Note != nil {
			u.Note = *in.Note
		}
		if err := tx.Save(u).Error; err != nil {
			return err
		}
		if in.InboundIDs != nil {
			if err := tx.Where("user_id = ?", u.ID).Delete(&models.UserInbound{}).Error; err != nil {
				return err
			}
			return associateInbounds(tx, u.ID, *in.InboundIDs)
		}
		return nil
	})
}

// ResetTraffic zeroes a user's usage counters.
func (s *UserService) ResetTraffic(ctx context.Context, id uint) (*models.User, error) {
	return s.txReturnUser(ctx, id, func(tx *gorm.DB, u *models.User) error {
		now := time.Now()
		u.UsedUp = 0
		u.UsedDown = 0
		u.LastResetAt = &now
		if u.Status == models.UserLimited {
			u.Status = models.UserActive
		}
		return tx.Save(u).Error
	})
}

// Delete removes a user and their inbound associations.
func (s *UserService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserInbound{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&models.User{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// RegenerateSubToken rotates a user's subscription token, invalidating old links.
func (s *UserService) RegenerateSubToken(ctx context.Context, id uint) (*models.User, error) {
	return s.txReturnUser(ctx, id, func(tx *gorm.DB, u *models.User) error {
		tok, err := security.NewToken(24)
		if err != nil {
			return err
		}
		u.SubToken = tok
		return tx.Save(u).Error
	})
}

// txReturnUser loads a user in a transaction, applies fn, then reloads it.
func (s *UserService) txReturnUser(ctx context.Context, id uint, fn func(tx *gorm.DB, u *models.User) error) (*models.User, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u models.User
		if err := tx.First(&u, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		return fn(tx, &u)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func associateInbounds(tx *gorm.DB, userID uint, inboundIDs []uint) error {
	if len(inboundIDs) == 0 {
		return nil
	}
	rows := make([]models.UserInbound, 0, len(inboundIDs))
	for _, iid := range inboundIDs {
		rows = append(rows, models.UserInbound{UserID: userID, InboundID: iid, Enabled: true})
	}
	return tx.Create(&rows).Error
}
