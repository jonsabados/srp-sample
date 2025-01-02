package bbom

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"
)

type CachingCreatureRepo struct {
	db            *sql.DB
	cacheDuration time.Duration

	cache      map[int64]CreatureLookupResult
	cacheMutex sync.RWMutex
}

func NewCachingCreatureRepo(db *sql.DB, cacheDuration time.Duration) *CachingCreatureRepo {
	return &CachingCreatureRepo{
		cache:         make(map[int64]CreatureLookupResult),
		db:            db,
		cacheDuration: cacheDuration,
	}
}

func (c *CachingCreatureRepo) CreateCreature(ctx context.Context, name, description string) (Creature, error) {
	stmt, err := c.db.PrepareContext(ctx, "insert into creatures (name, description) values ($1, $2) returning id")
	if err != nil {
		return Creature{}, err
	}
	defer stmt.Close()
	res := stmt.QueryRowContext(ctx, name, description)
	var id int64
	err = res.Scan(&id)
	if err != nil {
		return Creature{}, err
	}
	ret := Creature{
		ID:          id,
		Name:        name,
		Description: description,
	}
	c.cacheMutex.Lock()
	c.cache[id] = CreatureLookupResult{
		ResultFound: true,
		Creature:    ret,
		timestamp:   time.Now(),
	}
	c.cacheMutex.Unlock()
	return ret, nil
}

func (c *CachingCreatureRepo) GetCreature(ctx context.Context, id int64) (CreatureLookupResult, error) {
	c.cacheMutex.RLock()
	if result, cached := c.cache[id]; cached && time.Now().Sub(result.timestamp) < c.cacheDuration {
		c.cacheMutex.RUnlock()
		return result, nil
	}
	c.cacheMutex.RUnlock()
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	// let's make sure another concurrent request didn't already do the query, and if so lets return its result
	if result, cached := c.cache[id]; cached && time.Now().Sub(result.timestamp) < c.cacheDuration {
		return result, nil
	}
	stmt, err := c.db.PrepareContext(ctx, "select name, description from creatures where id=$1")
	if err != nil {
		return CreatureLookupResult{}, err
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, id)
	var name, description string
	err = row.Scan(&name, &description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ret := CreatureLookupResult{
				ResultFound: false,
				timestamp:   time.Now(),
			}
			c.cache[id] = ret
			return ret, nil
		} else {
			return CreatureLookupResult{}, err
		}
	}
	ret := CreatureLookupResult{
		ResultFound: true,
		Creature: Creature{
			ID:          id,
			Name:        name,
			Description: description,
		},
		timestamp: time.Now(),
	}
	c.cache[id] = ret
	return ret, nil
}
