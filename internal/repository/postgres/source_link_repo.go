package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SourceLinkRepo struct {
	pool *pgxpool.Pool
}

func NewSourceLinkRepo(pool *pgxpool.Pool) *SourceLinkRepo {
	return &SourceLinkRepo{pool: pool}
}

func (r *SourceLinkRepo) Create(ctx context.Context, linkID, sourceName, hash string) (*domain.SourceLink, error) {
	sourceName = strings.TrimSpace(strings.ToLower(sourceName))

	sl := &domain.SourceLink{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO source_links (link_id, source_name, hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, link_id, source_name, hash, is_active, created_at`,
		linkID, sourceName, hash,
	).Scan(&sl.ID, &sl.LinkID, &sl.SourceName, &sl.Hash, &sl.IsActive, &sl.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create source link: %w", err)
	}
	return sl, nil
}

func (r *SourceLinkRepo) GetByHash(ctx context.Context, hash string) (*domain.SourceLink, error) {
	sl := &domain.SourceLink{}
	err := r.pool.QueryRow(ctx,
		`SELECT sl.id, sl.link_id, sl.source_name, sl.hash, sl.is_active, sl.created_at
		 FROM source_links sl
		 JOIN links l ON l.id = sl.link_id
		 WHERE sl.hash = $1 AND sl.is_active = true AND l.is_active = true`,
		hash,
	).Scan(&sl.ID, &sl.LinkID, &sl.SourceName, &sl.Hash, &sl.IsActive, &sl.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get source link by hash: %w", err)
	}
	return sl, nil
}

func (r *SourceLinkRepo) ListByLinkID(ctx context.Context, linkID string) ([]domain.SourceLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, link_id, source_name, hash, is_active, created_at
		 FROM source_links WHERE link_id = $1 ORDER BY created_at DESC`,
		linkID,
	)
	if err != nil {
		return nil, fmt.Errorf("list source links: %w", err)
	}
	defer rows.Close()

	var sources []domain.SourceLink
	for rows.Next() {
		var sl domain.SourceLink
		if err := rows.Scan(&sl.ID, &sl.LinkID, &sl.SourceName, &sl.Hash, &sl.IsActive, &sl.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan source link: %w", err)
		}
		sources = append(sources, sl)
	}
	return sources, nil
}

func (r *SourceLinkRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM source_links WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete source link: %w", err)
	}
	return nil
}
