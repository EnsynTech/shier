# E2E：附件钓鱼（technique = `attachment`）

诱饵为邮件附件（Office / PDF / PNG 等）。打开附件可能记「打开」；附件内链接或宏叙事可再记「点击」。

| 项 | 内容 |
|---|---|
| 标签 | `attachment`（常与 `link` / `qr` 并存） |
| 推荐剧本 | `office_remote_image`（打开追踪）；`office_macro_openurl`、`office_excel_remote`（附件+链接） |
| 投递 | 邮件 → Mailpit |
| API | [`docs/openapi.yaml`](../openapi.yaml) |
| 公共前置 | [`e2e-local-mailpit.md`](../e2e-local-mailpit.md) |

---

## 1. 准备

1. Mailpit + 识饵 + Mailpit 发信配置。  
2. 导入其一：  
   - `office_remote_image`：Word + 远程图 → 侧重 **打开**  
   - `office_macro_openurl`：文档内链接 → **打开 + 点击**  
   - `office_excel_remote`：Excel 远程图 + 正文备用链接  

---

## 2. 手工路径

### A. 远程图片打开（`office_remote_image`）

1. 启动活动，Mailpit 下载 `.docx` 附件。  
2. 用 Word（或能加载远程图的客户端）打开；允许加载外部内容。  
3. 期望结果出现 **Email Opened**（依赖远程图请求钓鱼站追踪像素）。  
4. 无桌面 Word 时：可从附件解出关系里的图片 URL，对追踪地址 `curl` 模拟打开。

### B. 附件内链接（`office_macro_openurl`）

1. Mailpit 下载附件并打开；点击文中链接（或邮件正文备用链接）。  
2. 期望 **Clicked Link**；落地页可提交则 **Submitted Data**。

### C. Excel（`office_excel_remote`）

1. 打开工作簿并允许远程内容 → 打开事件。  
2. 或点邮件正文链接 → 点击事件。

---

## 3. API 冒烟要点

```bash
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/office_macro_openurl/import
```

创建活动后从 Mailpit API 取附件：

```bash
# 列出消息后按 ID 取原文 / 附件（见 Mailpit API）
curl -s http://127.0.0.1:8025/api/v1/messages
```

活动快照应含 `attachment`（及可能的 `link`）。

---

## 4. 期望事件

| 剧本侧重 | 期望 |
|----------|------|
| remote_image / excel 远程图 | Email Opened |
| macro / 正文备用链接 | Clicked Link |
| 落地页提交 | Submitted Data |

---

## 5. 验收清单

- [ ] Mailpit 邮件含预期附件且可下载  
- [ ] 打开或点击路径至少一种产生对应状态  
- [ ] `techniques` 含 `attachment`  
- [ ] breakdown 手法列可见 `attachment`（及并存的 `link`）  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 打开文档无 Opened | 客户端拦截远程图；用 curl 打追踪 URL 验证服务端 |
| 宏未启用 | 用正文备用链接验证点击路径即可 |
| 附件损坏 | 重新导入剧本；确认发送时未截断 |
