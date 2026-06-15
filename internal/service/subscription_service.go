package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/links"
	"github.com/phoenix-panel/phoenix/internal/models"
)

// SubscriptionService assembles a user's connectable links by joining their
// enabled inbounds with the nodes that host them.
type SubscriptionService struct {
	db    *gorm.DB
	users *UserService
}

// NewSubscriptionService constructs a SubscriptionService.
func NewSubscriptionService(db *gorm.DB, users *UserService) *SubscriptionService {
	return &SubscriptionService{db: db, users: users}
}

// SubscriptionData bundles a user with the rendered links and usage metadata
// surfaced to subscription clients (via headers).
type SubscriptionData struct {
	User  models.User
	URIs  []string
	Pairs []links.NodeInbound
}

// Build resolves a subscription by token. Returns ErrNotFound if the token is
// unknown. Disabled/expired users still receive an (empty) document so clients
// don't error; callers should reflect status in headers.
func (s *SubscriptionService) Build(ctx context.Context, token string) (*SubscriptionData, error) {
	user, err := s.users.GetBySubToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Record the fetch for "last subscription update" tracking.
	now := time.Now()
	s.db.WithContext(ctx).Model(user).Update("sub_last_at", &now)

	data := &SubscriptionData{User: *user}

	// Only build links if the user is currently allowed to connect.
	if !user.CanConnect(now) {
		return data, nil
	}

	pairs, err := s.resolvePairs(ctx, user.Inbounds)
	if err != nil {
		return nil, err
	}
	data.Pairs = pairs
	data.URIs = links.BuildAll(*user, pairs)
	return data, nil
}

// resolvePairs loads the node for each inbound the user is provisioned on.
func (s *SubscriptionService) resolvePairs(ctx context.Context, inbounds []models.Inbound) ([]links.NodeInbound, error) {
	if len(inbounds) == 0 {
		return nil, nil
	}
	nodeIDs := make([]uint, 0, len(inbounds))
	seen := map[uint]struct{}{}
	for _, in := range inbounds {
		if _, ok := seen[in.NodeID]; !ok {
			seen[in.NodeID] = struct{}{}
			nodeIDs = append(nodeIDs, in.NodeID)
		}
	}
	var nodes []models.Node
	if err := s.db.WithContext(ctx).Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return nil, err
	}
	nodeByID := make(map[uint]models.Node, len(nodes))
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	pairs := make([]links.NodeInbound, 0, len(inbounds))
	for _, in := range inbounds {
		node, ok := nodeByID[in.NodeID]
		if !ok {
			continue
		}
		pairs = append(pairs, links.NodeInbound{Node: node, Inbound: in})
	}
	return pairs, nil
}
