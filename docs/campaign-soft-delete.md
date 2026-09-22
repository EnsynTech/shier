# 识饵演练软删除与回收站

| 项 | 内容 |
|---|---|
| 状态 | **已批准，按本文落地**（2026-09-21） |
| 日期 | 2026-09-21 |
| 范围 | 演练（Campaign）删除语义、回收站、恢复、彻底删除。不含模板 / 人群 / 通道的回收站，不含定时自动清理 |
| 关联 | 产品定位为乙方交付工具；演练结果是给甲方看的证据。AI 策略见 `docs/ai-api-orchestration-decision.md`。列表交互见 `docs/ui-design.md` §5.2 |
| 结论摘要 | **默认删除改为软删除**：结果、时间线、步骤保留，演练移出日常列表并停止投放。管理后台增加**回收站**可恢复。**彻底删除**仅针对回收站中的记录，且请求体必须带确认字段。本期不做保留期自动清理 |

本文件为产品与实现规格。代码须对照本文，而不是沿用上游 Gophish 的物理删除语义。

---

## 1. 要回答的问题

1. 顾问或甲方培训负责人误删一场已跑完的演练后，如何把结果拿回来？
2. 「删除」与现有「完成 / 归档」如何分工，避免把停演当成销毁？
3. 已删演练会不会继续发信、继续记点击，或污染工作台 / 报告？
4. 外部智能体能否像开演一样，轻易把证据彻底抹掉？

---

## 2. 产品定位（决策基线）

| 项 | 约定 |
|---|---|
| 使用者 | 乙方交付顾问为主；甲方主要查看演练成果 |
| 演练数据 | 交付证据，不是可随手清空的缓存 |
| 误操作 | 必须能从产品内恢复，不能只依赖整库备份 |
| 发送权 | 与开演策略一致：停发可以自动做；销毁证据不能轻易自动化 |

---

## 3. 现状基线（代码事实）

| 层 | 现状 |
|---|---|
| 删除 API | `DELETE /api/campaigns/{id}` → `models.DeleteCampaign` |
| 物理清理 | 立刻删除 `results`、`events`、`mail_logs`、`delivery_logs`、`campaign_steps` 以及 `campaigns` 行 |
| 「归档」 | 仅 UI：状态为 `Completed` 的演练进第二张表。归档行仍带删除按钮，点了同样物理删除 |
| 完成演练 | `GET /api/campaigns/{id}/complete` 停发并标 `Completed`，**保留**结果 |
| 查询 | `GetCampaignSummaries`、`GetCampaignBreakdown` 等使用 `Table("campaigns")`，不过滤删除标记（今日也没有该列） |
| 钓鱼站 | `GetCampaign` 找不到或状态为 `Completed` 时不再记事件 |
| 用户删除 | `DeleteUser` 对其名下每场演练调用 `DeleteCampaign` |

界面虽提示 “This can't be undone”，但没有任何恢复路径。

---

## 4. 三层操作语义

| 操作 | 用户感知 | 系统行为 |
|---|---|---|
| 完成 / 归档 | 演练结束，仍在「已归档」里可看 | 停发；状态 `Completed`；数据保留 |
| 删除 | 从进行中 / 已归档消失 | **软删除**：写入 `deleted_at`；停发；结果保留；进入回收站 |
| 彻底删除 | 回收站里再确认一次 | **物理删除**，与今日 `DeleteCampaign` 相同，不可恢复 |

原则：

1. 日常「删除」= 移入回收站，**不是**销毁。
2. 软删除必须先停发，避免回收站里的演练继续触达目标。
3. 恢复**不重新开演、不重建待发队列**。恢复后出现在「已归档」（状态为 `Completed`）。若要再跑一轮，使用现有复制流程新建演练。
4. 彻底删除只接受已在回收站中的记录，防止一条 API 把进行中的演练直接抹掉。

---

## 5. 数据模型

### 5.1 字段

`campaigns` 增加可空时间列：

