package services

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/difyz9/edge-tts-go/pkg/communicate"
	"github.com/difyz9/edge-tts-go/pkg/types"
	"github.com/difyz9/edge-tts-go/pkg/voices"
)

const (
	// EdgeTTSDefaultVoice 默认中文女声（晓晓），Microsoft Edge 朗读在线音色
	// （长格式名，仅作兼容保留，Go 直连合成请使用 EdgeTTSDefaultVoiceShort）
	EdgeTTSDefaultVoice = "Microsoft Server Speech Text to Speech Voice (zh-CN, XiaoxiaoNeural)"
	// EdgeTTSDefaultVoiceShort 默认音色的 short name（用于设置项存储、前端下拉与 Go 直连合成）
	EdgeTTSDefaultVoiceShort = "zh-CN-XiaoxiaoNeural"
)

// EdgeTTSProxy 返回 edge-tts 专用代理地址：优先 EDGE_TTS_PROXY，回退通用
// HTTPS_PROXY / https_proxy。受限网络（国内/部分云主机）下微软端点会拒绝握手（403），
// 经代理出网可绕过；未配置时返回空（edge-tts 直连）。
// difyz9/edge-tts-go 在收到 403 时会依据服务端 Date 头自动校正本地时钟偏移并重试，
// 可规避因系统时钟偏差导致的 Sec-MS-GEC 令牌被拒（这正是旧实现偶发 403 的根因）。
func EdgeTTSProxy() string {
	for _, k := range []string{"EDGE_TTS_PROXY", "HTTPS_PROXY", "https_proxy"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// EdgeTTSVoice 微软 Edge TTS 可用音色（下发给前端展示）
type EdgeTTSVoice struct {
	ShortName    string `json:"short_name"`
	FriendlyName string `json:"friendly_name"`
	Gender       string `json:"gender"`
	Locale       string `json:"locale"`
}

// difyz9/edge-tts-go v0.0.3 的 voices.ListVoices 返回的 FriendlyName 为空，
// 需要基于 ShortName/Name 自行生成可读名称，否则前端下拉选项会显示空白。
var edgeTTSLocaleNames = map[string]string{
	"zh-CN":          "普通话",
	"zh-CN-liaoning": "东北话",
	"zh-CN-shaanxi":  "陕西话",
	"zh-CN-sichuan":  "四川话",
	"zh-CN-henan":    "河南话",
}

// normalizeEdgeTTSVoice 把旧版长格式音色名（如 Microsoft Server Speech Text to
// Speech Voice (zh-CN, XiaoxiaoNeural)）转成 difyz9 需要的 short name；已是 short
// name 则原样返回。用于兼容旧设置/任务中保存的长格式默认值。
func normalizeEdgeTTSVoice(voice string) string {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return EdgeTTSDefaultVoiceShort
	}
	prefix := "Microsoft Server Speech Text to Speech Voice ("
	if strings.HasPrefix(voice, prefix) && strings.HasSuffix(voice, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(voice, prefix), ")")
		parts := strings.Split(inner, ", ")
		if len(parts) == 2 {
			return fmt.Sprintf("%s-%s", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	return voice
}

// edgeTTSFriendlyName 为音色生成友好显示名（优先使用接口返回的 FriendlyName）。
func edgeTTSFriendlyName(v types.Voice) string {
	if v.FriendlyName != "" {
		return v.FriendlyName
	}
	// ShortName 形如 zh-CN-XiaoxiaoNeural 或 zh-CN-liaoning-XiaobeiNeural
	parts := strings.Split(v.ShortName, "-")
	if len(parts) >= 2 {
		voiceName := parts[len(parts)-1]
		locale := strings.Join(parts[:len(parts)-1], "-")
		localeName := locale
		if name, ok := edgeTTSLocaleNames[locale]; ok {
			localeName = name
		}
		return fmt.Sprintf("%s（%s）", voiceName, localeName)
	}
	if v.Name != "" {
		return v.Name
	}
	return v.ShortName
}

var (
	edgeTTSCache     []EdgeTTSVoice
	edgeTTSCacheTime time.Time
)

// ListEdgeTTSVoices 返回中文音色列表（带 1 小时缓存）。
// 失败时返回 nil，由调用方静默降级（前端显示空列表）。
func ListEdgeTTSVoices() []EdgeTTSVoice {
	if len(edgeTTSCache) > 0 && time.Since(edgeTTSCacheTime) < time.Hour {
		return edgeTTSCache
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	vlist, err := voices.ListVoices(ctx, EdgeTTSProxy())
	if err != nil {
		return nil
	}
	list := make([]EdgeTTSVoice, 0, len(vlist))
	for _, v := range vlist {
		// 只保留简体中文（含东北/四川/陕西/河南等方言音色）
		if !strings.HasPrefix(v.Locale, "zh-CN") {
			continue
		}
		list = append(list, EdgeTTSVoice{
			ShortName:    v.ShortName,
			FriendlyName: edgeTTSFriendlyName(v),
			Gender:       v.Gender,
			Locale:       v.Locale,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].FriendlyName < list[j].FriendlyName })
	edgeTTSCache = list
	edgeTTSCacheTime = time.Now()
	return list
}

// EdgeTTSSpeedToRate 语速倍率（0.5~2.0）转 edge-tts rate 参数（+20% / -30%）
func EdgeTTSSpeedToRate(speed float64) string {
	if speed <= 0 {
		speed = 1.0
	}
	pct := int((speed - 1.0) * 100)
	if pct >= 0 {
		return fmt.Sprintf("+%d%%", pct)
	}
	return fmt.Sprintf("%d%%", pct)
}

// EdgeTTSEmotion 情感对应的音色近似参数。
// Edge TTS 无真实情绪音色，只能通过语速 + 音量 + 音调组合近似模拟。
type EdgeTTSEmotion struct {
	SpeedFactor float64 // 相对用户语速的倍率（>1 更快，<1 更慢）
	Volume      string  // edge-tts volume 参数，如 "+5%"
	Pitch       string  // edge-tts pitch 参数，如 "+12Hz"
}

// EdgeTTSEmotionParams 返回情感对应的 Edge TTS 参数；未知情感按「平静」处理
func EdgeTTSEmotionParams(emotion string) EdgeTTSEmotion {
	switch emotion {
	case "愉悦":
		return EdgeTTSEmotion{1.10, "+5%", "+8Hz"} // 轻快、明亮
	case "悲伤":
		return EdgeTTSEmotion{0.90, "-5%", "-8Hz"} // 低沉、缓慢
	case "激昂":
		return EdgeTTSEmotion{1.15, "+8%", "+12Hz"} // 高亢、有力
	case "温柔":
		return EdgeTTSEmotion{0.95, "-3%", "+2Hz"} // 柔和
	case "自信":
		return EdgeTTSEmotion{1.05, "+3%", "+5Hz"}
	case "庄重":
		return EdgeTTSEmotion{0.90, "-3%", "-10Hz"} // 沉稳
	default: // 平静 / 未知
		return EdgeTTSEmotion{1.00, "+0%", "+0Hz"}
	}
}

// SynthesizeEdgeTTS 使用微软 Edge TTS 在线合成语音（Go 直连 difyz9/edge-tts-go，
// 无需本地 Python tts-service），返回 MP3 音频字节。voice 为空时使用默认中文女声
// short name（zh-CN-XiaoxiaoNeural）。
// 受限网络下微软端点可能 403：设置 EDGE_TTS_PROXY（或 HTTPS_PROXY）后自动经代理出网；
// 该库在收到 403 时会依据服务端 Date 头自动校正本地时钟偏移并重试，规避时钟偏差导致的
// Sec-MS-GEC 令牌被拒（旧实现偶发 403/bad handshake 的根因）。
// 文本内部的 XML 特殊字符（& < > 等）由库内部转义，并对超长文本按字节自动切分。
func SynthesizeEdgeTTS(text, voice, rate, volume, pitch string) ([]byte, error) {
	voice = normalizeEdgeTTSVoice(voice)
	if rate == "" {
		rate = "+0%"
	}
	if volume == "" {
		volume = "+0%"
	}
	if pitch == "" {
		pitch = "+0Hz"
	}

	comm, err := communicate.NewCommunicate(
		text, voice, rate, volume, pitch,
		EdgeTTSProxy(), 15, 120,
	)
	if err != nil {
		return nil, fmt.Errorf("初始化 Edge TTS 失败：%v", err)
	}

	// 设置整体超时，避免网络静默丢包时 websocket 握手/读取无限挂起（difyz9 内部
	// HandshakeTimeout 为 0 不兜底）；与既有 HTTP 转发 180s 超时保持一致。
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	chunkChan, errChan := comm.Stream(ctx)
	var audioBuf []byte
	for chunk := range chunkChan {
		if chunk.Type == "audio" {
			audioBuf = append(audioBuf, chunk.Data...)
		}
	}
	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("Edge TTS 合成失败：%v", err)
	}
	if len(audioBuf) == 0 {
		return nil, fmt.Errorf("Edge TTS 未返回音频数据")
	}
	return audioBuf, nil
}
