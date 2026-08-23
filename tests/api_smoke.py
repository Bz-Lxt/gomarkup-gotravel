#!/usr/bin/env python3
"""API smoke in Mock mode. Cost: ¥0."""
import json
import os
import urllib.error
import urllib.request

BASE = os.environ.get("API_BASE", "http://backend:8080").rstrip("/")


def req(method, path, data=None, token=None, status=200):
    body = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if data is not None:
        body = json.dumps(data).encode()
    r = urllib.request.Request(BASE + path, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=15) as resp:
            raw = json.loads(resp.read().decode())
            assert resp.status == status, (resp.status, raw)
            return raw
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        raise AssertionError(f"{method} {path} -> {e.code} {raw}") from e


def main():
    h = req("GET", "/api/v1/health")
    assert h["ok"] and h["data"]["service"] == "gotravel"
    assert h["data"]["tz"] == "Asia/Shanghai"

    login = req("POST", "/api/v1/auth/login", {"username": "captain", "password": "captain123"})
    token = login["data"]["token"]
    assert token

    me = req("GET", "/api/v1/auth/me", token=token)
    assert me["data"]["username"] == "captain"

    teams = req("GET", "/api/v1/teams", token=token)
    assert teams["data"]
    team = teams["data"][0]
    trips = req("GET", f"/api/v1/teams/{team['id']}/trips", token=token)
    assert trips["data"]
    trip = trips["data"][0]
    detail = req("GET", f"/api/v1/trips/{trip['id']}", token=token)
    assert len(detail["data"]["waypoints"]) >= 2
    dist = req("GET", f"/api/v1/trips/{trip['id']}/distance", token=token)
    assert dist["data"]["total_meters"] > 0

    sess = req("POST", f"/api/v1/trips/{trip['id']}/sessions", {}, token=token, status=201)
    sid = sess["data"]["id"]
    sim = req("POST", "/api/v1/sim/start", {"session_id": sid, "count": 6, "laggard": True}, token=token)
    assert sim["data"]["running"] is True
    metrics = req("GET", "/api/v1/metrics")
    assert "inbound" in metrics["data"]
    print("SMOKE_OK")


if __name__ == "__main__":
    main()
