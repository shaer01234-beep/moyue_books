# 飞牛 fpk 打包说明

将本项目构建为飞牛 fnOS 应用（fpk）。目录结构与打包脚本参见同级 `../build.sh`。

> 官方 schema 以 `fnpack create` 生成的官方模板为准，本目录按飞牛官方标准组织。

## 目录结构

```
fnos/
├── build.sh              # 一键构建 + 打包脚本（架构参数 / --pack / --skip-web）
├── icon-src/             # 图标源文件
└── moyue/                # 应用目录（= fpk 包根）
    ├── manifest          # 应用描述（key=value）：appname、version、service_port 等
    ├── config/
    │   ├── privilege     # 应用运行用户（JSON）
    │   └── resource      # 资源声明：data-share → 创建 shares/ 并注入 TRIM_APPSHARE
    ├── cmd/
    │   ├── main          # 应用主控：start/stop/status
    │   ├── install_init / install_callback
    │   ├── uninstall_init / uninstall_callback
    │   ├── upgrade_init / upgrade_callback
    │   └── config_init / config_callback
    ├── wizard/           # 安装向导（install / config / uninstall，JSON）
    ├── app/
    │   ├── bin/
    │   │   └── moyue     # Go 单二进制（前端已通过 //go:embed 内嵌）
    │   └── ui/           # 桌面入口：config（.url + 网关配置）+ images（图标）
    ├── ICON.PNG          # 64x64 应用图标
    └── ICON_256.PNG      # 256x256 应用图标
```

> 安装后映射：`app/` → `target/`（`TRIM_APPDEST`）、`etc/` → `@appconf`（`TRIM_PKGETC`）、`var/` → `@appdata`（`TRIM_PKGVAR`）、`shares/` → `@appshare`（`TRIM_APPSHARE`，需 resource 声明）

## 构建步骤

```bash
# 默认 arm64（manifest platform=arm）
./fnos/build.sh

# x86_64
./fnos/build.sh amd64

# 老款 32 位 ARM
./fnos/build.sh armv7

# 构建并生成 .fpk（有 fnpack 用官方工具，否则用内置打包器）
./fnos/build.sh --pack

# 跳过前端构建（沿用已有 frontend/dist，调试后端时提速）
./fnos/build.sh --skip-web
```

脚本会依次完成：

1. 构建前端（注入网关前缀 `VITE_BASE=/app/moyue/`、`VITE_API_BASE=/app/moyue/api`）
2. 同步 `frontend/dist` → `backend/html/dist`（由 `//go:embed` 内嵌进二进制）
3. 交叉编译 Go 服务 → `app/bin/moyue`（`CGO_ENABLED=0`，纯 Go SQLite，无需交叉工具链）
4. 按架构同步 `manifest` 的 `platform` 字段（amd64→x86，arm64/armv7→arm）
5. `--pack` 时生成 `moyue_<version>_<platform>.fpk`，并把 `app.tgz` 的 md5 写入 manifest

生成后在飞牛 **应用中心 → 手动安装** 上传 `.fpk` 即可。

## 访问方式

- **统一网关**：桌面图标打开，即 `https://<NAS>/app/moyue`（Unix Socket + `gatewayPrefix`）
- **直连端口**：`http://<NAS IP>:18080`（书籍分享链接、局域网直连）

两者由同一进程双入口提供，互不冲突；端口可在应用设置中修改（`TRIM_SERVICE_PORT`）。

## 环境变量约定（后端自动识别）

| 环境变量 | 安装后目录 | 用途 |
|---------|-----------|------|
| `TRIM_PKGVAR` | `/vol*/@appdata/moyue` | 数据库 `moyue.db`、JWT 密钥、日志 |
| `TRIM_PKGETC` | `/vol*/@appconf/moyue` | 只读配置 `config.yaml` |
| `TRIM_APPSHARE` | `/vol*/@appshare/moyue` | 上传文件 `uploads/`（用户可在"文件管理"直接访问） |
| `TRIM_APPDEST` | `/vol*/@appcenter/moyue` | 程序目录（含 `bin/moyue`） |
| `TRIM_SERVICE_PORT` | - | 服务端口（用户自定义） |

端口优先级：`TRIM_SERVICE_PORT` > `PORT` > 配置文件 `port` > 18080

## 上传文件在共享目录（TRIM_APPSHARE）

上传文件（图书/封面）默认存放于 `$TRIM_APPSHARE/uploads/`，即 `/vol*/@appshare/moyue/uploads/`，用户可在飞牛"文件管理"直接访问。

**关键前提**：必须在 `config/resource` 中声明 `data-share`（本目录已提供），系统才会创建 `shares/` 并注入 `TRIM_APPSHARE`。

### 从旧版升级（数据迁移）

旧版上传文件在 `var/` 下（`/vol*/@appdata/moyue/uploads/`），升级到新版后需手动迁移：

```bash
# 以 SSH 登录飞牛执行（先停止应用）
mv /vol1/@appdata/moyue/uploads/* /vol1/@appshare/moyue/uploads/
chown -R moyue:moyue /vol1/@appshare/moyue/uploads
```

数据库中的旧文件路径由后端 `ResolvePath` 的 `/uploads/` 重映射规则自动兼容，无需改库。
