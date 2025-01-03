package srp

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jonsabados/srp-sample/db"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatureRepo_CreateCreature(t *testing.T) {
	ctx := context.Background()
	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	connectionCfg, err := db.ConnectionParamsFromEnv()
	require.NoError(t, err)
	connectionOpener := db.NewConnectionOpener(connectionCfg)

	testInstance := NewCreatureRepo(connectionOpener)
	creature, err := testInstance.CreateCreature(ctx, name, description)
	require.NoError(t, err)
	assert.Equal(t, name, creature.Name)
	assert.Equal(t, description, creature.Description)

	conn, err := connectionOpener.OpenConnection()
	require.NoError(t, err)
	defer conn.Close()

	//let's ensure data was written to postgres as expected
	stmt, err := conn.PrepareContext(ctx, "select name, description from creatures where id=$1")
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
	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	connectionCfg, err := db.ConnectionParamsFromEnv()
	require.NoError(t, err)
	connectionOpener := db.NewConnectionOpener(connectionCfg)

	testInstance := NewCreatureRepo(connectionOpener)

	conn, err := connectionOpener.OpenConnection()
	require.NoError(t, err)
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, "insert into creatures (name, description) values ($1, $2)")
	require.NoError(t, err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, name, description)
	require.NoError(t, err)
	stmt, err = conn.PrepareContext(ctx, "select id from creatures where name=$1")
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

	connectionCfg, err := db.ConnectionParamsFromEnv()
	require.NoError(t, err)
	connectionOpener := db.NewConnectionOpener(connectionCfg)

	testInstance := NewCreatureRepo(connectionOpener)

	// let's hope no-one goes and does something silly like inserting a record with a negative id...
	result, err := testInstance.GetCreature(ctx, -1)
	require.NoError(t, err)
	assert.False(t, result.ResultFound)
}
