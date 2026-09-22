# E2E：二维码钓鱼（technique = `qr`）

邮件内嵌 CID 二维码、PDF/PNG 附件二维码或落地页二次扫码，扫码结果指向带 `rid` 的钓鱼 URL。

| 项 | 内容 |
|---|---|
| 标签 | `qr`（通常同时有 `link`） |
| 推荐剧本 | `qr_account_verify`（正文 QR）；`qr_png_attachment` / `qr_pdf_verify`（附件）；`qr_landing_rescan`（多跳） |
| 投递 | 邮件 → Mailpit |
| API | [`docs/openapi.yaml`](../openapi.yaml) |
| 自动化 | `python3 scripts/e2e_mailpit.py`（默认此手法包） |
| 公共前置 | [`e2e-local-mailpit.md`](../e2e-local-mailpit.md) |

---

## 1. 准备

1. Mailpit + 识饵。  
2. 导入 `qr_account_verify`（最快）或其它 QR 包。  
3. 目标组建议带部门。

---

## 2. 手工路径

### A. 正文内嵌 QR（`qr_account_verify`）

1. 启动活动 → Mailpit 打开 HTML。  
2. 确认内嵌二维码图片；可用手机扫码，或点邮件中的备用文字链接。  
3. 打开落地页 → **Clicked Link**；提交登录表单 → **Submitted Data**。

### B. PNG / PDF 附件 QR

1. Mailpit 下载 `qrcode.png` 或 PDF。  
2. 扫附件中的码，或解码确认 URL 含 `rid` 与 `:18000`。  
3. 浏览器打开该 URL，核对结果状态。

### C. 落地页二次扫码（`qr_landing_rescan`）

1. 先点邮件 QR / 链接进入落地页。  
2. 落地页再显示可扫码（`QRDataURI`）。  
3. 二次扫码仍落到同一演练域（多跳体验）。

---

## 3. API 冒烟要点

```bash
python3 scripts/e2e_mailpit.py
# 或
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/qr_account_verify/import
curl -s "http://127.0.0.1:18000/?rid=<RId>" -o /dev/null -w "%{http_code}\n"
```

---

## 4. 期望事件

| 动作 | 期望 |
|------|------|
| 扫码或点备用链接 | Clicked Link |
| 落地页提交 | Submitted Data |
| 活动标签 | `qr` + `link`（附件类再加 `attachment`） |

---

## 5. 验收清单

- [ ] Mailpit HTML 可见 QR 或附件含码  
- [ ] 解码 URL 含正确主机与 `rid`  
- [ ] 点击/扫码后状态更新  
- [ ] breakdown 出现 `qr`（及 `link`）  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 邮件里二维码裂图 | 确认发送走识饵渲染（CID）；勿只用纯文本预览 |
| 扫码到错误域名 | 活动 URL 是否写成 `http://127.0.0.1:18000` |
| PDF 无个性化码 | 确认使用会在发送时替换的 PDF 包，而非静态错包 |
