# 识饵（Shier）

识饵是面向钓鱼演练与安全意识培训的管理平台。咨询方用它组织演练、发出演练内容、收回结果，并把这些结果展示给甲方的安全运营和意识培训负责人。

## 主要功能

- 管理演练活动、邮件模板、落地页、收件人组和发信配置，并记录打开、点击与数据提交。
- 按部门汇总结果，并查看部门与手法的交叉点击率。
- 使用内置剧本包开展链接、附件、二维码、短信、即时通讯、日历和回复等演练。
- 除邮件外，可以通过短信 HTTP 网关和即时通讯 Webhook 风格通道发送。
- 管理界面支持中文和英文。默认语言跟随浏览器语言，顶部的中 / EN 切换会记住操作者的选择。
- 外部模型或自动化程序可以通过 REST API 配置演练。正式发送需要人在管理界面确认，或持有一次性确认令牌。

### 内置剧本包

剧本包放在 `models/scenariopacks/`。下面的名称和标识与 `manifest.json` 一致。

**链接**

- `link_mailbox_verify`：企业邮箱异地登录验证
- `link_mailbox_filing`：企业邮箱实名备案
- `link_sso_login`：统一身份认证
- `link_vpn_reauth`：SSL VPN 重新认证
- `link_it_password_reset`：IT 重置密码
- `link_hr_salary`：调薪确认与工资条
- `link_parcel_hold`：顺丰企业件滞留
- `link_invoice_check`：数电票查验
- `link_tax_settlement`：个税年度汇算确认
- `link_festival_gift`：中秋礼品领取
- `link_health_checkup`：年度体检预约
- `link_etc_notice`：企业 ETC 账单核对
- `link_travel_draw`：工会疗休养报名

**二维码**

- `qr_account_verify`：账号异常扫码验证
- `qr_parcel_pickup`：快递待取件扫码
- `qr_pdf_verify`：PDF 附件扫码验证
- `qr_landing_rescan`：落地页二次扫码
- `qr_png_attachment`：PNG 附件扫码

**附件**

- `office_remote_image`：远程图片追踪文档
- `office_macro_openurl`：宏启用打开链接
- `office_excel_remote`：Excel 远程图片追踪

**日历**

- `calendar_video_meeting`：视频会议邀请
- `calendar_hr_training`：人力资源强制培训

**多步骤与回复**

- `multistep_it_notice`：IT 维护多步骤确认
- `reply_meeting_confirm`：会议出席请回复

**短信**

- `sms_parcel_pickup`：短信快递待取件

**即时通讯**

- `im_meeting_card`：IM 会议确认卡片

## 相对 Gophish 增加的能力

- 中文优先的内置剧本包，覆盖链接、附件、二维码、短信、即时通讯、日历和回复。
- 短信与即时通讯两类非邮件发送通道。
- 收件人单独的部门字段，以及部门与手法的交叉分析。
- 管理端中英双语。
- 面向外部模型的配置接口。创建草稿不会发信；开演必须经过人工确认，或使用一次性确认令牌。
- 回复手法可以分成两步：第一封要求回复，记下回复后再发第二封。
- 把可疑邮件转发到指定上报邮箱后，可以记为已上报；回复记在发信邮箱，两条记录分开。
- 删除演练活动后进入回收站，彻底清除需要单独确认。

## 与 Gophish 的关系

本项目基于开源项目 [Gophish](https://github.com/gophish/gophish) 改造。上游项目主页是 <https://getgophish.com>，上游源码是 <https://github.com/gophish/gophish>。

识饵沿用 Gophish 的演练、追踪和记录能力，并按上一节扩展。产品名称、界面品牌和文档中的「识饵 / Shier」标识这个衍生作品。本仓库不代表 Gophish 上游项目或其作者。

代码中仍保留 `gophish` 作为模块路径、二进制名、配置文件名或内部标识，以便与上游兼容、便于合并更新。这些标识不改变本仓库的产品品牌。

定制开发在 `shier` 分支进行。官方 Gophish 远程通常命名为 `upstream`，用于拉取上游更新。请勿将本分支的定制提交直接推送到官方上游仓库。

## 免费使用与商业服务

识饵开源，可自行部署使用（见 `LICENSE`）。维护方为 [恩新智能](https://ensyntech.com/)，联系 ensyntech@outlook.com。若需要项目制钓鱼演练、意识培训配合，或渗透 / 咨询 / 定制开发，请通过官网或邮箱联系，按合同交付，而不是把本仓库当作付费软件授权。

## 许可

上游 Gophish Community Edition 使用 MIT License，版权为 Copyright (c) 2013-2020 Jordan Wright。完整许可文本在仓库根目录 [`LICENSE`](./LICENSE)。本分支新增与修改的部分，在适用范围内继续按 MIT 条款使用与分发。

对外发布二进制或源码包时，请一并附带 `LICENSE`，并保留本 README 中的上游出处说明。请勿删除、改写或隐藏 `LICENSE` 中的上游版权与许可文本。

感谢 [Jordan Wright](https://github.com/jordan-wright) 与 Gophish 社区提供的开源基础。
