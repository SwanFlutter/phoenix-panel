package core

import (
	"context"
	"log/slog"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// SingBoxAdapter drives a sing-box instance. sing-box exposes management via
// its Clash-API / V2 API (HTTP) and, for user provisioning, via config reload
// or the experimental management endpoints.
//
// NOTE: Like the Xray adapter, the HTTP/management wiring is the production
// integration point. sing-box commonly manages users by rewriting its config
// and triggering a hot reload, or via the experimental cache-file users API.
type SingBoxAdapter struct {
	node models.Node
	// client *http.Client // wired in production
}

// NewSingBoxAdapter constructs a sing-box adapter bound to a node.
func NewSingBoxAdapter(node models.Node) *SingBoxAdapter {
	return &SingBoxAdapter{node: node}
}

func (a *SingBoxAdapter) Type() models.CoreType { return models.CoreSingBox }

func (a *SingBoxAdapter) AddUser(ctx context.Context, inbound models.Inbound, user models.User) error {
	slog.Debug("singbox.AddUser (stub)", "node", a.node.Name, "inbound", inbound.Tag, "user", user.Username)
	return nil
}

func (a *SingBoxAdapter) RemoveUser(ctx context.Context, inbound models.Inbound, user models.User) error {
	slog.Debug("singbox.RemoveUser (stub)", "node", a.node.Name, "inbound", inbound.Tag, "user", user.Username)
	return nil
}

func (a *SingBoxAdapter) QueryUsage(ctx context.Context, reset bool) ([]Usage, error) {
	// Production: GET /traffic or read per-user counters from the Clash API.
	slog.Debug("singbox.QueryUsage (stub)", "node", a.node.Name, "reset", reset)
	return nil, nil
}

func (a *SingBoxAdapter) Stats(ctx context.Context) (SystemStats, error) {
	return SystemStats{Version: a.node.XrayVersion, Online: a.node.IsActive}, nil
}

func (a *SingBoxAdapter) Ping(ctx context.Context) error {
	return nil
}
