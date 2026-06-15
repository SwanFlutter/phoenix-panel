package core

import (
	"context"
	"log/slog"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// XrayAdapter drives an Xray-core instance via its gRPC HandlerService /
// StatsService API (the standard management API exposed by Xray's "api" inbound).
//
// NOTE: The gRPC wiring to xray-core's proto services is the production
// integration point. This adapter is structured so that swapping the stubbed
// methods for real gRPC calls requires no changes elsewhere in the panel.
// The xray-core proto clients (app/proxyman/command, app/stats/command) should
// be vendored and a *grpc.ClientConn dialed against node.APIHost:node.APIPort.
type XrayAdapter struct {
	node models.Node
	// conn *grpc.ClientConn  // wired in production
}

// NewXrayAdapter constructs an Xray adapter bound to a node.
func NewXrayAdapter(node models.Node) *XrayAdapter {
	return &XrayAdapter{node: node}
}

func (a *XrayAdapter) Type() models.CoreType { return models.CoreXray }

func (a *XrayAdapter) AddUser(ctx context.Context, inbound models.Inbound, user models.User) error {
	// Production: HandlerService.AlterInbound with an AddUserOperation carrying
	// the protocol-specific account (VLESS id, VMess id, Trojan/SS password).
	slog.Debug("xray.AddUser (stub)", "node", a.node.Name, "inbound", inbound.Tag, "user", user.Username)
	return nil
}

func (a *XrayAdapter) RemoveUser(ctx context.Context, inbound models.Inbound, user models.User) error {
	// Production: HandlerService.AlterInbound with a RemoveUserOperation by email/tag.
	slog.Debug("xray.RemoveUser (stub)", "node", a.node.Name, "inbound", inbound.Tag, "user", user.Username)
	return nil
}

func (a *XrayAdapter) QueryUsage(ctx context.Context, reset bool) ([]Usage, error) {
	// Production: StatsService.QueryStats with pattern "user>>>*>>>traffic>>>*".
	slog.Debug("xray.QueryUsage (stub)", "node", a.node.Name, "reset", reset)
	return nil, nil
}

func (a *XrayAdapter) Stats(ctx context.Context) (SystemStats, error) {
	// Production: StatsService.GetSysStats.
	return SystemStats{Version: a.node.XrayVersion, Online: a.node.IsActive}, nil
}

func (a *XrayAdapter) Ping(ctx context.Context) error {
	// Production: a lightweight GetSysStats call to verify connectivity.
	return nil
}
