# 个人图书管理系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

基于 **Go + Gin + GORM + SQLite** 后端和 **Vue 3 + Element Plus + Vite** 前端构建的现代化个人图书管理应用。

> 本项目以 **MIT 协议** 开源，欢迎 Fork、自部署与二次开发。详见 [LICENSE](LICENSE) 与 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 功能特性（已实现）

### P0 - 核心功能
- [x] **图书上传**：支持拖拽上传，批量导入
- [x] **格式支持**：EPUB、KEPUB、MOBI、AZW3、AZW、PDF、CBZ、CBR、M4B、MP3，以及 Markdown(.md) / 纯文本(.txt)（上传或扫描时自动转为 EPUB，按一级标题拆分为多章节，并自动生成主题色封面）
- [x] **自动元数据识别**：从 EPUB OPF 文件提取书名、作者、出版社、语言、ISBN
- [x] **文件名智能解析**：从文件名提取书名和作者
- [x] **图书 CRUD**：编辑书名、作者、标签、封面；删除图书
- [x] **书架视图**：网格/列表双视图，支持搜索、排序、分页
- [x] **在线阅读器**：EPUB 阅读支持，目录导航、翻页、进度条
- [x] **阅读进度保存**：自动保存当前章节和百分比进度
- [x] **阅读统计**：藏书总数、正在阅读、已读完、总页数

### P1 - 待实现
- [ ] 划线和批注功能
- [ ] 收藏夹管理
- [ ] 高级搜索和筛选
- [ ] 导出/导入备份
- [ ] PDF 在线阅读

### P2/P3 - 未来扩展
- [ ] 在线元数据获取（豆瓣、Google Books）
- [ ] 阅读统计和年度报告
- [ ] 移动端适配优化
- [ ] 云同步

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.17, Gin, GORM, SQLite3 |
| 前端 | Vue 3, Element Plus, Vite, Axios |
| 数据库 | SQLite3（零配置，本地存储）|

## 目录结构

```
my-books/
├── backend/              # Go 后端（可独立开发/调试/打包）
│   ├── config/           # 配置加载与数据库初始化
│   │   ├── config.go     # 配置文件 + 环境变量 + fpk 目录探测
│   │   └── db.go         # SQLite 初始化
│   ├── config.yaml.example   # 配置文件模板（位于 backend/ 根目录）
│   ├── data/             # 数据目录（数据库 + 上传文件）
│   │   ├── moyue.db    # SQLite 数据库
│   │   ├── jwt_secret    # JWT 签名密钥
│   │   └── uploads/      # 上传文件存储
│   │       ├── books/    # 图书文件
│   │       ├── covers/   # 封面图片
│   │       └── temp/     # 临时文件
│   ├── handlers/         # HTTP 处理器
│   ├── models/           # 数据模型
│   ├── routes/           # 路由配置
│   ├── services/         # 业务逻辑（元数据提取、EPUB 解析）
│   ├── utils/            # 工具函数
│   └── main.go           # 入口文件
├── frontend/             # Vue 3 前端
│   ├── src/
│   │   ├── api/          # API 封装
│   │   ├── views/        # 页面组件
│   │   ├── router/       # 路由配置
│   │   └── App.vue       # 根组件
│   ├── package.json
│   └── vite.config.js
└── README.md
```

## 快速启动

### 前置要求
- Go 1.17+
- Node.js 18+
- macOS / Linux / Windows

### 1. 生产构建（前端内嵌进二进制，单文件部署）

```bash
./build.sh
./backend/moyue-server
```

`build.sh` 会依次：构建前端（vite build）→ 将产物同步到 `backend/html/dist` → Go 编译。
前端资源通过 `go:embed` 内嵌进 `backend/moyue-server` 单个二进制，无需再部署静态文件。
后端默认运行在 `http://localhost:18080`，直接访问即可打开前端页面。

### 2. 配置文件（可选）

复制 `backend/config.yaml.example` 为 `backend/config.yaml`（backend 项目根目录，与二进制同目录），可自定义端口、数据目录等：

