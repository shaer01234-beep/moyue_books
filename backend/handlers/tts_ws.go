package handlers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"my-books/backend/middleware"
	"my-books/backend/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// wsTTSSem 限制并发合成请求数（edge-tts 经 tts-service 转发到微软共享服务，防止过载被限流）
var wsTTSSem = make(chan struct{}, 3)

var ttsWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 本服务已通过 query token 鉴权，允许任意 Origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TTSWebSocketHandler 语音合成 WebSocket 接口（Go 直连 Edge TTS）。
// 一条连接可连续发送多条合成请求并并发返回音频，复用握手、大幅提升逐句朗读效率。
//
// 协议：
//
//	客户端 -> 服务端（TextMessage JSON）：{"id":N,"text":"...","emotion":"平静","speed_factor":1.0}
//	                                   或 {"type":"cancel"} 取消全部进行中的合成
//	服务端 -> 客户端（BinaryMessage）：   [int32 BE id][MP3 音频]
//	服务端 -> 客户端（TextMessage JSON）：{"id":N,"error":"..."} 单条合成失败
func TTSWebSocketHandler(c *gin.Context) {
	uid := middleware.CurrentUserID(c)

	conn, err := ttsWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		id   int32
		data []byte
		err  error
	}
	results := make(chan result, 16)

	var wg sync.WaitGroup
	defer wg.Wait() // 等待进行中的合成收尾，避免 goroutine 泄漏

	// 写协程：把合成结果按 id 发给客户端
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for res := range results {
			if ctx.Err() != nil {
				continue // 已取消，丢弃结果
			}
			if res.err != nil {
				msg, _ := json.Marshal(gin.H{"id": res.id, "error": res.err.Error()})
				conn.WriteMessage(websocket.TextMessage, msg)
				continue
			}
			buf := make([]byte, 4+len(res.data))
			binary.BigEndian.PutUint32(buf[:4], uint32(res.id))
			copy(buf[4:], res.data)
			conn.WriteMessage(websocket.BinaryMessage, buf)
		}
	}()

	cfg := services.GetTTSConfig(uid)

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if mt != websocket.TextMessage {
			continue
		}
		var req struct {
			Type    string  `json:"type"`
			ID      int32   `json:"id"`
			Text    string  `json:"text"`
			Emotion string  `json:"emotion"`
			Speed   float64 `json:"speed_factor"`
			Voice   string  `json:"voice"` // Edge TTS 音色 short name，空则用保存配置
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Type == "cancel" {
			cancel() // 取消全部进行中的合成
			continue
		}
		if strings.TrimSpace(req.Text) == "" {
			continue
		}

		// 按用户配置的引擎分发合成（edge-tts 经 tts-service 转发，见 SynthesizeWithConfig）。
		// 请求参数可覆盖语速与音色；情感是否生效由 cfg.EmotionEnabled 内部控制。
		synthCfg := cfg
		if req.Voice != "" {
			synthCfg.Voice = req.Voice
		}
		if req.Speed > 0 {
			synthCfg.SpeedFactor = req.Speed
		}

		wg.Add(1)
		go func(id int32, text, emotion string, speed float64, synthCfg services.TTSConfig) {
			defer wg.Done()
			// 全局并发限流
			select {
			case wsTTSSem <- struct{}{}:
				defer func() { <-wsTTSSem }()
			case <-ctx.Done():
				return
			}
			data, _, err := services.SynthesizeWithConfig(synthCfg, text, emotion, speed, "", "mp3")
			if ctx.Err() != nil {
				return // 已取消，不再回写
			}
			results <- result{id: id, data: data, err: err}
		}(req.ID, req.Text, req.Emotion, req.Speed, synthCfg)
	}

	// 读循环结束（连接关闭）：取消进行中合成并关闭结果通道
	cancel()
	close(results)
	<-writeDone
}
