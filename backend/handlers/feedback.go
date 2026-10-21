package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"my-books/backend/config"
	"my-books/backend/logger"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"

	"github.com/gin-gonic/gin"
)

// 反馈正文长度上限（字符）
const feedbackContentMax = 2000

// GetFeedbackInfo 获取展示给使用者的反馈方式。
// 163 的 SMTP 配置内置于后端源码，此接口仅返回 QQ 群入口与邮件通道状态，
// 绝不返回任何邮箱明文。
func GetFeedbackInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"qq_group":      services.FeedbackQQGroup(),
		"email_enabled": services.FeedbackMailEnabled(),
	})
}

type sendFeedbackInput struct {
	Contact string `json:"contact"`
	Content string `json:"content"`
}

// SendFeedbackMail 使用者提交反馈：填写反馈内容与联系方式，
// 点击提交后由后端通过内置 163 通道自动发送至开发者邮箱。
func SendFeedbackMail(c *gin.Context) {
	var input sendFeedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "反馈内容不能为空"})
		return
	}
	if utf8.RuneCountInString(content) > feedbackContentMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "反馈内容过长（最多 2000 字）"})
		return
	}
	contact := strings.TrimSpace(input.Contact)
	if contact == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写联系方式（QQ / 邮箱等），方便作者回复你"})
		return
	}
	if utf8.RuneCountInString(contact) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "联系方式过长"})
		return
	}
	if !services.FeedbackMailEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "邮件通道尚未配置，请稍后再试"})
		return
	}

	// 反馈人信息
	nick := "匿名用户"
	if uid := middleware.CurrentUserID(c); uid > 0 {
		var user models.User
		if err := config.DB.First(&user, uid).Error; err == nil {
			nick = user.Nickname
			if nick == "" {
				nick = user.Username
			}
		}
	}
	subject := "【墨阅】用户反馈 - " + nick
	body := fmt.Sprintf("反馈人：%s\n联系方式：%s\n时间：%s\n来源 IP：%s\n\n——————————\n%s\n——————————\n\n此邮件由墨阅应用自动发送，请勿直接回复。",
		nick, contact, time.Now().Format("2006-01-02 15:04:05"), c.ClientIP(), content)

	if err := services.SendMail(services.FeedbackToEmail(), subject, body); err != nil {
		logger.Error("反馈邮件发送失败", "error", err.Error(), "from", nick, "ip", c.ClientIP())
		c.JSON(http.StatusBadGateway, gin.H{"error": "邮件发送失败：" + err.Error()})
		return
	}
	logger.Info("反馈邮件已发送", "from", nick, "ip", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "反馈已发送，感谢你的建议！"})
}
