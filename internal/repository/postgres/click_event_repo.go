package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClickEventRepo struct {
	pool *pgxpool.Pool
}

func NewClickEventRepo(pool *pgxpool.Pool) *ClickEventRepo {
	return &ClickEventRepo{pool: pool}
}

func (r *ClickEventRepo) BatchInsert(ctx context.Context, events []domain.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	values := make([]string, 0, len(events))
	args := make([]interface{}, 0, len(events)*13)
	for i, e := range events {
		base := i * 13
		values = append(values, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12, base+13,
		))
		args = append(args, e.Hash, e.LinkID, e.SourceLinkID, e.IPAddress, e.UserAgent,
			e.Referer, e.Browser, e.OS, e.Country, e.City, e.IsBot, e.ClickedAt, e.SourceName)
	}

	query := `INSERT INTO click_events (hash, link_id, source_link_id, ip_address, user_agent, referer, browser, os, country, city, is_bot, clicked_at, source_name) VALUES ` + strings.Join(values, ",")

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch insert click events: %w", err)
	}
	return nil
}

type AnalyticsSummary struct {
	TotalClicks    int        `json:"total_clicks"`
	UniqueVisitors int        `json:"unique_visitors"`
	BotClicks      int        `json:"bot_clicks"`
	LastClickedAt  *time.Time `json:"last_clicked_at"`
}

func (r *ClickEventRepo) GetSummary(ctx context.Context, linkID string, from, to time.Time) (*AnalyticsSummary, error) {
	summary := &AnalyticsSummary{}
	err := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) as total_clicks,
			COUNT(DISTINCT ip_address) as unique_visitors,
			COUNT(*) FILTER (WHERE is_bot = true) as bot_clicks,
			MAX(clicked_at) as last_clicked_at
		 FROM click_events
		 WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at <= $3`,
		linkID, from, to,
	).Scan(&summary.TotalClicks, &summary.UniqueVisitors, &summary.BotClicks, &summary.LastClickedAt)
	if err != nil {
		return nil, fmt.Errorf("get analytics summary: %w", err)
	}
	return summary, nil
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Clicks    int       `json:"clicks"`
}

func (r *ClickEventRepo) GetClickTimeSeries(ctx context.Context, linkID string, from, to time.Time, granularity string) ([]TimeSeriesPoint, error) {
	trunc := "day"
	if granularity == "hour" {
		trunc = "hour"
	}

	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT date_trunc('%s', clicked_at) as bucket, COUNT(*) as clicks
		 FROM click_events
		 WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at <= $3
		 GROUP BY bucket ORDER BY bucket`, trunc),
		linkID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get click time series: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Clicks); err != nil {
			return nil, fmt.Errorf("scan time series: %w", err)
		}
		points = append(points, p)
	}
	return points, nil
}

type BreakdownItem struct {
	Label  string `json:"label"`
	Clicks int    `json:"clicks"`
}

func (r *ClickEventRepo) GetSourceBreakdown(ctx context.Context, linkID string, from, to time.Time) ([]BreakdownItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(sl.source_name, NULLIF(ce.source_name, ''), 'direct') as label, COUNT(*) as clicks
		 FROM click_events ce
		 LEFT JOIN source_links sl ON sl.id = ce.source_link_id
		 WHERE ce.link_id = $1 AND ce.clicked_at >= $2 AND ce.clicked_at <= $3
		 GROUP BY label ORDER BY clicks DESC`,
		linkID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get source breakdown: %w", err)
	}
	defer rows.Close()

	return scanBreakdown(rows)
}

func (r *ClickEventRepo) GetReferrerBreakdown(ctx context.Context, linkID string, from, to time.Time) ([]BreakdownItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(NULLIF(referer, ''), 'direct') as label, COUNT(*) as clicks
		 FROM click_events
		 WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at <= $3
		 GROUP BY label ORDER BY clicks DESC LIMIT 20`,
		linkID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get referrer breakdown: %w", err)
	}
	defer rows.Close()

	return scanBreakdown(rows)
}

func (r *ClickEventRepo) GetLocationBreakdown(ctx context.Context, linkID string, from, to time.Time) ([]BreakdownItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(country, 'Unknown') as label, COUNT(*) as clicks
		 FROM click_events
		 WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at <= $3
		 GROUP BY label ORDER BY clicks DESC LIMIT 20`,
		linkID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get location breakdown: %w", err)
	}
	defer rows.Close()

	return scanBreakdown(rows)
}

func (r *ClickEventRepo) GetBrowserBreakdown(ctx context.Context, linkID string, from, to time.Time) ([]BreakdownItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(NULLIF(browser, ''), 'Unknown') as label, COUNT(*) as clicks
		 FROM click_events
		 WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at <= $3
		 GROUP BY label ORDER BY clicks DESC LIMIT 20`,
		linkID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("get browser breakdown: %w", err)
	}
	defer rows.Close()

	return scanBreakdown(rows)
}

func scanBreakdown(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
}) ([]BreakdownItem, error) {
	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.Label, &item.Clicks); err != nil {
			return nil, fmt.Errorf("scan breakdown: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

type RecentClickItem struct {
	ClickedAt time.Time `json:"clicked_at"`
	Source    string    `json:"source"`
	Referer   string    `json:"referer"`
	Browser   string    `json:"browser"`
	Country   string    `json:"country"`
	IsBot     bool      `json:"is_bot"`
}

func (r *ClickEventRepo) GetRecentActivity(ctx context.Context, linkID string, limit int) ([]RecentClickItem, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT ce.clicked_at,
		        COALESCE(sl.source_name, NULLIF(ce.source_name, ''), 'direct') as source,
		        COALESCE(NULLIF(ce.referer, ''), 'direct') as referer,
		        COALESCE(NULLIF(ce.browser, ''), 'Unknown') as browser,
		        COALESCE(NULLIF(ce.country, ''), 'Unknown') as country,
		        ce.is_bot
		 FROM click_events ce
		 LEFT JOIN source_links sl ON sl.id = ce.source_link_id
		 WHERE ce.link_id = $1
		 ORDER BY ce.clicked_at DESC
		 LIMIT $2`,
		linkID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get recent activity: %w", err)
	}
	defer rows.Close()

	items := make([]RecentClickItem, 0)
	for rows.Next() {
		var item RecentClickItem
		if err := rows.Scan(&item.ClickedAt, &item.Source, &item.Referer, &item.Browser, &item.Country, &item.IsBot); err != nil {
			return nil, fmt.Errorf("scan recent activity: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}
