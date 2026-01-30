package sqlc

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	Close()
	Ping(ctx context.Context) error
	ExecTx(ctx context.Context, fn func(*Queries) error) error
}

// SQLStore provides all functions to execute SQL queries and transactions
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

// NewStore creates a new store
func NewStore() (Store, error) {
	config, err := LoadRDBConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if config.DBName == "" || config.Host == "" {
		return nil, fmt.Errorf("invalid config for host:%v", config.Host)
	}
	dsn := fmt.Sprintf(dataSourceURIFmt, config.Username, config.Password, config.Host, config.DBName, config.AppName)

	connPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to db %w", err)
	}

	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}, nil
}

func (s SQLStore) Close() {
	s.connPool.Close()
}
func (s SQLStore) Ping(ctx context.Context) error {
	return s.connPool.Ping(ctx)
}

// Environment variable names for database configuration
const (
	RDB_MIGRATION_URL = "RDB_MIGRATION_URL"
	RDB_HOST          = "RDB_HOST"
	RDB_PORT          = "RDB_PORT"
	RDB_USER          = "RDB_USER"
	RDB_PASSWD        = "RDB_PASSWD"
	RDB_DB_NAME       = "RDB_DB_NAME"
	APP_NAME          = "APP_NAME"
	dataSourceURIFmt  = "postgresql://%s:%s@%s/%s?sslmode=disable&application_name=$%s"
)

type RDBConfigOptions struct {
	Host         string // e.g: hostname  // google.com
	Port         int    // e.g: port number // 5432
	Username     string
	Password     string
	DBName       string
	AppName      string
	MigrationURL string
}

func GetIntOsEnv(envName string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(envName)
	if !exists {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// RDBConfigOptions represents the configuration options for the relational database
func LoadRDBConfigFromEnv() (RDBConfigOptions, error) {
	host := os.Getenv(RDB_HOST)
	if host == "" {
		return RDBConfigOptions{}, fmt.Errorf("RDB_HOST environment variable is required")
	}
	port := GetIntOsEnv(RDB_PORT, 5432)
	if port < 1 || port > 65535 {
		return RDBConfigOptions{}, fmt.Errorf("port must be between 1 and 65535")
	}

	user := os.Getenv(RDB_USER)
	if user == "" {
		return RDBConfigOptions{}, fmt.Errorf("RDB_USER environment variable is required")
	}

	passwd := os.Getenv(RDB_PASSWD)
	if passwd == "" {
		return RDBConfigOptions{}, fmt.Errorf("RDB_PASSWD environment variable is required")
	}

	dbName := os.Getenv(RDB_DB_NAME)
	if dbName == "" {
		return RDBConfigOptions{}, fmt.Errorf("RDB_DB_NAME environment variable is required")
	}

	appName := os.Getenv(APP_NAME)
	if appName == "" {
		return RDBConfigOptions{}, fmt.Errorf("APP_NAME environment variable is required")
	}

	migrationURL := os.Getenv(RDB_MIGRATION_URL)
	if migrationURL == "" {
		return RDBConfigOptions{}, fmt.Errorf("RDB_MIGRATION_URL environment variable is required")
	}

	return newRDBConfigOptions(
		host,
		port,
		user,
		passwd,
		dbName,
		appName,
		migrationURL,
	), nil
}
func newRDBConfigOptions(host string, port int, username, password, dbname, appname, migrationUrl string) RDBConfigOptions {
	return RDBConfigOptions{
		Host:         host,
		Port:         port,
		DBName:       dbname,
		Username:     username,
		Password:     password,
		AppName:      appname,
		MigrationURL: migrationUrl,
	}
}
