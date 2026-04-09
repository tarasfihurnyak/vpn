package peer

import (
	"context"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"

	db "vpn/internal/db/sqlc"
)

type Service struct {
	q db.Querier
}

func NewService(q db.Querier) *Service {
	return &Service{q: q}
}

func toModel(p db.Peer) Peer {
	return Peer{
		ID:        p.ID,
		UserID:    p.UserID,
		Name:      p.Name,
		PublicKey: p.PublicKey,
		IPAddress: p.IpAddress,
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
}

func (s *Service) Create(ctx context.Context, userID pgtype.UUID, name, publicKey string, ipAddress netip.Addr) (Peer, error) {
	p, err := s.q.CreatePeer(ctx, db.CreatePeerParams{
		UserID:    userID,
		Name:      name,
		PublicKey: publicKey,
		IpAddress: ipAddress,
	})
	if err != nil {
		return Peer{}, err
	}
	return toModel(p), nil
}

func (s *Service) GetByID(ctx context.Context, id pgtype.UUID) (Peer, error) {
	p, err := s.q.GetPeer(ctx, id)
	if err != nil {
		return Peer{}, err
	}
	return toModel(p), nil
}

func (s *Service) GetByPublicKey(ctx context.Context, publicKey string) (Peer, error) {
	p, err := s.q.GetPeerByPublicKey(ctx, publicKey)
	if err != nil {
		return Peer{}, err
	}
	return toModel(p), nil
}

func (s *Service) GetByUser(ctx context.Context, userID pgtype.UUID) (Peer, error) {
	p, err := s.q.GetPeerByUser(ctx, userID)
	if err != nil {
		return Peer{}, err
	}
	return toModel(p), nil
}

func (s *Service) ListByUser(ctx context.Context, userID pgtype.UUID) ([]Peer, error) {
	peers, err := s.q.ListPeersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Peer, len(peers))
	for i, p := range peers {
		result[i] = toModel(p)
	}
	return result, nil
}

func (s *Service) ListEnabled(ctx context.Context) ([]Peer, error) {
	peers, err := s.q.ListEnabledPeers(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Peer, len(peers))
	for i, p := range peers {
		result[i] = toModel(p)
	}
	return result, nil
}

func (s *Service) Enable(ctx context.Context, id pgtype.UUID) error {
	return s.q.EnablePeer(ctx, id)
}

func (s *Service) Disable(ctx context.Context, id pgtype.UUID) error {
	return s.q.DisablePeer(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeletePeer(ctx, id)
}
