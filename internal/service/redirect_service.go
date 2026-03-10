package service

import (
	"context"

	"github.com/charan/url-shortener/internal/repository/postgres"
	rediscache "github.com/charan/url-shortener/internal/repository/redis"
)

type RedirectResult struct {
	OriginalURL  string
	LinkID       string
	SourceLinkID *string
}

type RedirectService struct {
	linkRepo       *postgres.LinkRepo
	sourceLinkRepo *postgres.SourceLinkRepo
	linkCache      *rediscache.LinkCache
}

func NewRedirectService(linkRepo *postgres.LinkRepo, sourceLinkRepo *postgres.SourceLinkRepo, linkCache *rediscache.LinkCache) *RedirectService {
	return &RedirectService{
		linkRepo:       linkRepo,
		sourceLinkRepo: sourceLinkRepo,
		linkCache:      linkCache,
	}
}

func (s *RedirectService) Resolve(ctx context.Context, hash string) (*RedirectResult, error) {
	cached, err := s.linkCache.Get(ctx, hash)
	if err == nil && cached != nil {
		return &RedirectResult{
			OriginalURL:  cached.OriginalURL,
			LinkID:       cached.LinkID,
			SourceLinkID: cached.SourceLinkID,
		}, nil
	}

	link, err := s.linkRepo.GetByHash(ctx, hash)
	if err == nil {
		result := &RedirectResult{
			OriginalURL: link.OriginalURL,
			LinkID:      link.ID,
		}
		_ = s.linkCache.Set(ctx, hash, &rediscache.CachedLink{
			OriginalURL: link.OriginalURL,
			LinkID:      link.ID,
		})
		return result, nil
	}

	sl, err := s.sourceLinkRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	link, err = s.linkRepo.GetByID(ctx, sl.LinkID)
	if err != nil {
		return nil, err
	}

	result := &RedirectResult{
		OriginalURL:  link.OriginalURL,
		LinkID:       link.ID,
		SourceLinkID: &sl.ID,
	}

	_ = s.linkCache.Set(ctx, hash, &rediscache.CachedLink{
		OriginalURL:  link.OriginalURL,
		LinkID:       link.ID,
		SourceLinkID: &sl.ID,
	})

	return result, nil
}
