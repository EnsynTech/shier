# E2E：日历钓鱼（technique = `calendar`）

邮件附带可打开的 `.ics` 日历邀请，描述/地点中含钓鱼链接；标签常为 `calendar` + `attachment` + `link`。

| 项 | 内容 |
|---|---|
| 标签 | `calendar` |
| 推荐剧本 | `calendar_video_meeting`、`calendar_hr_training` |
| 投递 | 邮件 → Mailpit |
| API | [`docs/openapi.yaml`](../openapi.yaml) |
| 公共前置 | [`e2e-local-mailpit.md`](../e2e-local-mailpit.md) |

---

## 1. 准备

1. Mailpit + 识饵 + Mailpit SMTP。  
2. 导入 `calendar_video_meeting` 或 `calendar_hr_training`。  
3. 目标组（建议填部门）。

---

## 2. 手工路径

1. 启动活动 → Mailpit 收信，下载 `.ics` 附件。  
2. 用文本编辑器或日历客户端打开 `.ics`：  
   - 含 `BEGIN:VCALENDAR`  
   - `DESCRIPTION` / `LOCATION` 等处含 `http://127.0.0.1:18000` 与 `rid`（或模板变量已渲染的 URL）  
3. 浏览器打开该 URL → **Clicked Link**。  
4. （可选）将 `.ics` 导入系统日历，从日历 UI 点链接，效果应相同。  
5. breakdown 可见 `calendar` / `attachment` / `link`。

---

## 3. API 冒烟要点

```bash
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/calendar_video_meeting/import
```

从 Mailpit 消息附件下载后：

```bash
grep -E 'DESCRIPTION|LOCATION|http' /path/to/invite.ics | head
```

---

## 4. 期望事件

| 动作 | 期望 |
|------|------|
| 邮件+ics 发出 | Email Sent；Mailpit 有 `.ics` |
| 打开 ics 内链接 | Clicked Link |
| 活动标签 | 含 `calendar` |

---

## 5. 验收清单

- [ ] 附件为有效 `.ics`  
- [ ] URL 已渲染且指向本机钓鱼站  
- [ ] 点击后状态更新  
- [ ] breakdown 含 `calendar`  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| ics 时间异常 | 导入包会刷新 DTSTART 等；以能打开为准 |
| 日历客户端不显示链接 | 用编辑器核对 DESCRIPTION 后手动打开 URL |
