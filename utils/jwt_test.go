package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// 业务方自定义 Claims， 需要实现GetRegistered方法
type MyClaims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	VerMd5 string `json:"ver_md5"`
	jwt.RegisteredClaims
}

func (c *MyClaims) GetRegistered() *jwt.RegisteredClaims {
	return &c.RegisteredClaims
}

func TestJWTGenerateToken(t *testing.T) {
	// 初始化 JWT 服务
	j := NewJWT(JWTConfig{
		Secret:     []byte("test_secret"),
		Issuer:     "test-service",
		ExpireTime: 2 * time.Hour,
	})

	// =========================
	// 测试自定义 MyClaims
	// =========================
	claims := &MyClaims{
		UserID: 1001,
		Role:   "admin",
		VerMd5: "123456adfasfsdfsd",
	}
	t.Logf("自定义 MyClaims: %+v", claims)

	tokenStr, err := j.GenerateToken(claims)
	assert.NoError(t, err)
	t.Logf("自定义 MyClaims Generated token (MyClaims): %s", tokenStr)

	var parsed MyClaims
	err = j.ParseToken(tokenStr, &parsed)
	t.Logf("自定义 MyClaims Parsed token (MyClaims): %+v", parsed)
	assert.NoError(t, err)
	assert.Equal(t, claims.UserID, parsed.UserID)
	assert.Equal(t, claims.Role, parsed.Role)
	assert.Equal(t, "test-service", parsed.Issuer) // 自动补
	assert.WithinDuration(t, time.Now(), parsed.IssuedAt.Time, time.Minute)
	assert.WithinDuration(t, time.Now().Add(2*time.Hour), parsed.ExpiresAt.Time, 2*time.Minute)

	// =========================
	// 测试直接传 RegisteredClaims
	// =========================
	customClaims := &jwt.RegisteredClaims{
		Issuer:    "custom-issuer",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	t.Logf("测试直接传 RegisteredClaims: %+v", customClaims)
	tokenStr2, err := j.GenerateToken(customClaims)
	assert.NoError(t, err)
	t.Logf("测试直接传 Generated token (RegisteredClaims): %s", tokenStr2)

	var parsed2 jwt.RegisteredClaims
	err = j.ParseToken(tokenStr2, &parsed2)
	t.Logf("测试直接传 Parsed token (RegisteredClaims): %+v", parsed2)
	assert.NoError(t, err)

	// 直接 RegisteredClaims 不自动补默认值，应该保持原值
	assert.Equal(t, "custom-issuer", parsed2.Issuer)
	assert.WithinDuration(t, time.Now(), parsed2.IssuedAt.Time, time.Minute)
	assert.WithinDuration(t, time.Now().Add(10*time.Minute), parsed2.ExpiresAt.Time, time.Minute)
}
