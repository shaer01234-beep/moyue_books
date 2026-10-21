package utils

import (
	"os"
	"path/filepath"
	"strings"

	"my-books/backend/config"
)

// ResolvePath 将数据库中存的路径解析为绝对路径，兼容迁移前后的存储形式：
//   - 绝对路径原样返回
//   - 含 "uploads/" 的路径（旧相对路径如 "uploads/covers/x.png"、旧绝对路径
//     如 "/xx/.../uploads/books/x.pdf"）统一重映射到数据目录下的
//     <data_dir>/uploads/...（上传文件已随版本迁移）
//   - 其余相对路径去掉 .. 前缀段后基于项目根目录拼接，
//     避免依赖进程工作目录导致文件读取失败
func ResolvePath(p string) string {
	if p == "" {
		return p
	}
	cleaned := filepath.Clean(p)
	slashed := filepath.ToSlash(cleaned)

	// 上传文件：统一指向数据目录下的 uploads
	if idx := strings.Index(slashed, "/uploads/"); idx >= 0 {
		rel := slashed[idx+len("/uploads/"):]
		if rel != "" {
			return filepath.Join(config.UploadDir(), rel)
		}
	}

	if filepath.IsAbs(cleaned) {
		return cleaned
	}

	var kept []string
	for _, seg := range strings.Split(slashed, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			continue
		}
		kept = append(kept, seg)
	}
	if len(kept) == 0 {
		return ProjectRoot
	}
	return filepath.Join(ProjectRoot, filepath.Join(kept...))
}

var ProjectRoot string

func init() {
	// 尝试从可执行文件路径推断项目根目录
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		dir := filepath.Dir(exe)
		// 含 go.mod 的目录即 backend 项目根（可独立开发/调试/打包）；
		// 无 go.mod（如 fpk target/ 部署）时以二进制所在目录为根
		ProjectRoot = dir
	} else {
		wd, _ := os.Getwd()
		ProjectRoot = wd
	}
}