```yaml
server:
  port: 18080
data:
  data_dir: ""       # 数据目录，默认 backend/data/（fpk 环境为 $TRIM_PKGVAR）
  upload_dir: ""     # 上传目录，默认 <data_dir>/uploads
  db_path: ""        # 数据库，默认 <data_dir>/moyue.db
```

所有配置项均可用环境变量覆盖：`MB_DATA_DIR`、`MB_UPLOAD_DIR`、`MB_DB_PATH`、`MB_JWT_SECRET`、`PORT`。
配置文件查找顺序：`MB_CONFIG` > `$TRIM_PKGETC/config.yaml` > `/app/config/config.yaml` > `backend/config.yaml`（backend 独立运行时默认读取）。

### 2.1 单独运行 backend（独立开发/调试）

backend 目录本身即一个完整的 Go 项目（含 `go.mod`），可脱离前端独立运行：

```bash
# 前端资源通过 go:embed 内嵌，单独编译 backend 前需先生成前端产物：
cd frontend && npm install && npm run build && cp -r dist ../backend/html/dist

cd backend
CGO_ENABLED=1 go build -o moyue-server .
./moyue-server
```

- 数据目录自动定位到 `backend/data/`（数据库、JWT 密钥、上传文件）
- 自动读取 `backend/config.yaml`（存在时）
- 不依赖工作目录、不依赖仓库根目录，任意位置启动结果一致
- 也可直接执行仓库根的 `./build.sh` 一步完成「构建前端 → 同步 → 编译二进制」

### 3. 启动前端（开发模式，热更新）

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:3000`，已配置代理到后端 API。

### 4. 访问应用

打开浏览器访问 http://localhost:3000 即可使用（生产模式直接访问 http://localhost:18080）。

### 5. 飞牛 fnOS（fpk 应用）打包兼容

后端已按飞牛 fpk 应用的目录约定设计，打包为 fpk 应用时无需修改代码。打包骨架见 `backend/fpk/`（manifest + 启动脚本 + 构建说明）：

**用户可自定义服务端口**：`manifest` 中 `service_port` 定义默认端口，用户在飞牛"应用管理"界面可修改端口映射，系统以 `TRIM_SERVICE_PORT` 注入，后端自动生效（优先级：`TRIM_SERVICE_PORT` > `PORT` > 配置文件 > 18080）。

| 内容 | 开发目录 | 安装后目录 | 环境变量 | 说明 |
|------|---------|-----------|---------|------|
| 程序（单二进制，前端已内嵌） | `app/` | `target/` | `TRIM_APPDEST` | `moyue-server` |
| 静态配置（只读） | `etc/` | `etc/` | `TRIM_PKGETC` | 放置 `config.yaml`，自动读取 |
| 运行时数据（持久化，卸载保留） | `var/` | `var/` | `TRIM_PKGVAR` | 数据库 `moyue.db`、JWT 密钥存放于此 |
| 用户共享数据 | `shares/` | `shares/` | `TRIM_APPSHARE` | 上传文件默认存放于 `$TRIM_APPSHARE/uploads`，可在飞牛"文件管理"直接访问 |

路径优先级规则：
- 配置路径：`MB_CONFIG` > `$TRIM_PKGETC/config.yaml` > `/app/config/config.yaml`（旧约定）> `backend/config.yaml`
- 数据目录：`MB_DATA_DIR` > 配置文件 `data_dir` > `$TRIM_PKGVAR` > `/app/data`（旧约定）> `backend/data`
- 上传目录：`MB_UPLOAD_DIR` > 配置文件 `upload_dir` > `$TRIM_APPSHARE/uploads`（飞牛环境）> `数据目录/uploads`
- 数据库默认 `数据目录/moyue.db`；上传目录（含 `books/covers/temp` 子目录）首次启动自动创建

## API 接口文档

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/books` | 图书列表（支持分页、搜索、排序） |
| POST | `/api/books` | 单本上传（multipart/form-data） |
| POST | `/api/books/batch` | 批量上传 |
| GET | `/api/books/:id` | 图书详情 |
| PUT | `/api/books/:id` | 更新图书信息 |
| DELETE | `/api/books/:id` | 删除图书 |
| GET | `/api/books/:id/cover` | 获取封面 |
| POST | `/api/books/:id/cover` | 上传/更换封面 |
| PUT | `/api/books/:id/progress` | 更新阅读进度 |
| GET | `/api/books/:id/content` | 获取 EPUB 目录/章节内容 |
| GET | `/api/books/:id/download` | 下载原文件 |
| GET | `/api/stats` | 统计信息 |

