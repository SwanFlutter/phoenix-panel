package api

import (
	"time"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// ---- Auth ----

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	Admin     adminDTO  `json:"admin"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type adminDTO struct {
	ID       uint             `json:"id"`
	Username string           `json:"username"`
	Role     models.AdminRole `json:"role"`
}

func toAdminDTO(a models.Admin) adminDTO {
	return adminDTO{ID: a.ID, Username: a.Username, Role: a.Role}
}

// ---- Users ----

type createUserRequest struct {
	Username     string                   `json:"username" binding:"required,min=1,max=64"`
	DataLimit    int64                    `json:"data_limit"` // bytes, 0 = unlimited
	ExpireAt     *time.Time               `json:"expire_at"`
	DataStrategy models.DataLimitStrategy `json:"data_strategy"`
	SSMethod     string                   `json:"ss_method"`
	Note         string                   `json:"note" binding:"max=500"`
	InboundIDs   []uint                   `json:"inbound_ids"`
}

type updateUserRequest struct {
	Status       *models.UserStatus        `json:"status"`
	DataLimit    *int64                    `json:"data_limit"`
	ExpireAt     *time.Time                `json:"expire_at"`
	ClearExpiry  bool                      `json:"clear_expiry"`
	DataStrategy *models.DataLimitStrategy `json:"data_strategy"`
	Note         *string                   `json:"note"`
	InboundIDs   *[]uint                   `json:"inbound_ids"`
}

// userDTO is the API representation of a user, including derived fields.
type userDTO struct {
	ID            uint              `json:"id"`
	Username      string            `json:"username"`
	Status        models.UserStatus `json:"status"`
	UUID          string            `json:"uuid"`
	SubToken      string            `json:"sub_token"`
	SubscriptionURL string          `json:"subscription_url"`
	DataLimit     int64             `json:"data_limit"`
	UsedUp        int64             `json:"used_up"`
	UsedDown      int64             `json:"used_down"`
	UsedTotal     int64             `json:"used_total"`
	RemainingData int64             `json:"remaining_data"`
	ExpireAt      *time.Time        `json:"expire_at"`
	Note          string            `json:"note"`
	CreatedAt     time.Time         `json:"created_at"`
	OnlineAt      *time.Time        `json:"online_at"`
}

type listUsersResponse struct {
	Items  []userDTO `json:"items"`
	Total  int64     `json:"total"`
	Offset int       `json:"offset"`
	Limit  int       `json:"limit"`
}

func toUserDTO(u models.User, baseURL string) userDTO {
	now := time.Now()
	return userDTO{
		ID:              u.ID,
		Username:        u.Username,
		Status:          u.EffectiveStatus(now),
		UUID:            u.UUID,
		SubToken:        u.SubToken,
		SubscriptionURL: baseURL + "/sub/" + u.SubToken,
		DataLimit:       u.DataLimit,
		UsedUp:          u.UsedUp,
		UsedDown:        u.UsedDown,
		UsedTotal:       u.UsedTotal(),
		RemainingData:   u.RemainingData(),
		ExpireAt:        u.ExpireAt,
		Note:            u.Note,
		CreatedAt:       u.CreatedAt,
		OnlineAt:        u.OnlineAt,
	}
}
