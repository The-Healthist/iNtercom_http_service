// @title           intercom_http_service API
// @version         1.0
// @description     A comprehensive intercom backend service with video calling capabilities
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.yourcompany.com/support
// @contact.email  support@yourcompany.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      39.108.49.167:20033
// @BasePath  /api

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Enter the token with the `Bearer: ` prefix
package main

import (
	"fmt"
	"intercom_http_service/internal/router"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/database"
	Logger "intercom_http_service/internal/logger"
	"log"
	"os"
	"runtime"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 设置最大处理器数量，提高并发性能
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 初始化日志配置
	if err := Logger.SetupLogger(); err != nil {
		fmt.Printf("初始化日志配置失败: %v\n", err)
		os.Exit(1)
	}

	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		Logger.Warning("无法加载.env文件: %v", err)
		// 即使加载失败也继续执行，可能环境变量已经通过其他方式设置
	} else {
		Logger.Info("成功加载.env文件")
	}

	// 获取配置
	cfg := config.GetConfig()

	// 创建优化的数据库连接池
	pool, err := database.NewConnectionPool(cfg)
	if err != nil {
		log.Fatalf("无法创建数据库连接池: %v", err)
	}
	db := pool.GetDB()

	// 根据配置执行不同的数据库操作
	if cfg.DBMigrationMode == "drop" {
		// 删除并重建表
		log.Println("警告: 在drop模式下运行，将删除并重建所有表")
		err = dropAndRecreateTables(db)
		if err != nil {
			log.Fatalf("删除并重建表失败: %v", err)
		}
	} else if cfg.DBMigrationMode == "alter" {
		// 执行高级迁移，包括修改列、删除列等
		log.Println("在alter模式下运行，将修改表结构以匹配模型")
		err = advancedMigrate(db, cfg)
		if err != nil {
			log.Fatalf("高级迁移失败: %v", err)
		}
	} else {
		// 默认AutoMigrate，只会添加新列和新表，不会删除或修改列
		log.Println("在标准模式下运行，将只添加新列和新表")
		if err := autoMigrate(db); err != nil {
			log.Fatalf("自动迁移失败: %v", err)
		}
	}

	// 确保系统中有管理员账户
	ensureAdminExists(db, cfg)

	// 初始化路由
	r := router.SetupRouter(db, cfg)

	// 使用配置中的端口，而不是直接从环境变量获取
	port := cfg.ServerPort

	// 打印系统信息
	printSystemInfo(pool)

	// 启动服务器 - 注意监听所有接口(0.0.0.0)而不是只监听localhost
	Logger.Info("服务器启动在: http://0.0.0.0:%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		Logger.Error("启动服务器失败: %v", err)
		os.Exit(1)
	}
}

// initDB 初始化数据库连接
func initDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	fmt.Println("Database connection established")
	return db, nil
}

// autoMigrate 自动迁移所有模型（只添加新列和新表）
func autoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.Admin{},
		&model.PropertyStaff{},
		&model.Device{},
		&model.Resident{},
		&model.CallRecord{},
		&model.AccessLog{},
		&model.EmergencyLog{},
		&model.SystemLog{},
	)

	if err != nil {
		return err
	}

	fmt.Println("Database migration completed")
	return nil
}