## 使用说明

### 导入图书
1. 点击左侧导航栏「导入图书」
2. 拖拽文件到上传区域，或点击选择文件
3. 支持批量选择多个文件同时上传
4. 系统会自动提取 EPUB 元数据（书名、作者、出版社等）

### 浏览书架
1. 点击「书架」进入图书列表
2. 切换网格/列表视图
3. 支持按时间、书名、最近阅读排序
4. 顶部搜索栏可按书名/作者搜索

### 阅读图书
1. 点击图书卡片或「阅读」按钮进入阅读器
2. 点击屏幕左右边缘翻页
3. 使用键盘 ← → 方向键翻页
4. 点击「目录」打开章节导航
5. 点击「设置」调整字体、字号、行距、主题
6. 退出时自动保存阅读进度

### 编辑图书信息
1. 在书架或详情页点击「编辑」
2. 修改书名、作者（逗号分隔）、出版社、描述、标签
3. 支持更换封面图片

## 注意事项

- 图书文件存储在 `backend/data/uploads/books/`，封面存储在 `backend/data/uploads/covers/`
- 数据库为 SQLite，位于 `backend/data/moyue.db`，可随时备份整个 `backend/data/` 目录
- `backend/data/` 为独立项目的运行时数据目录，后端二进制可随目录整体迁移部署
- PDF 格式暂不支持在线阅读，可通过详情页下载后阅读
- 首次启动会自动创建数据库表结构与数据目录

## 语音合成（TTS）配置

应用内置多种语音合成引擎，用于在线阅读朗读与有声书生成：

- **Edge TTS（云端，免费）**：微软在线语音，无需本地模型，默认开启。后端已改用 Go 原生库直连微软端点，**不再依赖 Python tts-service**，并在收到 403 时依据服务端时间自动校正本地时钟偏移并重试。
- **GPT-SoVITS / Fish-Speech / ChatTTS（本地部署）**：声音克隆或本地模型，需在 `tts.base_url` 配置自托管服务地址。

### 受限网络（403）代理配置

在国内或受限网络中，微软端点可能拒绝连接（403 / 握手失败）。可选两种解法：

1. **代理出网（推荐）**：在启动后端的环境中设置以下环境变量之一，后端会经代理连接微软：
   ```bash
   export EDGE_TTS_PROXY="http://127.0.0.1:7890"   # 优先（与 tts-service 取值一致）
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

当前完成度约 **60%**（P0 核心功能已全部实现）。如需继续开发 P1/P2 功能（批注、收藏夹、PDF 阅读器、阅读统计等），可在此代码基础上扩展。

## 开源许可证

本项目基于 **[MIT 许可证](LICENSE)** 发布。你可以自由地使用、修改、分发本软件（包括商业用途），
只需在副本中保留原始版权声明与许可声明。软件按「原样」提供，作者不承担任何担保或责任。

欢迎通过 Issue 与 Pull Request 参与贡献，详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 致谢与依赖

本项目使用了诸多优秀的开源库，包括但不限于：

- 后端：Gin、GORM、glebarez/sqlite、difyz9/edge-tts-go、pdfcpu 等（完整列表见 `backend/go.mod`）
- 前端：Vue 3、Element Plus、foliate-js、pdfjs-dist 等（完整列表见 `frontend/package.json`）

感谢这些项目的维护者与贡献者。
