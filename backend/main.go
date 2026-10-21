package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"my-books/backend/config"
	"my-books/backend/handlers"
	"my-books/backend/logger"
	"my-books/backend/routes"
	"my-books/backend/services"
)

func main() {
	// 加载配置（支持配置文件与环境变量，兼容飞牛 fpk 目录约定）
	config.Load()

	// 初始化运行日志：输出到控制台 + <data_dir>/logs/moyue.log（按大小轮转）。
	// config.Load 期间的诊断日志已由 logger 包缓冲，初始化后一并写入文件
	if err := logger.Init(config.LogLevel(), filepath.Join(config.DataDir(), "logs")); err != nil {
		log.Printf("warn: 运行日志初始化失败，日志仅输出到控制台: %v", err)
	}

	// 确保数据目录与上传目录存在
	for _, dir := range []string{
		config.DataDir(),
		filepath.Join(config.UploadDir(), "books"),
		filepath.Join(config.UploadDir(), "covers"),
		filepath.Join(config.UploadDir(), "temp"),
		services.AudiobookRoot(),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal("failed to create directory:", err)
		}
	}

	// 初始化数据库
	config.InitDB()

	// 恢复中断的有声书生成任务（服务重启场景）
	services.RecoverInterruptedAudiobooks()

	// 启动媒体库文件夹监听：目录下文件变动时自动触发一次扫描（替代原定时轮询）。
	handlers.StartLibraryWatcher()

	// 设置路由
	r := routes.SetupRouter()

	// 统一网关模式：监听 Unix Socket（由飞牛网关反代），
	// 同时监听 TCP 端口（飞牛 service_port / 书籍分享外部直连），两种入口互不冲突
	if socket := config.GatewaySocket(); socket != "" {
		os.Remove(socket) // 清理上次残留的 socket 文件
		listener, err := net.Listen("unix", socket)
		if err != nil {
			log.Fatal("failed to listen unix socket:", err)
		}
		os.Chmod(socket, 0660) // 应用用户与网关进程可读写，避免他用户访问
		log.Println("Server listening on unix socket:", socket)
		go func() {
			if err := (&http.Server{Handler: r}).Serve(listener); err != nil {
				log.Fatal("failed to serve unix socket:", err)
			}
		}()
	}

	// TCP 端口：本地开发 / 飞牛 service_port（TRIM_SERVICE_PORT）直连，供书籍分享等外部访问
	log.Println("Server starting on http://localhost:" + config.Port())
	if err := r.Run(":" + config.Port()); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
