# personal-automation

个人日常自动化脚本和服务的集合仓库，围绕 AI 编码助手（Claude / Codex / 自建 agent）的 token 用量统计、飞书 URL 预览 slot 更新等场景。

## 目录

| 路径 | 说明 |
|---|---|
| [`token-aggregator/`](./token-aggregator) | 多设备 token 用量聚合服务（Go + SQLite，Docker 部署）。支持采集 `ccusage` / `@ccusage/codex` / 自定义 agent，并可定时把汇总数据推送到飞书自定义 URL 预览 slot。|
| [`lark-token-slot/`](./lark-token-slot) | 早期单机版：shell 脚本 + cron 直接写飞书 slot。功能已被 `token-aggregator` 覆盖，保留供离线/单机场景参考。|

## 快速上手

部署聚合服务（VPS）：

```bash
cd token-aggregator
export INGEST_TOKEN="$(openssl rand -hex 24)"
export PUBLIC_URL="https://tokens.example.com"
docker compose up -d --build
```

或直接拉预构建的镜像：

```bash
docker pull ghcr.io/roccoon/personal-automation/token-aggregator:latest
```

采集端一键安装：

```bash
bash -c "$(curl -fsSL https://tokens.example.com/install.sh)"
```

详细使用说明见 [`token-aggregator/README.md`](./token-aggregator/README.md) 或部署后访问根路径 `/`（浏览器渲染 HTML，curl 返回 Markdown）。

## 构建与发布

`.github/workflows/build.yml` 在 push 到 `main` 或打 `v*` 标签时自动：

1. 跑 `go vet` + `go test ./...`
2. 构建 `linux/amd64` + `linux/arm64` 镜像
3. 推送到 `ghcr.io/<owner>/personal-automation/token-aggregator`（tag 形如 `latest` / `v1.2.3` / `sha-abcdef0`）

## 许可

[MIT](./LICENSE)
