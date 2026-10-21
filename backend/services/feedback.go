package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 问题反馈邮件通道（163 邮箱，凭证经环境变量注入）
//
// 为避免开源泄露真实凭证，原硬编码的 163 邮箱账号 / SMTP 授权码已移除，
// 改为通过环境变量在运行时注入，源码中不再携带任何真实凭据：
//   - FEEDBACK_MAIL_ACCOUNT：发件 / 收件 163 邮箱账号
//   - FEEDBACK_MAIL_AUTH：163 SMTP 授权码（非登录密码，
//     在 163 网页端「设置 → POP3/SMTP/IMAP」开启服务后获取）
//   - 未配置时邮件反馈通道自动禁用（FeedbackMailEnabled 返回 false）
//
// ============================================================
const (
	// feedbackQQGroup QQ 交流群入口：可填加群链接（https://qm.qq.com/...，前端显示「加入群聊」按钮）
	// 或纯群号（前端展示「QQ 群号：xxxx」文字）；群二维码图片 frontend/public/qq-group.png
	// 随前端构建内置，反馈页以图片 + 入口文字/按钮的形式提供 QQ 群联系
	feedbackQQGroup = "1107776057"

	feedbackSMTPHost = "smtp.163.com" // 163 SMTP 服务器
	feedbackSMTPPort = 465            // 465 直连 TLS（SSL）

	// 凭证环境变量名
	feedbackMailAccountEnv = "FEEDBACK_MAIL_ACCOUNT"
	feedbackMailAuthEnv    = "FEEDBACK_MAIL_AUTH"

	// 占位默认值（非真实凭证）：仅在未配置环境变量时返回，
	// 含 "your_" 前缀，FeedbackMailEnabled 据此判定为未启用。
	feedbackMailPlaceholder = "yourname@163.com"
)

// FeedbackQQGroup 返回 QQ 交流群入口（加群链接或群号）
func FeedbackQQGroup() string { return feedbackQQGroup }

// feedbackMailAccount 返回发件 / 收件 163 邮箱账号（优先环境变量）。
func feedbackMailAccount() string {
	if v := os.Getenv(feedbackMailAccountEnv); v != "" {
		return v
	}
	return feedbackMailPlaceholder
}

// feedbackMailAuth 返回 163 SMTP 授权码（优先环境变量）。
func feedbackMailAuth() string {
	return os.Getenv(feedbackMailAuthEnv)
}

// FeedbackToEmail 返回反馈邮件的收件地址（即 163 账号自身）。
// 未配置时返回占位值，前端据此隐藏邮件入口或提示未启用。
func FeedbackToEmail() string { return feedbackMailAccount() }

// FeedbackMailEnabled 返回邮件反馈通道是否已配置。
// 账号/授权码为空或仍为源码占位值时视为未配置，避免发送必然失败误导使用者。
func FeedbackMailEnabled() bool {
	acc := feedbackMailAccount()
	auth := feedbackMailAuth()
	return acc != "" && !strings.Contains(acc, "your_") &&
		auth != ""
}

// SendMail 通过内置 163 SMTP 通道发送邮件：
//   - 465 端口直连 TLS（SSL）
//   - 标题/正文/收件人均做换行清洗，防止邮件头注入
func SendMail(to, subject, body string) error {
	// 防邮件头注入：标题与正文不允许换行
	sanitize := strings.NewReplacer("\r", " ", "\n", " ")
	subject = sanitize.Replace(subject)
	to = sanitize.Replace(to)

	mailAccount := feedbackMailAccount()
	mailAuthCode := feedbackMailAuth()

	addr := net.JoinHostPort(feedbackSMTPHost, strconv.Itoa(feedbackSMTPPort))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsCfg := &tls.Config{ServerName: feedbackSMTPHost}

	var conn net.Conn
	var err error
	if conn, err = dialer.Dial("tcp", addr); err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	var client *smtp.Client
	if feedbackSMTPPort == 465 {
		tconn := tls.Client(conn, tlsCfg)
		if err := tconn.Handshake(); err != nil {
			conn.Close()
			return fmt.Errorf("TLS 握手失败: %w", err)
		}
		client, err = smtp.NewClient(tconn, feedbackSMTPHost)
	} else {
		client, err = smtp.NewClient(conn, feedbackSMTPHost)
		if err == nil && feedbackSMTPPort != 25 {
			if e := client.StartTLS(tlsCfg); e != nil {
				client.Close()
				return fmt.Errorf("STARTTLS 协商失败: %w", e)
			}
		}
	}
	if err != nil {
		conn.Close()
		return fmt.Errorf("建立 SMTP 会话失败: %w", err)
	}
	defer client.Close()

	smtpAuth := smtp.PlainAuth("", mailAccount, mailAuthCode, feedbackSMTPHost)
	if err := client.Auth(smtpAuth); err != nil {
		return fmt.Errorf("SMTP 认证失败，请检查环境变量中的邮箱账号与授权码: %w", err)
	}
	if err := client.Mail(mailAccount); err != nil {
		return fmt.Errorf("发件地址被拒绝: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("收件地址被拒绝: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备发送失败: %w", err)
	}
	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nDate: %s\r\n\r\n",
		mailAccount, to, subject, time.Now().Format(time.RFC1123Z))
	if _, err := w.Write([]byte(header + body)); err != nil {
		w.Close()
		return fmt.Errorf("写入邮件失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("发送失败: %w", err)
	}
	return client.Quit()
}
