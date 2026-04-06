package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns health status information.
	Health() HealthStats

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// Dashboard methods
	GetDashboardSummary(ctx context.Context, period PeriodType) (DashboardSummary, error)

	// Record methods
	GetRecord(ctx context.Context, recordID int64) (Record, error)
	GetRecords(ctx context.Context, filters *RecordFilters) ([]Record, error)
	CreateRecord(ctx context.Context, userID string, record Record) (int64, error)
	UpdateRecord(ctx context.Context, id int64, updates Record) error
	DeleteRecord(ctx context.Context, id int64) error

	// User methods
	CreateUser(ctx context.Context, user User) (string, error)
	ToggleUserStatus(ctx context.Context, userID string) error
	AssignUserRole(ctx context.Context, userID string, role RoleType) error
	GetUserByEmail(ctx context.Context, email string) (AuthUser, error)
	GetUserById(ctx context.Context, userID string) (AuthUser, error)
}

type service struct {
	db *sql.DB
}

type HealthStats struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	OpenConnections   string `json:"open_connections,omitempty"`
	InUse             string `json:"in_use,omitempty"`
	Idle              string `json:"idle,omitempty"`
	WaitCount         string `json:"wait_count,omitempty"`
	WaitDuration      string `json:"wait_duration,omitempty"`
	MaxIdleClosed     string `json:"max_idle_closed,omitempty"`
	MaxLifetimeClosed string `json:"max_lifetime_closed,omitempty"`
	Error             string `json:"error,omitempty"`
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
	dbUrl      = os.Getenv("DATABASE_URL")
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	var connStr string

	if dbUrl != "" {
		connStr = dbUrl
	} else {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
func (s *service) Health() HealthStats {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := HealthStats{}

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats.Status = "down"
		stats.Error = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats.Status = "up"
	stats.Message = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats.OpenConnections = strconv.Itoa(dbStats.OpenConnections)
	stats.InUse = strconv.Itoa(dbStats.InUse)
	stats.Idle = strconv.Itoa(dbStats.Idle)
	stats.WaitCount = strconv.FormatInt(dbStats.WaitCount, 10)
	stats.WaitDuration = dbStats.WaitDuration.String()
	stats.MaxIdleClosed = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats.MaxLifetimeClosed = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats.Message = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats.Message = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats.Message = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats.Message = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}
