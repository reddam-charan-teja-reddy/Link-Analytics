package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/charan/url-shortener/internal/repository/postgres"
	rediscache "github.com/charan/url-shortener/internal/repository/redis"
	"github.com/charan/url-shortener/pkg/hash"
	"github.com/jackc/pgx/v5"
)

type LinkService struct {
	linkRepo       *postgres.LinkRepo
	sourceLinkRepo *postgres.SourceLinkRepo
	groupRepo      *postgres.GroupRepo
	linkCache      *rediscache.LinkCache
	baseURL        string
}

func NewLinkService(linkRepo *postgres.LinkRepo, sourceLinkRepo *postgres.SourceLinkRepo, groupRepo *postgres.GroupRepo, linkCache *rediscache.LinkCache, baseURL string) *LinkService {
	return &LinkService{
		linkRepo:       linkRepo,
		sourceLinkRepo: sourceLinkRepo,
		groupRepo:      groupRepo,
		linkCache:      linkCache,
		baseURL:        baseURL,
	}
}

type LinkResponse struct {
	domain.Link
	ShortURL string `json:"short_url"`
}

type SourceLinkResponse struct {
	domain.SourceLink
	ShortURL string `json:"short_url"`
}

type BatchSourceResult struct {
	CreatedCount int                  `json:"created_count"`
	SkippedCount int                  `json:"skipped_count"`
	Items        []SourceLinkResponse `json:"items"`
}

func validateDestinationURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return errors.New("invalid url")
	}
	if parsed.Host == "" {
		return errors.New("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("only http and https urls are allowed")
	}

	return nil
}

func (s *LinkService) Create(ctx context.Context, userID, originalURL, title string) (*LinkResponse, error) {
	if err := validateDestinationURL(originalURL); err != nil {
		return nil, err
	}

	h, err := generateUniqueHash(ctx, s.linkRepo, 3)
	if err != nil {
		return nil, err
	}

	link, err := s.linkRepo.Create(ctx, userID, originalURL, h, title)
	if err != nil {
		return nil, err
	}

	return &LinkResponse{
		Link:     *link,
		ShortURL: s.baseURL + "/" + link.Hash,
	}, nil
}

func (s *LinkService) Get(ctx context.Context, userID, linkID string) (*LinkResponse, error) {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if link.UserID != userID {
		return nil, errors.New("not found")
	}

	sources, err := s.sourceLinkRepo.ListByLinkID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	link.Sources = sources

	return &LinkResponse{
		Link:     *link,
		ShortURL: s.baseURL + "/" + link.Hash,
	}, nil
}

func (s *LinkService) List(ctx context.Context, userID, groupID, sourceFilter string) ([]LinkResponse, error) {
	links, err := s.linkRepo.ListByUserID(ctx, userID, groupID, sourceFilter)
	if err != nil {
		return nil, err
	}

	responses := make([]LinkResponse, len(links))
	for i, link := range links {
		responses[i] = LinkResponse{
			Link:     link,
			ShortURL: s.baseURL + "/" + link.Hash,
		}
	}
	return responses, nil
}
func (s *LinkService) CreateGroup(ctx context.Context, userID, name string) (*domain.Group, error) {
	return s.groupRepo.Create(ctx, userID, strings.TrimSpace(name))
}

func (s *LinkService) ListGroups(ctx context.Context, userID string) ([]domain.Group, error) {
	return s.groupRepo.ListByUserID(ctx, userID)
}

func (s *LinkService) UpdateGroup(ctx context.Context, userID, groupID, name string) (*domain.Group, error) {
	return s.groupRepo.Update(ctx, userID, groupID, strings.TrimSpace(name))
}

func (s *LinkService) DeleteGroup(ctx context.Context, userID, groupID string) error {
	return s.groupRepo.Delete(ctx, userID, groupID)
}

func (s *LinkService) ListLinkGroups(ctx context.Context, userID, linkID string) ([]domain.Group, error) {
	return s.groupRepo.ListByLinkID(ctx, userID, linkID)
}

func (s *LinkService) AddLinkToGroup(ctx context.Context, userID, groupID, linkID string) error {
	return s.groupRepo.AddLink(ctx, userID, groupID, linkID)
}

func (s *LinkService) RemoveLinkFromGroup(ctx context.Context, userID, groupID, linkID string) error {
	return s.groupRepo.RemoveLink(ctx, userID, groupID, linkID)
}

