package config

import (
	"log"
	"os"

	"my-books/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库
func InitDB() {
	// 确保数据目录存在
	if err := os.MkdirAll(Cfg.Data.DataDir, 0755); err != nil {
		log.Fatal("failed to create data directory:", err)
	}

	dbPath := Cfg.Data.DBPath
	// SQLite 连接参数说明（修复"扫描导入锁库"问题）：
	//   - journal_mode(WAL)：写前日志模式，允许并发读 + 单写者，显著降低锁冲突；
	//   - busy_timeout(5000)：抢不到写锁时等待 5s 而非立即报 SQLITE_BUSY；
	//   - synchronous(NORMAL)：WAL 下安全且比 FULL 更快的刷盘策略；
	//   - foreign_keys(1)：保持外键约束（沿用旧配置）。
	dsn := dbPath + "?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// 连接池：SQLite 单写者限制，将最大打开连接数限制为 1，
	// 使所有读写都串行复用同一连接，从连接层面彻底消除并发写锁冲突。
	// 个人图书管理场景并发读压力小，1 连接足以胜任；如需更高读并发，
	// 可结合 WAL 放宽为少量连接（如 4），但写操作仍需在应用层串行化。
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB:", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// 自动迁移
	err = db.AutoMigrate(
		&models.User{},
		&models.Book{},
		&models.Annotation{},
		&models.Bookmark{},
		&models.Collection{},
		&models.CollectionBook{},
		&models.ReadingHistory{},
		&models.ReadingPlan{},
		&models.Setting{},
		&models.BookShare{},
		&models.BookReview{},
		&models.AudiobookShare{},
		&models.Category{},
		&models.AudiobookTask{},
		&models.Library{},
		&models.LibraryScanLog{},
	)
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	// 清理旧版本遗留的全局唯一索引 idx_settings_key（当时无 user_id 维度），
	// 该索引会导致不同用户的同名设置项互相冲突，组合索引 idx_user_key 已覆盖
	db.Exec("DROP INDEX IF EXISTS idx_settings_key")

	DB = db
	log.Println("Database initialized at", dbPath)
}
