package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
)

// catalogCacheTTL is long because the catalog only changes via migrations —
// stale-by-an-hour is acceptable and saves the 23-row scan on every page load.
const catalogCacheTTL = time.Hour

const catalogCacheKey = "catalog:cars"

// CatalogService serves the read-only EV catalog (cache-aside, 1 h TTL).
// Reuses the CacheStore interface the analytics dashboard defined.
type CatalogService struct {
	catalog repository.CatalogRepository
	cache   CacheStore
}

func NewCatalogService(catalog repository.CatalogRepository, cache CacheStore) *CatalogService {
	return &CatalogService{catalog: catalog, cache: cache}
}

// List returns every catalog car. Cache errors are non-fatal — the rows are
// recomputed from the source of truth.
func (s *CatalogService) List(ctx context.Context) ([]domain.CatalogCar, error) {
	if val, ok, err := s.cache.CacheGet(ctx, catalogCacheKey); err != nil {
		log.Printf("catalog: cache get: %v", err)
	} else if ok {
		var cached []domain.CatalogCar
		if jErr := json.Unmarshal([]byte(val), &cached); jErr == nil {
			return cached, nil
		}
		log.Printf("catalog: cache decode: corrupt, recomputing")
	}

	items, err := s.catalog.ListCatalog(ctx)
	if err != nil {
		return nil, err
	}

	if blob, err := json.Marshal(items); err == nil {
		if err := s.cache.CacheSet(ctx, catalogCacheKey, string(blob), catalogCacheTTL); err != nil {
			log.Printf("catalog: cache set: %v", err)
		}
	}
	return items, nil
}
