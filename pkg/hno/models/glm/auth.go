package glm

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// parseAPIKey parses the GLM API key into keyID and keySecret
// Format: {key_id}.{key_secret}
// parseAPIKey 解析 GLM API 密钥为 keyID 和 keySecret
// 格式: {key_id}.{key_secret}
func parseAPIKey(apiKey string) (string, string, error) {
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("API key must be in format {key_id}.{key_secret}, got %d parts", len(parts))
	}

	keyID := parts[0]
	keySecret := parts[1]

	if keyID == "" || keySecret == "" {
		return "", "", fmt.Errorf("API key parts cannot be empty")
	}

	return keyID, keySecret, nil
}

// generateJWT creates a JWT token for GLM API authentication
// The token is valid for 7 days and uses HMAC-SHA256 signing
// generateJWT 为 GLM API 认证创建 JWT 令牌
// 令牌有效期为 7 天，使用 HMAC-SHA256 签名
func generateJWT(keyID, keySecret string) (string, error) {
	// Calculate expiration time (7 days from now)
	// 计算过期时间（从现在起 7 天）
	timestamp := time.Now().UnixMilli()
	exp := timestamp + (7 * 24 * 60 * 60 * 1000) // 7 days in milliseconds

	// Create claims
	// 创建声明
	claims := jwt.MapClaims{
		"api_key":   keyID,
		"timestamp": timestamp,
		"exp":       exp,
	}

	// Create token with custom header
	// 创建带有自定义头的令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["alg"] = "HS256"
	token.Header["sign_type"] = "SIGN"

	// Sign the token with the key secret
	// 使用密钥签名令牌
	signedToken, err := token.SignedString([]byte(keySecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return signedToken, nil
}
