# E2E：链接钓鱼（technique = `link`）

正文或落地页中的可点击 URL（`{{.URL}}`）引导打开钓鱼站。多数邮件剧本同时带 `link`；本篇以「纯链接 / 多步骤链接」为主路径。

| 项 | 内容 |
|---|---|
| 标签 | `link` |
| 推荐剧本 | `multistep_it_notice`（主）；备选任意含 `link` 的包 |
| 投递 | 邮件 SMTP → Mailpit |
| 钓鱼站 | `http://127.0.0.1:18000` |
| API | [`docs/openapi.yaml`](../openapi.yaml)（`scenario_packs`、`campaigns`、`util/send_test_email`） |
| 公共前置 | [`e2e-local-mailpit.md`](../e2e-local-mailpit.md) |

---

## 1. 准备

1. 启动 Mailpit（正式版）与识饵。  
2. 发信配置指向 `127.0.0.1:1025`。  
3. 目标组至少 1 人，建议填 **部门**（便于 breakdown）。  
4. 导入剧本：`POST /api/scenario_packs/multistep_it_notice/import`  
   - 会生成多封模板；活动可只选「确认」那封带链接的模板做单步，或配置两步。

---

## 2. 手工路径

1. **行动** → 新建：模板（含 `{{.URL}}`）、落地页、组、Mailpit 发信配置，URL = `http://127.0.0.1:18000`，立即启动。  
2. Mailpit 打开邮件 → 点击「打开确认页」类链接。  
3. 落地页加载成功。  
4. 活动结果：该人状态为 **Clicked Link**（若再提交表单则为 **Submitted Data**）。  
5. 仪表盘 / `GET /api/campaigns/breakdown`：手法列出现 `link`，对应部门有点击。

### 多步骤（可选）

导入后配置两步：步骤 0 铺垫（可无强链接）、步骤 1 确认信（含链接），步骤 1 触发选「上一步发送后」并设延迟。期望 Mailpit 先后两封；仅步骤 1 的点击计入链接成功。

---

## 3. API 冒烟要点

```bash
# 导入
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/multistep_it_notice/import

# 创建活动（字段见 OpenAPI CampaignCreate；立即投放为管理端兼容路径）
# 模拟点击：
curl -s -o /dev/null -w "%{http_code}\n" "http://127.0.0.1:18000/?rid=<RId>"
```

或：`python3 scripts/e2e_mailpit.py`（默认二维码包也含 `link`，可改脚本 `PACK_ID` 为 `multistep_it_notice` 的带链接模板名后自测）。

---

## 4. 期望事件

| 动作 | 期望状态 / 事件 |
|------|-----------------|
| 邮件发出 | Email Sent |
| 打开含追踪图的邮件 | Email Opened（若模板有追踪像素） |
| 点击 `{{.URL}}` | Clicked Link |
| 落地页提交凭证 | Submitted Data |

---

## 5. 验收清单

- [ ] Mailpit 中链接指向 `:18000` 且带 `rid`  
- [ ] 点击后结果为 Clicked Link  
- [ ] 活动 `techniques` 含 `link`（剧本导入快照）  
- [ ] breakdown 中 `link` 桶有数据  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 点了没变状态 | 是否点了真实 `rid` 链接；钓鱼服务是否在 `:18000` |
| 多步骤只收到第一封 | 检查步骤触发与 worker；延迟未到属正常 |
