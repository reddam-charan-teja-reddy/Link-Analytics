package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshSessionRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshSessionRepo(pool *pgxpool.Pool) *RefreshSessionRepo {
	return &RefreshSessionRepo{pool: pool}
}

func (r *RefreshSessionRepo) Create(ctx context.Context, userID, familyID, tokenJTI string, expiresAt time.Time, parentJTI *string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_sessions (user_id, family_id, token_jti, parent_jti, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, familyID, tokenJTI, parentJTI, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}

	return nil
}

func (r *RefreshSessionRepo) Rotate(ctx context.Context, userID, familyID, currentJTI, nextJTI string, nextExpiresAt time.Time) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin refresh rotation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE refresh_sessions
		 SET revoked_at = NOW(), revoked_reason = 'rotated'
		 WHERE user_id = $1
		   AND family_id = $2
		   AND token_jti = $3
		   AND revoked_at IS NULL
		   AND expires_at > NOW()`,
		userID, familyID, currentJTI,
	)
	if err != nil {
		return false, fmt.Errorf("revoke current refresh session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return false, nil
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO refresh_sessions (user_id, family_id, token_jti, parent_jti, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, familyID, nextJTI, currentJTI, nextExpiresAt,
	)
	if err != nil {
		return false, fmt.Errorf("create rotated refresh session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit refresh rotation tx: %w", err)
	}

	return true, nil
}

func (r *RefreshSessionRepo) RevokeFamily(ctx context.Context, userID, familyID, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_sessions
		 SET revoked_at = NOW(), revoked_reason = $3
		 WHERE user_id = $1
		   AND family_id = $2
		   AND revoked_at IS NULL`,
		userID, familyID, reason,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh family: %w", err)
	}

	return nil
}
