# 识饵本地端到端测试（Mailpit）

本文说明如何在本机用 [Mailpit](https://github.com/axllent/mailpit) 作为 SMTP 收件箱，走通「配置 → 发信 → 打开/点击 → 看结果」的端到端路径。适用于功能验收与回归，不依赖真实外发邮件。

**API 完整规格不在本文维护**，见 [`docs/openapi.yaml`](openapi.yaml)。

**按钓鱼手法拆分的 E2E**（链接 / 附件 / 二维码 / 短信 / IM / 日历 / 回复）见 [`docs/e2e/README.md`](e2e/README.md)。

| 项 | 内容 |
|---|---|
| 环境 | macOS 本机；识饵管理端 `:13333`，钓鱼站 `:18000` |
| 邮件捕获 | Mailpit SMTP `:1025`，Web UI `:8025` |
| 前置 | 已安装正式版 Mailpit（含前端资源）；识饵已编译且数据库已迁移 |
| API 权威 | [`docs/openapi.yaml`](openapi.yaml) |
| 分手法 E2E | [`docs/e2e/README.md`](e2e/README.md) |

---

## 1. 服务启动

### 1.1 Mailpit（必须用正式版）

不要使用 `go install` 得到的 `dev` 二进制：该构建缺少 `/dist/app.js`、`/dist/app.css`，浏览器会显示空白页。

推荐 Homebrew：

```bash
# 若需代理（示例为本机 7890）
export http_proxy=http://127.0.0.1:7890 https_proxy=http://127.0.0.1:7890

brew install mailpit
/opt/homebrew/opt/mailpit/bin/mailpit \
  --smtp 127.0.0.1:1025 \
  --listen 127.0.0.1:8025 \
  --disable-version-check
```

自检：

```bash
curl -s http://127.0.0.1:8025/api/v1/info   # Version 应为 vX.Y.Z，不是 dev
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8025/dist/app.js  # 200
open http://127.0.0.1:8025/
```

### 1.2 识饵

```bash
cd /path/to/gophish
go build -o gophish .
./gophish
```

默认监听见 `config.json`：管理端 `0.0.0.0:13333`，钓鱼站 `0.0.0.0:18000`。浏览器打开 http://127.0.0.1:13333/ 登录。

---

## 2. 一次性配置

### 2.1 发信配置（SMTP → Mailpit）

管理端 **发信配置** → 新建，或确认已有 **Mailpit Local**：

| 字段 | 值 |
|---|---|
| 名称 | `Mailpit Local` |
| 主机 | `127.0.0.1:1025` |
| SMTP From | 任意演练域邮箱，如 `drill@shier.local` |
| 用户名 / 密码 | 留空 |
| 忽略证书错误 | 可勾选 |

API 自检（将 `API_KEY` 换成当前用户密钥，勿写入仓库）：

```bash
curl -s -X POST "http://127.0.0.1:13333/api/util/send_test_email" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "victim@example.com",
    "first_name": "测试",
    "last_name": "用户",
    "smtp": {"name": "Mailpit Local"}
  }'
```

期望：返回 `"success": true`；Mailpit UI 出现主题为默认测试信或指定模板的邮件。

### 2.2 目标组（含部门）

**用户组** → 新建组，至少 2 人，并填写 **部门**（独立字段，不是「职务」）：

| 邮箱 | 名 | 部门 |
|---|---|---|
| `fin1@example.com` | 甲 | 财务 |
| `it1@example.com` | 乙 | IT |

也可用 CSV（表头支持 `Department` 或 `部门`）：

```csv
First Name,Last Name,Email,Phone,Position,Department
甲,,fin1@example.com,,,财务
乙,,it1@example.com,,,IT
```

### 2.3 导入剧本

**邮件模板** 或 **落地页** → **从模板库导入**。建议先导入一个带链接的包，例如「账号异常扫码验证」或「IT 维护多步骤确认」。

确认筛选器（手法 / 行业 / 难度）可用，导入后生成可编辑的模板与落地页。

---

## 3. 主路径：邮件演练 E2E

按顺序执行并勾选结果。

### 3.1 创建并启动活动

1. **行动** → 新建。
2. 选择刚导入的模板、落地页、目标组、发信配置 **Mailpit Local**。
3. 钓鱼 URL 填：`http://127.0.0.1:18000`
4. 启动时间选「立即」并启动（管理端人工确认发送）。

### 3.2 收信（Mailpit）

1. 打开 http://127.0.0.1:8025/
2. 应看到发往组内目标的邮件；发件人与 **SMTP From** 一致。
3. 打开正文，确认链接指向 `http://127.0.0.1:18000/...`（含收件人 rid）。

### 3.3 打开 / 点击 / 提交

任选一封：

1. 在 Mailpit 中打开 HTML，或复制链接到浏览器访问（记为「打开」相关事件，视模板是否含追踪图而定）。
2. 点击钓鱼链接 → 落地页打开；活动结果中该人状态变为 **Clicked Link**（或等价中文展示）。
3. 若落地页开启凭证捕获：提交任意表单 → 状态变为 **Submitted Data**。

### 3.4 结果页

打开该活动的 **结果** 页，确认：

- 结果表有 **部门** 列，与组内填写一致。
- **按部门点击率** 表：财务 / IT（或「未填写」）行人数与点击合理。
- 若模板来自剧本库：页头有难度 / 手法 / 行业标签徽章。

### 3.5 仪表盘交叉报表

打开 **仪表盘**，在饼图下方查看 **部门 × 手法点击率** 矩阵：

- 至少出现本活动涉及的部门与手法列。
- 单元格为点击率；无数据为「—」。
- API 对照：`GET /api/campaigns/breakdown`（需 API Key）。

同一活动若有多种手法标签，同一结果会计入多个手法桶（产品约定，见界面说明文案）。

---

## 4. 扩展路径（按需）

各手法的完整步骤与验收清单见 **[`docs/e2e/`](e2e/README.md)**，勿在本文件重复维护。摘要：

| 场景 | 文档 |
|---|---|
| 链接 / 多步骤 | [e2e-technique-link.md](e2e/e2e-technique-link.md) |
| 附件 Office | [e2e-technique-attachment.md](e2e/e2e-technique-attachment.md) |
| 二维码 | [e2e-technique-qr.md](e2e/e2e-technique-qr.md) |
| 日历 .ics | [e2e-technique-calendar.md](e2e/e2e-technique-calendar.md) |
| 回复 + IMAP | [e2e-technique-reply.md](e2e/e2e-technique-reply.md) |
| 短信 HTTP | [e2e-technique-sms.md](e2e/e2e-technique-sms.md) |
| IM Webhook | [e2e-technique-im.md](e2e/e2e-technique-im.md) |

---

## 5. API 与自动化（可选）

识饵管理端 **完整 REST API**（含请求/响应结构、AI 编排、活动 / 组 / 模板 / 发信配置 / 通道 / 剧本库 / IMAP 等）以机器可读规格为准：

- **[`docs/openapi.yaml`](openapi.yaml)**（OpenAPI 3.0，本地默认 `http://127.0.0.1:13333`）
- 发送策略与「草稿 vs 开演」决策见 [`docs/ai-api-orchestration-decision.md`](ai-api-orchestration-decision.md)

下文仅列与 **Mailpit 本机冒烟** 相关的几条调用，便于对照收件箱；完整路径与字段请查 OpenAPI，勿在此文档复制维护。

认证：`Authorization: Bearer $API_KEY` 或 Query `api_key`（密钥勿写入仓库）。

```bash
# 测试发信 → 应出现在 Mailpit
curl -s -X POST "http://127.0.0.1:13333/api/util/send_test_email" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"email":"victim@example.com","first_name":"测试","last_name":"用户","smtp":{"name":"Mailpit Local"}}'

# 部门 × 手法交叉（仪表盘同源）
curl -s -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/campaigns/breakdown

# Mailpit 侧收件数（非识饵 API）
curl -s http://127.0.0.1:8025/api/v1/messages | python3 -c \
  'import sys,json; print(json.load(sys.stdin)["total"])'
```

活动 **真正开演** 须管理端确认，或按 OpenAPI 中 AI 路径携带一次性 `confirm_token`；勿对生产环境用无确认脚本群发。

---

## 6. 验收清单

- [ ] Mailpit UI 非空白（`Version` 非 `dev`，`/dist/app.js` 为 200）
- [ ] 测试邮件或活动邮件进入 Mailpit
- [ ] 点击链接后结果页状态与部门统计更新
- [ ] 仪表盘出现部门 × 手法矩阵（有标签的活动）
- [ ] 剧本库可按手法 / 行业 / 难度筛选并导入

---

## 7. 常见问题

| 现象 | 处理 |
|---|---|
| Mailpit 空白页 | 停掉 `go install` 的 `dev` 进程；改用 `brew` 正式版并重启 |
| 识饵连不上 SMTP | 确认 Mailpit 在 `127.0.0.1:1025`；发信配置主机写成 `127.0.0.1:1025` |
| 邮件有但点链接无效 | 活动 URL 是否为 `http://127.0.0.1:18000`；钓鱼服务是否在监听 |
| 部门统计全是「未填写」 | 组目标是否填了 **部门** 字段；仅改职务不会写入部门 |
| 矩阵无手法列 | 手工模板无标签；请用剧本库导入，或确认活动快照了 `techniques` |
| 重启 Mailpit 后邮件消失 | 默认使用临时库；需要持久化时为 Mailpit 指定 `--database` 路径 |

---

## 8. 建议最小回归集（每次发版前）

1. Mailpit 测试信一封。  
2. 导入一个链接类剧本 + 两组部门目标 → 启动活动 → 一人点击。  
3. 结果页部门表 + 仪表盘矩阵各看一眼。  

约 10–15 分钟可完成。

自动化（本机 Mailpit + 识饵已启动时）：

```bash
python3 scripts/e2e_mailpit.py
```

脚本会建组、导入剧本、启动活动、模拟打开/点击，并打印 PASS/FAIL 与 breakdown；完整 API 见 `docs/openapi.yaml`。
