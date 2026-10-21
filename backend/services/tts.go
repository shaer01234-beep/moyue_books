package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/models"
)

// TTS 配置键（存储在 Setting 表中）
const (
	TTSSettingBaseURL        = "tts.base_url"
	TTSSettingRefAudioPath   = "tts.ref_audio_path"
	TTSSettingPromptText     = "tts.prompt_text"
	TTSSettingPromptLang     = "tts.prompt_lang"
	TTSSettingTextLang       = "tts.text_lang"
	TTSSettingSpeedFactor    = "tts.speed_factor"
	TTSSettingEmotionEnabled = "tts.emotion_enabled"
	TTSSettingEngine         = "tts.engine"
	TTSSettingReferenceID    = "tts.reference_id"
	TTSSettingVoice          = "tts.voice" // Edge TTS 音色 short name
)

// 支持的语音引擎
const (
	EngineEdgeTTS    = "edge-tts"
	EngineGPTSovits  = "gpt-sovits"
	EngineFishSpeech = "fish-speech"
	EngineChatTTS    = "chattts"
	EngineAuto = "auto"
)

// EmotionPresets 情感预设：key -> 注入到合成文本前的情感描述
// GPT-SoVITS 通过对文本中括号情感标注的语义感知来实现情感控制
var EmotionPresets = map[string]string{
	"平静": "（平静地）",
	"愉悦": "（开心地）",
	"悲伤": "（悲伤地）",
	"激昂": "（激昂地）",
	"温柔": "（温柔地）",
	"自信": "（自信地）",
	"庄重": "（庄重地）",
}

// EmotionList 提供给前端的可选情感列表
func EmotionList() []string {
	list := make([]string, 0, len(EmotionPresets))
	for k := range EmotionPresets {
		list = append(list, k)
	}
	return list
}

// TTSConfig 语音引擎连接配置
type TTSConfig struct {
	BaseURL        string  `json:"base_url"`
	RefAudioPath   string  `json:"ref_audio_path"`
	PromptText     string  `json:"prompt_text"`
	PromptLang     string  `json:"prompt_lang"`
	TextLang       string  `json:"text_lang"`
	SpeedFactor    float64 `json:"speed_factor"`
	EmotionEnabled bool    `json:"emotion_enabled"`
	// Engine 语音工具：""/"auto"=自动、edge-tts、gpt-sovits、fish-speech、chattts
	Engine      string `json:"engine"`
	ReferenceID string `json:"reference_id"` // fish-speech 预置音色 ID / ChatTTS 固定音色种子
	// Voice Edge TTS 音色 short name（如 zh-CN-XiaoxiaoNeural）
	Voice string `json:"voice"`
}

// TTSEngines 提供给前端的可选语音引擎列表
func TTSEngines() []map[string]interface{} {
	return []map[string]interface{}{
		{"key": EngineEdgeTTS, "label": "Edge TTS（在线，免费）", "desc": "微软在线语音，Go 直连无需本地服务；受限网络请设 EDGE_TTS_PROXY/HTTPS_PROXY 代理出网。情感/语速为近似模拟。", "available": true, "default": true},
		// {"key": EngineGPTSovits, "label": "GPT-SoVITS（声音克隆）", "desc": "官方 api_v2 服务，参考音频克隆音色，需本地部署模型。", "available": true, "default": false},
		// {"key": EngineFishSpeech, "label": "Fish-Speech（声音克隆）", "desc": "官方 API server，支持预置音色 reference_id 或参考音频克隆。", "available": true, "default": false},
		// {"key": EngineChatTTS, "label": "ChatTTS（本地模型）", "desc": "本地 ChatTTS 模型合成，支持参考音色、固定音色种子与语速控制。需安装 ChatTTS 并下载模型。", "available": true, "default": false},
	}
}

// 读取单个设置项（按用户隔离）
func getSetting(uid uint, key, def string) string {
	var s models.Setting
	err := config.DB.Where("user_id = ? AND key = ?", uid, key).First(&s).Error
	if err != nil || s.Value == "" {
		return def
	}
	return s.Value
}

