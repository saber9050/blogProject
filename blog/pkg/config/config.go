package config

// Config 应用配置结构体
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Coze     CozeConfig     `mapstructure:"coze"`
	Email    EmailConfig    `mapstructure:"email"`
	WeChat   WeChatConfig   `mapstructure:"wechat"`
	Minio    MinioConfig    `mapstructure:"minio"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Mode    string `mapstructure:"mode"` // debug, release, test
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	ExpireHours        int    `mapstructure:"expire_hours"`         // Access Token 过期时间（小时）
	RefreshExpireHours int    `mapstructure:"refresh_expire_hours"` // Refresh Token 过期时间（小时）
	RememberMeHours    int    `mapstructure:"remember_me_hours"`    // "记住我"模式过期时间（小时）
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Filename   string `mapstructure:"filename"`    // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留旧日志文件的最大天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
}

// CORSConfig CORS 配置
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// CozeConfig 扣子配置
type CozeConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	APIKey     string `mapstructure:"api_key"`
	WorkflowID string `mapstructure:"workflow_id"`
	BotID      string `mapstructure:"cozeBotID"`
	TimeoutSec int    `mapstructure:"timeout_sec"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// WeChatConfig 微信小程序配置
type WeChatConfig struct {
	AppID               string `mapstructure:"app_id"`
	AppSecret           string `mapstructure:"app_secret"`
	Code2SessionURL     string `mapstructure:"code2session_url"`
	AccessTokenURL      string `mapstructure:"access_token_url"`
	SubscribeSendURL    string `mapstructure:"subscribe_send_url"`
	SubscribeTemplateID string `mapstructure:"subscribe_template_id"`
	SubscribeResultPage string `mapstructure:"subscribe_result_page"`
	TimeoutSec          int    `mapstructure:"timeout_sec"`
}

// MinioConfig MinIO 配置
type MinioConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	BaseURL         string `mapstructure:"base_url"` // 用于拼接文件访问 URL，为空时自动根据 endpoint 和 use_ssl 生成
}

// CryptoConfig 数据传输加密配置
type CryptoConfig struct {
	RSAPrivateKey string `mapstructure:"rsa_private_key"` // PEM 格式 RSA 私钥
}
