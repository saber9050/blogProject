package utils

import (
	"crypto/rand"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomString 生成数字字符串
func GenerateRandomString(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

// Contains 检查字符串是否在切片中
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TrimSpace 去除首尾空格
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// IsEmpty 检查字符串是否为空
func IsEmpty(s string) bool {
	return TrimSpace(s) == ""
}

// HashPassword 对密码进行哈希加密
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// UniqueStrings 字符串切片去重
func UniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