func (s *LinkService) BatchCreateSources(ctx context.Context, userID, sourceName, scopeType, scopeID string) (*BatchSourceResult, error) {
	var (
		linkIDs []string
		err     error
	)

	scopeType = strings.TrimSpace(scopeType)
	if scopeType == "all" {
		linkIDs, err = s.linkRepo.ListLinkIDsByUserID(ctx, userID)
	} else if scopeType == "group" {
		if scopeID == "" {
			return nil, errors.New("scope_id is required for group scope")
		}
		linkIDs, err = s.groupRepo.ListLinkIDsByGroup(ctx, userID, scopeID)
	} else {
		return nil, errors.New("invalid scope_type")
	}
	if err != nil {
		return nil, err
	}

	result := &BatchSourceResult{Items: make([]SourceLinkResponse, 0)}
	for _, linkID := range linkIDs {
		item, createErr := s.CreateSource(ctx, userID, linkID, sourceName)
		if createErr != nil {
			if postgres.IsUniqueViolation(createErr) {
				result.SkippedCount++
				continue
			}
			return nil, createErr
		}
		result.CreatedCount++
		result.Items = append(result.Items, *item)
	}

	return result, nil
}

func (s *LinkService) Update(ctx context.Context, userID, linkID, title string, isActive bool) (*LinkResponse, error) {
	existing, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if existing.UserID != userID {
		return nil, errors.New("not found")
	}

	link, err := s.linkRepo.Update(ctx, linkID, title, isActive)
	if err != nil {
		return nil, err
	}

	// bust cache if link gets disabled
	if !isActive {
		_ = s.linkCache.Invalidate(ctx, link.Hash)
	}

	return &LinkResponse{
		Link:     *link,
		ShortURL: s.baseURL + "/" + link.Hash,
	}, nil
}

func (s *LinkService) Delete(ctx context.Context, userID, linkID string) error {
	existing, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return errors.New("not found")
	}

	// clear cache for this link + all its sources
	_ = s.linkCache.Invalidate(ctx, existing.Hash)
	sources, _ := s.sourceLinkRepo.ListByLinkID(ctx, linkID)
	for _, src := range sources {
		_ = s.linkCache.Invalidate(ctx, src.Hash)
	}

	return s.linkRepo.Delete(ctx, linkID)
}

func (s *LinkService) CreateSource(ctx context.Context, userID, linkID, sourceName string) (*SourceLinkResponse, error) {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if link.UserID != userID {
		return nil, errors.New("not found")
	}

	h, err := generateUniqueSourceHash(ctx, s.linkRepo, s.sourceLinkRepo, 5)
	if err != nil {
		return nil, err
	}

	sl, err := s.sourceLinkRepo.Create(ctx, linkID, sourceName, h)
	if err != nil {
		return nil, err
	}

	return &SourceLinkResponse{
		SourceLink: *sl,
		ShortURL:   s.baseURL + "/" + sl.Hash,
	}, nil
}

func (s *LinkService) ListSources(ctx context.Context, userID, linkID string) ([]SourceLinkResponse, error) {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if link.UserID != userID {
		return nil, errors.New("not found")
	}

	sources, err := s.sourceLinkRepo.ListByLinkID(ctx, linkID)
	if err != nil {
		return nil, err
	}

	responses := make([]SourceLinkResponse, len(sources))
	for i, src := range sources {
		responses[i] = SourceLinkResponse{
			SourceLink: src,
			ShortURL:   s.baseURL + "/" + src.Hash,
		}
	}
	return responses, nil
}

func (s *LinkService) DeleteSource(ctx context.Context, userID, linkID, sourceID string) error {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}
	if link.UserID != userID {
		return errors.New("not found")
	}

	sources, err := s.sourceLinkRepo.ListByLinkID(ctx, linkID)
	if err != nil {
		return err
	}
	for _, src := range sources {
		if src.ID == sourceID {
			_ = s.linkCache.Invalidate(ctx, src.Hash)
			return s.sourceLinkRepo.Delete(ctx, sourceID)
		}
	}
	return errors.New("source not found")
}

func generateUniqueHash(ctx context.Context, linkRepo *postgres.LinkRepo, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		h, err := hash.Generate()
		if err != nil {
			return "", fmt.Errorf("generate hash: %w", err)
		}

		_, err = linkRepo.GetByHash(ctx, h)
		if errors.Is(err, pgx.ErrNoRows) {
			return h, nil
		}
		if err != nil {
			return "", fmt.Errorf("check hash availability: %w", err)
		}
	}
	return "", fmt.Errorf("failed to generate unique hash after %d retries", maxRetries)
}

func generateUniqueSourceHash(ctx context.Context, linkRepo *postgres.LinkRepo, sourceLinkRepo *postgres.SourceLinkRepo, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		h, err := hash.Generate()
		if err != nil {
			return "", fmt.Errorf("generate hash: %w", err)
		}

		_, err = linkRepo.GetByHash(ctx, h)
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("check link hash availability: %w", err)
		}

		_, err = sourceLinkRepo.GetByHash(ctx, h)
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("check source hash availability: %w", err)
		}

		return h, nil
	}

	return "", fmt.Errorf("failed to generate unique source hash after %d retries", maxRetries)
}