// advancedMigrate 执行高级迁移，包括修改列、删除列等
func advancedMigrate(db *gorm.DB, cfg *config.Config) error {
	// 获取底层SQL连接
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB connection: %w", err)
	}

	// 禁用外键约束检查
	_, err = sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		log.Printf("禁用外键约束检查失败: %v", err)
	}
	defer sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 1") // 确保在函数结束时重新启用外键约束

	// 处理 property_staffs 表的特殊迁移
	log.Println("开始处理property_staffs表的特殊迁移")

	// 1. 检查表是否存在
	var tableExists bool
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'property_staffs'", cfg.DBName).Scan(&tableExists)
	if err != nil {
		log.Printf("检查表是否存在失败: %v", err)
	}

	if tableExists {
		// 2. 查询表中的所有列
		rows, err := sqlDB.Query(`
			SELECT COLUMN_NAME, IS_NULLABLE, COLUMN_DEFAULT 
			FROM INFORMATION_SCHEMA.COLUMNS 
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'property_staffs'
		`, cfg.DBName)

		if err != nil {
			log.Printf("查询表列失败: %v", err)
		} else {
			defer rows.Close()

			// 定义应该存在于模型中的列名
			modelColumns := map[string]bool{
				"id": true, "phone": true, "property_name": true, "position": true,
				"role": true, "status": true, "remark": true, "username": true,
				"password": true, "created_at": true, "updated_at": true,
				// 不包含 name 和 property_id，它们在模型中已经被移除
			}

			// 处理每一列
			for rows.Next() {
				var columnName, isNullable string
				var columnDefault interface{}
				if err := rows.Scan(&columnName, &isNullable, &columnDefault); err != nil {
					log.Printf("扫描列信息失败: %v", err)
					continue
				}

				// 检查列是否应该在模型中存在
				if !modelColumns[columnName] && columnName != "id" &&
					columnName != "created_at" && columnName != "updated_at" {
					log.Printf("在property_staffs表中发现多余列: %s，准备修改", columnName)

					// 对于property_id列，我们可以先将其设置为可为NULL，再删除
					if columnName == "property_id" && isNullable == "NO" {
						log.Printf("将property_id列修改为可为NULL")
						_, err = sqlDB.Exec("ALTER TABLE property_staffs MODIFY COLUMN property_id INT NULL")
						if err != nil {
							log.Printf("修改property_id列失败: %v", err)
						}
					}

					// 尝试删除列
					log.Printf("尝试删除列: %s", columnName)
					_, err = sqlDB.Exec(fmt.Sprintf("ALTER TABLE property_staffs DROP COLUMN %s", columnName))
					if err != nil {
						log.Printf("删除列失败: %v", err)
					}
				}
			}
		}
	}

	// 自动迁移其他表
	return autoMigrate(db)
}

// dropAndRecreateTables 删除并重建所有表
func dropAndRecreateTables(db *gorm.DB) error {
	// 获取底层SQL连接
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB connection: %w", err)
	}

	// 禁用外键约束检查
	_, err = sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		log.Printf("禁用外键约束检查失败: %v", err)
	}
	defer sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 1") // 确保在函数结束时重新启用外键约束

	// 删除所有表
	tables := []string{
		"admins", "property_staffs", "devices", "residents", "call_records",
		"access_logs", "emergency_logs", "system_logs", "buildings", "households",
	}

	for _, table := range tables {
		log.Printf("删除表: %s", table)
		_, err := sqlDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			log.Printf("删除表失败: %v", err)
		}
	}

	// 重新创建表
	return autoMigrate(db)
}

// ensureAdminExists 确保系统中有管理员账户
func ensureAdminExists(db *gorm.DB, cfg *config.Config) {
	var count int64
	db.Model(&model.Admin{}).Count(&count)

	if count == 0 {
		// 如果没有管理员，创建默认管理员
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.DefaultAdminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("生成密码哈希失败: %v", err)
		}

		admin := model.Admin{
			Username: "admin",
			Password: string(hashedPassword),
			Role:     "system_admin",
			Status:   "active",
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Fatalf("创建默认管理员失败: %v", err)
		}

		log.Println("已创建默认管理员账户")
	}
}

// printSystemInfo 打印系统信息
func printSystemInfo(pool *database.ConnectionPool) {
	// 打印数据库连接池信息
	stats, err := pool.Stats()
	if err == nil {
		log.Printf("数据库连接池状态: %+v", stats)
	}

	// 打印系统资源信息
	log.Printf("系统CPU核心数: %d", runtime.NumCPU())
	log.Printf("当前Go协程数: %d", runtime.NumGoroutine())

	// 打印内存信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("系统内存使用: Alloc=%v MiB, TotalAlloc=%v MiB, Sys=%v MiB",
		m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024)
}