// GetTTSConfig 读取语音引擎配置
func GetTTSConfig(uid uint) TTSConfig {
	speed, _ := strconv.ParseFloat(getSetting(uid, TTSSettingSpeedFactor, "1.0"), 64)
	if speed <= 0 {
		speed = 1.0
	}
	emotionEnabled, _ := strconv.ParseBool(getSetting(uid, TTSSettingEmotionEnabled, "true"))
	return TTSConfig{
		BaseURL:        getSetting(uid, TTSSettingBaseURL, ""),
		RefAudioPath:   getSetting(uid, TTSSettingRefAudioPath, ""),
		PromptText:     getSetting(uid, TTSSettingPromptText, ""),
		PromptLang:     getSetting(uid, TTSSettingPromptLang, "zh"),
		TextLang:       getSetting(uid, TTSSettingTextLang, "zh"),
		SpeedFactor:    speed,
		EmotionEnabled: emotionEnabled,
		Engine:         getSetting(uid, TTSSettingEngine, EngineEdgeTTS),
		ReferenceID:    getSetting(uid, TTSSettingReferenceID, ""),
		Voice:          getSetting(uid, TTSSettingVoice, EdgeTTSDefaultVoiceShort),
	}
}

// DefaultTTSConfig 返回内置默认配置：Go 直连 Edge TTS，无需用户配置语音服务地址。
func DefaultTTSConfig() TTSConfig {
	return TTSConfig{
		SpeedFactor:    1.0,
		EmotionEnabled: true,
		Engine:         EngineEdgeTTS,
		Voice:          EdgeTTSDefaultVoiceShort,
	}
}

// NormalizeEngine 归一化引擎名，未设置时返回空（保持旧行为）
func NormalizeEngine(e string) string {
	switch e {
	case EngineEdgeTTS, EngineGPTSovits, EngineFishSpeech, EngineChatTTS, EngineAuto:
		return e
	case "edge", "edge_tts":
		return EngineEdgeTTS
	case "gpt", "gpt_sovits":
		return EngineGPTSovits
	case "fish", "fish_speech":
		return EngineFishSpeech
	case "chat", "chat_tts":
		return EngineChatTTS
	default:
		return ""
	}
}

// SynthesizeTTS 按已保存配置的语音引擎分发合成请求，返回音频字节与 Content-Type。
func SynthesizeTTS(uid uint, text, emotion string, speedFactor float64, textLang, mediaType string) ([]byte, string, error) {
	return SynthesizeWithConfig(GetTTSConfig(uid), text, emotion, speedFactor, textLang, mediaType)
}

// EngineCandidates 返回合成引擎候选链（按优先级降序排列）。
// engine=auto 时：edge-tts 优先，其后为已配置服务地址（base_url）的本地引擎兜底
// （gpt-sovits / fish-speech / chattts，同一地址按序尝试，后端未启用则快速失败跳过）；
// 指定单一引擎时只返回该引擎；未设置时按 edge-tts。
func EngineCandidates(cfg TTSConfig) []string {
	engine := NormalizeEngine(cfg.Engine)
	if engine == "" {
		engine = EngineEdgeTTS
	}
	if engine != EngineAuto {
		return []string{engine}
	}
	list := []string{EngineEdgeTTS}
	if cfg.BaseURL != "" {
		list = append(list, EngineGPTSovits, EngineFishSpeech, EngineChatTTS)
	}
	return list
}

// ttsFailureHint 合成失败时附带的排障指引。
// 命中微软区域限制特征（403 / 握手失败）时给出代理与本地引擎两种解法，
// 其余错误（如未配置服务地址）由调用方按自身信息处理，不追加。
func ttsFailureHint(errs ...error) string {
	const hint = "（微软 Edge TTS 在当前网络不可用：可设置 EDGE_TTS_PROXY/HTTPS_PROXY 代理，或配置本地语音引擎 GPT-SoVITS / Fish-Speech / ChatTTS）"
	for _, err := range errs {
		if err == nil {
			continue
		}
		msg := err.Error()
		if strings.Contains(msg, "403") || strings.Contains(msg, "bad handshake") ||
			strings.Contains(msg, "Invalid response status") {
			return hint
		}
	}
	return ""
}

// SynthesizeWithConfig 按给定配置分发合成请求（handler 可用请求参数覆盖 engine/reference_id）。
// engine=auto 时按 EngineCandidates 依次尝试：edge-tts → 已配置的本地引擎，全部失败才报错；
// edge-tts 优先转发项目内置 tts-service（Python edge-tts），服务不可达、或配置了
// EDGE_TTS_PROXY 代理时回退 Go 内置直连（同样走代理）。
func SynthesizeWithConfig(cfg TTSConfig, text, emotion string, speedFactor float64, textLang, mediaType string) ([]byte, string, error) {
	candidates := EngineCandidates(cfg)
	var lastErr, firstErr error
	for i, engine := range candidates {
		data, contentType, err := synthesizeWithEngine(cfg, engine, text, emotion, speedFactor, textLang, mediaType)
		if err == nil {
			return data, contentType, nil
		}
		if firstErr == nil {
			firstErr = err
		}
		lastErr = err
		if i < len(candidates)-1 {
			log.Printf("tts: engine %s 合成失败，回退下一个引擎 %s：%v", engine, candidates[i+1], err)
		}
	}
	return nil, "", fmt.Errorf("%v%v", lastErr, ttsFailureHint(firstErr, lastErr))
}

