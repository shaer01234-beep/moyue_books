package handlers

import (
	"net/http"
	"strings"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type registerInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

type loginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册
func Register(c *gin.Context) {
	var input registerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if len(input.Username) < 2 || len(input.Username) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度需为 2-32 个字符"})
		return
	}
	if len(input.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少 6 位"})
		return
	}

	// 用户名唯一
	var count int64
	config.DB.Model(&models.User{}).Where("username = ?", input.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 是否首个用户（用于接管旧数据）
	var userCount int64
	config.DB.Model(&models.User{}).Count(&userCount)
	isFirstUser := userCount == 0

	nickname := strings.TrimSpace(input.Nickname)
	if nickname == "" {
		nickname = input.Username
	}
	user := models.User{
		Username:     input.Username,
		PasswordHash: string(hash),
		Nickname:     nickname,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	if isFirstUser {
		models.AdoptLegacyData(config.DB, user.ID)
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录凭证生成失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user.PublicUser(),
	})
}

// Login 用户登录
func Login(c *gin.Context) {
	var input loginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)

	var user models.User
	if err := config.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录凭证生成失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user.PublicUser(),
	})
}

// GetMe 获取当前登录用户信息
func GetMe(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var user models.User
	if err := config.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user.PublicUser()})
}

type updateProfileInput struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// UpdateProfile 更新当前用户资料
func UpdateProfile(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var input updateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var user models.User
	if err := config.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	updates := map[string]interface{}{}
	if input.Nickname != "" {
		nickname := strings.TrimSpace(input.Nickname)
		if len(nickname) > 32 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称过长"})
			return
		}
		updates["nickname"] = nickname
	}
	if input.Avatar != "" {
		updates["avatar"] = input.Avatar
	}
	if len(updates) > 0 {
		config.DB.Model(&user).Updates(updates)
	}
	c.JSON(http.StatusOK, gin.H{"user": user.PublicUser()})
}

type changePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改当前用户密码
func ChangePassword(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var input changePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写原密码和新密码"})
		return
	}
	if len(input.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码长度至少 6 位"})
		return
	}
	var user models.User
	if err := config.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "原密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	config.DB.Model(&user).Update("password_hash", string(hash))
	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}
