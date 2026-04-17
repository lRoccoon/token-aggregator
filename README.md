# token-aggregator

多设备 token 消耗聚合服务。中心节点跑在 VPS 上用 Docker 部署，各设备通过 `curl | bash` 一键安装采集脚本，定时把 ccusage / @ccusage/codex / 自定义 agent 的用量 push 到中心。

## 架构

```
[设备 A: claude+codex] ─┐
[设备 B: codex]        ─┼─> POST /ingest ─> [VPS: token-aggregator + SQLite]
[设备 C: hermes-agent] ─┘                              │
                                                       └─> GET /report
```

- Push 模型：各设备不需要开放端口
- 每个设备每 30 分钟跑一次采集；ccusage/codex 离线模式，秒级完成
- 采集端依赖：`bash` + `curl` 为核心，`npx`（Node）仅在采集 claude/codex 时需要
- 新增 agent：在 `config.env` 中定义 `adapter_<name>` 函数并加入 `SOURCES`

## 一、部署中心节点（VPS）

```bash
# 在 VPS 上 clone 仓库后进入仓库根目录
export INGEST_TOKEN="$(openssl rand -hex 24)"    # 客户端鉴权 token
export PUBLIC_URL="https://tokens.example.com"   # 反向代理后的公网地址

docker compose up -d --build
```

或者直接拉 CI 产出的镜像：

```bash
docker pull ghcr.io/<owner>/<repo>:latest
# 按需在 docker-compose.yml 里把 build: . 换成 image: ghcr.io/<owner>/<repo>:latest
```

建议前置 nginx/caddy 做 TLS。服务容器监听 8080，数据写入 `./data/usage.db`。

### 接口

| 路径 | 方法 | 鉴权 | 说明 |
|---|---|---|---|
| `/healthz` | GET | 否 | 健康检查 |
| `/ingest` | POST | Bearer | 采集上报（见下） |
| `/report` | GET | Bearer | 聚合报告 JSON |
| `/install.sh` | GET | 否 | 安装脚本（embed 的 SERVER_URL 自动替换） |
| `/collector.sh` | GET | 否 | 采集脚本（供 install.sh 下载） |

### /ingest 请求协议

请求体是**原始的 ccusage / codex JSON**，由 header 决定如何解析：

| Header | 说明 |
|---|---|
| `Authorization: Bearer <TOKEN>` | 鉴权 |
| `X-Device-Id` | 设备标识（如 `macbook-why`、`vps-1`） |
| `X-Source` | 用量来源（`claude` / `codex` / `hermes-agent` / 自定义） |
| `X-Format` | 解析格式：`ccusage` / `codex` / `standard`（缺省走 standard） |
| `X-Timezone` | 设备时区（记录用，服务端 `/report` 的 today 以自己 TIMEZONE 为准） |

`standard` 格式（用于自定义 agent）：
```json
{ "daily": [ { "date": "YYYY-MM-DD", "total_tokens": 12345, "cost_usd": 0.12 } ] }
```

### /report 响应示例

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

## 二、安装采集端

```bash
curl -fsSL https://tokens.example.com/install.sh | bash -s -- \
  --token <INGEST_TOKEN> \
  --device-id my-macbook \
  --sources claude,codex
```

支持参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--token` | 必填 | 中心节点的 Bearer token |
| `--server` | 下载时自动注入 | 手动覆盖 |
| `--device-id` | 主机名 | 设备标识 |
| `--sources` | `claude,codex` | 逗号分隔的采集源 |
| `--interval` | 30 | cron 间隔（分钟） |
| `--install-dir` | `~/.local/bin` | 安装目录 |
| `--no-cron` | - | 跳过 cron 安装 |
| `--no-run` | - | 跳过首次验证运行 |

安装后：
- 脚本：`~/.local/bin/token-collector.sh`
- 配置：`~/.config/token-collector/config.env`
- 日志：`~/.local/share/token-collector/collector.log`, `cron.log`

## 三、添加自定义 agent

在 `~/.config/token-collector/config.env` 末尾追加函数并把名字加入 `SOURCES`：

```bash
# 追加到 SOURCES
SOURCES="claude,codex,my-agent"