// synthesizeWithEngine 使用单一指定引擎合成（不含 auto 链，链式回退由 SynthesizeWithConfig 负责）
func synthesizeWithEngine(cfg TTSConfig, engine, text, emotion string, speedFactor float64, textLang, mediaType string) ([]byte, string, error) {
	// edge-tts 由 Go/Python 直连微软在线合成，无需本地语音服务地址；其他引擎需要 base_url
	if engine != EngineEdgeTTS && cfg.BaseURL == "" {
		return nil, "", fmt.Errorf("语音引擎 %s 未配置服务地址", engine)
	}

	// 情感控制：将情感描述注入合成文本（本地引擎对括号情感标注有语义感知）。
	// edge-tts 会把「（平静地）」等括号内容原样朗读出来，因此不注入。
	finalText := text
	if cfg.EmotionEnabled && emotion != "" && engine != EngineEdgeTTS {
		if preset, ok := EmotionPresets[emotion]; ok {
			finalText = preset + text
		} else {
			finalText = "（" + emotion + "）" + text
		}
	}

	if textLang == "" {
		textLang = cfg.TextLang
	}
	if speedFactor <= 0 {
		speedFactor = cfg.SpeedFactor
	}
	if speedFactor <= 0 {
		speedFactor = 1.0
	}
	if mediaType == "" {
		mediaType = "wav"
	}

	switch engine {
	case EngineFishSpeech:
		return synthesizeFishSpeech(cfg, finalText, textLang, mediaType, speedFactor)
	case EngineChatTTS:
		return synthesizeLegacy(cfg, finalText, textLang, mediaType, speedFactor, EngineChatTTS)
	case EngineGPTSovits:
		return synthesizeLegacy(cfg, finalText, textLang, mediaType, speedFactor, "")
	case EngineEdgeTTS:
		// 优先使用 Go 直连 difyz9/edge-tts-go：该库在收到 403 时会依据服务端
		// Date 头自动校正本地时钟偏移并重试，规避旧 Python 中转偶发的 403/bad
		// handshake；且无需本地 tts-service、自带 XML 转义与按字节切分。
		edgeEmotion := ""
		if cfg.EmotionEnabled {
			edgeEmotion = emotion
		}
		data, contentType, err := synthesizeEdgeTTSDirect(text, speedFactor, edgeEmotion, cfg.Voice)
		if err == nil {
			return data, contentType, nil
		}
		// Go 直连失败：仅当显式配置了 tts-service（base_url 或 TTS_SERVICE_URL）
		// 时才回退到该 Python 服务，兼容仍部署旧服务的环境；否则直接返回 Go 错误。
		if cfg.BaseURL == "" && strings.TrimSpace(os.Getenv("TTS_SERVICE_URL")) == "" {
			return nil, "", err
		}
		log.Printf("tts: Go 直连 Edge TTS(difyz9) 失败，回退 tts-service：%v", err)
		d2, ct2, _, e2 := synthesizeEdgeTTSService(cfg, text, emotion, speedFactor, textLang, mediaType)
		if e2 == nil {
			return d2, ct2, nil
		}
		return nil, "", err
	default: // 未知/异常引擎：回退转发 /tts
		return synthesizeLegacy(cfg, finalText, textLang, mediaType, speedFactor, "")
	}
}

// edgeTTSServiceURL edge-tts 引擎转发的 tts-service 地址：
// 用户配置的 base_url 优先，其次环境变量 TTS_SERVICE_URL，默认本机 9881。
func edgeTTSServiceURL(cfg TTSConfig) string {
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	if v := strings.TrimSpace(os.Getenv("TTS_SERVICE_URL")); v != "" {
		return v
	}
	return "http://127.0.0.1:9881"
}

