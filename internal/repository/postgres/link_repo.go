package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepo struct {
	pool *pgxpool.Pool
}

func NewLinkRepo(pool *pgxpool.Pool) *LinkRepo {
	return &LinkRepo{pool: pool}
}

func (r *LinkRepo) Create(ctx context.Context, userID, originalURL, hash, title string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO links (user_id, original_url, hash, title)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, original_url, hash, title, is_active, created_at, updated_at`,
		userID, originalURL, hash, title,
	).Scan(&link.ID, &link.UserID, &link.OriginalURL, &link.Hash, &link.Title, &link.IsActive, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create link: %w", err)
	}
	return link, nil
}

func (r *LinkRepo) GetByID(ctx context.Context, id string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.pool.QueryRow(ctx,
		`SELECT l.id, l.user_id, l.original_url, l.hash, l.title, l.is_active, l.created_at, l.updated_at
		 FROM links l
		 WHERE l.id = $1`,
		id,
	).Scan(&link.ID, &link.UserID, &link.OriginalURL, &link.Hash, &link.Title, &link.IsActive, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get link: %w", err)
	}
	return link, nil
}

func (r *LinkRepo) GetByHash(ctx context.Context, hash string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.pool.QueryRow(ctx,
		`SELECT l.id, l.user_id, l.original_url, l.hash, l.title, l.is_active, l.created_at, l.updated_at
		 FROM links l
		 WHERE l.hash = $1 AND l.is_active = true`,
		hash,
	).Scan(&link.ID, &link.UserID, &link.OriginalURL, &link.Hash, &link.Title, &link.IsActive, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get link by hash: %w", err)
	}
	return link, nil
}

func (r *LinkRepo) ListByUserID(ctx context.Context, userID, groupID, sourceFilter string) ([]domain.Link, error) {
	query := `SELECT l.id,
			l.user_id,
			l.original_url,
			l.hash,
			l.title,
			l.is_active,
			COALESCE(COUNT(ce.id), 0) AS total_clicks,
			MAX(ce.clicked_at) AS last_clicked_at,
			l.created_at,
			l.updated_at
		 FROM links l
		 LEFT JOIN click_events ce ON ce.link_id = l.id`

	conditions := []string{"l.user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	if groupID != "" {
		query += ` JOIN link_groups lg ON lg.link_id = l.id`
		conditions = append(conditions, fmt.Sprintf("lg.group_id = $%d", argIndex))
		args = append(args, groupID)
		argIndex++
	}

	if sourceFilter != "" {
		conditions = append(conditions, fmt.Sprintf(`(
			EXISTS (SELECT 1 FROM source_links sl WHERE sl.link_id = l.id AND sl.source_name ILIKE $%d)
			OR EXISTS (SELECT 1 FROM click_events ce2 WHERE ce2.link_id = l.id AND ce2.source_name ILIKE $%d)
		)`, argIndex, argIndex))
		args = append(args, "%"+strings.TrimSpace(sourceFilter)+"%")
		argIndex++
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " GROUP BY l.id"
	query += " ORDER BY l.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()

	var links []domain.Link
	for rows.Next() {
		var link domain.Link
		if err := rows.Scan(
			&link.ID,
			&link.UserID,
			&link.OriginalURL,
			&link.Hash,
			&link.Title,
			&link.IsActive,
			&link.TotalClicks,
			&link.LastClickedAt,
			&link.CreatedAt,
			&link.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
}

func (r *LinkRepo) Update(ctx context.Context, id, title string, isActive bool) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.pool.QueryRow(ctx,
		`UPDATE links SET title = $2, is_active = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, user_id, original_url, hash, title, is_active, created_at, updated_at`,
		id, title, isActive,
	).Scan(&link.ID, &link.UserID, &link.OriginalURL, &link.Hash, &link.Title, &link.IsActive, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update link: %w", err)
	}
	return link, nil
}

func (r *LinkRepo) ListLinkIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM links WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list link ids: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan link id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *LinkRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM links WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}
