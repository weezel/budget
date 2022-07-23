package dbengine

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golobby/dotenv"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresConfig provides postgresql configurations
// read from the file or from environmental variables.
type PostgresConfig struct {
	Username string `env:"DB_USERNAME"`
	Password string `env:"DB_PASSWORD"`
	Hostname string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	DBName   string `env:"DB_NAME"`
}

func (conf PostgresConfig) Validate() {
	if conf.Username == "" {
		log.Fatal("Missing DB_USERNAME")
	}
	if conf.Password == "" {
		log.Fatal("Missing DB_PASSWORD")
	}
	if conf.Hostname == "" {
		log.Fatal("Missing DB_HOST")
	}
	if conf.Port == "" {
		log.Fatal("Missing DB_PORT")
	}
	if conf.DBName == "" {
		log.Fatal("Missing DB_NAME")
	}
}

func InitPostgresConnection(ctx context.Context, filePath string) (*pgxpool.Pool, error) {
	dbConf := PostgresConfig{}
	fhandle, err := os.Open(filepath.Join(filePath, ".env"))
	if err != nil {
		return nil, err
	}
	err = dotenv.NewDecoder(fhandle).Decode(&dbConf)
	if err != nil {
		return nil, err
	}
	dbConf.Validate()

	pgConfigURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbConf.Username, dbConf.Password, dbConf.Hostname, dbConf.Port, dbConf.DBName)
	psqlDB, err := pgxpool.Connect(ctx, pgConfigURL)
	if err != nil {
		return nil, err
	}
	return psqlDB, psqlDB.Ping(ctx)
}
