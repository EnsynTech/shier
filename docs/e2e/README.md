# 识饵端到端测试（按钓鱼手法）

按剧本标签 `techniques` 拆分的本地验收说明。公共环境（Mailpit、识饵端口、发信配置）见 [`../e2e-local-mailpit.md`](../e2e-local-mailpit.md)。**完整 REST API** 见 [`../openapi.yaml`](../openapi.yaml)，本文不重复维护接口字段。

| 手法 | 文档 | 推荐剧本（manifest id） | 投递通道 |
|------|------|-------------------------|----------|
| 链接 `link` | [e2e-technique-link.md](e2e-technique-link.md) | `multistep_it_notice`、多数含链接剧本 | 邮件 → Mailpit |
| 附件 `attachment` | [e2e-technique-attachment.md](e2e-technique-attachment.md) | `office_remote_image`、`office_macro_openurl`、`office_excel_remote` | 邮件 → Mailpit |
| 二维码 `qr` | [e2e-technique-qr.md](e2e-technique-qr.md) | `qr_account_verify`、`qr_pdf_verify`、`qr_png_attachment` | 邮件 → Mailpit |
| 短信 `sms` | [e2e-technique-sms.md](e2e-technique-sms.md) | `sms_parcel_pickup` | Sending Channel（HTTP） |
| 即时通讯 `im` | [e2e-technique-im.md](e2e-technique-im.md) | `im_meeting_card` | Sending Channel（Webhook） |
| 日历 `calendar` | [e2e-technique-calendar.md](e2e-technique-calendar.md) | `calendar_video_meeting`、`calendar_hr_training` | 邮件 → Mailpit（.ics） |
| 回复 `reply` | [e2e-technique-reply.md](e2e-technique-reply.md) | `reply_meeting_confirm` | 邮件 + IMAP 监控 |

自动化冒烟（链接类示例）：

```bash
python3 scripts/e2e_mailpit.py
```

发版前建议：每种手法至少跑通对应文档「验收清单」一次；邮件类依赖 Mailpit 正式版（非 `go install` 的 blank UI）。
