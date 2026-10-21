# moyue-books（墨月图书）

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

一个基于 **Go + Gin + GORM + SQLite** 后端和 **Vue 3 + Element Plus + Vite** 前端的现代化个人图书管理与阅读应用。支持多种电子书格式上传、自动元数据识别、在线阅读、阅读进度同步，以及基于云端 / 本地引擎的语音合成（TTS）与有声书生成。

> 本项目以 **MIT 协议** 开源，欢迎 Fork、自部署与二次开发。详见 [LICENSE](LICENSE) 与 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 项目截图

| 书架 | 在线阅读器 |
|------|------------|
| ![书架](docs/screenshots/bookshelf.png) | ![在线阅读器](docs/screenshots/reader.png) |
| 导入与编辑 | 设置 |
| ![导入与编辑](docs/screenshots/edit.png) | ![设置](docs/screenshots/settings.png) |

> 截图请放置于 `docs/screenshots/`（建议 PNG 格式，宽度约 1200px）。未提供时上方为占位链接，提交图片后将自动渲染。

## 功能特性

### 已实现（P0 核心）

- **图书上传**：支持拖拽上传、批量导入。
- **格式支持**：EPUB、KEPUB、MOBI、AZW3、AZW、PDF、CBZ、CBR、M4B、MP3，以及 Markdown（`.md`）/ 纯文本（`.txt`）。后两类在上传或扫描时自动转为 EPUB，按一级标题拆分为多章节，并自动生成主题色封面。
- **自动元数据识别**：从 EPUB OPF 提取书名、作者、出版社、语言、ISBN；并从文件名智能解析书名与作者。
- **图书管理**：编辑书名、作者、标签、封面；删除图书。
- **书架视图**：网格 / 列表双视图，支持搜索、排序、分页。
- **在线阅读器**：EPUB 阅读，目录导航、翻页、进度条、阅读偏好设置。
- **阅读进度保存**：自动保存当前章节与百分比进度。
- **阅读统计**：藏书总数、正在阅读、已读完、总页数。

### 规划中（P1 / P2）

- [ ] 划线、批注功能
- [ ] 收藏夹管理
- [ ] 高级搜索与筛选
- [ ] 导出 / 导入备份
- [ ] PDF 在线阅读
- [ ] 在线元数据获取（豆瓣、Google Books）
- [ ] 阅读统计与年度报告
- [ ] 移动端适配优化、云同步

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go、Gin、GORM、glebarez/sqlite（纯 Go SQLite）、difyz9/edge-tts-go |
| 前端 | Vue 3、Element Plus、Vite、Axios、foliate-js、pdfjs-dist |
| 数据库 | SQLite（零配置，本地文件存储） |
| 部署 | 单二进制（前端经 `go:embed` 内嵌），支持飞牛 fnOS 打包 |

## 目录结构

```
moyue-books/
├── backend/                 # Go 后端（独立 Go 模块，可单独开发 / 调试 / 打包）
│   ├── cmd/                 # 辅助命令行工具
│   ├── config/              # 配置加载与数据库初始化
│   ├── handlers/            # HTTP 处理器
│   ├── html/                # 前端产物内嵌（embed.go + dist/，dist 由构建生成）
│   ├── logger/              # 日志组件
│   ├── middleware/          # 中间件（鉴权、访问日志等）
│   ├── models/              # 数据模型
│   ├── routes/              # 路由配置
│   ├── services/            # 业务逻辑（元数据提取、EPUB 解析、TTS 等）
│   ├── utils/               # 工具函数
│   ├── config.yaml.example  # 配置文件模板
│   ├── go.mod / go.sum
│   └── main.go              # 入口
├── frontend/                # Vue 3 前端
│   ├── public/              # 静态资源（pdf.worker、群二维码等）
│   ├── src/
│   │   ├── api/             # API 封装
│   │   ├── components/      # 公共组件
│   │   ├── composables/     # 组合式函数
│   │   ├── player/          # 播放器逻辑
│   │   ├── router/          # 前端路由
│   │   ├── styles/          # 样式
│   │   ├── utils/           # 工具函数
│   │   ├── views/           # 页面组件
│   │   ├── App.vue
│   │   ├── config.js
│   │   ├── main.js
│   │   └── polyfills.js
│   ├── capacitor.config.json
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── build.sh                 # 一键构建（前端 → 同步 → Go 编译）
├── start.sh                 # 本地一键启动（后端 + 前端开发服务器）
├── LICENSE
├── CONTRIBUTING.md
└── README.md
```

## 快速开始

### 前置要求

- Go 1.21+（首次构建会自动拉取 1.25 工具链）
- Node.js 18+
- 操作系统：macOS / Linux / Windows

### 一键构建与运行（生产）

```bash
./build.sh
./backend/moyue-server
```

`build.sh` 会依次执行：构建前端（`vite build`）→ 将产物同步到 `backend/html/dist` → 编译 Go 二进制。前端资源通过 `go:embed` 内嵌进 `backend/moyue-server` 单个二进制，无需额外部署静态文件。后端默认运行在 `http://localhost:18080`，直接访问即可打开前端页面。

