package isolation

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
)

type DatabaseIsolationManager struct {
	databases    map[string]*IsolatedDatabase
	mutex        sync.RWMutex
	basePath     string
	dbType       string // "sqlite" or "postgres"
	postgresURI  string // PostgreSQL connection string
}

type IsolatedDatabase struct {
	InstanceID     string `json:"instance_id"`
	DatabasePath   string `json:"database_path"`
	KeysPath       string `json:"keys_path"`
	DatabaseName   string `json:"database_name"`   // For PostgreSQL
	KeysDBName     string `json:"keys_db_name"`    // For PostgreSQL
	Connection     *sql.DB `json:"-"`
	KeysConn       *sql.DB `json:"-"`
	DBType         string `json:"db_type"`         // "sqlite" or "postgres"
	ConnectionURI  string `json:"connection_uri"`  // Full connection string
	KeysURI        string `json:"keys_uri"`        // Keys connection string
	mutex          sync.RWMutex `json:"-"`
}

func NewDatabaseIsolationManager(basePath string) *DatabaseIsolationManager {
	return &DatabaseIsolationManager{
		databases: make(map[string]*IsolatedDatabase),
		basePath:  basePath,
		dbType:    "sqlite", // Default to SQLite
	}
}

// NewPostgresDatabaseIsolationManager creates a new database isolation manager with PostgreSQL support
func NewPostgresDatabaseIsolationManager(basePath, postgresURI string) *DatabaseIsolationManager {
	return &DatabaseIsolationManager{
		databases:   make(map[string]*IsolatedDatabase),
		basePath:    basePath,
		dbType:      "postgres",
		postgresURI: postgresURI,
	}
}

// CreateIsolatedDatabase creates isolated database for an instance (SQLite or PostgreSQL)
func (dim *DatabaseIsolationManager) CreateIsolatedDatabase(instanceID string) (*IsolatedDatabase, error) {
	dim.mutex.Lock()
	defer dim.mutex.Unlock()

	if _, exists := dim.databases[instanceID]; exists {
		return nil, fmt.Errorf("database for instance %s already exists", instanceID)
	}

	var isolatedDB *IsolatedDatabase
	var err error

	switch dim.dbType {
	case "postgres":
		isolatedDB, err = dim.createPostgresDatabase(instanceID)
	default: // sqlite
		isolatedDB, err = dim.createSQLiteDatabase(instanceID)
	}

	if err != nil {
		return nil, err
	}

	// Initialize databases
	if err := dim.initializeDatabase(isolatedDB); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	dim.databases[instanceID] = isolatedDB
	logrus.Infof("[DB_ISOLATION] Created isolated %s database for instance: %s", dim.dbType, instanceID)
	return isolatedDB, nil
}

// createSQLiteDatabase creates SQLite databases for an instance
func (dim *DatabaseIsolationManager) createSQLiteDatabase(instanceID string) (*IsolatedDatabase, error) {
	// Create instance database directory
	dbDir := filepath.Join(dim.basePath, "instances", instanceID, "storages")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Database paths
	dbPath := filepath.Join(dbDir, fmt.Sprintf("whatsapp_%s.db", instanceID))
	keysPath := filepath.Join(dbDir, fmt.Sprintf("keys_%s.db", instanceID))

	return &IsolatedDatabase{
		InstanceID:    instanceID,
		DatabasePath:  dbPath,
		KeysPath:      keysPath,
		DBType:        "sqlite",
		ConnectionURI: fmt.Sprintf("file:%s?_foreign_keys=on", dbPath),
		KeysURI:       fmt.Sprintf("file:%s?_foreign_keys=on", keysPath),
	}, nil
}

// createPostgresDatabase creates PostgreSQL databases for an instance
func (dim *DatabaseIsolationManager) createPostgresDatabase(instanceID string) (*IsolatedDatabase, error) {
	// Generate unique database names for this instance
	dbName := fmt.Sprintf("whatsapp_%s", strings.ReplaceAll(instanceID, "-", "_"))
	keysDBName := fmt.Sprintf("keys_%s", strings.ReplaceAll(instanceID, "-", "_"))

	// Create databases in PostgreSQL
	if err := dim.createPostgresDatabaseSchema(dbName); err != nil {
		return nil, fmt.Errorf("failed to create main database: %w", err)
	}

	if err := dim.createPostgresDatabaseSchema(keysDBName); err != nil {
		return nil, fmt.Errorf("failed to create keys database: %w", err)
	}

	// Build connection URIs for the specific databases
	mainURI := dim.buildPostgresURI(dbName)
	keysURI := dim.buildPostgresURI(keysDBName)

	return &IsolatedDatabase{
		InstanceID:    instanceID,
		DatabaseName:  dbName,
		KeysDBName:    keysDBName,
		DBType:        "postgres",
		ConnectionURI: mainURI,
		KeysURI:       keysURI,
	}, nil
}

