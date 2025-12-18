package database

import (
	"fmt"
	"server_monitor/utils"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB // DB Global Connection. Becarefull to use this because it open New connection while used

func log_level() logger.LogLevel {
	log_db_mode := utils.Getenv("LOG_DB_MODE", "silent")
	log_level := logger.Silent
	switch log_db_mode {
	case "silent":
		log_level = logger.Silent
	case "error":
		log_level = logger.Error
	case "warn":
		log_level = logger.Warn
	case "info":
		log_level = logger.Info
	case "debug":
		log_level = logger.Info
	}
	return log_level
}

// InitDB initializes and returns a database connection
func InitDB(dbURI string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	DB = db
	return db, nil
}

func InitWebDB(dbURI string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return db, nil
}

func InitAndCheckDB(dbUser, dbPass, dbHost, dbPort, dbName string) (*gorm.DB, error) {
	// Connect to information_schema
	infoSchemaURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/information_schema?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPass,
		dbHost,
		dbPort,
	)
	infoSchemaDB, err := gorm.Open(mysql.Open(infoSchemaURI), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to information_schema: %v", err)
	}

	// Check if the database exists
	var dbExists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT SCHEMA_NAME FROM SCHEMATA WHERE SCHEMA_NAME = '%s')", dbName)
	err = infoSchemaDB.Raw(query).Scan(&dbExists).Error
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to check if database exists: %v", err)
	}

	// Create the database if it does not exist
	if !dbExists {
		createDBQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
		err = infoSchemaDB.Exec(createDBQuery).Error
		if err != nil {
			logrus.Error(err)
			return nil, fmt.Errorf("failed to create database: %v", err)
		}
		fmt.Printf("Database %s created successfully\n", dbName)
	}

	// Close the connection to information_schema
	dbSQL, err := infoSchemaDB.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}
	dbSQL.Close()

	// Connect to the specified database
	dbURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPass,
		dbHost,
		dbPort,
		dbName,
	)
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Get the underlying sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get db instance: %v", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)           // Set the maximum number of idle connections
	sqlDB.SetMaxOpenConns(100)          // Set the maximum number of open connections
	sqlDB.SetConnMaxLifetime(time.Hour) // Set the maximum lifetime of a connection

	return db, nil
}

func InitMySqlDB(dbUser, dbPass, dbHost, dbPort, dbName string) (*gorm.DB, error) {
	// Try connecting to information_schema
	infoSchemaURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/information_schema?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort,
	)
	infoSchemaDB, err := gorm.Open(mysql.Open(infoSchemaURI), &gorm.Config{Logger: logger.Default.LogMode(log_level())})
	if err == nil {
		// Check if the database exists
		var dbExists bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT SCHEMA_NAME FROM SCHEMATA WHERE SCHEMA_NAME = '%s')", dbName)
		if err = infoSchemaDB.Raw(query).Scan(&dbExists).Error; err == nil {
			if !dbExists {
				createDBQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
				if err = infoSchemaDB.Exec(createDBQuery).Error; err == nil {
					fmt.Printf("Database %s created successfully\n", dbName)
				}
			}
		}

		// Close the connection to information_schema
		dbSQL, _ := infoSchemaDB.DB()
		dbSQL.Close()
	}

	// Connect directly to the specified database
	dbURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{Logger: logger.Default.LogMode(log_level())})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Get the underlying sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get db instance: %v", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("Connected to the database successfully : " + dbName)
	return db, nil
}
func InitPostgreSqlDB(dbUser, dbPass, dbHost, dbPort, dbName string) (*gorm.DB, error) {
	// Connect to the PostgreSQL information_schema
	infoSchemaURI := fmt.Sprintf("host=%s port=%s user=%s dbname=postgres password=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass,
	)
	infoSchemaDB, err := gorm.Open(postgres.Open(infoSchemaURI), &gorm.Config{Logger: logger.Default.LogMode(log_level())})
	if err == nil {
		// Check if the database exists
		var dbExists bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", dbName)
		err = infoSchemaDB.Raw(query).Scan(&dbExists).Error
		if err != nil {
			logrus.Error(err)
			return nil, fmt.Errorf("failed to check if database exists: %v", err)
		}

		// Create the database if it does not exist
		if !dbExists {
			createDBQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
			err = infoSchemaDB.Exec(createDBQuery).Error
			if err != nil {
				logrus.Error(err)
				return nil, fmt.Errorf("failed to create database: %v", err)
			}
			fmt.Printf("Database %s created successfully\n", dbName)
		}

		// Close the connection to information_schema
		dbSQL, err := infoSchemaDB.DB()
		if err != nil {
			logrus.Error(err)
			return nil, fmt.Errorf("failed to get database instance: %v", err)
		}
		dbSQL.Close()
	}

	// Connect to the specified database
	dbURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbName, dbPass,
	)
	db, err := gorm.Open(postgres.Open(dbURI), &gorm.Config{Logger: logger.Default.LogMode(log_level())})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Get the underlying sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get db instance: %v", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("Connected to the PostgreSQL database successfully")
	return db, nil
}
func InitSqlLiteDB(dbPath string) (*gorm.DB, error) {
	// Connect to the SQLite database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to SQLite database: %v", err)
	}

	// Get the underlying sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get db instance: %v", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("Connected to the SQLite database successfully")
	return db, nil
}

func InitMsSqlDB(dbUser, dbPass, dbHost, dbPort, dbName string) (*gorm.DB, error) {
	// Validasi parameter penting
	if dbUser == "" || dbPass == "" || dbHost == "" || dbPort == "" || dbName == "" {
		return nil, fmt.Errorf("missing one or more required database connection parameters")
	}

	// Buat DSN
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s", dbUser, dbPass, dbHost, dbPort, dbName)

	// Koneksi ke SQL Server
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to connect to SQL Server: %v", err)
	}

	// Ambil *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		logrus.Error(err)
		return nil, fmt.Errorf("failed to get underlying DB instance: %v", err)
	}

	// Atur connection pooling
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Tes koneksi
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQL Server: %v", err)
	}

	fmt.Println("Connected to SQL Server successfully")
	return db, nil
}
