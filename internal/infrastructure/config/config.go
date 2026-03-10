package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	config     *Config
	configOnce sync.Once
)

// Config stores all configuration of the application
type Config struct {
	// Environment type
	EnvType string

	// Database
	DBHost          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBPort          string
	DBMigrationMode string // 数据库迁移模式: "auto"(默认), "alter"(修改), "drop"(删除重建)

	// Server
	ServerPort string

	// Redis
	RedisHost string
	RedisPort string
	RedisDB   int

	// Aliyun RTC
	AliyunAccessKey string
	AliyunRTCAppID  string
	AliyunRTCRegion string

	// Tencent Cloud RTC
	TencentSDKAppID   int    // 腾讯云 SDKAppID
	TencentSecretKey  string // 腾讯云 SDKAppID 对应的密钥
	TencentRTCEnabled bool   // 是否启用腾讯云RTC，如果为false则使用阿里云RTC

	// MQTT配置
	MQTTBrokerURL  string // MQTT服务器地址，如 tcp://broker.example.com:1883
	MQTTClientID   string // MQTT客户端ID
	MQTTUsername   string // MQTT用户名
	MQTTPassword   string // MQTT密码
	MQTTQoS        int    // 服务质量 (0, 1, 2)
	MQTTRetained   bool   // 是否保留消息
	MQTTSSLEnabled bool   // 是否启用SSL/TLS
	MQTTCACertPath string // CA证书路径，用于SSL/TLS验证

	// JWT Authentication
	JWTSecretKey string

	// Admin
	DefaultAdminPassword string
}

// LoadConfig loads config from environment variables based on ENV_TYPE
func LoadConfig() *Config {
	// Get environment type (default to LOCAL if not set)
	envType := getEnv("ENV_TYPE", "LOCAL")
	prefix := ""

	// Set prefix based on environment type
	if strings.ToUpper(envType) == "LOCAL" {
		prefix = "LOCAL_"
	} else if strings.ToUpper(envType) == "SERVER" {
		prefix = "SERVER_"
	} else {
		fmt.Printf("Warning: Unknown ENV_TYPE '%s', defaulting to LOCAL environment\n", envType)
		prefix = "LOCAL_"
		envType = "LOCAL"
	}

	fmt.Printf("Loading configuration for environment: %s\n", envType)

	// 解析腾讯云SDKAppID
	tencentAppID, _ := strconv.Atoi(getEnv("TENCENT_SDKAPPID", "0"))

	return &Config{
		// Environment type
		EnvType: envType,

		// Database config - use environment-specific variables if available
		DBHost:          getEnvRequired(prefix + "DB_HOST"),
		DBUser:          getEnvRequired(prefix + "DB_USER"),
		DBPassword:      getEnvRequired(prefix + "DB_PASSWORD"),
		DBName:          getEnvRequired(prefix + "DB_NAME"),
		DBPort:          getEnvRequired(prefix + "DB_PORT"),
		DBMigrationMode: getEnv(prefix+"DB_MIGRATION_MODE", "auto"),

		// Server config
		ServerPort: getEnv(prefix+"SERVER_PORT", getEnv("SERVER_PORT", "8080")),

		// Redis config
		RedisHost: getEnv(prefix+"REDIS_HOST", getEnv("	REDIS_HOST", "localhost")),
		RedisPort: getEnv(prefix+"REDIS_PORT", getEnv("REDIS_PORT", "6380")),
		RedisDB:   getEnvAsInt("REDIS_DB", 0),

		// Aliyun RTC config（启用腾讯云 RTC 时可留空）
		AliyunAccessKey: getEnv("ALIYUN_ACCESS_KEY", ""),
		AliyunRTCAppID:  getEnv("ALIYUN_RTC_APP_ID", ""),
		AliyunRTCRegion: getEnv("ALIYUN_RTC_REGION", "cn-hangzhou"),

		// Tencent Cloud RTC config
		TencentSDKAppID:   tencentAppID,
		TencentSecretKey:  getEnv("TENCENT_SECRET_KEY", ""),
		TencentRTCEnabled: getEnvAsBool("TENCENT_RTC_ENABLED", false),

		// MQTT配置
		MQTTBrokerURL:  getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTClientID:   getEnv("MQTT_CLIENT_ID", "intercom_server"),
		MQTTUsername:   getEnv("MQTT_USERNAME", ""),
		MQTTPassword:   getEnv("MQTT_PASSWORD", ""),
		MQTTQoS:        getEnvAsInt("MQTT_QOS", 1),
		MQTTRetained:   getEnvAsBool("MQTT_RETAINED", false),
		MQTTSSLEnabled: getEnvAsBool("MQTT_SSL_ENABLED", false),
		MQTTCACertPath: getEnv("MQTT_CA_CERT_PATH", ""),

		// JWT Config
		JWTSecretKey: getEnv("JWT_SECRET_KEY", "intercom-secret-key-change-in-production"),

		// Admin Config
		DefaultAdminPassword: getEnvRequired("DEFAULT_ADMIN_PASSWORD"),
	}
}

// GetConfig returns the application configuration as a singleton
func GetConfig() *Config {
	configOnce.Do(func() {
		config = LoadConfig()
	})
	return config
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local&allowNativePasswords=true&multiStatements=true"
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as integer with default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as boolean with default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// 要求必须提供环境变量的辅助函数
func getEnvRequired(key string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	panic(fmt.Sprintf("Required environment variable %s is not set", key))
}