// createPostgresDatabaseSchema creates a new database in PostgreSQL
func (dim *DatabaseIsolationManager) createPostgresDatabaseSchema(dbName string) error {
	// Connect to PostgreSQL server (without specific database)
	db, err := sql.Open("postgres", dim.postgresURI)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	// Check if database already exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		// Create the database
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
		logrus.Infof("[DB_ISOLATION] Created PostgreSQL database: %s", dbName)
	}

	return nil
}

// buildPostgresURI builds a PostgreSQL connection URI for a specific database
func (dim *DatabaseIsolationManager) buildPostgresURI(dbName string) string {
	// Parse the base URI and replace the database name
	baseURI := dim.postgresURI
	if strings.Contains(baseURI, "?") {
		// Has query parameters
		parts := strings.Split(baseURI, "?")
		if strings.Contains(parts[0], "/") {
			// Replace database name
			uriParts := strings.Split(parts[0], "/")
			uriParts[len(uriParts)-1] = dbName
			return strings.Join(uriParts, "/") + "?" + parts[1]
		}
		return parts[0] + "/" + dbName + "?" + parts[1]
	} else {
		// No query parameters
		if strings.Contains(baseURI, "/") {
			// Replace database name
			uriParts := strings.Split(baseURI, "/")
			uriParts[len(uriParts)-1] = dbName
			return strings.Join(uriParts, "/")
		}
		return baseURI + "/" + dbName
	}
}

// GetIsolatedDatabase retrieves the isolated database for an instance
func (dim *DatabaseIsolationManager) GetIsolatedDatabase(instanceID string) (*IsolatedDatabase, error) {
	dim.mutex.RLock()
	defer dim.mutex.RUnlock()

	db, exists := dim.databases[instanceID]
	if !exists {
		return nil, fmt.Errorf("database for instance %s not found", instanceID)
	}

	return db, nil
}

// DeleteIsolatedDatabase removes the isolated database for an instance
func (dim *DatabaseIsolationManager) DeleteIsolatedDatabase(instanceID string) error {
	dim.mutex.Lock()
	defer dim.mutex.Unlock()

	db, exists := dim.databases[instanceID]
	if !exists {
		return fmt.Errorf("database for instance %s not found", instanceID)
	}

	// Close connections
	if db.Connection != nil {
		db.Connection.Close()
	}
	if db.KeysConn != nil {
		db.KeysConn.Close()
	}

	switch db.DBType {
	case "postgres":
		// Drop PostgreSQL databases
		if err := dim.dropPostgresDatabase(db.DatabaseName); err != nil {
			logrus.Warnf("[DB_ISOLATION] Failed to drop main database: %v", err)
		}
		if err := dim.dropPostgresDatabase(db.KeysDBName); err != nil {
			logrus.Warnf("[DB_ISOLATION] Failed to drop keys database: %v", err)
		}

	default: // sqlite
		// Remove SQLite database files
		if err := os.Remove(db.DatabasePath); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("[DB_ISOLATION] Failed to remove database file: %v", err)
		}
		if err := os.Remove(db.KeysPath); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("[DB_ISOLATION] Failed to remove keys file: %v", err)
		}
	}

	delete(dim.databases, instanceID)
	logrus.Infof("[DB_ISOLATION] Deleted isolated %s database for instance: %s", db.DBType, instanceID)
	return nil
}

// dropPostgresDatabase drops a PostgreSQL database
func (dim *DatabaseIsolationManager) dropPostgresDatabase(dbName string) error {
	// Connect to PostgreSQL server (without specific database)
	db, err := sql.Open("postgres", dim.postgresURI)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	// Terminate all connections to the database before dropping
	_, err = db.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, dbName)
	if err != nil {
		logrus.Warnf("[DB_ISOLATION] Failed to terminate connections to database %s: %v", dbName, err)
	}

	// Drop the database
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to drop database %s: %w", dbName, err)
	}

	logrus.Infof("[DB_ISOLATION] Dropped PostgreSQL database: %s", dbName)
	return nil
}

// BackupDatabase creates a backup of the isolated database
func (dim *DatabaseIsolationManager) BackupDatabase(instanceID, backupPath string) error {
	db, err := dim.GetIsolatedDatabase(instanceID)
	if err != nil {
		return err
	}

	db.mutex.RLock()
	defer db.mutex.RUnlock()

	// Create backup directory
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy database file
	if err := copyFile(db.DatabasePath, filepath.Join(backupPath, "whatsapp.db")); err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	// Copy keys file
	if err := copyFile(db.KeysPath, filepath.Join(backupPath, "keys.db")); err != nil {
		return fmt.Errorf("failed to backup keys: %w", err)
	}

	logrus.Infof("[DB_ISOLATION] Backed up database for instance: %s", instanceID)
	return nil
}

