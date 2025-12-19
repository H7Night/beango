package model

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db      *gorm.DB
	once    sync.Once
	dbMutex sync.RWMutex
)

type SqliteConfig struct {
	Path string `yaml:"path"`
	// 新增配置项
	MaxIdleConns    int    `yaml:"max_idle_conns"`    // 最大空闲连接数
	MaxOpenConns    int    `yaml:"max_open_conns"`    // 最大打开连接数
	ConnMaxLifetime string `yaml:"conn_max_lifetime"` // 连接最大生命周期
	Debug           bool   `yaml:"debug"`             // 是否开启调试模式
}

// ConnectDatabase 连接数据库（SQLite）- 线程安全的单例模式
func ConnectDatabase() {
	once.Do(func() {
		var err error
		db, err = initDatabaseConnection()
		if err != nil {
			log.Fatalf("Failed to connect database: %v", err)
		}
		log.Println("✅ Connect SQLite database success")
	})
}

// initDatabaseConnection 初始化数据库连接
func initDatabaseConnection() (*gorm.DB, error) {
	// 1. 加载配置
	sqliteConfig, err := loadSqliteConfig("config/sqlite.yml")
	if err != nil {
		log.Printf("Warning: %v, using default configuration", err)
		sqliteConfig = getDefaultConfig()
	}

	// 2. 确保数据库文件目录存在
	if err := ensureDatabaseDir(sqliteConfig.Path); err != nil {
		return nil, fmt.Errorf("ensure database directory failed: %w", err)
	}

	// 3. 创建GORM配置
	gormConfig := &gorm.Config{}
	if !sqliteConfig.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Error)
	}

	// 4. 连接数据库
	db, err := gorm.Open(sqlite.Open(sqliteConfig.Path), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect database failed: %w", err)
	}

	// 5. 获取底层sql.DB对象并配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 设置连接池参数
	if sqliteConfig.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(sqliteConfig.MaxIdleConns)
	}
	if sqliteConfig.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(sqliteConfig.MaxOpenConns)
	}
	if sqliteConfig.ConnMaxLifetime != "" {
		if duration, err := time.ParseDuration(sqliteConfig.ConnMaxLifetime); err == nil {
			sqlDB.SetConnMaxLifetime(duration)
		}
	}

	// 6. 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// 7. 初始化数据库表
	if err := initDatabaseTables(db); err != nil {
		return nil, fmt.Errorf("init database tables failed: %w", err)
	}

	// 8. 设置连接状态回调（可选）
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	log.Printf("Database initialized at: %s", sqliteConfig.Path)
	return db, nil
}

// loadSqliteConfig 读取配置文件
func loadSqliteConfig(path string) (*SqliteConfig, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var config SqliteConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("parse config file failed: %w", err)
	}

	// 使用默认值填充未设置的配置项
	config = fillDefaultValues(config)

	return &config, nil
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *SqliteConfig {
	// 获取可执行文件所在目录
	exeDir, err := os.Executable()
	if err != nil {
		exeDir = "."
	} else {
		exeDir = filepath.Dir(exeDir)
	}

	// 默认将数据库文件放在可执行文件同级的data目录下
	dbPath := filepath.Join(exeDir, "data", "beango.db")

	return &SqliteConfig{
		Path:            dbPath,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: "1h",
		Debug:           false,
	}
}

// fillDefaultValues 填充默认值
func fillDefaultValues(config SqliteConfig) SqliteConfig {
	if config.Path == "" {
		config.Path = "beango.db"
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 10
	}
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 100
	}
	if config.ConnMaxLifetime == "" {
		config.ConnMaxLifetime = "1h"
	}
	return config
}

// ensureDatabaseDir 确保数据库文件所在目录存在
func ensureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." {
		// 如果路径中没有目录部分，就不需要创建目录
		return nil
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s failed: %w", dir, err)
	}

	log.Printf("Database directory ensured: %s", dir)
	return nil
}

// initDatabaseTables 初始化数据库表
func initDatabaseTables(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 按依赖顺序初始化表
		models := []interface{}{
			&BeangoConfig{},
			&AccountMap{},
			// &BeancountTransaction{},
		}

		for _, model := range models {
			if err := tx.AutoMigrate(model); err != nil {
				return fmt.Errorf("migrate %T failed: %w", model, err)
			}
		}

		log.Printf("Database tables migrated successfully")
		return nil
	})
}

// GetDB 获取数据库连接实例（线程安全）
func GetDB() *gorm.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if db == nil {
		log.Println("Warning: Database not initialized, calling ConnectDatabase...")
		ConnectDatabase()
	}
	return db
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDatabaseStats 获取数据库连接池统计信息
func GetDatabaseStats() map[string]interface{} {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
