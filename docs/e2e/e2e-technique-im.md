# E2E：即时通讯钓鱼（technique = `im`）

经 **Sending Channel（IM / Webhook）** 投递标题+正文（含链接）。不走 SMTP / Mailpit。

| 项 | 内容 |
|---|---|
| 标签 | `im` + 通常 `link` |
| 推荐剧本 | `im_meeting_card` |
| 投递 | `channel=im` + Webhook 类发送通道 |
| API | OpenAPI：`SendingChannels`、`scenario_packs`、`campaigns` |
| 公共前置 | 识饵；企业微信/钉钉机器人 Webhook，或本地 mock |

---

## 1. 准备

1. Webhook URL（群机器人或 mock）。  
2. **发送通道** → 类型 **即时通讯（Webhook）**：URL、Method、Content-Type、Body 模板（标题/正文/链接占位符按通道说明填写）。  
3. 通道「测试」成功。  
4. 导入 `im_meeting_card`。  
5. 目标组：邮箱仍用于结果标识；IM 用户字段按模板使用邮箱或扩展字段（以当前 Body 模板为准）。

---

## 2. 手工路径

1. 新建活动：通道 **IM**，绑定 IM 发送通道，选导入的模板/页，URL = `http://127.0.0.1:18000`。  
2. 启动后在群聊 / mock 日志看到卡片或文本及链接。  
3. 点击链接 → **Clicked Link**。  
4. breakdown 含 `im`（及 `link`）。

---

## 3. API 冒烟要点

```bash
curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://127.0.0.1:13333/api/scenario_packs/im_meeting_card/import

# 创建通道、测试、创建 channel=im 活动：见 openapi.yaml
```

---

## 4. 期望事件

| 动作 | 期望 |
|------|------|
| Webhook 2xx | 投递成功 |
| 点链接 | Clicked Link |
| Webhook 非成功码 | 结果错误 / 重试策略按实现 |

---

## 5. 验收清单

- [ ] IM 通道测试通过  
- [ ] 活动为 `im` 通道  
- [ ] 机器人或 mock 收到消息  
- [ ] 链接可点且结果更新  
- [ ] `techniques` 含 `im`  

## 6. 常见问题

| 现象 | 处理 |
|------|------|
| 群无消息 | 查机器人权限、Body JSON 是否被网关拒绝 |
| 与短信通道混淆 | 活动 channel 与通道类型必须同为 `im` |
| Mailpit 无信 | 正常 |
