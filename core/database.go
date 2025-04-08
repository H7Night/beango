package core

import (
	"beango/model"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

type mysqlConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
}

// ConnectDatabase 连接数据库
func ConnectDatabase() {
	mysqlConfig, err := loadMysqlConfig("config/mysql.yml")
	if err != nil {
		panic("faild to load database config")
	}
	db, err = connectMysql(mysqlConfig)
	if err != nil {
		panic("faild to connect database")
	}
	fmt.Println("成功连接到数据库")
	initDatabase(db)
}

// 读取配置
func loadMysqlConfig(path string) (*mysqlConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置错误: %w", err)
	}
	var config mysqlConfig
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("读取配置错误: %w", err)
	}
	return &config, nil
}

// 根据配置连接数据库
func connectMysql(config *mysqlConfig) (*gorm.DB, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&loc=Local&parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset)
	db, err := gorm.Open(mysql.Open(url), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败:%w", err)
	}
	return db, err
}

func initDatabase(db *gorm.DB) {
	err := db.AutoMigrate(&model.ImportTranscation{})
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
}

func GetDB() *gorm.DB {
	return db
}
