# E2E：回复钓鱼（technique = `reply`）

引导用户 **直接回复** 诱饵邮件；识饵通过 **IMAP 监控** 收件箱检测回复，结果标记为已回复。不仅依赖点击链接。

| 项 | 内容 |
|---|---|
| 标签 | `reply` |
| 推荐剧本 | `reply_meeting_confirm` |
| 投递 | 邮件 → Mailpit（发信）+ **真实或测试 IMAP**（收回复） |
| API | OpenAPI：`/api/imap/`、`/api/imap/validate`、发信配置自定义头 Reply-To |
| 公共前置 | Mailpit 可验证「发出」；回复检测必须另配 IMAP |

---

## 1. 准备

1. **发信**：Mailpit SMTP，活动能发出邮件。  
2. **收回复**：在识饵 **设置 → IMAP** 配置一台邮箱（测试邮箱即可）：主机、端口、用户、密码、文件夹；`POST /api/imap/validate` 成功。  
3. **发信配置**：为 SMTP 配置增加 **Reply-To** 指向该 IMAP 监控地址（管理端有「使用 IMAP 监控地址作为 Reply-To」时可一键写入）。  
4. 导入 `reply_meeting_confirm`。

> Mailpit 默认只做 SMTP 捕获，**不会**把用户的回复送进 IMAP。本机完整回复链需要：另用一封可 IMAP 登录的信箱收 Reply-To，或用支持收信的测试栈。可分两段验收：① Mailpit 确认发出与文案引导回复；② IMAP 环境确认「已回复」状态。

---

## 2. 手工路径

### 段 A：发出（Mailpit）

1. 启动活动（模板引导「请回复本邮件确认」）。  
2. Mailpit 可见邮件；检查 Reply-To 头为监控邮箱。  

### 段 B：回复检测（IMAP）

1. 从任意客户端向 **Reply-To / From 可回线程** 回复该邮件（按 IMAP 监控匹配规则，通常需 Message-ID / 主题 / 收件人关联）。  
2. 等待 IMAP 轮询（见产品默认间隔）。  
3. 活动结果：**Replied / 已回复** 勾选或状态更新。  

---

## 3. API 冒烟要点

```bash
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/reply_meeting_confirm/import

# 读写 IMAP、validate：见 openapi.yaml tags IMAP
```

---

## 4. 期望事件

| 动作 | 期望 |
|------|------|
| 邮件发出 | Email Sent（Mailpit 可见） |
| 用户回复被 IMAP 抓到 | Reported/Replied 语义中的 **已回复**（`replied=true`） |
| 活动标签 | `reply` |

---

## 5. 验收清单

- [ ] IMAP validate 成功  
- [ ] 发信配置 Reply-To = 监控邮箱  
- [ ] Mailpit（或真实收件）看到诱饵信  
- [ ] 回复后结果页显示已回复  
- [ ] `techniques` 含 `reply`  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 一直未标记回复 | IMAP 未启用、文件夹错、Reply-To 未指向监控箱、轮询未到 |
| 仅有 Mailpit | 只能验发送；回复检测需 IMAP，不能只靠 Mailpit |
| 与「用户举报邮件」混淆 | 举报是转发到安全邮箱；本手法是 **回复原信** 检测 |
