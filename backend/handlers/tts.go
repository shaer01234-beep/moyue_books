package handlers

import (
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// GetTTSSettings 获取语音朗读可选项（引擎/情感/音色）。Go 后端直连 Edge TTS，无需用户配置。
func GetTTSSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"config":          services.DefaultTTSConfig(),
			"emotions":        services.EmotionList(),
			"engines":         services.TTSEngines(),
			"available_langs": []string{"zh", "en", "ja", "ko", "yue", "auto"},
			"voices":          services.ListEdgeTTSVoices(),
		},
	})
}

// SynthesizeTTSHandler 语音合成代理：按引擎分发到对应服务。
// engine / reference_id 为可选覆盖参数（未保存配置时用于测试）。
func SynthesizeTTSHandler(c *gin.Context) {
	var req struct {
		Text        string  `json:"text"`
		Emotion     string  `json:"emotion"`
		SpeedFactor float64 `json:"speed_factor"`
		TextLang    string  `json:"text_lang"`
		MediaType   string  `json:"media_type"`
		Engine      string  `json:"engine"`
		ReferenceID string  `json:"reference_id"`
		Voice       string  `json:"voice"` // Edge TTS 音色 short name（可覆盖保存配置）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	cfg := services.GetTTSConfig(middleware.CurrentUserID(c))
	if req.Engine != "" {
		cfg.Engine = req.Engine
	}
	if req.ReferenceID != "" {
		cfg.ReferenceID = req.ReferenceID
	}
	if req.Voice != "" {
		cfg.Voice = req.Voice
	}

	data, contentType, err := services.SynthesizeWithConfig(
		cfg, req.Text, req.Emotion, req.SpeedFactor, req.TextLang, req.MediaType,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

var (
	// Go 的 regexp（RE2）不支持反向引用，故拆成两个正则：
	// 先整体剥离 <script>/<style> 内容块，再剥离剩余标签。
	htmlBlockRe = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(?:script|style)>`)
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
	htmlSpaceRe = regexp.MustCompile(`[ \t\r\f\v]+`)
	htmlEmptyRe = regexp.MustCompile(`\n{3,}`)
)

// GetBookChapterText 获取 EPUB 章节纯文本（用于语音朗读）
// GET /api/books/:id/text?chapter=N
func GetBookChapterText(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	if book.FileType != "epub" && book.FileType != "kepub" && book.FileType != "azw3" && book.FileType != "mobi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only epub/azw3/mobi chapter text is supported"})
		return
	}

	reader, err := openBookReader(utils.ResolvePath(book.FilePath), book.FileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open book"})
		return
	}
	defer reader.Close()

	chapter := 0
	if cs := c.Query("chapter"); cs != "" {
		chapter, _ = strconv.Atoi(cs)
	}

	content, err := reader.GetContent(chapter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 章节标题：取 <title> 标签
	title := ""
	if m := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`).FindSubmatch(content); len(m) > 1 {
		title = strings.TrimSpace(html.UnescapeString(string(m[1])))
	}

	// 剥离 script/style 与所有标签，转纯文本
	text := string(content)
	text = htmlBlockRe.ReplaceAllString(text, "\n")
	text = htmlTagRe.ReplaceAllString(text, "\n")
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = htmlSpaceRe.ReplaceAllString(text, " ")
	// 按行合并，保留段落感
	lines := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "。") &&
			!strings.HasSuffix(lines[len(lines)-1], "！") &&
			!strings.HasSuffix(lines[len(lines)-1], "？") {
			lines[len(lines)-1] += line
		} else {
			lines = append(lines, line)
		}
	}
	text = strings.Join(lines, "\n")
	text = htmlEmptyRe.ReplaceAllString(text, "\n")

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"chapter": chapter,
			"title":   title,
			"text":    text,
		},
	})
}