// RestoreDatabase restores a database from backup
func (dim *DatabaseIsolationManager) RestoreDatabase(instanceID, backupPath string) error {
	db, err := dim.GetIsolatedDatabase(instanceID)
	if err != nil {
		return err
	}

	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Close existing connections
	if db.Connection != nil {
		db.Connection.Close()
		db.Connection = nil
	}
	if db.KeysConn != nil {
		db.KeysConn.Close()
		db.KeysConn = nil
	}

	// Restore database file
	if err := copyFile(filepath.Join(backupPath, "whatsapp.db"), db.DatabasePath); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	// Restore keys file
	if err := copyFile(filepath.Join(backupPath, "keys.db"), db.KeysPath); err != nil {
		return fmt.Errorf("failed to restore keys: %w", err)
	}

	// Reinitialize connections
	if err := dim.initializeDatabase(db); err != nil {
		return fmt.Errorf("failed to reinitialize database: %w", err)
	}

	logrus.Infof("[DB_ISOLATION] Restored database for instance: %s", instanceID)
	return nil
}

// ListDatabases returns all isolated databases
func (dim *DatabaseIsolationManager) ListDatabases() []*IsolatedDatabase {
	dim.mutex.RLock()
	defer dim.mutex.RUnlock()

	databases := make([]*IsolatedDatabase, 0, len(dim.databases))
	for _, db := range dim.databases {
		databases = append(databases, db)
	}

	return databases
}

// Private methods

func (dim *DatabaseIsolationManager) initializeDatabase(db *IsolatedDatabase) error {
	var driver string
	var mainConnStr, keysConnStr string

	switch db.DBType {
	case "postgres":
		driver = "postgres"
		mainConnStr = db.ConnectionURI
		keysConnStr = db.KeysURI
	default: // sqlite
		driver = "sqlite3"
		mainConnStr = db.ConnectionURI
		keysConnStr = db.KeysURI
	}

	// Initialize main database
	conn, err := sql.Open(driver, mainConnStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.Connection = conn

	// Initialize keys database
	keysConn, err := sql.Open(driver, keysConnStr)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open keys database: %w", err)
	}

	// Test keys connection
	if err := keysConn.Ping(); err != nil {
		conn.Close()
		keysConn.Close()
		return fmt.Errorf("failed to ping keys database: %w", err)
	}

	db.KeysConn = keysConn

	// Create basic tables if they don't exist
	if err := dim.createBasicTables(db); err != nil {
		conn.Close()
		keysConn.Close()
		return fmt.Errorf("failed to create basic tables: %w", err)
	}

	return nil
}

func (dim *DatabaseIsolationManager) createBasicTables(db *IsolatedDatabase) error {
	var queries, keysQueries []string

	switch db.DBType {
	case "postgres":
		// PostgreSQL-specific table creation
		queries = []string{
			`CREATE TABLE IF NOT EXISTS instance_info (
				id VARCHAR(255) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				phone VARCHAR(50),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS sessions (
				id VARCHAR(255) PRIMARY KEY,
				data BYTEA,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS messages (
				id VARCHAR(255) PRIMARY KEY,
				chat_id VARCHAR(255),
				message_data BYTEA,
				timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id)`,
			`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)`,
			`CREATE TABLE IF NOT EXISTS contacts (
				id VARCHAR(255) PRIMARY KEY,
				name VARCHAR(255),
				phone VARCHAR(50),
				data BYTEA,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		}

		keysQueries = []string{
			`CREATE TABLE IF NOT EXISTS encryption_keys (
				id VARCHAR(255) PRIMARY KEY,
				key_data BYTEA,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS session_keys (
				session_id VARCHAR(255) PRIMARY KEY,
				key_data BYTEA,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		}

	default: // sqlite
		// SQLite-specific table creation
		queries = []string{
			`CREATE TABLE IF NOT EXISTS instance_info (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				phone TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				data BLOB,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS messages (
				id TEXT PRIMARY KEY,
				chat_id TEXT,
				message_data BLOB,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id)`,
			`CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)`,
			`CREATE TABLE IF NOT EXISTS contacts (
				id TEXT PRIMARY KEY,
				name TEXT,
				phone TEXT,
				data BLOB,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		}

		keysQueries = []string{
			`CREATE TABLE IF NOT EXISTS encryption_keys (
				id TEXT PRIMARY KEY,
				key_data BLOB,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS session_keys (
				session_id TEXT PRIMARY KEY,
				key_data BLOB,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		}
	}

	// Execute main database queries
	for _, query := range queries {
		if _, err := db.Connection.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Execute keys database queries
	for _, query := range keysQueries {
		if _, err := db.KeysConn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute keys query: %w", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// Stop gracefully closes all database connections
func (dim *DatabaseIsolationManager) Stop() {
	dim.mutex.Lock()
	defer dim.mutex.Unlock()

	for instanceID, db := range dim.databases {
		if db.Connection != nil {
			db.Connection.Close()
		}
		if db.KeysConn != nil {
			db.KeysConn.Close()
		}
		logrus.Infof("[DB_ISOLATION] Closed database connections for instance: %s", instanceID)
	}

	logrus.Info("[DB_ISOLATION] Database isolation manager stopped")
}