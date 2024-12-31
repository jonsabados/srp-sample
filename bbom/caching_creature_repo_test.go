package bbom

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachingCreatureRepo_CreateCreature(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()
	testCacheDuration := time.Second

	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	testInstance := NewCachingCreatureRepo(db, testCacheDuration)
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

	//let's ensure the creature that was created was cached.... were going to have to get a bit creative
	stmt, err = db.PrepareContext(ctx, "delete from creatures where id=$1")
	require.NoError(t, err)
	res, err := stmt.ExecContext(ctx, creature.ID)
	require.NoError(t, err)
	impacted, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), impacted)

	// guess were sorta gonna have to test GetCreature too...
	result, err := testInstance.GetCreature(ctx, creature.ID)
	require.NoError(t, err)
	assert.True(t, result.ResultFound)
	assert.Equal(t, creature, result.Creature)

	time.Sleep(testCacheDuration)
	result, err = testInstance.GetCreature(ctx, creature.ID)
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

func TestCachingCreatureRepo_GetCreature(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()
	testCacheDuration := time.Second

	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	testInstance := NewCachingCreatureRepo(db, testCacheDuration)

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

	// now let's verify caching, gotta get creative...
	stmt, err = db.PrepareContext(ctx, "delete from creatures where id=$1")
	require.NoError(t, err)
	_, err = stmt.ExecContext(ctx, id)
	require.NoError(t, err)

	// repeat our lookup
	result, err = testInstance.GetCreature(ctx, id)
	require.NoError(t, err)
	assert.True(t, result.ResultFound)
	assert.Equal(t, Creature{
		ID:          id,
		Name:        name,
		Description: description,
	}, result.Creature)

	// and finally let the cache expire and repeat the lookup
	time.Sleep(testCacheDuration)
	result, err = testInstance.GetCreature(ctx, id)
	require.NoError(t, err)
	assert.False(t, result.ResultFound)
}

func TestCachingCreatureRepo_GetCreature_Concurrency(t *testing.T) {
	ctx := context.Background()
	db := openTestDBConnection(t)
	defer db.Close()
	testCacheDuration := time.Hour

	name := fmt.Sprintf("creature_test_%s", uuid.NewString())
	description := "a creature for testing purposes"

	testInstance := NewCachingCreatureRepo(db, testCacheDuration)

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

	selectCount := 1000
	barrier := sync.WaitGroup{}
	barrier.Add(1)
	wg := sync.WaitGroup{}
	for i := 0; i < selectCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			barrier.Wait()
			result, err := testInstance.GetCreature(ctx, id)
			require.NoError(t, err)
			assert.True(t, result.ResultFound)
			assert.Equal(t, Creature{
				ID:          id,
				Name:        name,
				Description: description,
			}, result.Creature)
		}()
	}
	barrier.Done()
	wg.Wait()
	// note - it would be good to ensure that only one call to the DB was made, but we can't :sad-panda:
}