# 定义 adapter：函数名 = adapter_<source 中的 '-' 换成 '_'>
adapter_my_agent() {
  # 从任意数据源生成 standard JSON
  cat <<EOF
{
  "daily": [
    { "date": "$(date +%F)", "total_tokens": 12345, "cost_usd": 0.12 }
  ]
}
EOF
}
```

## 四、hermes-agent

内置适配器直接读取 `~/.hermes/state.db`（hermes-agent 自己的会话库），按 (日, model) 聚合 token 明细上报；**cost 由服务端用 LiteLLM 价格表算**，因为 hermes 本身通常没写入 cost。依赖：`sqlite3` CLI（macOS 自带；Linux 大多数发行版有）。

可选覆盖项（`config.env`）：
```bash
HERMES_STATE_DB="${HOME}/.hermes/state.db"   # 非默认路径时设置
HERMES_DAYS=30                               # 回溯窗口，默认 30 天
```

### LiteLLM 价格表

服务端懒加载 https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json 并缓存到 `/data/litellm_prices.json`，24h 自动刷新。无网络时用磁盘副本兜底。

**本地价格覆盖**：把自定义价格写到 `/data/price_overrides.json`（与 LiteLLM 同 schema，key 是模型名），优先级高于 LiteLLM。例子：
```json
{
  "gpt-5.4": {
    "input_cost_per_token": 0.00000125,
    "output_cost_per_token": 0.000010,
    "cache_read_input_token_cost": 0.000000125,
    "cache_creation_input_token_cost": 0.00000156
  }
}
```

**查看未识别的模型**：`GET /unknown-models`（Bearer 鉴权）列出命中过但没价格的模型名和次数，用来决定哪些要加到 overrides。

## 五、飞书 Slot 定时推送（可选）

Aggregator 可以把聚合后的 `/report` 结果定时推送到飞书自定义 URL 预览的 slot（替代单机 `lark-token-slot` 脚本）。未配置凭据时自动禁用。

### 环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `LARK_SLOT_ID` | - | Slot ID；配合 `LARK_SLOT_CREDENTIAL` 同时提供才启用推送 |
| `LARK_SLOT_CREDENTIAL` | - | Slot 凭据（Bearer token） |
| `LARK_SLOT_API_URL` | `https://l.garyyang.work/api/slot/update` | Slot 更新接口 |
| `LARK_SLOT_INTERVAL` | `5m` | 推送间隔，Go duration 格式 |
| `LARK_SLOT_TEMPLATE_PATH` | `<data>/slot_template.tmpl` | 模板文件路径（相对容器内挂载的 `/data`） |

启用后容器启动会立即推送一次，之后按间隔循环。失败只记日志，不影响主服务。

### 模板

使用 Go `text/template`，数据对象是 `/report` 返回的 `Report` 结构。若 `LARK_SLOT_TEMPLATE_PATH` 文件不存在则使用内置默认模板：

```
已经消耗词元：今日 {{millions .TodayTokens}} / 总计 {{millions .TotalTokens}}，白赚 {{money .TotalCost}}
```

内置模板函数：

| 函数 | 输入 | 输出 |
|---|---|---|
| `millions` | `int64` tokens | `"XM"`（整数百万） |
| `money` | `float64` USD | `"$X.XX"` |

自定义例子（放到 `./data/slot_template.tmpl`）：

```
今日 {{millions .TodayTokens}} · 总计 {{millions .TotalTokens}} · 花了 {{money .TotalCost}}
{{range $dev, $agg := .Devices}}  [{{$dev}}] {{millions $agg.TodayTokens}}
{{end}}
```

热加载：每次推送都会重读模板文件，改完无需重启容器。

## 本地开发

```bash
go run .  --addr :8080 --db ./dev.db --token dev-token --public-url http://localhost:8080
```

## 许可

[MIT](./LICENSE)

