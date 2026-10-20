// Package auth 实现单用户认证：密码生成、bcrypt 校验、JWT 双 token 签发。
package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// passwordAlphabet 初始密码字符集（去掉容易混淆的 0O1lI）
const passwordAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// DefaultPasswordLength 初始密码长度
const DefaultPasswordLength = 16

// ErrPasswordTooShort 密码长度不足
var ErrPasswordTooShort = errors.New("密码长度不足")

// GeneratePassword 生成随机密码。
//
// 使用 crypto/rand 保证不可预测，字符集排除了易混淆字符。
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		length = DefaultPasswordLength
	}
	max := big.NewInt(int64(len(passwordAlphabet)))
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("生成随机密码失败: %w", err)
		}
		out[i] = passwordAlphabet[n.Int64()]
	}
	return string(out), nil
}
