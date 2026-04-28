package database

import (
	"context"
	"intercom_http_service/internal/config"
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectionPool 数据库连接池管理
type ConnectionPool struct {
	DB              *gorm.DB
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewConnectionPool 创建新的数据库连接池
func NewConnectionPool(cfg *config.Config) (*ConnectionPool, error) {
	// 创建数据库连接
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(resolveLogLevel(cfg.DBLogLevel)),
	})
	if err != nil {
		return nil, err
	}

	// 创建连接池
	pool := &ConnectionPool{
		DB:              db,
		MaxIdleConns:    10,               // 默认空闲连接数
		MaxOpenConns:    100,              // 默认最大连接数
		ConnMaxLifetime: 1 * time.Hour,    // 连接最大生命周期
		ConnMaxIdleTime: 30 * time.Minute, // 空闲连接最大生命周期
	}

	// 初始化连接池配置
	pool.ConfigurePool()

	return pool, nil
}

// ConfigurePool 配置连接池参数
func (p *ConnectionPool) ConfigurePool() error {
	// 获取底层SQL连接
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(p.MaxIdleConns)
	sqlDB.SetMaxOpenConns(p.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(p.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(p.ConnMaxIdleTime)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	log.Printf("数据库连接池已配置: 最大空闲连接数=%d, 最大连接数=%d", p.MaxIdleConns, p.MaxOpenConns)
	return nil
}

// UpdatePoolConfig 更新连接池配置
func (p *ConnectionPool) UpdatePoolConfig(maxIdle, maxOpen int, maxLifetime, maxIdleTime time.Duration) error {
	p.MaxIdleConns = maxIdle
	p.MaxOpenConns = maxOpen
	p.ConnMaxLifetime = maxLifetime
	p.ConnMaxIdleTime = maxIdleTime

	return p.ConfigurePool()
}

// Stats 获取连接池统计信息
func (p *ConnectionPool) Stats() (map[string]interface{}, error) {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return nil, err
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
	}, nil
}

// Close 关闭连接池
func (p *ConnectionPool) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// WithTransaction 在事务中执行函数
func (p *ConnectionPool) WithTransaction(fn func(tx *gorm.DB) error) error {
	return p.DB.Transaction(fn)
}

// HealthCheck 健康检查
func (p *ConnectionPool) HealthCheck() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

// GetDB 获取GORM数据库实例
func (p *ConnectionPool) GetDB() *gorm.DB {
	return p.DB
}

func resolveLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