| 列 | 类型 | 含义 |
|---|---|---|
| `deleted_at` | `DATETIME NULL` | `NULL` = 可见；非空 = 在回收站 |

索引：`(user_id, deleted_at)`，供列表与回收站查询。

结构体使用 GORM v1 约定字段名 `DeletedAt *time.Time`，以便 `Find` / `First` 自动排除已删行。所有 `Table("campaigns")` 或 `JOIN campaigns` 的查询**仍须显式** `deleted_at IS NULL`（`Table()` 不会走模型回调）。

`CampaignSummary` 对回收站列表输出 `deleted_at`。

### 5.2 迁移

- SQLite：`db/db_sqlite3/migrations/20260921120000_1.0.5_campaign_deleted_at.sql`
- MySQL：`db/db_mysql/migrations/20260921120000_1.0.5_campaign_deleted_at.sql`

已有演练 `deleted_at` 为 `NULL`，行为与今日可见列表一致。

### 5.3 软删除时对关联表的处理

| 表 | 软删除 | 恢复 | 彻底删除 |
|---|---|---|---|
| `campaigns` | 写 `deleted_at` | 清空 `deleted_at` | 删行 |
| `results` / `events` / `campaign_steps` | **保留** | 不动 | 删行 |
| `mail_logs` / `delivery_logs` | **删除**（停发） | 不重建 | 若仍有残留则删 |

软删除时若状态不是 `Completed`，同时把状态改为 `Completed` 并写入 `completed_date`（仅当原先为空或需要标记结束时写入当前 UTC 时间）。这样即便查询漏过滤 `deleted_at`，worker 也不会按 `Queued` / `In progress` 再投。

---

## 6. API

认证、写权限与现有 `/api/` 相同（API Key + `modify_objects`）。

### 6.1 行为变更

| Method | Path | 变更后 |
|---|---|---|
| GET | `/api/campaigns/` | 只返回未删除演练 |
| GET | `/api/campaigns/summary` | 只返回未删除演练 |
| GET | `/api/campaigns/breakdown` | 聚合时排除已删演练 |
| GET | `/api/campaigns/{id}` | 已删返回 404（与「不存在」相同，避免探测） |
| GET | `/api/campaigns/{id}/results` | 同上 |
| GET | `/api/campaigns/{id}/summary` | 同上 |
| GET | `/api/campaigns/{id}/complete` | 已删返回 404；未删行为不变 |
| POST | `/api/campaigns/{id}/launch` | 已删返回 404 |
| DELETE | `/api/campaigns/{id}` | **改为软删除**；成功文案说明已移入回收站 |

