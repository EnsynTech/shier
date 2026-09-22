## Learned User Preferences

- Prefers working and communicating in Chinese for this project.
- Wants bilingual admin UI (English and Chinese) with language chosen automatically from the browser Accept-Language header; do not show a manual language switcher in the page chrome.
- Keeps fork customizations on a dedicated branch and release tags so upstream updates do not overwrite local work; treat the official Gophish remote as upstream, not the push target for custom commits.
- Product branding for this fork should be 识饵 (Shier), not the earlier Xiangbi / 象比 naming.
- Prefers built-in phishing scenario and template pack copy to be Chinese-first.
- Treats the product mainly as a consultant-side (乙方) service tool for showing drill results to clients (甲方), not as a near-term self-serve product sold into client organizations.
- Admin UI should be oriented toward client security-operations and awareness-training owners (甲方安全运营、意识培训负责人).
- Wants AI-ready capabilities via model-agnostic REST APIs only (no built-in chat/Copilot UI in-product); any external model or agent may call the APIs to configure drills and drive workflows.
- Campaign/drill launch must not be freely triggerable by AI: sending requires a human click in the admin UI, or an explicit confirmation token on the launch API.
- Prefers phishing-report flows based on forwarding suspicious mail to a designated security mailbox over relying on Outlook/browser extensions, because many enterprises lack a unified mail client.
- Admin UI redesign (approved): progressive enhancement (方案 B)—Go templates for shell/routing, optional single-runtime islands on hot pages (wizard/workbench/results); lighter custom visual design (not Ant Design–dense kits); modern browsers only; no full admin SPA or Node SSR for now; designs and prototypes should include charts and visual statistics (not layout-only); research mainstream competing tools before major redesign iterations. Specs: `docs/ui-frontend-stack-decision.md`, `docs/ui-design.md`, `docs/ui-competitive-research.md`; static prototype: `docs/ui-redesign-demo.html`; walkthrough issues: `docs/ui-frontend-walkthrough-issues.md`.
- Wants red-team style trusted-sender setup guidance (SPF/DKIM/DMARC, lookalike vs customer-domain drills as config/docs) to raise realism; treat high-trust customer-domain sending and lookalike-domain scenarios as separate difficulty narratives in reporting, not as new payloads.

## Learned Workspace Facts

- This repo is a Gophish fork; custom work lives on the `shier` branch with release tag `shier-v0.12.1`, and the official remote is named `upstream`.
- Local default listen ports are admin on `:13333` and phishing on `:18000` (see `config.json`).
- UI i18n lives under `i18n/` with locale files `i18n/locales/en.json` and `i18n/locales/zh.json`; unsupported browser languages fall back to Chinese.
- Fetching upstream often requires the machine’s local HTTP/HTTPS git proxy.
- Built-in scenario packs live under `models/scenariopacks/` with a root `manifest.json` and per-pack assets.
- Inline email QR phishing uses CID-embedded images that encode `{{.URL}}`.
- Non-email delivery uses Sending Channels: SMS via a generic HTTP gateway and IM via a generic webhook-style config (not SMTP MailLog).
- Scenario/campaign tagging covers industry, technique (link / attachment / QR / SMS / IM / calendar / reply), and difficulty; targets and results use a dedicated Department field (separate from Position); results support department × technique click-rate breakdown (`GET /api/campaigns/breakdown`).
- AI-configurable API decision (approved): model-agnostic REST only; configure allowed, launch requires UI confirm (`POST /api/campaigns/{id}/launch`) or one-time token (`/api/ai/campaigns/{id}/prepare_launch` + `launch`). See `docs/ai-api-orchestration-decision.md`.
- Machine-readable API surface for agents: `docs/openapi.yaml`.
- Local technique e2e uses Mailpit; shared setup in `docs/e2e-local-mailpit.md`, per-technique docs under `docs/e2e/` (see `docs/e2e/README.md`), runners `scripts/e2e_mailpit.py` and `scripts/e2e_all_techniques.py`.
- Campaign delete is soft-delete into a recycle bin; spec is `docs/campaign-soft-delete.md`. Permanent purge requires `{ "confirm": "purge" }`. On Active Campaigns the row action is Archive (complete/stop sending), not Delete; delete-to-trash is for Archived campaigns, and purge is Trash-only.
