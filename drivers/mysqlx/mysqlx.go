package mysqlx

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Config 提供最常用的 MySQL 连接配置项。作为第三方包，配置由使用方传入。
type Config struct {
	// 基本连接
	User     string
	Password string
	Net      string // tcp 或 unix，默认 tcp
	Addr     string // host:port 或 unix socket path
	DBName   string

	// 连接参数
	Params map[string]string // 额外 DSN 参数，例如 {"parseTime":"true"}

	// 连接池设置
	MaxOpenConns    int           // 0 表示不限制
	MaxIdleConns    int           // 0 表示使用数据库驱动默认值
	ConnMaxLifetime time.Duration // 0 表示不限制
	ConnMaxIdleTime time.Duration // 0 表示不限制

	// 启动探活
	PingTimeout time.Duration // 0 表示不 Ping
}

// Option 函数式选项，用于在基础 Config 上叠加修改
type Option func(*Config)

// WithParam 追加 DSN 参数
func WithParam(key, value string) Option {
	return func(c *Config) {
		if c.Params == nil {
			c.Params = make(map[string]string)
		}
		c.Params[key] = value
	}
}

// WithPingTimeout 设置初始化 Ping 的超时
func WithPingTimeout(d time.Duration) Option {
	return func(c *Config) { c.PingTimeout = d }
}

// BuildDSN 根据配置构建 DSN 字符串（使用 go-sql-driver/mysql 的 Config 保证转义正确）
func BuildDSN(cfg Config) string {
	mcfg := mysql.NewConfig()
	mcfg.User = cfg.User
	mcfg.Passwd = cfg.Password
	if cfg.Net != "" {
		mcfg.Net = cfg.Net
	} else {
		mcfg.Net = "tcp"
	}
	mcfg.Addr = cfg.Addr
	mcfg.DBName = cfg.DBName

	// 默认常用参数
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	if _, ok := cfg.Params["parseTime"]; !ok {
		cfg.Params["parseTime"] = "true"
	}
	if _, ok := cfg.Params["loc"]; !ok {
		cfg.Params["loc"] = "Local"
	}
	if _, ok := cfg.Params["charset"]; !ok {
		cfg.Params["charset"] = "utf8mb4"
	}
	for k, v := range cfg.Params {
		mcfg.Params[k] = v
	}

	return mcfg.FormatDSN()
}

// New 返回初始化好的 *sql.DB，并在需要时执行一次 Ping。
// 使用方应在进程退出时调用 db.Close()。
func New(base Config, options ...Option) (*sql.DB, error) {
	// 叠加函数式选项
	for _, opt := range options {
		opt(&base)
	}

	dsn := BuildDSN(base)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 连接池设置
	if base.MaxOpenConns > 0 {
		db.SetMaxOpenConns(base.MaxOpenConns)
	}
	if base.MaxIdleConns > 0 {
		db.SetMaxIdleConns(base.MaxIdleConns)
	}
	if base.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(base.ConnMaxLifetime)
	}
	if base.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(base.ConnMaxIdleTime)
	}

	// 初始化探活
	if base.PingTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), base.PingTimeout)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return db, nil
}
