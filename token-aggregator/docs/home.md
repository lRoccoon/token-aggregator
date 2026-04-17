# token-aggregator

多设备 token 用量聚合服务：各设备定时采集 `ccusage` / `@ccusage/codex` / 自定义 agent 用量，push 到本服务；聚合后的今日/累计数据可通过 `/report` 查询，或由内置 pusher 定时推送到飞书自定义 URL 预览的 slot。

## 一键安装采集端

在需要采集的机器上（macOS / 主流 Linux）运行：

```bash
curl -fsSL __SERVER_URL__/install.sh | bash -s -- --token YOUR_INGEST_TOKEN
```

交互式安装（有 tty 时缺少必填项会提示输入）：

```bash
bash -c "$(curl -fsSL __SERVER_URL__/install.sh)"
```

常用参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--token` | 必填 | 本服务的 Bearer token（问运维） |
| `--device-id` | 主机名 | 设备标识 |
| `--sources` | `claude,codex` | 采集源，逗号分隔；支持 `claude` / `codex` / `hermes-agent` / 自定义 |
| `--interval` | `30` | cron 间隔（分钟） |
| `--install-dir` | `~/.local/bin` | 安装目录 |
| `--no-cron` | - | 跳过 cron |
| `--no-run` | - | 跳过首次验证运行 |

安装后文件位置：

- 采集脚本：`~/.local/bin/token-collector.sh`
- 配置：`~/.config/token-collector/config.env`
- 日志：`~/.local/share/token-collector/{collector,cron}.log`

## 依赖

| 需求 | 用途 |
|---|---|
| `bash`, `curl` | 必需 |
| `npx` (Node 18+) | 采 `claude` / `codex` 时需要 |
| `sqlite3` CLI | 采 `hermes-agent` 时需要 |
| `crontab` | 自动定时执行（可用 `--no-cron` 跳过） |

安装脚本会检测缺失的依赖并给出对应发行版的安装命令，不会擅自 `sudo`。

## 自定义 agent

在 `~/.config/token-collector/config.env` 末尾追加适配器：

```bash
SOURCES="claude,codex,my-agent"

adapter_my_agent() {
  cat <<EOF
{ "daily": [ { "date": "$(date +%F)", "total_tokens": 12345, "cost_usd": 0.12 } ] }
EOF
}
```

函数名规则：`adapter_<source 名称中的 '-' 换成 '_'>`。

## 飞书自定义 URL 预览 slot

Aggregator 可以把 `/report` 聚合后的数据定时推送到飞书 slot，替代单机脚本。启动时读取以下环境变量：

| 变量 | 默认 | 说明 |
|---|---|---|
| `LARK_SLOT_ID` | - | Slot ID；配合下一项同时配置才启用推送 |
| `LARK_SLOT_CREDENTIAL` | - | Slot Bearer 凭据 |
| `LARK_SLOT_API_URL` | `https://l.garyyang.work/api/slot/update` | Slot 更新接口 |
| `LARK_SLOT_INTERVAL` | `5m` | 推送间隔（Go duration） |
| `LARK_SLOT_TEMPLATE_PATH` | `/data/slot_template.tmpl` | 模板文件；不存在则用内置模板 |

模板采用 Go `text/template`，数据是 `/report` 返回的 Report 结构。内置默认模板：

```
已经消耗词元：今日 {{millions .TodayTokens}} / 总计 {{millions .TotalTokens}}，白赚 {{money .TotalCost}}
```

模板函数：

| 函数 | 输入 | 输出 |
|---|---|---|
| `millions` | `int64` tokens | `"XM"`（整数百万） |
| `money` | `float64` USD | `"$X.XX"` |

优化：每轮渲染后会和上次推送的文案比对，未变化则跳过 HTTP 调用，不污染 slot。

## 接口

| 路径 | 方法 | 鉴权 | 说明 |
|---|---|---|---|
| `/healthz` | GET | - | 健康检查 |
| `/ingest` | POST | Bearer | 采集上报 |
| `/report` | GET | Bearer | 聚合报告 JSON |
| `/unknown-models` | GET | Bearer | 列出未命中价格表的模型名（供扩 overrides） |
| `/install.sh` | GET | - | 安装脚本（已注入本服务地址） |
| `/collector.sh` | GET | - | 采集脚本 |

### `/ingest` 请求头

| Header | 说明 |
|---|---|
| `Authorization: Bearer <TOKEN>` | 鉴权 |
| `X-Device-Id` | 设备标识 |
| `X-Source` | 来源名（`claude` / `codex` / `hermes-agent` / 自定义） |
| `X-Format` | 解析格式：`ccusage` / `codex` / `hermes` / `standard`（默认 standard） |
| `X-Timezone` | 设备时区 |

`standard` 格式：

```json
{ "daily": [ { "date": "YYYY-MM-DD", "total_tokens": 12345, "cost_usd": 0.12 } ] }
```

### `/report` 响应示例

```json
{
  "today": "2026-04-17",
  "updated_at": 1744857600,
  "today_tokens": 48000000,
  "total_tokens": 925000000,
  "total_cost_usd": 520.31,
  "devices": {
    "macbook-why": {
      "today_tokens": 22000000,
      "total_tokens": 510000000,
      "total_cost_usd": 301.22,
      "sources": {
        "claude": { "today_tokens": 15000000, "total_tokens": 300000000, "total_cost_usd": 180.0 },
        "codex":  { "today_tokens":  7000000, "total_tokens": 210000000, "total_cost_usd": 121.22 }
      }
    }
  }
}
```

## 价格表

服务端懒加载 [LiteLLM 价格表](https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json)，24h 自动刷新，无网络时用磁盘副本兜底。

自定义覆盖：把价格写到容器内的 `/data/price_overrides.json`（同 LiteLLM schema），优先级高于 LiteLLM。
