# Lark Token Slot Automation

> 早期单机版：用 shell + cron 把本机 `ccusage` / `@ccusage/codex` 的 token 汇总写入飞书自定义 URL 预览 slot。多设备场景已被 [`token-aggregator`](../token-aggregator) 覆盖，这里保留供离线/单机用户参考。

## 文件

- `update_lark_token_slot.sh` — 拉取统计、生成文案、更新 slot
- `install_cron.sh` — 安装每 30 分钟执行一次的用户 cron
- `tests/test_update_lark_token_slot.sh` — 文案格式测试

## 配置

脚本从环境变量或配置文件读取 slot 凭据，路径默认为 `~/.config/lark-token-slot/config.env`（可用 `LARK_TOKEN_SLOT_CONFIG` 覆盖）：

```bash
# ~/.config/lark-token-slot/config.env
LARK_SLOT_ID=slot_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
LARK_SLOT_CREDENTIAL=cred_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# 可选
# LARK_SLOT_API_URL=https://l.garyyang.work/api/slot/update
# LARK_SLOT_INFO_URL=https://l.garyyang.work/api/slot/info
# CCUSAGE_BIN=/home/linuxbrew/.linuxbrew/bin/npx
```

申请 slot 与凭据：参考你使用的自定义 URL 预览服务文档（例如 `l.garyyang.work`）。

## 使用

```bash
# 一次性运行
./update_lark_token_slot.sh

# 只打印文案不推送
./update_lark_token_slot.sh --dry-run

# 装 cron（每 30 分钟执行）
./install_cron.sh
```

## URL 预览模板示例

```handlebars
{{slot id="<YOUR_SLOT_ID>"}} · {{time_now f="HH:mm"}}
```

预览链接（把 `<YOUR_SLOT_ID>` 替换为实际 slot id 后整体 URL-encode 即可）：

```
https://l.garyyang.work/?t=<url-encoded-template>
```

## 日志

- `logs/update.log` — 手动执行日志
- `logs/cron.log` — cron 执行日志
