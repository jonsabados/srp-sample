package srp

import (
	"context"
	"sync"
	"time"
)

type cachedLookupResult struct {
	result    CreatureLookupResult
	timestamp time.Time
}

type RawCreatureRepo interface {
	CreateCreature(ctx context.Context, name, description string) (Creature, error)
	GetCreature(ctx context.Context, id int64) (CreatureLookupResult, error)
}

type CachingCreatureRepo struct {
	rawRepo       RawCreatureRepo
	cacheDuration time.Duration

	cache      map[int64]cachedLookupResult
	cacheMutex sync.RWMutex
}

func NewCachingCreatureRepo(rawRepo RawCreatureRepo, cacheDuration time.Duration) *CachingCreatureRepo {
	return &CachingCreatureRepo{
		cache:         make(map[int64]cachedLookupResult),
		rawRepo:       rawRepo,
		cacheDuration: cacheDuration,
	}
}

func (c *CachingCreatureRepo) CreateCreature(ctx context.Context, name, description string) (Creature, error) {
	res, err := c.rawRepo.CreateCreature(ctx, name, description)
	if err != nil {
		return res, err
	}
	c.cacheMutex.Lock()
	c.cache[res.ID] = cachedLookupResult{
		result: CreatureLookupResult{
			ResultFound: true,
			Creature:    res,
		},
		timestamp: time.Now(),
	}
	c.cacheMutex.Unlock()
	return res, err
}

func (c *CachingCreatureRepo) GetCreature(ctx context.Context, id int64) (CreatureLookupResult, error) {
	c.cacheMutex.RLock()
	if result, cached := c.cache[id]; cached && time.Now().Sub(result.timestamp) < c.cacheDuration {
		c.cacheMutex.RUnlock()
		return result.result, nil
	}
	c.cacheMutex.RUnlock()
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	// let's make sure another concurrent request didn't already do the query, and if so lets return its result
	if result, cached := c.cache[id]; cached && time.Now().Sub(result.timestamp) < c.cacheDuration {
		return result.result, nil
	}
	result, err := c.rawRepo.GetCreature(ctx, id)
	if err != nil {
		return result, err
	}
	c.cache[id] = cachedLookupResult{
		result:    result,
		timestamp: time.Now(),
	}
	return result, err
}
