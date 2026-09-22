# E2E：短信钓鱼（technique = `sms`）

经 **Sending Channel（SMS / HTTP 网关）** 投递纯文本，内含钓鱼链接。不走 SMTP / Mailpit。

| 项 | 内容 |
|---|---|
| 标签 | `sms` + 通常 `link` |
| 推荐剧本 | `sms_parcel_pickup` |
| 投递 | `channel=sms` + 已配置的发送通道 |
| API | OpenAPI：`/api/sending_channels/`、`/api/scenario_packs/`、`/api/campaigns/` |
| 公共前置 | 识饵运行中；需可访问的 HTTP 短信网关或本地 mock |

---

## 1. 准备

1. 准备网关：真实短信 HTTP API，或本地 mock（例如接收 POST 并打日志的小服务）。  
2. 管理端 **发送通道** → 新建类型 **短信（HTTP）**：  
   - Endpoint、Method、鉴权头、Body 模板（含 `{{.Phone}}`、`{{.URL}}` 等，以产品字段为准）  
   - 成功状态码  
3. 用通道「测试」按钮验证网关可达（OpenAPI：`POST /api/sending_channels/test`）。  
4. 目标组：每位目标必须有 **手机号**（否则 SMS 活动会失败）。  
5. 导入 `sms_parcel_pickup`。

---

## 2. 手工路径

1. 新建活动：通道选 **SMS**，选择短信通道（不是 SMTP），模板/落地页来自导入，URL = `http://127.0.0.1:18000`。  
2. 启动后：在网关日志 / mock 中看到请求，正文含链接。  
3. 浏览器打开链接中的 `rid` URL → **Clicked Link**。  
4. breakdown 手法含 `sms`（及 `link`）。

---

## 3. API 冒烟要点

```bash
# 导入短信剧本
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/sms_parcel_pickup/import

# 通道 CRUD / test、活动创建见 openapi.yaml
# CampaignCreate: channel=sms, sending_channel.name=..., 无 smtp
```

本地无真实短信商时：用 mock 只验证「识饵发起 HTTP」与「链接可点」，不必真收到手机短信。

---

## 4. 期望事件

| 动作 | 期望 |
|------|------|
| 通道投递成功 | 结果为已发送类状态（与邮件 Sent 语义对应） |
| 打开链接 | Clicked Link |
| 缺手机号 | 创建/发送报错（Phone 必填） |

---

## 5. 验收清单

- [ ] SMS 通道测试成功  
- [ ] 活动 `channel=sms`，未误选 SMTP  
- [ ] 目标均有手机号  
- [ ] 网关收到正文且链接可点  
- [ ] 点击后结果与 `sms`/`link` breakdown 更新  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 一直失败 | 查 delivery 日志、通道成功码、TLS/证书勾选 |
| 无 Phone | 组目标补手机号 |
| Mailpit 收不到 | 正常：SMS 不走 Mailpit |
