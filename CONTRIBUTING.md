# 贡献指南

感谢你考虑为 **moyue_books（个人图书管理系统）** 做出贡献！

## 行为准则

请在所有交流中保持友善与尊重。我们采用常见的开源社区礼仪，
不接纳任何形式的骚扰、歧视或攻击性言论。

## 如何贡献

1. **Fork** 本仓库到你的账号。
2. 从 `main`（或 `master`）分支切出特性分支：`git checkout -b feat/your-feature`。
3. 提交改动，遵循清晰的提交信息（建议前缀 `feat:` / `fix:` / `docs:` / `chore:`）。
4. 推送分支并发起 **Pull Request**，描述改动目的与测试情况。

## 开发环境

- 后端：Go 1.17+（`backend/` 为独立 Go 模块）
- 前端：Node.js 18+，`frontend/` 使用 Vite
- 一键构建：`./build.sh`；本地开发：`./start.sh`

## 安全相关

- **请勿提交任何密钥、凭证或个人信息**。本仓库的反馈邮件账号 / SMTP 授权码通过环境变量
  `FEEDBACK_MAIL_ACCOUNT`、`FEEDBACK_MAIL_AUTH` 注入，源码中不含真实凭据。
- 如发现安全漏洞，请通过 Issue 或私下联系维护者，不要公开披露细节。

## 代码风格

- Go：使用 `gofmt` 格式化；保持现有分层（handlers / services / models / routes）。
- 前端：遵循 Vue 3 `<script setup>` 风格与现有目录结构。

## Issue 规范

提交 Issue 时请尽量包含：复现步骤、预期 / 实际结果、环境信息（系统、Go / Node 版本）。
