package srp

import (
	"context"
	"database/sql"
	"errors"
)

type ConnectionOpener interface {
	OpenConnection() (*sql.DB, error)
}

type CreatureRepo struct {
	connectionOpener ConnectionOpener
}

func NewCreatureRepo(connectionOpener ConnectionOpener) *CreatureRepo {
	return &CreatureRepo{
		connectionOpener: connectionOpener,
	}
}

func (c *CreatureRepo) CreateCreature(ctx context.Context, name, description string) (Creature, error) {
	db, err := c.connectionOpener.OpenConnection()
	if err != nil {
		return Creature{}, err
	}
	defer db.Close()

	stmt, err := db.PrepareContext(ctx, "insert into creatures (name, description) values ($1, $2) returning id")
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
	return Creature{
		ID:          id,
		Name:        name,
		Description: description,
	}, nil
}

func (c *CreatureRepo) GetCreature(ctx context.Context, id int64) (CreatureLookupResult, error) {
	db, err := c.connectionOpener.OpenConnection()
	if err != nil {
		return CreatureLookupResult{}, err
	}
	defer db.Close()

	stmt, err := db.PrepareContext(ctx, "select name, description from creatures where id=$1")
	if err != nil {
		return CreatureLookupResult{}, err
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, id)
	var name, description string
	err = row.Scan(&name, &description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CreatureLookupResult{
				ResultFound: false,
			}, nil
		} else {
			return CreatureLookupResult{}, err
		}
	}
	return CreatureLookupResult{
		ResultFound: true,
		Creature: Creature{
			ID:          id,
			Name:        name,
			Description: description,
		},
	}, nil
}
