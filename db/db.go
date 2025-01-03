package db

import (
	"database/sql"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
)

type ConnectionParams struct {
	Host     string
	User     string
	Password string
	Port     int
	DB       string
}

func ConnectionParamsFromEnv() (ConnectionParams, error) {
	dbCfg := struct {
		Host     string `envconfig:"POSTGRES_HOST" default:"127.0.0.1"`
		User     string `envconfig:"POSTGRES_USER" default:"postgres"`
		Password string `envconfig:"POSTGRES_PW" default:"postgres"`
		Port     int    `envconfig:"POSTGRES_PORT" default:"5433"`
		DB       string `envconfig:"POSTGRES_DB" default:"postgres"`
	}{}
	err := envconfig.Process("", &dbCfg)
	if err != nil {
		return ConnectionParams{}, err
	}
	return ConnectionParams{
		Host:     dbCfg.Host,
		User:     dbCfg.User,
		Password: dbCfg.Password,
		Port:     dbCfg.Port,
		DB:       dbCfg.DB,
	}, nil
}

type ConnectionOpener struct {
	connectionParams ConnectionParams
}

func NewConnectionOpener(connectionParams ConnectionParams) *ConnectionOpener {
	return &ConnectionOpener{connectionParams: connectionParams}
}

func (c *ConnectionOpener) OpenConnection() (*sql.DB, error) {
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", c.connectionParams.Host, c.connectionParams.Port, c.connectionParams.User, c.connectionParams.Password, c.connectionParams.User)
	return sql.Open("postgres", psqlconn)
}
