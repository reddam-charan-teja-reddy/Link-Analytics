package service

import (
	"context"
	"errors"
	"time"

	"github.com/charan/url-shortener/internal/repository/postgres"
)

type AnalyticsService struct {
	clickEventRepo *postgres.ClickEventRepo
	linkRepo       *postgres.LinkRepo
}

var ErrLinkNotFound = errors.New("link not found")

func NewAnalyticsService(clickEventRepo *postgres.ClickEventRepo, linkRepo *postgres.LinkRepo) *AnalyticsService {
	return &AnalyticsService{clickEventRepo: clickEventRepo, linkRepo: linkRepo}
}

func (s *AnalyticsService) ensureOwnership(ctx context.Context, userID, linkID string) error {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil || link.UserID != userID {
		return ErrLinkNotFound
	}
	return nil
}

func (s *AnalyticsService) GetSummary(ctx context.Context, userID, linkID string, from, to time.Time) (*postgres.AnalyticsSummary, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetSummary(ctx, linkID, from, to)
}

func (s *AnalyticsService) GetClickTimeSeries(ctx context.Context, userID, linkID string, from, to time.Time, granularity string) ([]postgres.TimeSeriesPoint, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetClickTimeSeries(ctx, linkID, from, to, granularity)
}

func (s *AnalyticsService) GetSourceBreakdown(ctx context.Context, userID, linkID string, from, to time.Time) ([]postgres.BreakdownItem, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetSourceBreakdown(ctx, linkID, from, to)
}

func (s *AnalyticsService) GetReferrerBreakdown(ctx context.Context, userID, linkID string, from, to time.Time) ([]postgres.BreakdownItem, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetReferrerBreakdown(ctx, linkID, from, to)
}

func (s *AnalyticsService) GetLocationBreakdown(ctx context.Context, userID, linkID string, from, to time.Time) ([]postgres.BreakdownItem, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetLocationBreakdown(ctx, linkID, from, to)
}

func (s *AnalyticsService) GetBrowserBreakdown(ctx context.Context, userID, linkID string, from, to time.Time) ([]postgres.BreakdownItem, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetBrowserBreakdown(ctx, linkID, from, to)
}

func (s *AnalyticsService) GetRecentActivity(ctx context.Context, userID, linkID string, limit int) ([]postgres.RecentClickItem, error) {
	if err := s.ensureOwnership(ctx, userID, linkID); err != nil {
		return nil, err
	}
	return s.clickEventRepo.GetRecentActivity(ctx, linkID, limit)
}
