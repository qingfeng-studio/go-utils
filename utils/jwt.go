package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config 定义 JWT 全局配置
type JWTConfig struct {
	Secret     []byte        // 签名秘钥
	Issuer     string        // 签发者
	ExpireTime time.Duration // 默认过期时间（如 2 * time.Hour）
}

// JWTService 封装 jwt 操作
type JWTService struct {
	cfg JWTConfig
}

// NewJwt 创建实例
func NewJWT(cfg JWTConfig) *JWTService {
	return &JWTService{cfg: cfg}
}

// GenerateToken 生成 token
func (j *JWTService) GenerateToken(payload jwt.Claims) (string, error) {
	switch claims := payload.(type) {
	case *jwt.RegisteredClaims:
		// 如果业务方直接传 RegisteredClaims，就认为完全由他控制，不补默认值
	case interface{ GetRegistered() *jwt.RegisteredClaims }:
		// 嵌套结构体（例如 MyClaims），则只补未设置的字段
		j.fillMissingDefaults(claims.GetRegistered())
	default:
		// 无法识别类型，不处理默认字段
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(j.cfg.Secret)
}

// ParseToken 验证 token
func (j *JWTService) ParseToken(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return j.cfg.Secret, nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("invalid token")
	}
	return nil
}

// 只补未设置的字段（区别于“全覆盖”）
func (j *JWTService) fillMissingDefaults(c *jwt.RegisteredClaims) {
	now := time.Now()
	if c.Issuer == "" && j.cfg.Issuer != "" {
		c.Issuer = j.cfg.Issuer
	}
	if c.IssuedAt == nil {
		c.IssuedAt = jwt.NewNumericDate(now)
	}
	if c.ExpiresAt == nil && j.cfg.ExpireTime > 0 {
		c.ExpiresAt = jwt.NewNumericDate(now.Add(j.cfg.ExpireTime))
	}
}
