package postgres

import (
	"context"
	"fmt"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepo struct {
	pool *pgxpool.Pool
}

func NewGroupRepo(pool *pgxpool.Pool) *GroupRepo {
	return &GroupRepo{pool: pool}
}

func (r *GroupRepo) Create(ctx context.Context, userID, name string) (*domain.Group, error) {
	group := &domain.Group{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO groups (user_id, name)
		 VALUES ($1, $2)
		 RETURNING id, user_id, name, created_at`,
		userID, name,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return group, nil
}

func (r *GroupRepo) ListByUserID(ctx context.Context, userID string) ([]domain.Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.user_id, g.name, g.created_at, COUNT(lg.link_id) AS link_count
		 FROM groups g
		 LEFT JOIN link_groups lg ON lg.group_id = g.id
		 WHERE g.user_id = $1
		 GROUP BY g.id
		 ORDER BY g.name ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var items []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.UserID, &g.Name, &g.CreatedAt, &g.LinkCount); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		items = append(items, g)
	}

	return items, nil
}

func (r *GroupRepo) AddLink(ctx context.Context, userID, groupID, linkID string) error {
	var allowed bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM groups g
			JOIN links l ON l.user_id = g.user_id
			WHERE g.id = $1 AND l.id = $2 AND g.user_id = $3
		)`,
		groupID, linkID, userID,
	).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("check group link ownership: %w", err)
	}
	if !allowed {
		return fmt.Errorf("group or link not found")
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO link_groups (group_id, link_id)
		 VALUES ($1, $2)
		 ON CONFLICT (group_id, link_id) DO NOTHING`,
		groupID, linkID,
	)
	if err != nil {
		return fmt.Errorf("add link to group: %w", err)
	}

	return nil
}

func (r *GroupRepo) RemoveLink(ctx context.Context, userID, groupID, linkID string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM link_groups lg
		 USING groups g, links l
		 WHERE lg.group_id = g.id
		   AND lg.link_id = l.id
		   AND g.id = $1
		   AND l.id = $2
		   AND g.user_id = $3
		   AND l.user_id = $3`,
		groupID, linkID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove link from group: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("group or link not found")
	}
	return nil
}

func (r *GroupRepo) ListLinkIDsByGroup(ctx context.Context, userID, groupID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT lg.link_id
		 FROM link_groups lg
		 JOIN groups g ON g.id = lg.group_id
		 JOIN links l ON l.id = lg.link_id
		 WHERE g.id = $1 AND g.user_id = $2 AND l.user_id = $2`,
		groupID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list link ids by group: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan group link id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (r *GroupRepo) Update(ctx context.Context, userID, groupID, name string) (*domain.Group, error) {
	group := &domain.Group{}
	err := r.pool.QueryRow(ctx,
		`UPDATE groups
		 SET name = $3
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, user_id, name, created_at`,
		groupID, userID, name,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}

	return group, nil
}

func (r *GroupRepo) Delete(ctx context.Context, userID, groupID string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM groups WHERE id = $1 AND user_id = $2`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

func (r *GroupRepo) ListByLinkID(ctx context.Context, userID, linkID string) ([]domain.Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.user_id, g.name, g.created_at
		 FROM groups g
		 JOIN link_groups lg ON lg.group_id = g.id
		 JOIN links l ON l.id = lg.link_id
		 WHERE g.user_id = $1 AND l.user_id = $1 AND l.id = $2
		 ORDER BY g.name ASC`,
		userID, linkID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups by link: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Group, 0)
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.UserID, &g.Name, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan group by link: %w", err)
		}
		items = append(items, g)
	}

	return items, nil
}
