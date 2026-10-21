package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config 应用配置
type Config struct {
	Server ServerConfig `yaml:"server"`
	Data   DataConfig   `yaml:"data"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

// DataConfig 数据目录配置（数据库、JWT 密钥、上传文件均存放于数据目录）
type DataConfig struct {
	DataDir       string `yaml:"data_dir"`
	UploadDir     string `yaml:"upload_dir"`
	DBPath        string `yaml:"db_path"`
	JWTSecretPath string `yaml:"jwt_secret_path"`
}

// Cfg 全局配置
var Cfg *Config

// GatewaySocket 返回飞牛统一网关的 Unix Socket 路径（FNNAS_GATEWAY_SOCKET，fpk 启动脚本注入）。
// 非空时后端仅监听该 socket，不绑定 TCP 端口，由飞牛网关反代外部访问，彻底规避端口冲突。
func GatewaySocket() string { return os.Getenv("FNNAS_GATEWAY_SOCKET") }

// GatewayPrefix 返回飞牛统一网关的 URL 前缀（FNNAS_GATEWAY_PREFIX，如 /app/moyue）。
// 网关把 <前缀>/* 的请求转发到 GatewaySocket，后端据此注册带前缀的路由并剥离前缀。
func GatewayPrefix() string { return os.Getenv("FNNAS_GATEWAY_PREFIX") }

// GatewayMode 是否启用飞牛统一网关（Unix Socket）模式
func GatewayMode() bool { return GatewaySocket() != "" }

// projectRoot 定位后端根目录：
//   - 开发/调试：exe 所在目录含 go.mod 时即为该目录本身（backend 独立项目）
//   - 打包部署（fpk target/ 等）：无 go.mod 时以二进制所在目录为根，
//     保证二进制同目录的 config.yaml 与相对数据路径始终可用，与工作目录无关
func projectRoot() string {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		return dir
	}
	wd, _ := os.Getwd()
	return wd
}

// Load 加载配置，优先级：环境变量 > 配置文件 > 默认值。
// 配置文件查找顺序：MB_CONFIG > $TRIM_PKGETC/config.yaml（飞牛 fpk 静态配置）> /app/config/config.yaml（旧约定）> 项目根 config.yaml。
// 数据目录默认：MB_DATA_DIR > 配置文件 data_dir > $TRIM_PKGVAR（飞牛 fpk 运行时数据）> /app/data（旧约定）> 项目根 data。
// 上传目录默认：MB_UPLOAD_DIR > 配置文件 upload_dir > $TRIM_APPSHARE/uploads（飞牛 fpk 共享目录）
//
//	> 从 TRIM_PKGVAR 推导的共享目录（兜底）> <data_dir>/uploads。
//
// 端口默认：$TRIM_SERVICE_PORT（飞牛 fpk，用户可自定义）> $PORT > 配置文件 port > 18080。
func Load() *Config {
	Cfg = &Config{
		Server: ServerConfig{Port: 18080, LogLevel: "info"},
	}

	// 0. 环境诊断日志：确认飞牛 fpk 各 TRIM_* 环境变量是否注入
	logTrimEnv()

	// 1. 定位并解析配置文件
	if configPath := locateConfigFile(); configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("warn: read config file %s failed: %v", configPath, err)
		} else if err := yaml.Unmarshal(data, Cfg); err != nil {
			log.Printf("warn: parse config file %s failed: %v", configPath, err)
		} else {
			log.Println("Loaded config file:", configPath)
		}
	}

	// 2. 环境变量覆盖
	applyEnv()

	// 3. 补齐默认路径
	resolvePaths()

	log.Printf("data dir: %s | upload dir: %s | db: %s", Cfg.Data.DataDir, Cfg.Data.UploadDir, Cfg.Data.DBPath)
	return Cfg
}

// logTrimEnv 打印与路径/端口相关的环境变量，用于排查 fpk 部署时
// TRIM_APPSHARE / TRIM_PKGVAR 等未被注入的问题。
func logTrimEnv() {
	keys := []string{
		"TRIM_APPDEST", "TRIM_PKGVAR", "TRIM_PKGETC",
		"TRIM_APPSHARE", "TRIM_SERVICE_PORT", "TRIM_USERNAME", "TRIM_GROUPNAME",
		"MB_DATA_DIR", "MB_UPLOAD_DIR", "MB_DB_PATH", "MB_CONFIG", "PORT",
	}
	for _, k := range keys {
		v := os.Getenv(k)
		if v != "" {
			log.Printf("[env] %s=%s", k, v)
		} else {
			log.Printf("[env] %s=<未设置>", k)
		}
	}
	if os.Getenv("TRIM_APPSHARE") == "" {
		log.Println("[env] 提示: TRIM_APPSHARE 未注入，上传目录将回退到 var 目录。" +
			" 请确认 fpk 的 config/resource 已声明 data-share 并重新安装应用（升级不注入，需重装）。")
	}
}

// locateConfigFile 按优先级查找配置文件
func locateConfigFile() string {
	if p := os.Getenv("MB_CONFIG"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		log.Printf("warn: MB_CONFIG=%s not found, fallback to default", p)
	}
	// 飞牛 fpk：静态配置目录（etc/，只读）
	if p := os.Getenv("TRIM_PKGETC"); p != "" {
		if _, err := os.Stat(filepath.Join(p, "config.yaml")); err == nil {
			return filepath.Join(p, "config.yaml")
		}
	}
	// 兼容旧 fpk 容器内约定配置目录
	if _, err := os.Stat("/app/config/config.yaml"); err == nil {
		return "/app/config/config.yaml"
	}
	// 飞牛 fpk：程序目录（target/，对应开发时 app/），兼容 config.yaml
	// 与二进制同目录或位于 app/server/ 子目录（与二进制同目录时上面 projectRoot 分支已覆盖）
	if p := os.Getenv("TRIM_APPDEST"); p != "" {
		for _, rel := range []string{"config.yaml", "server/config.yaml"} {
			if _, err := os.Stat(filepath.Join(p, rel)); err == nil {
				return filepath.Join(p, rel)
			}
		}
	}
	// 二进制所在目录（含 go.mod 时为 backend 项目根）
	if p := filepath.Join(projectRoot(), "config.yaml"); fileExists(p) {
		return p
	}
	return ""
}

// applyEnv 环境变量覆盖配置项
func applyEnv() {
	if v := os.Getenv("MB_DATA_DIR"); v != "" {
		Cfg.Data.DataDir = v
	}
	if v := os.Getenv("MB_UPLOAD_DIR"); v != "" {
		Cfg.Data.UploadDir = v
	}
	if v := os.Getenv("MB_DB_PATH"); v != "" {
		Cfg.Data.DBPath = v
	}
	if v := os.Getenv("MB_JWT_SECRET"); v != "" {
		Cfg.Data.JWTSecretPath = v
	}
	// 飞牛 fpk：系统注入的 service_port（manifest 定义，用户可在应用设置中修改端口映射）
	if v := os.Getenv("TRIM_SERVICE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			Cfg.Server.Port = n
		}
	} else if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			Cfg.Server.Port = n
		}
	}
}

// resolvePaths 补齐未配置的路径默认值
func resolvePaths() {
	if Cfg.Data.DataDir == "" {
		if p := os.Getenv("TRIM_PKGVAR"); p != "" {
			// 飞牛 fpk：运行时数据目录（var/，持久化，卸载保留，首次启动自动创建）
			Cfg.Data.DataDir = p
		} else if fileExists("/app/data") {
			// 兼容旧 fpk 容器内约定数据目录
			Cfg.Data.DataDir = "/app/data"
		} else {
			Cfg.Data.DataDir = filepath.Join(projectRoot(), "data")
		}
	}
	if Cfg.Data.UploadDir == "" {
		if p := os.Getenv("TRIM_APPSHARE"); p != "" {
			// 飞牛 fpk：上传文件存放于共享目录（shares/，用户可在"文件管理"直接访问）
			Cfg.Data.UploadDir = filepath.Join(p, "uploads")
			log.Printf("upload dir from TRIM_APPSHARE: %s", Cfg.Data.UploadDir)
		} else if p := deriveAppShare(); p != "" {
			// 兜底：TRIM_APPSHARE 未注入时，从 TRIM_PKGVAR 推导共享目录（目录存在才使用）
			Cfg.Data.UploadDir = filepath.Join(p, "uploads")
			log.Printf("upload dir derived from TRIM_PKGVAR: %s", Cfg.Data.UploadDir)
		} else {
			Cfg.Data.UploadDir = filepath.Join(Cfg.Data.DataDir, "uploads")
			log.Printf("upload dir fallback to data dir: %s", Cfg.Data.UploadDir)
		}
	} else {
		log.Printf("upload dir from config/env: %s", Cfg.Data.UploadDir)
	}
	if Cfg.Data.DBPath == "" {
		Cfg.Data.DBPath = filepath.Join(Cfg.Data.DataDir, "moyue.db")
	}
	if Cfg.Data.JWTSecretPath == "" {
		Cfg.Data.JWTSecretPath = filepath.Join(Cfg.Data.DataDir, "jwt_secret")
	}
}

// fileExists 判断路径存在
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// deriveAppShare 从 TRIM_PKGVAR（如 /vol1/@appdata/moyue）推导共享目录
// （/vol1/@appshare/moyue）。仅当推导出的目录已存在时返回，避免误用。
// 用于 TRIM_APPSHARE 未注入但共享目录已创建（resource 声明生效）的场景。
func deriveAppShare() string {
	pkgVar := os.Getenv("TRIM_PKGVAR")
	if pkgVar == "" {
		return ""
	}
	app := filepath.Base(pkgVar)                // moyue
	base := filepath.Base(filepath.Dir(pkgVar)) // @appdata
	if !strings.HasPrefix(base, "@appdata") {
		return ""
	}
	shareBase := "@appshare" + strings.TrimPrefix(base, "@appdata")
	share := filepath.Join(filepath.Dir(filepath.Dir(pkgVar)), shareBase, app)
	if fileExists(share) {
		return share
	}
	return ""
}

// 便捷访问器

func DataDir() string       { return Cfg.Data.DataDir }
func UploadDir() string     { return Cfg.Data.UploadDir }
func DBPath() string        { return Cfg.Data.DBPath }
func JWTSecretPath() string { return Cfg.Data.JWTSecretPath }
func Port() string          { return strconv.Itoa(Cfg.Server.Port) }
func LogLevel() string      { return Cfg.Server.LogLevel }
