package srp

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatureRepo_CreateCreature(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()

	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	testInstance := NewCreatureRepo(db)
	creature, err := testInstance.CreateCreature(ctx, name, description)
	require.NoError(t, err)
	assert.Equal(t, name, creature.Name)
	assert.Equal(t, description, creature.Description)

	//let's ensure data was written to postgres as expected
	stmt, err := db.PrepareContext(ctx, "select name, description from creatures where id=$1")
	require.NoError(t, err)
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, creature.ID)
	var pName, pDescription string
	err = row.Scan(&pName, &pDescription)
	require.NoError(t, err)
	assert.Equal(t, name, pName)
	assert.Equal(t, description, pDescription)
}

func TestCreatureRepo_GetCreature_ResultFound(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()

	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	testInstance := NewCreatureRepo(db)

	stmt, err := db.PrepareContext(ctx, "insert into creatures (name, description) values ($1, $2)")
	require.NoError(t, err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, name, description)
	require.NoError(t, err)
	stmt, err = db.PrepareContext(ctx, "select id from creatures where name=$1")
	require.NoError(t, err)
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, name)
	var id int64
	err = row.Scan(&id)
	require.NoError(t, err)

	result, err := testInstance.GetCreature(ctx, id)
	require.NoError(t, err)
	assert.True(t, result.ResultFound)
	assert.Equal(t, Creature{
		ID:          id,
		Name:        name,
		Description: description,
	}, result.Creature)
}

func TestCreatureRepo_GetCreature_NoResultFound(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()

	testInstance := NewCreatureRepo(db)

	// let's hope no-one goes and does something silly like inserting a record with a negative id...
	result, err := testInstance.GetCreature(ctx, -1)
	require.NoError(t, err)
	assert.False(t, result.ResultFound)
}

func openTestDBConnection(t *testing.T) *sql.DB {
	dbCfg := struct {
		Host     string `envconfig:"POSTGRES_HOST" default:"127.0.0.1"`
		User     string `envconfig:"POSTGRES_USER" default:"postgres"`
		Password string `envconfig:"POSTGRES_PW" default:"postgres"`
		Port     int    `envconfig:"POSTGRES_PORT" default:"5433"`
		DB       string `envconfig:"POSTGRES_DB" default:"postgres"`
	}{}
	err := envconfig.Process("", &dbCfg)
	require.NoError(t, err)

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.User)

	// open database
	db, err := sql.Open("postgres", psqlconn)
	require.NoError(t, err)
	return db
}
