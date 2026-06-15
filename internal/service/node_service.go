package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// NodeService manages nodes and their inbounds.
type NodeService struct {
	db *gorm.DB
}

// NewNodeService constructs a NodeService.
func NewNodeService(db *gorm.DB) *NodeService { return &NodeService{db: db} }

// ListNodes returns all nodes with their inbounds.
func (s *NodeService) ListNodes(ctx context.Context) ([]models.Node, error) {
	var nodes []models.Node
	err := s.db.WithContext(ctx).Preload("Inbounds").Order("id ASC").Find(&nodes).Error
	return nodes, err
}

// GetNode loads a single node with inbounds.
func (s *NodeService) GetNode(ctx context.Context, id uint) (*models.Node, error) {
	var n models.Node
	err := s.db.WithContext(ctx).Preload("Inbounds").First(&n, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &n, err
}

// CreateNode adds a node.
func (s *NodeService) CreateNode(ctx context.Context, n *models.Node) error {
	if n.Name == "" || n.Address == "" {
		return fmt.Errorf("%w: name and address are required", ErrValidation)
	}
	if n.Core != models.CoreXray && n.Core != models.CoreSingBox {
		return fmt.Errorf("%w: core must be xray or sing-box", ErrValidation)
	}
	return s.db.WithContext(ctx).Create(n).Error
}

// DeleteNode removes a node and its inbounds (cascade via FK).
func (s *NodeService) DeleteNode(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&models.Node{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateInbound adds an inbound to a node, validating protocol/core compat.
func (s *NodeService) CreateInbound(ctx context.Context, in *models.Inbound) error {
	if !in.Protocol.Valid() {
		return fmt.Errorf("%w: unknown protocol %q", ErrValidation, in.Protocol)
	}
	node, err := s.GetNode(ctx, in.NodeID)
	if err != nil {
		return err
	}
	if !in.Protocol.SupportedBy(node.Core) {
		return fmt.Errorf("%w: protocol %s is not supported by core %s", ErrValidation, in.Protocol, node.Core)
	}
	if in.Port <= 0 || in.Port > 65535 {
		return fmt.Errorf("%w: port must be 1-65535", ErrValidation)
	}
	return s.db.WithContext(ctx).Create(in).Error
}

// DeleteInbound removes an inbound.
func (s *NodeService) DeleteInbound(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&models.Inbound{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