### 配置文件（可选）

复制模板为本地配置（位于 `backend/` 根目录，与二进制同目录）：

```bash
cp backend/config.yaml.example backend/config.yaml
```

```yaml
server:
  port: 18080
  log_level: info          # debug / info / warn / error
data:
  data_dir: ""             # 数据目录，默认 backend/data/（飞牛环境为 $TRIM_PKGVAR）
  upload_dir: ""           # 上传目录，默认 <data_dir>/uploads
  db_path: ""              # 数据库，默认 <data_dir>/moyue.db
  jwt_secret_path: ""      # JWT 签名密钥文件，默认 <data_dir>/jwt_secret
```

所有配置项均可用环境变量覆盖（见下文「配置说明」）。配置文件查找顺序：
`MB_CONFIG` > `$TRIM_PKGETC/config.yaml` > `/app/config/config.yaml` > `backend/config.yaml`。

### 开发模式

启动后端（独立调试）：

```bash
# 前端资源经 go:embed 内嵌，单独编译 backend 前需先生成前端产物：
cd frontend && npm install && npm run build && cp -r dist ../backend/html/dist

cd backend
CGO_ENABLED=1 go build -o moyue-server .
./moyue-server
```

或直接用 `start.sh` 一键启动后端与前端开发服务器（热更新，前端运行在 `http://localhost:3000`，已配置代理到后端 API）。

### 飞牛 fnOS（fnpk）部署

仓库已包含飞牛打包骨架（`fnos/`），提供 `fnos/build.sh` 一键交叉编译并打包 `.fpk`，详见 `fnos/moyue/README.md`。后端按飞牛 fpk 应用的目录约定设计，打包为 fpk 应用时无需修改代码。路径与环境变量约定如下：

| 内容 | 环境变量 | 说明 |
|------|----------|------|
| 程序（单二进制，前端已内嵌） | `TRIM_APPDEST` | `moyue-server` |
| 静态配置（只读） | `TRIM_PKGETC` | 放置 `config.yaml`，自动读取 |
| 运行时数据（持久化，卸载保留） | `TRIM_PKGVAR` | 数据库 `moyue.db`、JWT 密钥存放于此 |
| 用户共享数据 | `TRIM_APPSHARE` | 上传文件默认存放于 `$TRIM_APPSHARE/uploads` |

用户可在飞牛「应用管理」中修改服务端口，系统以 `TRIM_SERVICE_PORT` 注入并自动生效（优先级：`TRIM_SERVICE_PORT` > `PORT` > 配置文件 > 18080）。

#### 发布（自动构建 .fpk）

本项目已配置 GitHub Actions 发布流程（`.github/workflows/release.yml`）。打版本 tag 并推送即可自动交叉编译并打包各架构 `.fpk`，并上传到 GitHub Release：

- **触发**：推送 `v*` tag（如 `v1.0.2`），或在 Actions 页面手动触发 `Release` 工作流（可填版本号）。
- **产物**：`amd64`（x86）、`arm64`（arm）两个架构的 `moyue_<version>_<platform>.fpk`。
- **版本号**：自动与 tag 对齐（手动触发时可用输入覆盖），并同步进 `fnos/moyue/manifest` 与前端 `APP_VERSION`。

本地手动打包（不发布）示例：

```bash
./fnos/build.sh                 # 构建 arm64 应用目录（默认）
./fnos/build.sh amd64 --pack    # 构建 x86_64 并生成 .fpk
```

详细打包参数见 `fnos/moyue/README.md`。

## 配置说明

除 `config.yaml` 外，以下环境变量均可覆盖对应配置：

| 环境变量 | 作用 |
|----------|------|
| `MB_CONFIG` | 指定配置文件路径（最高优先级） |
| `MB_DATA_DIR` | 数据目录 |
| `MB_UPLOAD_DIR` | 上传目录 |
| `MB_DB_PATH` | 数据库文件路径 |
| `MB_JWT_SECRET` | JWT 签名密钥（环境变量优先于密钥文件） |
| `PORT` | 服务监听端口 |
| `EDGE_TTS_PROXY` | Edge TTS 代理地址（受限网络出网用） |
| `HTTPS_PROXY` | 通用 HTTPS 代理（Edge TTS 亦可经此出网） |
| `FEEDBACK_MAIL_ACCOUNT` | 反馈邮件通道的 163 邮箱账号 |
| `FEEDBACK_MAIL_AUTH` | 反馈邮件通道的 163 SMTP 授权码 |

路径解析优先级：
- 配置路径：`MB_CONFIG` > `$TRIM_PKGETC/config.yaml` > `/app/config/config.yaml` > `backend/config.yaml`
- 数据目录：`MB_DATA_DIR` > 配置文件 `data_dir` > `$TRIM_PKGVAR` > `/app/data` > `backend/data`
- 上传目录：`MB_UPLOAD_DIR` > 配置文件 `upload_dir` > `$TRIM_APPSHARE/uploads`（飞牛环境）> `数据目录/uploads`
- 数据库默认 `数据目录/moyue.db`；上传目录（含 `books/covers/temp` 子目录）首次启动自动创建