// synthesizeEdgeTTSService 转发 edge-tts 合成到项目内置 tts-service（Python edge-tts）。
// 情感以（XX地）前缀注入，由服务端 edge 后端解析前缀并映射音色/音调/语速；
// voice 透传用户选择的音色 short name（服务端支持后优先于情感默认音色）。
// 返回 (音频, contentType, 服务可达, error)：reachable=false 表示连接失败，调用方可回退其它路径。
func synthesizeEdgeTTSService(cfg TTSConfig, text, emotion string, speedFactor float64, textLang, mediaType string) ([]byte, string, bool, error) {
	base := edgeTTSServiceURL(cfg)
	finalText := text
	if cfg.EmotionEnabled && emotion != "" {
		if preset, ok := EmotionPresets[emotion]; ok {
			finalText = preset + text
		} else {
			finalText = "（" + emotion + "）" + text
		}
	}
	payload := map[string]interface{}{
		"text":         finalText,
		"text_lang":    textLang,
		"engine":       EngineEdgeTTS,
		"speed_factor": speedFactor,
		"media_type":   mediaType,
	}
	if cfg.Voice != "" {
		payload["voice"] = cfg.Voice
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", true, err
	}
	req, err := http.NewRequest("POST", strings.TrimRight(base, "/")+"/tts", bytes.NewReader(body))
	if err != nil {
		return nil, "", true, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", false, fmt.Errorf("无法连接语音合成服务（%s）：%v", base, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", true, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", true, fmt.Errorf("语音合成失败（%d）：%s", resp.StatusCode, truncate(string(data), 300))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/" + mediaType
	}
	return data, contentType, true, nil
}

// synthesizeEdgeTTSDirect 使用 Go 内置 Edge TTS 直连微软在线合成（返回 MP3）。
// emotion 通过语速/音量/音调近似模拟（详见 EdgeTTSEmotionParams）；voice 为音色 short name，空则用默认。
func synthesizeEdgeTTSDirect(text string, speedFactor float64, emotion, voice string) ([]byte, string, error) {
	ep := EdgeTTSEmotionParams(emotion)
	rate := EdgeTTSSpeedToRate(speedFactor * ep.SpeedFactor)
	data, err := SynthesizeEdgeTTS(text, voice, rate, ep.Volume, ep.Pitch)
	if err != nil {
		return nil, "", fmt.Errorf("Edge TTS 合成失败：%v", err)
	}
	return data, "audio/mpeg", nil
}

// synthesizeLegacy 转发到 /tts（兼容 edge-tts 与本项目 tts-service、官方 GPT-SoVITS api_v2）
func synthesizeLegacy(cfg TTSConfig, text, textLang, mediaType string, speedFactor float64, engine string) ([]byte, string, error) {
	payload := map[string]interface{}{
		"text":           text,
		"text_lang":      textLang,
		"ref_audio_path": cfg.RefAudioPath,
		"prompt_lang":    cfg.PromptLang,
		"prompt_text":    cfg.PromptText,
		"speed_factor":   speedFactor,
		"media_type":     mediaType,
		"streaming_mode": false,
	}
	if engine != "" {
		payload["engine"] = engine
	}
	// reference_id：ChatTTS 用作固定音色种子，Fish-Speech 由其专属接口转发
	if cfg.ReferenceID != "" {
		payload["reference_id"] = cfg.ReferenceID
	}
	return postAudio(cfg.BaseURL+"/tts", payload, mediaType, cfg.BaseURL)
}

// synthesizeFishSpeech 转发到 Fish-Speech /v1/tts（JSON）
func synthesizeFishSpeech(cfg TTSConfig, text, textLang, mediaType string, speedFactor float64) ([]byte, string, error) {
	if !isFishFormat(mediaType) {
		mediaType = "wav"
	}
	lang := textLang
	if lang == "" || lang == "yue" {
		lang = "auto"
	}
	payload := map[string]interface{}{
		"text":         text,
		"language":     lang,
		"format":       mediaType,
		"streaming":    false,
		"speed_factor": speedFactor,
	}
	if cfg.ReferenceID != "" {
		payload["reference_id"] = cfg.ReferenceID
	}
	if cfg.RefAudioPath != "" {
		payload["reference_audio"] = cfg.RefAudioPath
	}
	if cfg.PromptText != "" {
		payload["reference_text"] = cfg.PromptText
	}
	return postAudio(cfg.BaseURL+"/v1/tts", payload, mediaType, cfg.BaseURL)
}

func isFishFormat(m string) bool {
	switch m {
	case "wav", "mp3", "pcm":
		return true
	}
	return false
}

func postAudio(url string, payload map[string]interface{}, mediaType, baseURL string) ([]byte, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("无法连接语音服务（%s）：%v", baseURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("语音合成失败（%d）：%s", resp.StatusCode, truncate(string(data), 300))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/" + mediaType
	}
	return data, contentType, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
