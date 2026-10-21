package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-books/backend/config"
)

var jwtSecret []byte

// getJWTSecret 获取签名密钥：优先环境变量，其次本地文件（重启后 token 依然有效）
func getJWTSecret() []byte {
	if jwtSecret != nil {
		return jwtSecret
	}
	if env := os.Getenv("JWT_SECRET"); env != "" {
		jwtSecret = []byte(env)
		return jwtSecret
	}
	secretPath := config.JWTSecretPath()
	data, err := os.ReadFile(secretPath)
	if err != nil {
		buf := make([]byte, 32)
		if _, e := rand.Read(buf); e != nil {
			// 极低概率：回退固定密钥，保证可用性
			jwtSecret = []byte("my-books-fallback-secret-2026")
			return jwtSecret
		}
		data = []byte(base64.StdEncoding.EncodeToString(buf))
		_ = os.MkdirAll(filepath.Dir(secretPath), 0755)
		_ = os.WriteFile(secretPath, data, 0600)
	}
	jwtSecret = data
	return jwtSecret
}

// Claims token 载荷
type Claims struct {
	UserID    uint   `json:"uid"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"exp"`
}

// RandomSecret 生成 n 字节的安全随机串（base64url 编码，无 padding）。
// 用于网关自动建号的随机密码等场景。
func RandomSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateToken 生成签名 token（默认 7 天有效期）
func GenerateToken(userID uint, username string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, getJWTSecret())
	mac.Write([]byte(b64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return b64 + "." + sig, nil
}

// ParseToken 解析并校验 token
func ParseToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, getJWTSecret())
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if claims.UserID == 0 || claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}