> 数据与备份：图书文件存于 `backend/data/uploads/books/`，封面存于 `backend/data/uploads/covers/`，数据库位于 `backend/data/moyue.db`。备份整个 `backend/data/` 目录即可完整迁移。

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/books` | 图书列表（支持分页、搜索、排序） |
| POST | `/api/books` | 单本上传（multipart/form-data） |
| POST | `/api/books/batch` | 批量上传 |
| GET | `/api/books/:id` | 图书详情 |
| PUT | `/api/books/:id` | 更新图书信息 |
| DELETE | `/api/books/:id` | 删除图书 |
| GET | `/api/books/:id/cover` | 获取封面 |
| POST | `/api/books/:id/cover` | 上传 / 更换封面 |
| PUT | `/api/books/:id/progress` | 更新阅读进度 |
| GET | `/api/books/:id/content` | 获取 EPUB 目录 / 章节内容 |
| GET | `/api/books/:id/download` | 下载原文件 |
| GET | `/api/stats` | 统计信息 |

## 使用说明

### 导入图书
1. 点击左侧导航栏「导入图书」。
2. 拖拽文件到上传区域，或点击选择文件（支持批量）。
3. 系统会自动提取 EPUB 元数据（书名、作者、出版社等）。

### 浏览书架
1. 点击「书架」进入图书列表。
2. 切换网格 / 列表视图。
3. 支持按时间、书名、最近阅读排序；顶部搜索栏可按书名 / 作者搜索。

### 阅读图书
1. 点击图书卡片或「阅读」按钮进入阅读器。
2. 点击屏幕左右边缘或按键盘 ← → 翻页。
3. 点击「目录」打开章节导航；点击「设置」调整字体、字号、行距、主题。
4. 退出时自动保存阅读进度。

### 编辑图书信息
1. 在书架或详情页点击「编辑」。
2. 修改书名、作者（逗号分隔）、出版社、描述、标签。
3. 支持更换封面图片。

> 注：PDF 格式暂不支持在线阅读，可通过详情页下载后阅读。

## 语音合成（TTS）配置

应用内置多种语音合成引擎，用于在线阅读朗读与有声书生成：

- **Edge TTS（云端，免费）**：微软在线语音，无需本地模型，默认开启。后端使用 Go 原生库直连微软端点，**不依赖 Python tts-service**，并在收到 403 时依据服务端时间自动校正本地时钟偏移并重试。
- **GPT-SoVITS / Fish-Speech / ChatTTS（本地部署）**：声音克隆或本地模型，需在 `tts.base_url` 配置自托管服务地址。

### 受限网络（403）代理配置

在国内或受限网络中，微软端点可能拒绝连接（403 / 握手失败）。可选两种解法：

1. **代理出网（推荐）**：在启动后端的环境中设置以下环境变量之一，后端会经代理连接微软：
   ```bash
   export EDGE_TTS_PROXY="http://127.0.0.1:7890"   # 优先
   # 或通用代理（二选一）
   export HTTPS_PROXY="http://127.0.0.1:7890"
   ```
2. **改用本机引擎**：Edge TTS 不可用时，有声书生成选「自动」会回退本地引擎；阅读朗读可选「浏览器离线」。

> 说明：音色列表接口（`voices.ListVoices`）在当前版本会忽略代理参数，但其同样带时钟偏移重试，通常不受影响。

## 问题反馈（邮件通道）配置

应用内置一个 163 邮箱反馈通道，用于接收用户反馈。**源码中不含任何真实邮箱凭证**，需通过环境变量自行注入：

```bash
export FEEDBACK_MAIL_ACCOUNT="yourname@163.com"   # 发件 / 收件 163 邮箱账号
export FEEDBACK_MAIL_AUTH="your_smtp_auth_code"   # 163 SMTP 授权码（非登录密码）
```

- 授权码在 163 网页端「设置 → POP3/SMTP/IMAP」开启服务后获取。
- 未配置时反馈邮件通道自动禁用（`FeedbackMailEnabled` 返回 `false`），不影响其他功能。

## 开发计划

当前完成度约 **60%**（P0 核心功能已全部实现）。如需继续开发 P1/P2 功能（批注、收藏夹、PDF 阅读器、阅读统计等），可在此代码基础上扩展。欢迎通过 Issue 与 Pull Request 参与贡献。

## 贡献

欢迎参与贡献！提交 Issue、提出建议或发起 Pull Request 前，请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 开源许可证

本项目基于 **[MIT 许可证](LICENSE)** 发布。你可以自由地使用、修改、分发本软件（包括商业用途），只需在副本中保留原始版权声明与许可声明。软件按「原样」提供，作者不承担任何担保或责任。

## 致谢与依赖

本项目使用了诸多优秀的开源库，包括但不限于：

- 后端：Gin、GORM、glebarez/sqlite、difyz9/edge-tts-go、pdfcpu 等（完整列表见 `backend/go.mod`）
- 前端：Vue 3、Element Plus、foliate-js、pdfjs-dist 等（完整列表见 `frontend/package.json`）

感谢这些项目的维护者与贡献者。
