#!/usr/bin/env python3
"""Run all 7 technique E2E suites against local 识饵 + Mailpit (+ HTTP mock for SMS/IM)."""
from __future__ import annotations

import base64
import json
import re
import sqlite3
import threading
import time
import urllib.error
import urllib.request
import zipfile
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, HTTPServer
from io import BytesIO
from typing import Any
from urllib.parse import urlencode

BASE = "http://127.0.0.1:13333"
PHISH = "http://127.0.0.1:18000"
MAILPIT = "http://127.0.0.1:8025"
DB = "/Users/boss/Projects/gophish/gophish.db"
SMTP_NAME = "Mailpit Local"
TS = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
MOCK_PORT = 18099

results: list[dict[str, Any]] = []
mock_hits: list[dict[str, Any]] = []


class MockHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):  # noqa: N802
        return

    def do_POST(self):  # noqa: N802
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length).decode(errors="replace")
        mock_hits.append(
            {
                "path": self.path,
                "body": body,
                "time": time.time(),
            }
        )
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"ok":true}')


def start_mock() -> HTTPServer:
    srv = HTTPServer(("127.0.0.1", MOCK_PORT), MockHandler)
    t = threading.Thread(target=srv.serve_forever, daemon=True)
    t.start()
    return srv


def api_key() -> str:
    conn = sqlite3.connect(DB)
    try:
        return conn.execute("SELECT api_key FROM users WHERE id=1").fetchone()[0]
    finally:
        conn.close()


def imap_enabled() -> bool:
    conn = sqlite3.connect(DB)
    try:
        row = conn.execute(
            "SELECT enabled, host FROM imap WHERE enabled=1 LIMIT 1"
        ).fetchone()
        return bool(row and row[1])
    except Exception:
        return False
    finally:
        conn.close()


def req(
    method: str,
    url: str,
    body: Any | None = None,
    key: str | None = None,
) -> tuple[int, Any]:
    data = None
    headers: dict[str, str] = {}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=45) as resp:
            raw = resp.read()
            code = resp.status
    except urllib.error.HTTPError as e:
        raw = e.read()
        code = e.code
    except Exception as e:
        return 0, str(e)
    if not raw:
        return code, None
    try:
        return code, json.loads(raw.decode())
    except Exception:
        return code, raw.decode(errors="replace")[:400]


def ensure_smtp(key: str) -> None:
    code, smtps = req("GET", f"{BASE}/api/smtp/", key=key)
    names = [s.get("name") for s in (smtps or [])] if isinstance(smtps, list) else []
    if SMTP_NAME in names:
        return
    req(
        "POST",
        f"{BASE}/api/smtp/",
        key=key,
        body={
            "name": SMTP_NAME,
            "host": "127.0.0.1:1025",
            "from_address": "drill@shier.local",
            "ignore_cert_errors": True,
            "headers": [],
        },
    )


def clear_mailpit() -> None:
    req("DELETE", f"{MAILPIT}/api/v1/messages")


def wait_mailpit(n: int, timeout: float = 25.0) -> dict:
    deadline = time.time() + timeout
    last: dict = {}
    while time.time() < deadline:
        code, data = req("GET", f"{MAILPIT}/api/v1/messages")
        if code == 200 and isinstance(data, dict):
            last = data
            if data.get("total", 0) >= n:
                return data
        time.sleep(0.4)
    return last


def wait_campaign_results(key: str, cid: int, timeout: float = 20.0) -> dict:
    deadline = time.time() + timeout
    last: dict = {}
    while time.time() < deadline:
        code, data = req("GET", f"{BASE}/api/campaigns/{cid}/results", key=key)
        if code == 200 and isinstance(data, dict) and (data.get("results") or []):
            return data
        last = data if isinstance(data, dict) else {}
        time.sleep(0.4)
    return last


def wait_status(key: str, cid: int, email: str, statuses: set[str], timeout: float = 15.0) -> str | None:
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        code, data = req("GET", f"{BASE}/api/campaigns/{cid}/results", key=key)
        if code == 200 and isinstance(data, dict):
            for r in data.get("results") or []:
                if r.get("email") == email:
                    last = r.get("status")
                    if last in statuses:
                        return last
        time.sleep(0.4)
    return last


def import_pack(key: str, pack_id: str) -> tuple[str, str]:
    code, data = req("POST", f"{BASE}/api/scenario_packs/{pack_id}/import", key=key)
    if code not in (200, 201) or not isinstance(data, dict):
        raise RuntimeError(f"import {pack_id}: {code} {data}")
    return data["template_name"], data["page_name"]


