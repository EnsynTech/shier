#!/usr/bin/env python3
"""识饵本地 E2E：API + Mailpit。只向 stdout 打印结果摘要。"""
from __future__ import annotations

import json
import sqlite3
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from typing import Any

BASE = "http://127.0.0.1:13333"
PHISH = "http://127.0.0.1:18000"
MAILPIT = "http://127.0.0.1:8025"
DB = "/Users/boss/Projects/gophish/gophish.db"
PACK_ID = "qr_account_verify"
TS = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")

checks: list[tuple[str, bool, str]] = []


def record(name: str, ok: bool, detail: str = "") -> None:
    checks.append((name, ok, detail))


def api_key() -> str:
    conn = sqlite3.connect(DB)
    try:
        row = conn.execute("SELECT api_key FROM users WHERE id=1").fetchone()
        if not row:
            raise SystemExit("no api_key in db")
        return row[0]
    finally:
        conn.close()


def req(
    method: str,
    url: str,
    body: Any | None = None,
    key: str | None = None,
    form: dict[str, str] | None = None,
) -> tuple[int, Any]:
    data = None
    headers: dict[str, str] = {}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    if form is not None:
        from urllib.parse import urlencode

        data = urlencode(form).encode()
        headers["Content-Type"] = "application/x-www-form-urlencoded"
    elif body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=30) as resp:
            raw = resp.read()
            code = resp.status
    except urllib.error.HTTPError as e:
        raw = e.read()
        code = e.code
    if not raw:
        return code, None
    try:
        return code, json.loads(raw.decode())
    except Exception:
        return code, raw.decode(errors="replace")[:500]


def wait_mailpit(min_count: int, timeout: float = 20.0) -> dict:
    deadline = time.time() + timeout
    last: dict = {}
    while time.time() < deadline:
        code, data = req("GET", f"{MAILPIT}/api/v1/messages")
        if code == 200 and isinstance(data, dict):
            last = data
            if data.get("total", 0) >= min_count:
                return data
        time.sleep(0.5)
    return last


def wait_results(key: str, cid: int, expect_status: str | None = None, timeout: float = 25.0):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        code, data = req("GET", f"{BASE}/api/campaigns/{cid}/results", key=key)
        if code == 200 and isinstance(data, dict):
            last = data
            results = data.get("results") or []
            if not expect_status:
                if results:
                    return data
            else:
                if any(r.get("status") == expect_status for r in results):
                    return data
        time.sleep(0.5)
    return last


