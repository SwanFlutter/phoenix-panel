// Package core abstracts the proxy backend so the panel can drive either
// Xray-core or sing-box through one interface. Adapters translate the panel's
// domain models into core-specific user provisioning and traffic queries.
package core

import (
	"context"
	"fmt"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// Usage is a per-user traffic reading pulled from a core.
type Usage struct {
	UserTag string // typically the user's UUID or username tag in the core
	Up      int64  // bytes
	Down    int64  // bytes
}

// SystemStats is a coarse health/stat snapshot of a running core.
type SystemStats struct {
	Version       string
	Uptime        int64 // seconds
	NumGoroutine  int
	MemAllocBytes uint64
	Online        bool
}

// ProxyCore is the contract every backend adapter must satisfy. Implementations
// must be safe for concurrent use.
type ProxyCore interface {
	// Type identifies the backing core.
	Type() models.CoreType

	// AddUser provisions a user on the given inbound.
	AddUser(ctx context.Context, inbound models.Inbound, user models.User) error
	// RemoveUser deprovisions a user from the given inbound.
	RemoveUser(ctx context.Context, inbound models.Inbound, user models.User) error

	// QueryUsage returns per-user traffic counters, optionally resetting them.
	QueryUsage(ctx context.Context, reset bool) ([]Usage, error)

	// Stats returns core health information.
	Stats(ctx context.Context) (SystemStats, error)

	// Ping reports whether the core API is reachable.
	Ping(ctx context.Context) error
}

// Factory builds a ProxyCore for a node based on its configured core type.
func Factory(node models.Node) (ProxyCore, error) {
	switch node.Core {
	case models.CoreXray:
		return NewXrayAdapter(node), nil
	case models.CoreSingBox:
		return NewSingBoxAdapter(node), nil
	default:
		return nil, fmt.Errorf("core: unsupported core type %q", node.Core)
	}
}
