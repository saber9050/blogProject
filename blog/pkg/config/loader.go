package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

var globalConfig *Config

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 环境变量前缀
	v.SetEnvPrefix("core-coach")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖敏感配置
	//
	// 这里每个变量都是显式 os.Getenv，不能指望上面的 v.AutomaticEnv()：
	// viper 的 Unmarshal 不会读取 AutomaticEnv 注册的键，
	// 所以 compose 里注入的 CORE_COACH_* 变量对结构体字段实际是无效的。
	// （历史上 MYSQL_HOST 就是这么「配了但没生效」的。）
	if val := os.Getenv("PG_HOST"); val != "" {
		config.Database.Postgres.Host = val
	}
	if val := os.Getenv("PG_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			config.Database.Postgres.Port = p
		}
	}
	if val := os.Getenv("PG_USER"); val != "" {
		config.Database.Postgres.Username = val
	}
	if val := os.Getenv("PG_PASSWORD"); val != "" {
		config.Database.Postgres.Password = val
	}
	if val := os.Getenv("PG_DATABASE"); val != "" {
		config.Database.Postgres.Database = val
	}
	if val := os.Getenv("REDIS_PASSWORD"); val != "" {
		config.Database.Redis.Password = val
	}
	if val := os.Getenv("JWT_SECRET"); val != "" {
		config.JWT.Secret = val
	}
	if val := os.Getenv("COZE_API_KEY"); val != "" {
		config.Coze.APIKey = val
	}
	if val := os.Getenv("CRYPTO_RSA_PRIVATE_KEY"); val != "" {
		config.Crypto.RSAPrivateKey = val
	}
	// MinIO 对外访问地址（浏览器拼图片 URL 用的前缀）。
	// 注意：这里必须显式覆盖，不能指望上面的 AutomaticEnv ——
	// viper 的 Unmarshal 不会读取 AutomaticEnv 注册的键，
	// 所以 compose 里注入的 CORE_COACH_* 变量对结构体字段是无效的。
	if val := os.Getenv("MINIO_PUBLIC_URL"); val != "" {
		config.Minio.BaseURL = val
	}
	globalConfig = config
	return config, nil
}

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		panic("配置未初始化，请先调用 Load() 加载配置")
	}
	return globalConfig
}

// MustLoad 加载配置，失败时 panic
func MustLoad(configPath string) *Config {
	config, err := Load(configPath)
	if err != nil {
		panic(err)
	}
	return config
}