def main() -> int:
    key = api_key()

    # 0. health
    code, info = req("GET", f"{MAILPIT}/api/v1/info")
    record("Mailpit 可用", code == 200 and isinstance(info, dict) and str(info.get("Version", "")).startswith("v"), str((info or {}).get("Version")))
    code, _ = req("GET", f"{BASE}/login")
    # /login returns HTML 200
    record("识饵管理端可用", code == 200, f"HTTP {code}")

    # clear mailpit for clean count
    req("DELETE", f"{MAILPIT}/api/v1/messages")

    # 1. SMTP
    code, smtps = req("GET", f"{BASE}/api/smtp/", key=key)
    smtp_name = "Mailpit Local"
    names = [s.get("name") for s in (smtps or [])] if isinstance(smtps, list) else []
    if smtp_name not in names:
        code, created = req(
            "POST",
            f"{BASE}/api/smtp/",
            key=key,
            body={
                "name": smtp_name,
                "interface_type": "SMTP",
                "host": "127.0.0.1:1025",
                "from_address": "drill@shier.local",
                "ignore_cert_errors": True,
                "headers": [],
            },
        )
        record("创建发信配置 Mailpit Local", code in (200, 201), str(created)[:120])
    else:
        record("发信配置 Mailpit Local 已存在", True, smtp_name)

    # 2. group with departments
    group_name = f"E2E-Dept-{TS}"
    code, group = req(
        "POST",
        f"{BASE}/api/groups/",
        key=key,
        body={
            "name": group_name,
            "targets": [
                {
                    "email": f"fin1+{TS}@example.com",
                    "first_name": "财务甲",
                    "last_name": "测",
                    "department": "财务",
                },
                {
                    "email": f"it1+{TS}@example.com",
                    "first_name": "IT乙",
                    "last_name": "测",
                    "department": "IT",
                },
            ],
        },
    )
    record("创建带部门的目标组", code in (200, 201) and isinstance(group, dict), group_name if code in (200, 201) else str(group)[:160])

    # 3. import scenario pack
    code, packs = req("GET", f"{BASE}/api/scenario_packs/", key=key)
    pack_ok = isinstance(packs, list) and any(p.get("id") == PACK_ID for p in packs)
    record("剧本库含 qr_account_verify", pack_ok, f"packs={len(packs) if isinstance(packs, list) else 0}")
    code, imported = req("POST", f"{BASE}/api/scenario_packs/{PACK_ID}/import", key=key)
    tmpl = (imported or {}).get("template_name") if isinstance(imported, dict) else None
    page = (imported or {}).get("page_name") if isinstance(imported, dict) else None
    record("导入剧本", code in (200, 201) and bool(tmpl and page), f"{tmpl} / {page}")
    if not tmpl or not page:
        print_report()
        return 1

    # 4. create+launch campaign (admin compatible immediate path)
    camp_name = f"E2E-Mailpit-{TS}"
    code, camp = req(
        "POST",
        f"{BASE}/api/campaigns/",
        key=key,
        body={
            "name": camp_name,
            "url": PHISH,
            "channel": "email",
            "groups": [{"name": group_name}],
            "template": {"name": tmpl},
            "page": {"name": page},
            "smtp": {"name": smtp_name},
        },
    )
    cid = camp.get("id") if isinstance(camp, dict) else None
    record(
        "创建并启动活动",
        code in (200, 201) and bool(cid),
        f"id={cid} status={(camp or {}).get('status') if isinstance(camp, dict) else camp}",
    )
    if not cid:
        print_report()
        return 1

    # tags snapshot
    techs = camp.get("techniques") if isinstance(camp, dict) else None
    record("活动快照手法标签", isinstance(techs, list) and len(techs) > 0, str(techs))

    # 5. wait mail
    mail = wait_mailpit(2, timeout=25)
    total = mail.get("total", 0) if isinstance(mail, dict) else 0
    record("Mailpit 收到 2 封演练邮件", total >= 2, f"total={total}")

    # 6. wait results with rid
    results_payload = wait_results(key, cid, timeout=20)
    results = (results_payload or {}).get("results") or []
    record("活动结果已生成", len(results) >= 2, f"n={len(results)}")
    depts = {r.get("email"): r.get("department") for r in results}
    record(
        "结果含部门字段",
        "财务" in depts.values() and "IT" in depts.values(),
        str(depts),
    )

    # 7. open + click 财务
    fin = next((r for r in results if r.get("department") == "财务"), None)
    if not fin:
        record("定位财务收件人", False, "missing")
        print_report(cid)
        return 1
    rid = fin.get("id")
    open_code, _ = req("GET", f"{PHISH}/track?rid={rid}")
    record("模拟打开邮件 (track)", open_code == 200, f"HTTP {open_code} rid={rid}")
    click_code, _ = req("GET", f"{PHISH}/?rid={rid}")
    record("模拟点击链接", click_code == 200, f"HTTP {click_code}")

    # wait clicked
    after = wait_results(key, cid, expect_status="Clicked Link", timeout=15)
    statuses = {r.get("email"): r.get("status") for r in ((after or {}).get("results") or [])}
    fin_status = statuses.get(fin.get("email"))
    # Submitted Data also counts as clicked in stats; Clicked Link is direct
    clicked_ok = fin_status in ("Clicked Link", "Submitted Data")
    record("财务目标状态为已点击", clicked_ok, f"status={fin_status}")

    # 8. breakdown
    code, bd = req("GET", f"{BASE}/api/campaigns/breakdown", key=key)
    by_dept = {b.get("department"): b for b in ((bd or {}).get("by_department") or [])} if isinstance(bd, dict) else {}
    fin_b = by_dept.get("财务") or {}
    record(
        "breakdown 财务有点击",
        code == 200 and fin_b.get("clicked", 0) >= 1,
        f"财务={fin_b}",
    )
    cross = (bd or {}).get("department_x_technique") if isinstance(bd, dict) else []
    record(
        "breakdown 含部门×手法交叉",
        isinstance(cross, list) and len(cross) > 0,
        f"rows={len(cross) if isinstance(cross, list) else 0}",
    )

    print_report(cid, camp_name, bd if isinstance(bd, dict) else None, after or results_payload)
    return 0 if all(ok for _, ok, _ in checks) else 1


def print_report(cid: int | None = None, name: str | None = None, bd: dict | None = None, results_payload: dict | None = None) -> None:
    passed = sum(1 for _, ok, _ in checks if ok)
    failed = sum(1 for _, ok, _ in checks if not ok)
    print("=" * 60)
    print("识饵 E2E 结果（Mailpit + API）")
    print("=" * 60)
    if name or cid:
        print(f"活动: {name or ''}  id={cid}")
    print(f"合计: {passed} 通过 / {failed} 失败 / {len(checks)} 项")
    print("-" * 60)
    for title, ok, detail in checks:
        mark = "PASS" if ok else "FAIL"
        extra = f"  — {detail}" if detail else ""
        print(f"[{mark}] {title}{extra}")
    if results_payload:
        print("-" * 60)
        print("结果明细:")
        for r in results_payload.get("results") or []:
            print(
                f"  · {r.get('email')}  dept={r.get('department')!r}  "
                f"status={r.get('status')}"
            )
    if bd:
        print("-" * 60)
        print("by_department:")
        for b in bd.get("by_department") or []:
            rate = b.get("click_rate")
            rate_s = f"{rate*100:.1f}%" if isinstance(rate, (int, float)) else rate
            print(
                f"  · {b.get('department')}: total={b.get('total')} "
                f"clicked={b.get('clicked')} rate={rate_s}"
            )
        print("department_x_technique (前 8 行):")
        for b in (bd.get("department_x_technique") or [])[:8]:
            rate = b.get("click_rate")
            rate_s = f"{rate*100:.1f}%" if isinstance(rate, (int, float)) else rate
            print(
                f"  · {b.get('department')} × {b.get('technique')}: "
                f"{b.get('clicked')}/{b.get('total')} ({rate_s})"
            )
    print("=" * 60)
    print("Mailpit UI: http://127.0.0.1:8025/")
    if cid:
        print(f"活动结果: http://127.0.0.1:13333/campaigns/{cid}")


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as e:
        record("脚本异常", False, repr(e))
        print_report()
        sys.exit(1)
