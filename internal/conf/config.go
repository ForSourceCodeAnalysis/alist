package conf

import (
	"path/filepath"

	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/pkg/utils/random"
)

// 数据库配置
type Database struct {
	Type        string `json:"type" env:"TYPE"`
	Host        string `json:"host" env:"HOST"`
	Port        int    `json:"port" env:"PORT"`
	User        string `json:"user" env:"USER"`
	Password    string `json:"password" env:"PASS"`
	Name        string `json:"name" env:"NAME"`
	DBFile      string `json:"db_file" env:"FILE"`
	TablePrefix string `json:"table_prefix" env:"TABLE_PREFIX"`
	SSLMode     string `json:"ssl_mode" env:"SSL_MODE"`
}

// web服务配置信息
type Scheme struct {
	Address      string `json:"address" env:"ADDR"`                  //默认监听全网ipv4(ipv6修改成 :: )，如果只想监听本地，可以修改成127.0.0.1(ipv6 ::1)
	HttpPort     int    `json:"http_port" env:"HTTP_PORT"`           //http监听端口 默认5244
	HttpsPort    int    `json:"https_port" env:"HTTPS_PORT"`         //https监听端口
	ForceHttps   bool   `json:"force_https" env:"FORCE_HTTPS"`       //强制使用https
	CertFile     string `json:"cert_file" env:"CERT_FILE"`           //使用https时需要设置，证书路径
	KeyFile      string `json:"key_file" env:"KEY_FILE"`             //使用https时需要设置，密钥
	UnixFile     string `json:"unix_file" env:"UNIX_FILE"`           //unix socket
	UnixFilePerm string `json:"unix_file_perm" env:"UNIX_FILE_PERM"` //unix socket 权限
}

type LogConfig struct {
	Enable     bool   `json:"enable" env:"LOG_ENABLE"`       //是否启用日志
	Name       string `json:"name" env:"LOG_NAME"`           //日志文件
	MaxSize    int    `json:"max_size" env:"MAX_SIZE"`       //单位MB，单个日志文件达到MaxSize后，会自动创建下一个新的文件
	MaxBackups int    `json:"max_backups" env:"MAX_BACKUPS"` //日志文件最大备份数 因为每个文件有最大大小限制，所以会产生多个日志文件，但最多有MaxBackups个，超过会删除
	MaxAge     int    `json:"max_age" env:"MAX_AGE"`         //日志最大保存天数
	Compress   bool   `json:"compress" env:"COMPRESS"`       //是否压缩日志
}

type TaskConfig struct {
	Workers  int `json:"workers" env:"WORKERS"`
	MaxRetry int `json:"max_retry" env:"MAX_RETRY"`
}

type TasksConfig struct {
	Download TaskConfig `json:"download" envPrefix:"DOWNLOAD_"`
	Transfer TaskConfig `json:"transfer" envPrefix:"TRANSFER_"`
	Upload   TaskConfig `json:"upload" envPrefix:"UPLOAD_"`
	Copy     TaskConfig `json:"copy" envPrefix:"COPY_"`
}

type Cors struct {
	AllowOrigins []string `json:"allow_origins" env:"ALLOW_ORIGINS"`
	AllowMethods []string `json:"allow_methods" env:"ALLOW_METHODS"`
	AllowHeaders []string `json:"allow_headers" env:"ALLOW_HEADERS"`
}

// 总配置
type Config struct {
	Force                 bool        `json:"force" env:"FORCE"`                       //强制从环境变量读取配置
	SiteURL               string      `json:"site_url" env:"SITE_URL"`                 //web服务网址
	Cdn                   string      `json:"cdn" env:"CDN"`                           //cdn地址，如果设置，alist web服务的一些前端资源会从cdn地址加载，而不是从你搭建的服务地址下载
	JwtSecret             string      `json:"jwt_secret" env:"JWT_SECRET"`             //jwt token
	TokenExpiresIn        int         `json:"token_expires_in" env:"TOKEN_EXPIRES_IN"` //token过期时间，单位 天
	Database              Database    `json:"database"`
	Scheme                Scheme      `json:"scheme"`
	TempDir               string      `json:"temp_dir" env:"TEMP_DIR"`
	BleveDir              string      `json:"bleve_dir" env:"BLEVE_DIR"` //这是一个文本索引库，具体用法还需要研究
	Log                   LogConfig   `json:"log"`
	DelayedStart          int         `json:"delayed_start" env:"DELAYED_START"`                       //延迟启动，主要是自启动时，考虑到有些时候网络连接好，如果启动太快会导致一些驱动连接失败
	MaxConnections        int         `json:"max_connections" env:"MAX_CONNECTIONS"`                   //同一时间最大连接并发数
	TlsInsecureSkipVerify bool        `json:"tls_insecure_skip_verify" env:"TLS_INSECURE_SKIP_VERIFY"` //是否跳过检查tls证书
	Tasks                 TasksConfig `json:"tasks" envPrefix:"TASKS_"`
	Cors                  Cors        `json:"cors" envPrefix:"CORS_"`
}

func DefaultConfig() *Config {
	tempDir := filepath.Join(flags.DataDir, "temp")
	indexDir := filepath.Join(flags.DataDir, "bleve")
	logPath := filepath.Join(flags.DataDir, "log/log.log")
	dbPath := filepath.Join(flags.DataDir, "data.db")
	return &Config{
		Scheme: Scheme{
			Address:    "0.0.0.0",
			UnixFile:   "",
			HttpPort:   5244,
			HttpsPort:  -1,
			ForceHttps: false,
			CertFile:   "",
			KeyFile:    "",
		},
		JwtSecret:      random.String(16),
		TokenExpiresIn: 48,
		TempDir:        tempDir,
		Database: Database{
			Type:        "sqlite3",
			Port:        0,
			TablePrefix: "x_",
			DBFile:      dbPath,
		},
		BleveDir: indexDir,
		Log: LogConfig{
			Enable:     true,
			Name:       logPath,
			MaxSize:    50,
			MaxBackups: 30,
			MaxAge:     28,
		},
		MaxConnections:        0,
		TlsInsecureSkipVerify: true,
		Tasks: TasksConfig{
			Download: TaskConfig{
				Workers:  5,
				MaxRetry: 1,
			},
			Transfer: TaskConfig{
				Workers:  5,
				MaxRetry: 2,
			},
			Upload: TaskConfig{
				Workers: 5,
			},
			Copy: TaskConfig{
				Workers:  5,
				MaxRetry: 2,
			},
		},
		Cors: Cors{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"*"},
			AllowHeaders: []string{"*"},
		},
	}
}
