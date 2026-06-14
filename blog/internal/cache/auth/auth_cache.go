package auth

import (
	"blog/internal/constant"
	"blog/pkg/database"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

// loginCache 登录验证缓存
type loginCache struct {
	redisClient *redis.Client
}

// NewLoginCache 新建登录缓存
func NewLoginCache() AuthCache {
	return &loginCache{
		redisClient: database.GetRedis(),
	}
}

// GetCaptchaKey 获取图形验证码的缓存键
func (c *loginCache) GetCaptchaKey(captchaKey string) string {
	return fmt.Sprintf("captcha:login:%s", captchaKey)
}

// GetEmailCaptchaKey 获取邮箱验证码的缓存键
// purpose: login登录 reset_password重设密码 reset_email重设邮箱
func (c *loginCache) GetEmailCaptchaKey(email, purpose string) string {
	return fmt.Sprintf("captcha:email:%s:%s", email, purpose)
}

// GetEmailSendLimitKey 获取邮箱发送频率限制的缓存键
func (c *loginCache) GetEmailSendLimitKey(email string) string {
	return fmt.Sprintf("captcha:limit:%s", email)
}

// GetBlacklistKey 获取token黑名单的缓存键
func (c *loginCache) GetBlacklistKey(token string) string {
	return fmt.Sprintf("blacklist:token:%s", token)
}

// StoreCaptcha 存储图形验证码
func (c *loginCache) StoreCaptcha(captchaKey, captchaCode string, expireSecond int64) error {
	ctx := context.Background()
	key := c.GetCaptchaKey(captchaKey)

	return c.redisClient.Set(ctx, key, captchaCode, time.Duration(expireSecond)*time.Second).Err()
}

// GetCaptcha 获取图形验证码
func (c *loginCache) GetCaptcha(captchaKey string) (string, error) {
	ctx := context.Background()
	key := c.GetCaptchaKey(captchaKey)

	code, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return code, nil
}

// DeleteCaptcha 删除图形验证码
func (c *loginCache) DeleteCaptcha(captchaKey string) error {
	ctx := context.Background()
	key := c.GetCaptchaKey(captchaKey)

	return c.redisClient.Del(ctx, key).Err()
}

// StoreEmailCaptcha 存储邮箱验证码
func (c *loginCache) StoreEmailCaptcha(email string, captcha string, purpose string, expireSeconds int64) error {
	ctx := context.Background()
	key := c.GetEmailCaptchaKey(email, purpose)

	return c.redisClient.Set(ctx, key, captcha, time.Duration(expireSeconds)*time.Second).Err()
}

// GetEmailCaptcha 获取邮箱验证码
func (c *loginCache) GetEmailCaptcha(email string, purpose string) (string, error) {
	ctx := context.Background()
	key := c.GetEmailCaptchaKey(email, purpose)

	code, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return code, nil
}

// DeleteEmailCaptcha 删除邮箱验证码
func (c *loginCache) DeleteEmailCaptcha(email string, purpose string) error {
	ctx := context.Background()
	key := c.GetEmailCaptchaKey(email, purpose)

	return c.redisClient.Del(ctx, key).Err()
}

// CheckEmailSendLimit 检查邮箱发送频率限制（60秒）
func (c *loginCache) CheckEmailSendLimit(email string) (bool, error) {
	ctx := context.Background()
	key := c.GetEmailSendLimitKey(email)

	ok, err := c.redisClient.SetNX(ctx, key, 1, time.Duration(constant.CaptchaSendLimit)*time.Second).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// DeleteEmailSendLimit 删除邮箱发送频率限制
func (c *loginCache) DeleteEmailSendLimit(email string) error {
	ctx := context.Background()
	key := c.GetEmailSendLimitKey(email)

	return c.redisClient.Del(ctx, key).Err()
}

// BlacklistToken 将 JWT TOKEN 加入黑名单
// expireSecond 该token剩余过期时间,单位秒
func (c *loginCache) BlacklistToken(token string, expireSecond int64) error {
	ctx := context.Background()
	key := c.GetBlacklistKey(token)

	return c.redisClient.Set(ctx, key, 1, time.Duration(expireSecond)*time.Second).Err()
}

// CheckBlacklist 检查 JWT TOKEN 是否在黑名单中
func (c *loginCache) CheckBlacklist(token string) (bool, error) {
	ctx := context.Background()
	key := c.GetBlacklistKey(token)

	_, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return true, nil
}
