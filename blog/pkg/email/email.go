package email

import (
	"blog/pkg/config"
	"blog/pkg/logger"
	"fmt"
	"net/url"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// SendCaptchaEmail 发送验证码邮件
func SendCaptchaEmail(to string, captcha string, purpose string) error {
	cfg := config.Get().Email

	// 构建邮件内容
	var body string
	if purpose == "login" {
		body = fmt.Sprintf("您的登录验证码是：%s，有效期5分钟。请勿将验证码泄露给他人。", captcha)
	} else if purpose == "reset_password" {
		body = fmt.Sprintf("您的密码重置验证码是：%s，有效期5分钟。请勿将验证码泄露给他人。", captcha)
	} else {
		body = fmt.Sprintf("您的验证码是：%s，有效期5分钟。请勿将验证码泄露给他人。", captcha)
	}

	// 创建邮件消息
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "【个人博客】验证码通知")
	m.SetBody("text/plain; charset=UTF-8", body)

	// 创建邮件发送器
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	// 如果使用 SSL 端口（465），启用 SSL
	if cfg.Port == 465 {
		d.SSL = true
	}

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		logger.Error("发送邮件失败", zap.Error(err), zap.String("to", to))
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	logger.Info("验证码邮件发送成功", zap.String("to", to), zap.String("purpose", purpose))
	return nil
}

// SendVerificationEmail 发送确认消息到邮箱
func SendVerificationEmail(toEmail, token string) error {
	cfg := config.Get().Email
	//	构建邮件内容
	app2 := config.Get().App
	baseURL := fmt.Sprintf("http://%s:%d", app2.Host, app2.Port)
	// URL 参数编码
	escapedToken := url.QueryEscape(token)

	confirmLink := fmt.Sprintf("%s/api/v1/user/email?token=%s",
		baseURL, escapedToken)

	htmlBody := fmt.Sprintf(`
        <h2>邮箱修改确认</h2>
        <p>您好：</p>
        <p>您正在将账户的绑定邮箱修改为 <strong>%s</strong>。<br>
        请点击下方链接完成确认，该链接 <strong>5分钟内</strong> 有效：</p>
        <p><a href="%s">点击确认</a></p>
        <p>如果不是您本人操作，请忽略此邮件。</p>
        <br>
        <p>核心力量团队</p>
    `, toEmail, confirmLink)

	plainBody := fmt.Sprintf(
		"您好：\n\n您正在将账户的绑定邮箱修改为 %s。\n请点击完成确认（5分钟内有效）：\n%s\n\n如果不是您本人操作，请忽略此邮件。\n\n核心力量团队",
		toEmail, confirmLink,
	)

	subject := "【个人博客】邮箱修改确认"

	m := gomail.NewMessage()
	// 发件人带显示名称
	m.SetHeader("From", m.FormatAddress(cfg.From, "个人博客"))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)
	// 先设置纯文本正文，再添加 HTML 作为富文本替代
	m.SetBody("text/plain; charset=UTF-8", plainBody)
	m.AddAlternative("text/html; charset=UTF-8", htmlBody)

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if cfg.Port == 465 {
		d.SSL = true
	}
	if err := d.DialAndSend(m); err != nil {
		logger.Error("发送邮件失败", zap.Error(err), zap.String("to", toEmail))
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	logger.Info("邮件发送成功", zap.String("to", toEmail), zap.String("subject", subject))
	return nil
}