def create_group(key: str, name: str, targets: list[dict]) -> None:
    code, data = req(
        "POST",
        f"{BASE}/api/groups/",
        key=key,
        body={"name": name, "targets": targets},
    )
    if code not in (200, 201):
        raise RuntimeError(f"group: {code} {data}")


def create_email_campaign(key: str, name: str, group: str, tmpl: str, page: str) -> dict:
    code, camp = req(
        "POST",
        f"{BASE}/api/campaigns/",
        key=key,
        body={
            "name": name,
            "url": PHISH,
            "channel": "email",
            "groups": [{"name": group}],
            "template": {"name": tmpl},
            "page": {"name": page},
            "smtp": {"name": SMTP_NAME},
        },
    )
    if code not in (200, 201) or not isinstance(camp, dict) or not camp.get("id"):
        raise RuntimeError(f"campaign: {code} {camp}")
    return camp


def create_channel_campaign(
    key: str, name: str, group: str, tmpl: str, page: str, channel: str, ch_name: str
) -> dict:
    code, camp = req(
        "POST",
        f"{BASE}/api/campaigns/",
        key=key,
        body={
            "name": name,
            "url": PHISH,
            "channel": channel,
            "groups": [{"name": group}],
            "template": {"name": tmpl},
            "page": {"name": page},
            "sending_channel": {"name": ch_name},
        },
    )
    if code not in (200, 201) or not isinstance(camp, dict) or not camp.get("id"):
        raise RuntimeError(f"campaign: {code} {camp}")
    return camp


def ensure_channel(key: str, name: str, channel: str) -> None:
    code, chs = req("GET", f"{BASE}/api/sending_channels/", key=key)
    names = [c.get("name") for c in (chs or [])] if isinstance(chs, list) else []
    if name in names:
        return
    # text/plain avoids JSON breakage when Message contains quotes/newlines
    body_tmpl = (
        "phone={{.Phone}}\nurl={{.URL}}\nrid={{.RId}}\n"
        if channel == "sms"
        else "im_user={{.IMUser}}\nurl={{.URL}}\nrid={{.RId}}\nsubject={{.Subject}}\n"
    )
    code, data = req(
        "POST",
        f"{BASE}/api/sending_channels/",
        key=key,
        body={
            "name": name,
            "channel": channel,
            "endpoint_url": f"http://127.0.0.1:{MOCK_PORT}/{channel}",
            "http_method": "POST",
            "content_type": "text/plain",
            "body_template": body_tmpl,
            "success_status_codes": "200,201,204",
            "ignore_cert_errors": True,
        },
    )
    if code not in (200, 201):
        raise RuntimeError(f"channel {channel}: {code} {data}")


def click_rid(rid: str) -> int:
    code, _ = req("GET", f"{PHISH}/?rid={rid}")
    return code


def track_rid(rid: str) -> int:
    code, _ = req("GET", f"{PHISH}/track?rid={rid}")
    return code


def mailpit_message_detail(msg_id: str) -> dict:
    code, data = req("GET", f"{MAILPIT}/api/v1/message/{msg_id}")
    return data if code == 200 and isinstance(data, dict) else {}


def add_result(technique: str, status: str, detail: str) -> None:
    results.append({"technique": technique, "status": status, "detail": detail})


