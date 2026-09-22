# 管理端前端走查问题记录

日期：2026-09-21  
环境：本机 `http://127.0.0.1:13333/`（Mailpit + 识饵已启动）  
方式：Cursor 内置浏览器（英文 `Accept-Language`）全页点击走查 + Playwright（`zh-CN` / `en-US`）对照审计  
账号：`admin`

## 结论摘要

主流程可登录、导航、查看工作台 / 演练 / 场景库 / 人群 / 报告 / 发信与通道 / 设置等页面。部门×手法矩阵、场景库中文文案、分组「部门」字段、报告页 Highcharts 均可用。中英文切换随 `Accept-Language` 正常。

下列缺陷已于同日修复（见下文「已修复」）。

## 已修复

### P1 — 活动结果页浏览器标题仍写 Gophish

- **修复**：`campaign_results.js` 改为 `$("title").text(c.name + " - " + t("brand.name"))`。

### P1 — 设置 / 用户页密码强度脚本运行时报错

- **修复**：`zxcvbn.js` 并入 `vendor.min.js`；`passwords.js` 去掉 `import`，使用全局 `zxcvbn`，并对 DOM 做空值守卫。

### P2 — 场景库标签重复

- **修复**：`scenarios.js` 对 `category` 与 `techniques` 渲染前去重；难度标签单独保留。

### P2 — 发起演练 / 测试邮件表单无部门字段

- **修复**：`campaigns.html` / `sending_profiles.html` 增加 `to_department`；对应 `sendTestEmail()` 提交 `department`。

### P2 — 列表图标按钮缺少无障碍名称

- **修复**：演练 / 人群 / 模板 / 落地页 / 发信配置 / 发送通道 / 用户 / Webhooks 等操作按钮补充 `aria-label`（及 i18n `title`），避免 Bootstrap tooltip 挪走 `title` 后无障碍名丢失。新增 `js.impersonate`。

## 观察（非缺陷或已知限制）

| 项 | 说明 |
|---|---|
| 管理端需 HTTP | `config.json` 中 `use_tls: false`；访问 `https://127.0.0.1:13333` 会失败，属配置预期。 |
| 语言随浏览器 | 内置浏览器英文导航（Workbench…）；`Accept-Language: zh-CN` 时为「工作台 / 演练 / 场景库…」。无手动切换器，符合产品约定。 |
| 报告页 | 趋势图与部门对比图有数据；文案写明完整 brief 导出为后续版本；部分入口按钮偏「占位/跳转」属产品阶段。 |
| 工作台 | 易中招率、部门对比、部门×手法矩阵正常；活动结果页有难度/手法标签与部门点击率。 |
| 场景库 | 14 个内置包中文标题与说明正常；「Launch」可进发起演练。 |

## 已走查页面

登录 → 工作台 → 演练（含 `?new=1` 弹窗）→ 活动结果 → 场景库 → 人群 → 报告 → 邮件模版 → 钓鱼页面 → 发信配置 → 发送通道 → 设置 → 用户管理 → Webhooks。