### 6.2 新增

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/campaigns/trash` | 当前用户回收站摘要列表（结构同 `CampaignSummaries`，条目含 `deleted_at`） |
| POST | `/api/campaigns/{id}/restore` | 从回收站恢复。未删或不存在：404 |
| DELETE | `/api/campaigns/{id}/purge` | 彻底删除。必须已在回收站 |

彻底删除请求体：

```json
{ "confirm": "purge" }
```

缺字段或值不是 `purge` 时：HTTP 403，`message` 为 `purge_confirmation_required`（与开演缺令牌同一类保护）。

### 6.3 AI 策略

| 功能 | AI 策略 |
|---|---|
| 软删除 `DELETE /api/campaigns/{id}` | 允许配置（移入回收站） |
| 列出回收站 / 恢复 | 允许配置 |
| 彻底删除 `.../purge` | **禁止作为默认编排动作**。接口仍供管理后台使用，但必须带 `confirm`；`GET /api/ai/schema` 的 `forbidden` 列出该路径 |

外部模型需要「清掉一场演练」时，只应调用软删除。销毁证据必须由人在回收站确认。

### 6.4 错误

| 条件 | HTTP | message |
|---|---|---|
| 未删记录上 restore / purge | 404 | `Campaign not found` |
| 已删记录上 GET 详情 / 开演 / 完成 | 404 | `Campaign not found` |
| purge 未带确认 | 403 | `purge_confirmation_required` |

不向客户端区分「没有这场演练」和「在别人的回收站里」。

---

## 7. 投放、钓鱼站、报表

| 路径 | 要求 |
|---|---|
| Worker `GetQueuedCampaigns` | 模型 `Find` 自动排除 `deleted_at` 非空行；软删除前已改 `Completed` 并清队列 |
| 钓鱼点击 / 打开 / 提交 | `GetCampaign` 找不到已删演练 → 不记事件（与今日无效 rid 类似） |
| 工作台、报告、部门 × 手法汇总 | 只统计未删除演练 |
| 删用户 | 对该用户**含回收站**的全部演练调用彻底删除（不经过 `confirm`，属账号销毁） |

---

## 8. 管理后台

演练列表 `/campaigns` 三个 Tab：

1. 进行中（非 `Completed`、未删除）
2. 已归档（`Completed`、未删除）
3. 回收站（`deleted_at` 非空）

| Tab | 行操作 |
|---|---|
| 进行中 | 查看结果、复制、归档（停发，进入已归档） |
| 已归档 | 查看结果、复制、删除（软删除，确认文案写明可在回收站恢复） |
| 回收站 | 恢复；彻底删除（二次确认，文案写明结果与时间线将消失且无法恢复） |

回收站不提供「查看完整结果页」入口（结果页仍走 `GET /api/campaigns/{id}`，已删为 404）。需要看结果时先恢复。

工作台删除按钮走同一软删除 API；成功后刷新列表即可。

结果页删除按钮同样软删除，成功后回到 `/campaigns`。

中英文文案同步；无手动语言开关。

---

## 9. 明确不做什么（本期）

1. **不做**按天数自动清空回收站（可在后续加配置项）。
2. **不**在恢复时重建 `mail_logs` / `delivery_logs`，不把状态改回 `Draft` / `Queued` / `In progress`。
3. **不**给模板、人群、页面、通道做回收站。
4. **不**把整库备份或导出文件当作本问题的产品解。
5. **不**在钓鱼站对已删演练返回特殊提示页（避免暴露这是演练）。

---

## 10. 实现清单

| 项 | 位置 |
|---|---|
| 迁移 | `db/db_sqlite3/migrations/`、`db/db_mysql/migrations/` |
| 模型 | `models/campaign.go`：`DeletedAt`；`DeleteCampaign` 改为软删除；新增 `RestoreCampaign`、`PurgeCampaign`、`GetDeletedCampaignSummaries`；所有 `Table`/`JOIN` 列表过滤 |
| 用户删除 | `models/user.go`：卸载时彻底删除含回收站演练 |
| API | `controllers/api/server.go`、`controllers/api/campaign.go` |
| AI schema | `controllers/api/ai.go`：forbidden + `purge_confirmation_required` |
| OpenAPI | `docs/openapi.yaml` |
| AI 决策表 | `docs/ai-api-orchestration-decision.md` §4.2 |
| UI | `templates/campaigns.html`、`static/js/src/app/campaigns.js`、`campaign_results.js`、`dashboard.js`、`gophish.js` |
| i18n | `i18n/locales/en.json`、`zh.json` |
| 测试清理 | `models/models_test.go` 对 `Campaign` 使用 `Unscoped().Delete`，否则软删除会污染后续用例 |
| 测试 | 软删除隐藏、恢复可见、purge 物理删除、breakdown 排除已删、邮件队列被清 |

---

## 11. 验收

- 删除一场进行中或已归档演练后：列表与工作台、报告中消失；回收站可见；`results` 行仍在库中；待发队列为空。
- 恢复后：出现在已归档；结果页可打开；不会自动再发信。
- 彻底删除：无 `confirm` 返回 403；带 `confirm` 后库中无该 `campaign_id` 的结果 / 事件 / 步骤。
- 已删演练的钓鱼链接不再写入新事件。
- 中文、英文界面回收站文案完整。