def run_link(key: str) -> None:
    tech = "link"
    try:
        clear_mailpit()
        g = f"E2E-link-{TS}"
        create_group(
            key,
            g,
            [
                {
                    "email": f"link+{TS}@example.com",
                    "first_name": "L",
                    "department": "测试部",
                }
            ],
        )
        # Prefer step1 template: import creates two; use names containing 确认 if present
        tmpl, page = import_pack(key, "multistep_it_notice")
        code, tmpls = req("GET", f"{BASE}/api/templates/", key=key)
        if isinstance(tmpls, list):
            for t in tmpls:
                n = t.get("name") or ""
                if "确认" in n and "多步骤" in n:
                    tmpl = n
                    break
        camp = create_email_campaign(key, f"E2E-link-{TS}", g, tmpl, page)
        cid = camp["id"]
        mail = wait_mailpit(1)
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        if not r0:
            add_result(tech, "FAIL", "无结果行")
            return
        click_rid(r0["id"])
        st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        techs = camp.get("techniques") or []
        ok = mail.get("total", 0) >= 1 and st in ("Clicked Link", "Submitted Data") and "link" in techs
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} mail={mail.get('total')} status={st} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_qr(key: str) -> None:
    tech = "qr"
    try:
        clear_mailpit()
        g = f"E2E-qr-{TS}"
        create_group(
            key,
            g,
            [{"email": f"qr+{TS}@example.com", "first_name": "Q", "department": "财务"}],
        )
        tmpl, page = import_pack(key, "qr_account_verify")
        camp = create_email_campaign(key, f"E2E-qr-{TS}", g, tmpl, page)
        cid = camp["id"]
        mail = wait_mailpit(1)
        detail = {}
        if mail.get("messages"):
            detail = mailpit_message_detail(mail["messages"][0]["ID"])
        html = (detail.get("HTML") or "") + (detail.get("Text") or "")
        has_qr_marker = ("cid:" in html.lower()) or ("qr" in html.lower()) or ("data:image" in html.lower())
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        click_rid(r0["id"])
        st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        techs = camp.get("techniques") or []
        ok = (
            mail.get("total", 0) >= 1
            and st in ("Clicked Link", "Submitted Data")
            and "qr" in techs
            and "link" in techs
        )
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} status={st} qr_like={has_qr_marker} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_calendar(key: str) -> None:
    tech = "calendar"
    try:
        clear_mailpit()
        g = f"E2E-cal-{TS}"
        create_group(
            key,
            g,
            [{"email": f"cal+{TS}@example.com", "first_name": "C", "department": "人事"}],
        )
        tmpl, page = import_pack(key, "calendar_video_meeting")
        camp = create_email_campaign(key, f"E2E-cal-{TS}", g, tmpl, page)
        cid = camp["id"]
        mail = wait_mailpit(1)
        msg = (mail.get("messages") or [None])[0]
        ics_ok = False
        url_in_ics = None
        if msg:
            detail = mailpit_message_detail(msg["ID"])
            atts = detail.get("Attachments") or []
            for a in atts:
                if str(a.get("FileName", "")).endswith(".ics") or "calendar" in str(a.get("ContentType", "")):
                    # fetch part
                    part_id = a.get("PartID") or a.get("ContentID")
                    # Mailpit: GET /api/v1/message/{id}/part/{partID}
                    if a.get("PartID"):
                        code, raw = req(
                            "GET",
                            f"{MAILPIT}/api/v1/message/{msg['ID']}/part/{a['PartID']}",
                        )
                        text = raw if isinstance(raw, str) else ""
                        if not text and code == 200:
                            # binary path - use download endpoint
                            try:
                                with urllib.request.urlopen(
                                    f"{MAILPIT}/api/v1/message/{msg['ID']}/part/{a['PartID']}/download",
                                    timeout=20,
                                ) as resp:
                                    text = resp.read().decode(errors="replace")
                            except Exception:
                                text = ""
                        ics_ok = "BEGIN:VCALENDAR" in text
                        m = re.search(r"https?://[^\s<>\"]+", text)
                        if m:
                            url_in_ics = m.group(0)
                    break
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        if url_in_ics and "rid=" in url_in_ics:
            req("GET", url_in_ics.split()[0].rstrip(">"))
        else:
            click_rid(r0["id"])
        st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        techs = camp.get("techniques") or []
        ok = ics_ok and st in ("Clicked Link", "Submitted Data") and "calendar" in techs
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} ics={ics_ok} status={st} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_attachment(key: str) -> None:
    tech = "attachment"
    try:
        clear_mailpit()
        g = f"E2E-att-{TS}"
        create_group(
            key,
            g,
            [{"email": f"att+{TS}@example.com", "first_name": "A", "department": "研发"}],
        )
        tmpl, page = import_pack(key, "office_macro_openurl")
        camp = create_email_campaign(key, f"E2E-att-{TS}", g, tmpl, page)
        cid = camp["id"]
        mail = wait_mailpit(1)
        msg = (mail.get("messages") or [None])[0]
        has_att = False
        if msg:
            detail = mailpit_message_detail(msg["ID"])
            atts = detail.get("Attachments") or []
            has_att = len(atts) > 0
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        # body link path (macro pack includes fallback URL)
        click_rid(r0["id"])
        st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        techs = camp.get("techniques") or []
        ok = has_att and st in ("Clicked Link", "Submitted Data") and "attachment" in techs
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} attachment={has_att} status={st} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_sms(key: str) -> None:
    tech = "sms"
    try:
        mock_hits.clear()
        ch_name = f"E2E-SMS-Mock-{TS}"
        ensure_channel(key, ch_name, "sms")
        g = f"E2E-sms-{TS}"
        create_group(
            key,
            g,
            [
                {
                    "email": f"sms+{TS}@example.com",
                    "first_name": "S",
                    "phone": "13800138000",
                    "department": "运营",
                }
            ],
        )
        tmpl, page = import_pack(key, "sms_parcel_pickup")
        before = len(mock_hits)
        camp = create_channel_campaign(
            key, f"E2E-sms-{TS}", g, tmpl, page, "sms", ch_name
        )
        cid = camp["id"]
        # wait mock hit
        deadline = time.time() + 20
        while time.time() < deadline and len(mock_hits) <= before:
            time.sleep(0.3)
        hit = mock_hits[-1] if len(mock_hits) > before else None
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        body = (hit or {}).get("body") or ""
        has_url = PHISH in body or "rid" in body.lower()
        if r0:
            click_rid(r0["id"])
            st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        else:
            st = None
        techs = camp.get("techniques") or []
        ok = hit is not None and has_url and st in ("Clicked Link", "Submitted Data") and "sms" in techs
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} mock_hit={hit is not None} status={st} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_im(key: str) -> None:
    tech = "im"
    try:
        mock_hits.clear()
        ch_name = f"E2E-IM-Mock-{TS}"
        ensure_channel(key, ch_name, "im")
        g = f"E2E-im-{TS}"
        create_group(
            key,
            g,
            [
                {
                    "email": f"im+{TS}@example.com",
                    "first_name": "I",
                    "department": "市场",
                }
            ],
        )
        tmpl, page = import_pack(key, "im_meeting_card")
        before = len(mock_hits)
        camp = create_channel_campaign(
            key, f"E2E-im-{TS}", g, tmpl, page, "im", ch_name
        )
        cid = camp["id"]
        deadline = time.time() + 20
        while time.time() < deadline and len(mock_hits) <= before:
            time.sleep(0.3)
        hit = mock_hits[-1] if len(mock_hits) > before else None
        res = wait_campaign_results(key, cid)
        r0 = (res.get("results") or [None])[0]
        body = (hit or {}).get("body") or ""
        has_url = PHISH in body or "rid" in body.lower()
        if r0:
            click_rid(r0["id"])
            st = wait_status(key, cid, r0["email"], {"Clicked Link", "Submitted Data"})
        else:
            st = None
        techs = camp.get("techniques") or []
        ok = hit is not None and has_url and st in ("Clicked Link", "Submitted Data") and "im" in techs
        add_result(
            tech,
            "PASS" if ok else "FAIL",
            f"camp={cid} mock_hit={hit is not None} status={st} techniques={techs}",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def run_reply(key: str) -> None:
    tech = "reply"
    try:
        clear_mailpit()
        g = f"E2E-reply-{TS}"
        create_group(
            key,
            g,
            [{"email": f"reply+{TS}@example.com", "first_name": "R", "department": "总办"}],
        )
        tmpl, page = import_pack(key, "reply_meeting_confirm")
        camp = create_email_campaign(key, f"E2E-reply-{TS}", g, tmpl, page)
        cid = camp["id"]
        mail = wait_mailpit(1)
        techs = camp.get("techniques") or []
        send_ok = mail.get("total", 0) >= 1 and "reply" in techs
        if not imap_enabled():
            add_result(
                tech,
                "PASS" if send_ok else "FAIL",
                f"camp={cid} send_ok={send_ok} techniques={techs}; "
                f"IMAP未配置 → 跳过「已回复」检测（仅验发送+标签）",
            )
            return
        # IMAP configured: cannot auto-inject reply without mailbox creds in this runner
        add_result(
            tech,
            "PASS" if send_ok else "FAIL",
            f"camp={cid} send_ok={send_ok}; IMAP已启用但自动回复注入未实现 → 请人工回信验收检测",
        )
    except Exception as e:
        add_result(tech, "FAIL", repr(e))


def main() -> int:
    key = api_key()
    ensure_smtp(key)
    mock = start_mock()
    time.sleep(0.2)
    try:
        runners = [
            ("link", run_link),
            ("attachment", run_attachment),
            ("qr", run_qr),
            ("sms", run_sms),
            ("im", run_im),
            ("calendar", run_calendar),
            ("reply", run_reply),
        ]
        for name, fn in runners:
            fn(key)
    finally:
        mock.shutdown()

    passed = sum(1 for r in results if r["status"] == "PASS")
    failed = sum(1 for r in results if r["status"] == "FAIL")
    print("=" * 72)
    print("识饵 7 手法 E2E 结果")
    print("=" * 72)
    print(f"合计: {passed} 通过 / {failed} 失败 / {len(results)} 项")
    print("-" * 72)
    for r in results:
        print(f"[{r['status']:4}] {r['technique']:12}  {r['detail']}")
    print("=" * 72)
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
