package rediscluster

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config Redis Cluster 配置
type Config struct {
	Addrs    []string
	Username string
	Password string

	// 连接/池
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int

	// 启动探活
	PingTimeout time.Duration // 0 表示不 Ping
}

// Option 允许对配置进行增量修改
type Option func(*Config)

// WithPoolSize 设置连接池大小
func WithPoolSize(n int) Option { return func(c *Config) { c.PoolSize = n } }

// WithTimeouts 设置读写/拨号超时
func WithTimeouts(dial, read, write time.Duration) Option {
	return func(c *Config) {
		c.DialTimeout, c.ReadTimeout, c.WriteTimeout = dial, read, write
	}
}

// WithPingTimeout 设置初始化 Ping 的超时
func WithPingTimeout(d time.Duration) Option { return func(c *Config) { c.PingTimeout = d } }

// New 初始化并返回 *redis.ClusterClient
func New(base Config, options ...Option) (*redis.ClusterClient, error) {
	for _, opt := range options {
		opt(&base)
	}
	cli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        base.Addrs,
		Username:     base.Username,
		Password:     base.Password,
		DialTimeout:  base.DialTimeout,
		ReadTimeout:  base.ReadTimeout,
		WriteTimeout: base.WriteTimeout,
		PoolSize:     base.PoolSize,
		MinIdleConns: base.MinIdleConns,
	})

	if base.PingTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), base.PingTimeout)
		defer cancel()
		if err := cli.Ping(ctx).Err(); err != nil {
			_ = cli.Close()
			return nil, err
		}
	}
	return cli, nil
}
