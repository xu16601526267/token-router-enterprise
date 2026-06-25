#!/usr/bin/env python3
"""Live A->B->C reseller/customer chain smoke test for Token Router.

A is the live platform, B is a temporary downstream enterprise router, and C
is an employee/customer created inside B. The script keeps credentials and full
API keys out of the generated report.
"""

from __future__ import annotations

import argparse
import copy
import concurrent.futures
import http.cookiejar
import json
import os
import secrets
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROLE_COMMON_USER = 1
CHANNEL_TYPE_OPENAI = 1


@dataclass
class Response:
    status: int
    headers: dict[str, str]
    body: bytes
    elapsed: float
    error: str = ""

    def json(self) -> Any:
        try:
            return json.loads(self.body.decode("utf-8"))
        except Exception:
            return None

    def text(self) -> str:
        return self.body.decode("utf-8", errors="replace")


@dataclass
class AUpstreamKey:
    token_id: int
    name: str
    key: str
    models: list[str]

    @property
    def auth_key(self) -> str:
        return self.key if self.key.startswith("sk-") else "sk-" + self.key

    @property
    def masked(self) -> str:
        return mask_key(self.auth_key)


@dataclass
class BCustomerKey:
    token_id: int
    name: str
    key: str
    models: list[str]

    @property
    def masked(self) -> str:
        return mask_key(self.key)


@dataclass
class BMeshInstance:
    index: int
    runner: "ChainRunner"
    a_key: AUpstreamKey
    c_keys: list[BCustomerKey]


@dataclass
class CKeyRequestPlan:
    b_index: int
    key_index: int
    key: BCustomerKey
    model_sequence: list[str]
    before_usage: Response
    after_usage: Response | None = None
    log_response: Response | None = None


def mask_key(key: str) -> str:
    if len(key) <= 12:
        return "***"
    return f"{key[:6]}...{key[-4:]}"


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


class Client:
    def __init__(self, base_url: str, timeout: int) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.cookies = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self.cookies)
        )
        self.user_id: int | None = None

    def request(
        self,
        method: str,
        path: str,
        *,
        json_body: Any | None = None,
        api_key: str | None = None,
        user_auth: bool = False,
        headers: dict[str, str] | None = None,
    ) -> Response:
        url = self.base_url + path
        data = None
        req_headers = {"Content-Type": "application/json"}
        if headers:
            req_headers.update(headers)
        if json_body is not None:
            data = json.dumps(json_body, ensure_ascii=False).encode("utf-8")
        if api_key:
            req_headers["Authorization"] = f"Bearer {api_key}"
        if user_auth:
            if self.user_id is None:
                raise RuntimeError("user_id is not set; login first")
            req_headers["New-Api-User"] = str(self.user_id)

        req = urllib.request.Request(
            url, data=data, headers=req_headers, method=method.upper()
        )
        start = time.monotonic()
        try:
            with self.opener.open(req, timeout=self.timeout) as resp:
                body = resp.read()
                return Response(
                    status=resp.status,
                    headers={k.lower(): v for k, v in resp.headers.items()},
                    body=body,
                    elapsed=time.monotonic() - start,
                )
        except urllib.error.HTTPError as exc:
            return Response(
                status=exc.code,
                headers={k.lower(): v for k, v in exc.headers.items()},
                body=exc.read(),
                elapsed=time.monotonic() - start,
                error=str(exc),
            )
        except Exception as exc:
            return Response(
                status=0,
                headers={},
                body=str(exc).encode("utf-8", errors="replace"),
                elapsed=time.monotonic() - start,
                error=str(exc),
            )

    def login(self, username: str, password: str) -> dict[str, Any]:
        resp = self.request(
            "POST",
            "/api/user/login",
            json_body={"username": username, "password": password},
        )
        payload = resp.json()
        if resp.status != 200 or not isinstance(payload, dict) or not payload.get("success"):
            raise RuntimeError(f"login failed: status={resp.status}, body={resp.text()[:300]}")
        data = payload.get("data") or {}
        self.user_id = int(data["id"])
        return data


class ChainRunner:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.run_dir = Path(args.run_dir).resolve()
        self.responses_dir = self.run_dir / "responses"
        self.payloads_dir = self.run_dir / "payloads"
        self.a_client = Client(args.a_base_url, args.timeout_seconds)
        self.b_client: Client | None = None
        self.b_process: subprocess.Popen[bytes] | None = None
        self.b_base_url = ""
        self.a_key: AUpstreamKey | None = None
        self.c_key: BCustomerKey | None = None
        self.results: list[dict[str, Any]] = []
        self.warnings: list[str] = []
        self.created_b_username = ""

    def setup_dirs(self) -> None:
        self.responses_dir.mkdir(parents=True, exist_ok=True)
        self.payloads_dir.mkdir(parents=True, exist_ok=True)

    def save_response(self, name: str, resp: Response) -> None:
        (self.responses_dir / f"{name}.body").write_bytes(resp.body)
        meta = {
            "status": resp.status,
            "elapsed": round(resp.elapsed, 6),
            "content_type": resp.headers.get("content-type", ""),
            "error": resp.error,
        }
        (self.responses_dir / f"{name}.meta.json").write_text(
            json.dumps(meta, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )

    def save_payload(self, name: str, payload: Any) -> None:
        (self.payloads_dir / f"{name}.json").write_text(
            json.dumps(payload, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )

    def record(
        self,
        name: str,
        segment: str,
        resp: Response | None,
        ok: bool,
        detail: str,
        *,
        expected: str,
        save: bool = True,
    ) -> None:
        if resp is not None and save:
            self.save_response(name, resp)
        self.results.append(
            {
                "name": name,
                "segment": segment,
                "status": resp.status if resp else 0,
                "elapsed": resp.elapsed if resp else 0,
                "ok": ok,
                "expected": expected,
                "detail": detail,
            }
        )

    @staticmethod
    def has_business_error(payload: Any) -> bool:
        if not isinstance(payload, dict):
            return False
        return bool(payload.get("error")) or payload.get("success") is False

    @staticmethod
    def api_success(resp: Response) -> bool:
        payload = resp.json()
        return resp.status == 200 and isinstance(payload, dict) and payload.get("success") is True

    @staticmethod
    def usage_total(payload: Any) -> float | None:
        if isinstance(payload, dict) and isinstance(payload.get("total_usage"), (int, float)):
            return float(payload["total_usage"])
        return None

    @staticmethod
    def log_rows(payload: Any) -> list[dict[str, Any]]:
        if isinstance(payload, dict) and isinstance(payload.get("data"), list):
            return [row for row in payload["data"] if isinstance(row, dict)]
        return []

    def request_with_key(
        self,
        client: Client,
        name: str,
        segment: str,
        key: str,
        method: str,
        path: str,
        *,
        body: Any | None = None,
        expect_success: bool = True,
        expected: str,
        headers: dict[str, str] | None = None,
    ) -> Response:
        resp = client.request(method, path, json_body=body, api_key=key, headers=headers)
        payload = resp.json()
        if expect_success:
            ok = 200 <= resp.status < 300 and not self.has_business_error(payload)
            detail = "ok" if ok else f"unexpected failure: {resp.text()[:240]}"
        else:
            ok = resp.status >= 400 or self.has_business_error(payload)
            detail = "rejected as expected" if ok else "unexpectedly accepted"
        self.record(name, segment, resp, ok, detail, expected=expected)
        return resp

    def chat_payload(
        self,
        model: str,
        prompt: str,
        *,
        max_tokens: int,
        stream: bool = False,
    ) -> dict[str, Any]:
        return {
            "model": model,
            "messages": [
                {"role": "system", "content": "You are a concise API test assistant."},
                {"role": "user", "content": prompt},
            ],
            "temperature": 0,
            "max_tokens": max_tokens,
            "stream": stream,
        }

    def user_get_models(self) -> list[str]:
        resp = self.a_client.request("GET", "/api/user/models", user_auth=True)
        payload = resp.json()
        if resp.status != 200 or not isinstance(payload, dict) or not payload.get("success"):
            raise RuntimeError(f"A user model list failed: status={resp.status}, body={resp.text()[:300]}")
        models = payload.get("data") or []
        if not isinstance(models, list) or not models:
            raise RuntimeError("A test user has no usable models")
        return [str(model) for model in models]

    def select_models(self, available: list[str]) -> list[str]:
        selected: list[str] = []
        for preferred in [self.args.primary_model, self.args.secondary_model, "kimi-test", "moonlight-16b"]:
            if preferred and preferred in available and preferred not in selected:
                selected.append(preferred)
        for model in available:
            if model not in selected:
                selected.append(model)
            if len(selected) >= 2:
                break
        if not selected:
            raise RuntimeError("no model selected")
        return selected[:2]

    def create_a_key(self, models: list[str]) -> AUpstreamKey:
        name = f"codex-b-upstream-{int(time.time())}-{secrets.token_hex(3)}"
        payload = {
            "name": name,
            "expired_time": int(time.time()) + 86400,
            "remain_quota": self.args.a_key_quota,
            "unlimited_quota": False,
            "model_limits_enabled": True,
            "model_limits": ",".join(models),
            "allow_ips": "",
            "group": "default",
            "cross_group_retry": False,
        }
        self.save_payload("a_create_b_upstream_key", {**payload, "key": "***"})
        resp = self.a_client.request("POST", "/api/token/", json_body=payload, user_auth=True)
        if not self.api_success(resp):
            raise RuntimeError(f"A create upstream key failed: status={resp.status}, body={resp.text()[:300]}")
        token_id = self.find_a_token_id(name)
        key_resp = self.a_client.request("POST", f"/api/token/{token_id}/key", user_auth=True)
        key_payload = key_resp.json()
        key = ((key_payload or {}).get("data") or {}).get("key")
        if key_resp.status != 200 or not key:
            raise RuntimeError(f"A fetch upstream key failed: status={key_resp.status}, body={key_resp.text()[:300]}")
        self.a_key = AUpstreamKey(token_id=token_id, name=name, key=str(key), models=models)
        self.record(
            "a_create_b_upstream_key",
            "A侧给B开上游Key",
            resp,
            True,
            f"token_id={token_id}, key={self.a_key.masked}",
            expected="A test account can issue a temporary upstream key for B",
        )
        return self.a_key

    def find_a_token_id(self, name: str) -> int:
        resp = self.a_client.request("GET", "/api/token/?p=1&size=100", user_auth=True)
        payload = resp.json()
        data = payload.get("data") if isinstance(payload, dict) else None
        items = data.get("items") if isinstance(data, dict) else data if isinstance(data, list) else []
        for item in items:
            if isinstance(item, dict) and item.get("name") == name:
                return int(item["id"])
        raise RuntimeError(f"A created key not found in token list: {name}")

    def cleanup_a_key(self) -> None:
        if not self.a_key:
            return
        resp = self.a_client.request("DELETE", f"/api/token/{self.a_key.token_id}", user_auth=True)
        if not self.api_success(resp):
            self.warnings.append(
                f"A temporary upstream key cleanup failed: id={self.a_key.token_id}, status={resp.status}"
            )

    def build_b_binary(self) -> Path:
        if self.args.b_binary:
            binary = Path(self.args.b_binary).resolve()
            if not binary.exists():
                raise RuntimeError(f"B binary does not exist: {binary}")
            return binary

        binary = self.run_dir / "bin" / "token-router"
        binary.parent.mkdir(parents=True, exist_ok=True)
        build_log = self.run_dir / "b-build.log"
        with build_log.open("wb") as log_file:
            proc = subprocess.run(
                ["go", "build", "-o", str(binary), "."],
                cwd=self.args.repo_root,
                stdout=log_file,
                stderr=subprocess.STDOUT,
                timeout=self.args.build_timeout_seconds,
            )
        if proc.returncode != 0:
            raise RuntimeError(f"B binary build failed; see {build_log}")
        self.record(
            "b_build_binary",
            "B侧临时企业站",
            None,
            True,
            str(binary),
            expected="current repository builds a runnable B router binary",
            save=False,
        )
        return binary

    def start_b_router(self, binary: Path) -> None:
        port = find_free_port()
        self.b_base_url = f"http://127.0.0.1:{port}"
        log_dir = self.run_dir / "b-logs"
        log_dir.mkdir(parents=True, exist_ok=True)
        stdout_path = self.run_dir / "b-router.stdout.log"
        db_path = self.run_dir / "b-router.db"
        env = os.environ.copy()
        for key in [
            "SQL_DSN",
            "LOG_SQL_DSN",
            "REDIS_CONN_STRING",
            "SESSION_SECRET",
            "CRYPTO_SECRET",
        ]:
            env.pop(key, None)
        env.update(
            {
                "SQLITE_PATH": str(db_path) + "?_busy_timeout=30000",
                "SESSION_SECRET": secrets.token_urlsafe(32),
                "CRYPTO_SECRET": secrets.token_urlsafe(32),
                "GIN_MODE": "release",
                "MEMORY_CACHE_ENABLED": "false",
                "GLOBAL_API_RATE_LIMIT_ENABLE": "false",
                "GLOBAL_WEB_RATE_LIMIT_ENABLE": "false",
                "CRITICAL_RATE_LIMIT_ENABLE": "false",
                "SEARCH_RATE_LIMIT_ENABLE": "false",
                "UPDATE_TASK": "false",
                "GENERATE_DEFAULT_TOKEN": "false",
                "SYNC_FREQUENCY": "3600",
                "CHANNEL_UPDATE_FREQUENCY": "",
            }
        )
        stdout_file = stdout_path.open("wb")
        self.b_process = subprocess.Popen(
            [str(binary), "--port", str(port), "--log-dir", str(log_dir)],
            cwd=self.run_dir,
            env=env,
            stdout=stdout_file,
            stderr=subprocess.STDOUT,
        )
        self.b_client = Client(self.b_base_url, self.args.timeout_seconds)

        deadline = time.monotonic() + self.args.startup_timeout_seconds
        last_error = ""
        while time.monotonic() < deadline:
            if self.b_process.poll() is not None:
                raise RuntimeError(f"B router exited early with code {self.b_process.returncode}; see {stdout_path}")
            resp = self.b_client.request("GET", "/api/setup")
            if resp.status == 200:
                self.record(
                    "b_router_startup",
                    "B侧临时企业站",
                    resp,
                    True,
                    self.b_base_url,
                    expected="temporary B router exposes setup endpoint",
                )
                return
            last_error = resp.text()[:200]
            time.sleep(0.5)
        raise RuntimeError(f"B router did not become ready: {last_error}; see {stdout_path}")

    def stop_b_router(self) -> None:
        if not self.b_process:
            return
        if self.b_process.poll() is None:
            self.b_process.terminate()
            try:
                self.b_process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.b_process.kill()
                self.b_process.wait(timeout=10)

    def require_b_client(self) -> Client:
        if self.b_client is None:
            raise RuntimeError("B client is not initialized")
        return self.b_client

    def setup_b_root(self) -> None:
        client = self.require_b_client()
        username = f"br{int(time.time()) % 1000000000}"
        password = secrets.token_urlsafe(12)
        payload = {
            "username": username,
            "password": password,
            "confirmPassword": password,
            "SelfUseModeEnabled": True,
            "DemoSiteEnabled": False,
        }
        self.save_payload("b_setup_root", {**payload, "password": "***", "confirmPassword": "***"})
        resp = client.request("POST", "/api/setup", json_body=payload)
        if not self.api_success(resp):
            raise RuntimeError(f"B setup failed: status={resp.status}, body={resp.text()[:300]}")
        user = client.login(username, password)
        self.record(
            "b_setup_root",
            "B侧临时企业站",
            resp,
            True,
            f"root_user_id={user.get('id')}",
            expected="B router can be initialized as an enterprise customer",
        )

    def add_b_channel_to_a(self, a_key: AUpstreamKey) -> None:
        client = self.require_b_client()
        payload = {
            "mode": "single",
            "multi_key_mode": "",
            "batch_add_set_key_prefix_2_name": False,
            "channel": {
                "type": CHANNEL_TYPE_OPENAI,
                "key": a_key.auth_key,
                "name": f"codex-a-upstream-{int(time.time())}",
                "base_url": self.a_client.base_url,
                "models": ",".join(a_key.models),
                "group": "default",
                "status": 1,
                "weight": 100,
                "priority": 0,
                "auto_ban": 0,
            },
        }
        redacted = json.loads(json.dumps(payload))
        redacted["channel"]["key"] = a_key.masked
        self.save_payload("b_add_a_upstream_channel", redacted)
        resp = client.request("POST", "/api/channel/", json_body=payload, user_auth=True)
        if not self.api_success(resp):
            raise RuntimeError(f"B add A upstream channel failed: status={resp.status}, body={resp.text()[:500]}")
        self.record(
            "b_add_a_upstream_channel",
            "B侧配置A为上游",
            resp,
            True,
            f"models={','.join(a_key.models)}",
            expected="B can configure A as an OpenAI-compatible upstream channel",
        )

    def create_b_c_user(self) -> int:
        client = self.require_b_client()
        username = f"c{secrets.token_hex(4)}"
        password = secrets.token_urlsafe(12)
        self.created_b_username = username
        payload = {
            "username": username,
            "password": password,
            "display_name": "Codex C Employee",
            "role": ROLE_COMMON_USER,
        }
        self.save_payload("b_create_c_user", {**payload, "password": "***"})
        resp = client.request("POST", "/api/user/", json_body=payload, user_auth=True)
        if not self.api_success(resp):
            raise RuntimeError(f"B create C user failed: status={resp.status}, body={resp.text()[:300]}")

        search_path = "/api/user/search?" + urllib.parse.urlencode(
            {"keyword": username, "p": 1, "size": 10}
        )
        search = client.request("GET", search_path, user_auth=True)
        payload = search.json()
        data = payload.get("data") if isinstance(payload, dict) else None
        items = data.get("items") if isinstance(data, dict) else []
        user_id = 0
        for item in items:
            if isinstance(item, dict) and item.get("username") == username:
                user_id = int(item["id"])
                break
        if user_id == 0:
            raise RuntimeError(f"B C user not found after create: status={search.status}, body={search.text()[:300]}")

        quota_resp = client.request(
            "POST",
            "/api/user/manage",
            json_body={
                "id": user_id,
                "action": "add_quota",
                "mode": "override",
                "value": self.args.c_user_quota,
            },
            user_auth=True,
        )
        if not self.api_success(quota_resp):
            raise RuntimeError(f"B set C user quota failed: status={quota_resp.status}, body={quota_resp.text()[:300]}")
        self.record(
            "b_create_c_user",
            "B侧给C建员工账号",
            resp,
            True,
            f"c_user_id={user_id}",
            expected="B enterprise admin can create a C employee/customer account",
        )
        self.record(
            "b_set_c_user_quota",
            "B侧给C建员工账号",
            quota_resp,
            True,
            f"quota={self.args.c_user_quota}",
            expected="B enterprise admin can assign C user quota",
        )
        return user_id

    def create_b_c_key(self, c_user_id: int, models: list[str]) -> BCustomerKey:
        client = self.require_b_client()
        allow_ips = ""
        name = f"codex-c-employee-{int(time.time())}-{secrets.token_hex(3)}"
        payload = {
            "user_id": c_user_id,
            "name": name,
            "status": 1,
            "expired_time": int(time.time()) + 86400,
            "remain_quota": self.args.c_key_quota,
            "unlimited_quota": False,
            "model_limits_enabled": True,
            "model_limits": ",".join(models),
            "allow_ips": allow_ips,
            "group": "default",
            "cross_group_retry": False,
        }
        self.save_payload("b_create_c_api_key", payload)
        resp = client.request("POST", "/api/enterprise/api-keys", json_body=payload, user_auth=True)
        payload_resp = resp.json()
        data = payload_resp.get("data") if isinstance(payload_resp, dict) else None
        secret_key = data.get("secret_key") if isinstance(data, dict) else ""
        item = data.get("item") if isinstance(data, dict) else {}
        token_id = int(item.get("id") or 0) if isinstance(item, dict) else 0
        if not self.api_success(resp) or not secret_key or token_id == 0:
            raise RuntimeError(f"B create C API key failed: status={resp.status}, body={resp.text()[:500]}")
        self.c_key = BCustomerKey(token_id=token_id, name=name, key=str(secret_key), models=models)
        self.record(
            "b_create_c_api_key",
            "B侧给C开API Key",
            resp,
            True,
            f"token_id={token_id}, key={self.c_key.masked}",
            expected="B enterprise admin can issue a governed API key to C",
        )
        return self.c_key

    def check_chat_json(self, name: str, segment: str, resp: Response) -> None:
        payload = resp.json()
        ok = (
            resp.status == 200
            and isinstance(payload, dict)
            and isinstance(payload.get("choices"), list)
            and len(payload.get("choices")) > 0
            and isinstance(payload.get("usage"), dict)
        )
        detail = "OpenAI JSON choices/usage present" if ok else f"bad chat shape: {resp.text()[:240]}"
        self.results.append(
            {
                "name": f"{name}_shape",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "OpenAI-compatible JSON with choices and usage",
                "detail": detail,
            }
        )

    def check_stream(self, name: str, segment: str, resp: Response) -> None:
        text = resp.text()
        content_type = resp.headers.get("content-type", "")
        ok = resp.status == 200 and "data:" in text and "[DONE]" in text
        detail = f"SSE stream ok ({content_type})" if ok else f"stream invalid ({content_type}): {text[:240]}"
        self.results.append(
            {
                "name": f"{name}_stream_shape",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "SSE data frames with [DONE]",
                "detail": detail,
            }
        )

    def check_models(self, name: str, segment: str, resp: Response, expected_models: list[str]) -> None:
        payload = resp.json()
        listed: set[str] = set()
        if isinstance(payload, dict) and isinstance(payload.get("data"), list):
            for item in payload["data"]:
                if isinstance(item, dict) and item.get("id"):
                    listed.add(str(item["id"]))
        missing = [model for model in expected_models if model not in listed]
        ok = resp.status == 200 and not missing
        detail = f"listed={sorted(listed)[:12]}" if ok else f"missing={missing}, listed_sample={sorted(listed)[:12]}"
        self.results.append(
            {
                "name": f"{name}_model_scope",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "C sees only models exposed by B key/channel",
                "detail": detail,
            }
        )

    def accounting_check(
        self,
        name: str,
        segment: str,
        before_resp: Response,
        after_resp: Response,
    ) -> None:
        before = self.usage_total(before_resp.json())
        after = self.usage_total(after_resp.json())
        ok = isinstance(before, float) and isinstance(after, float) and after > before
        self.results.append(
            {
                "name": f"{name}_usage_delta",
                "segment": segment,
                "status": after_resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "usage increases after billable requests",
                "detail": f"before={before}, after={after}",
            }
        )

    def ledger_check(
        self,
        name: str,
        segment: str,
        resp: Response,
        *,
        min_rows: int,
        expected_models: list[str],
        forbidden_model: str,
    ) -> None:
        rows = self.log_rows(resp.json())
        model_names = {str(row.get("model_name")) for row in rows if row.get("model_name")}
        quota_sum = sum(int(row.get("quota") or 0) for row in rows)
        missing = [model for model in expected_models if model not in model_names]
        forbidden_seen = forbidden_model in model_names
        ok = len(rows) >= min_rows and not missing and not forbidden_seen and quota_sum > 0
        detail = (
            f"rows={len(rows)}, quota_sum={quota_sum}, models={sorted(model_names)}"
            if ok
            else f"rows={len(rows)}, missing={missing}, forbidden_seen={forbidden_seen}, quota_sum={quota_sum}, models={sorted(model_names)}"
        )
        self.results.append(
            {
                "name": f"{name}_ledger_coverage",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "ledger covers successful calls and does not include B-blocked invalid model",
                "detail": detail,
            }
        )

    def run_c_to_b_to_a(self, a_key: AUpstreamKey, c_key: BCustomerKey) -> None:
        b_client = self.require_b_client()
        segment = "C员工调用B企业站再转发A"
        primary = c_key.models[0]
        secondary = c_key.models[1] if len(c_key.models) > 1 else primary
        expected_successful_requests = self.args.concurrency + 2 + (1 if secondary != primary else 0)

        b_usage_before = self.request_with_key(
            b_client,
            "b_c_usage_before",
            segment,
            c_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="C can query B-side usage before requests",
        )
        a_usage_before = self.request_with_key(
            self.a_client,
            "a_b_usage_before",
            segment,
            a_key.auth_key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="B upstream key can query A-side usage before requests",
        )
        models_resp = self.request_with_key(
            b_client,
            "b_c_models",
            segment,
            c_key.key,
            "GET",
            "/v1/models",
            expected="C can discover B-exposed models",
        )
        self.check_models("b_c_models", segment, models_resp, c_key.models)

        chat_resp = self.request_with_key(
            b_client,
            "c_chat_primary_via_b",
            segment,
            c_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(primary, "Return exactly: c-via-b-primary-ok", max_tokens=48),
            expected="C primary non-stream request succeeds through B to A",
            headers={"X-Request-Id": f"codex-cb-primary-{int(time.time())}"},
        )
        self.check_chat_json("c_chat_primary_via_b", segment, chat_resp)

        if secondary != primary:
            secondary_resp = self.request_with_key(
                b_client,
                "c_chat_secondary_via_b",
                segment,
                c_key.key,
                "POST",
                "/v1/chat/completions",
                body=self.chat_payload(secondary, "Return exactly: c-via-b-secondary-ok", max_tokens=48),
                expected="C secondary model request succeeds through B to A",
            )
            self.check_chat_json("c_chat_secondary_via_b", segment, secondary_resp)

        stream_resp = self.request_with_key(
            b_client,
            "c_chat_stream_via_b",
            segment,
            c_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(primary, "Stream a short response through B.", max_tokens=64, stream=True),
            expected="C stream request succeeds through B to A",
        )
        self.check_stream("c_chat_stream_via_b", segment, stream_resp)

        def concurrent_call(index: int) -> tuple[int, Response]:
            payload = self.chat_payload(
                primary,
                f"C employee concurrent request {index}. Return ok-{index}.",
                max_tokens=self.args.max_tokens,
            )
            resp = b_client.request(
                "POST",
                "/v1/chat/completions",
                json_body=payload,
                api_key=c_key.key,
                headers={"X-Request-Id": f"codex-cb-concurrent-{int(time.time())}-{index}"},
            )
            return index, resp

        with concurrent.futures.ThreadPoolExecutor(max_workers=self.args.concurrency) as executor:
            futures = [executor.submit(concurrent_call, i) for i in range(1, self.args.concurrency + 1)]
            for future in concurrent.futures.as_completed(futures):
                index, resp = future.result()
                payload = resp.json()
                ok = resp.status == 200 and isinstance(payload, dict) and isinstance(payload.get("choices"), list)
                self.record(
                    f"c_concurrent_via_b_{index}",
                    segment,
                    resp,
                    ok,
                    "ok" if ok else f"bad concurrent response: {resp.text()[:200]}",
                    expected="C concurrent OpenAI-compatible requests succeed through B to A",
                )

        invalid_model = f"codex-invalid-c-model-{int(time.time())}"
        self.request_with_key(
            b_client,
            "c_invalid_model_blocked_by_b",
            segment,
            c_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(invalid_model, "This model must be blocked by B.", max_tokens=16),
            expect_success=False,
            expected="B blocks C key model whitelist violation before forwarding to A",
        )

        time.sleep(self.args.settle_seconds)
        b_usage_after = self.request_with_key(
            b_client,
            "b_c_usage_after",
            segment,
            c_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="C can query B-side usage after requests",
        )
        a_usage_after = self.request_with_key(
            self.a_client,
            "a_b_usage_after",
            segment,
            a_key.auth_key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="B upstream key can query A-side usage after requests",
        )
        b_logs = self.request_with_key(
            b_client,
            "b_c_logs",
            segment,
            c_key.key,
            "GET",
            "/api/log/token",
            expected="C key can query B-side token ledger",
        )
        a_logs = self.request_with_key(
            self.a_client,
            "a_b_logs",
            segment,
            a_key.auth_key,
            "GET",
            "/api/log/token",
            expected="B upstream key can query A-side token ledger",
        )

        self.accounting_check("b_c", segment, b_usage_before, b_usage_after)
        self.accounting_check("a_b", segment, a_usage_before, a_usage_after)
        self.ledger_check(
            "b_c",
            segment,
            b_logs,
            min_rows=expected_successful_requests,
            expected_models=c_key.models,
            forbidden_model=invalid_model,
        )
        self.ledger_check(
            "a_b",
            segment,
            a_logs,
            min_rows=expected_successful_requests,
            expected_models=c_key.models,
            forbidden_model=invalid_model,
        )

    def write_summary(self, a_user: dict[str, Any], selected_models: list[str]) -> None:
        passed = sum(1 for result in self.results if result["ok"])
        failed = [result for result in self.results if not result["ok"]]
        lines = [
            "# A平台-B企业客户-C员工真实链路测试",
            "",
            f"- A 平台地址：`{self.a_client.base_url}`",
            f"- 临时 B 企业站地址：`{self.b_base_url}`",
            f"- A 测试用户 ID：`{a_user.get('id')}`",
            f"- B 内 C 用户名：`{self.created_b_username}`",
            f"- 测试模型：`{', '.join(selected_models)}`",
            f"- 并发数：`{self.args.concurrency}`",
            f"- A 侧 B 上游 Key：`{self.a_key.masked if self.a_key else ''}`",
            f"- B 侧 C 员工 Key：`{self.c_key.masked if self.c_key else ''}`",
            f"- 通过/总数：`{passed}/{len(self.results)}`",
            "",
            "## 测试矩阵",
            "",
            "| 链路 | 用例 | HTTP | 耗时秒 | 结果 | 期望 | 观察 |",
            "| --- | --- | ---: | ---: | --- | --- | --- |",
        ]
        for result in self.results:
            status = result["status"]
            elapsed = result["elapsed"]
            mark = "PASS" if result["ok"] else "FAIL"
            lines.append(
                "| {segment} | `{name}` | {status} | {elapsed:.3f} | {mark} | {expected} | {detail} |".format(
                    segment=result["segment"],
                    name=result["name"],
                    status=status,
                    elapsed=elapsed,
                    mark=mark,
                    expected=str(result["expected"]).replace("|", "/"),
                    detail=str(result["detail"]).replace("|", "/").replace("\n", " ")[:260],
                )
            )
        if self.warnings:
            lines += ["", "## 清理/环境警告", ""]
            lines.extend(f"- {warning}" for warning in self.warnings)
        lines += ["", "## 结论", ""]
        if failed:
            lines.append(
                "存在失败项，A->B->C 企业客户链路不能判定为通过；需要先处理失败项再给下游客户使用。"
            )
        else:
            lines.append(
                "A 平台给 B 企业客户开上游 Key、B 企业站给 C 员工开户并开 Key、C 通过 B 调用 A、B 侧 C-Key 记账、A 侧 B-Key 记账、并发、流式、模型白名单拦截和双侧日志对账均通过。"
            )
        summary = "\n".join(lines) + "\n"
        (self.run_dir / "summary.md").write_text(summary, encoding="utf-8")
        print(summary)

    def run(self) -> int:
        self.setup_dirs()
        a_user = self.a_client.login(self.args.a_username, self.args.a_password)
        available_models = self.user_get_models()
        selected_models = self.select_models(available_models)
        a_key = self.create_a_key(selected_models)
        binary = self.build_b_binary()
        try:
            self.start_b_router(binary)
            self.setup_b_root()
            self.add_b_channel_to_a(a_key)
            c_user_id = self.create_b_c_user()
            c_key = self.create_b_c_key(c_user_id, selected_models)
            self.run_c_to_b_to_a(a_key, c_key)
        finally:
            self.cleanup_a_key()
            self.stop_b_router()
        self.write_summary(a_user, selected_models)
        return 1 if any(not result["ok"] for result in self.results) else 0


class MultiChainRunner:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.run_dir = Path(args.run_dir).resolve()
        self.responses_dir = self.run_dir / "responses"
        self.a_client = Client(args.a_base_url, args.timeout_seconds)
        self.instances: list[BMeshInstance] = []
        self.results: list[dict[str, Any]] = []
        self.warnings: list[str] = []
        self.a_user: dict[str, Any] = {}
        self.selected_models: list[str] = []
        self.a_usage_before_by_b: dict[int, Response] = {}
        self.a_usage_after_by_b: dict[int, Response] = {}
        self.a_logs_by_b: dict[int, Response] = {}
        self.invalid_model_by_b: dict[int, str] = {}

    def setup_dirs(self) -> None:
        self.responses_dir.mkdir(parents=True, exist_ok=True)

    def save_response(self, name: str, resp: Response) -> None:
        safe_name = name.replace("/", "_").replace(" ", "_")
        (self.responses_dir / f"{safe_name}.body").write_bytes(resp.body)
        meta = {
            "status": resp.status,
            "elapsed": round(resp.elapsed, 6),
            "content_type": resp.headers.get("content-type", ""),
            "error": resp.error,
        }
        (self.responses_dir / f"{safe_name}.meta.json").write_text(
            json.dumps(meta, indent=2, ensure_ascii=False),
            encoding="utf-8",
        )

    def record(
        self,
        name: str,
        segment: str,
        resp: Response | None,
        ok: bool,
        detail: str,
        *,
        expected: str,
        save: bool = True,
    ) -> None:
        if resp is not None and save:
            self.save_response(name, resp)
        self.results.append(
            {
                "name": name,
                "segment": segment,
                "status": resp.status if resp else 0,
                "elapsed": resp.elapsed if resp else 0,
                "ok": ok,
                "expected": expected,
                "detail": detail,
            }
        )

    def adopt_child_results(self, child: ChainRunner, b_index: int, start_idx: int) -> int:
        for result in child.results[start_idx:]:
            copied = dict(result)
            copied["name"] = f"b{b_index}_{copied['name']}"
            copied["segment"] = f"B{b_index} {copied['segment']}"
            self.results.append(copied)
        return len(child.results)

    def select_models(self) -> list[str]:
        self.a_user = self.a_client.login(self.args.a_username, self.args.a_password)
        helper = ChainRunner(self.args)
        helper.a_client = self.a_client
        available_models = helper.user_get_models()
        self.selected_models = helper.select_models(available_models)
        return self.selected_models

    def build_binary(self) -> Path:
        helper = ChainRunner(self.args)
        helper.setup_dirs()
        binary = helper.build_b_binary()
        self.results.extend(helper.results)
        return binary

    def child_args(self, b_index: int, binary: Path) -> argparse.Namespace:
        child_args = copy.copy(self.args)
        child_args.run_dir = str(self.run_dir / f"B{b_index:02d}")
        child_args.b_binary = str(binary)
        return child_args

    def prepare_instances(self, binary: Path, models: list[str]) -> None:
        for b_index in range(1, self.args.b_count + 1):
            child = ChainRunner(self.child_args(b_index, binary))
            child.setup_dirs()
            child.a_client.login(self.args.a_username, self.args.a_password)
            cursor = 0

            a_key = child.create_a_key(models)
            child.start_b_router(binary)
            child.setup_b_root()
            child.add_b_channel_to_a(a_key)

            c_keys: list[BCustomerKey] = []
            for _ in range(self.args.c_per_b):
                c_user_id = child.create_b_c_user()
                for _ in range(self.args.keys_per_c):
                    c_keys.append(child.create_b_c_key(c_user_id, models))

            cursor = self.adopt_child_results(child, b_index, cursor)
            self.instances.append(
                BMeshInstance(index=b_index, runner=child, a_key=a_key, c_keys=c_keys)
            )
            self.record(
                f"b{b_index}_mesh_ready",
                "多B多C多Key拓扑准备",
                None,
                True,
                f"B{b_index}: C_users={self.args.c_per_b}, C_keys={len(c_keys)}, base={child.b_base_url}, adopted={cursor}",
                expected="one B enterprise router is ready with C users and C keys",
                save=False,
            )

    def request_with_key(
        self,
        client: Client,
        name: str,
        segment: str,
        key: str,
        method: str,
        path: str,
        *,
        body: Any | None = None,
        expect_success: bool = True,
        expected: str,
        headers: dict[str, str] | None = None,
    ) -> Response:
        resp = client.request(method, path, json_body=body, api_key=key, headers=headers)
        payload = resp.json()
        if expect_success:
            ok = 200 <= resp.status < 300 and not ChainRunner.has_business_error(payload)
            detail = "ok" if ok else f"unexpected failure: {resp.text()[:240]}"
        else:
            ok = resp.status >= 400 or ChainRunner.has_business_error(payload)
            detail = "rejected as expected" if ok else "unexpectedly accepted"
        self.record(name, segment, resp, ok, detail, expected=expected)
        return resp

    def check_models(self, name: str, segment: str, resp: Response, expected_models: list[str]) -> None:
        payload = resp.json()
        listed: set[str] = set()
        if isinstance(payload, dict) and isinstance(payload.get("data"), list):
            for item in payload["data"]:
                if isinstance(item, dict) and item.get("id"):
                    listed.add(str(item["id"]))
        missing = [model for model in expected_models if model not in listed]
        ok = resp.status == 200 and not missing
        detail = f"listed={sorted(listed)[:12]}" if ok else f"missing={missing}, listed_sample={sorted(listed)[:12]}"
        self.results.append(
            {
                "name": f"{name}_model_scope",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "each C key sees B-exposed models",
                "detail": detail,
            }
        )

    def accounting_check(self, name: str, segment: str, before_resp: Response, after_resp: Response) -> None:
        before = ChainRunner.usage_total(before_resp.json())
        after = ChainRunner.usage_total(after_resp.json())
        ok = isinstance(before, float) and isinstance(after, float) and after > before
        self.results.append(
            {
                "name": f"{name}_usage_delta",
                "segment": segment,
                "status": after_resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "usage increases after successful concurrent calls",
                "detail": f"before={before}, after={after}",
            }
        )

    def ledger_check(
        self,
        name: str,
        segment: str,
        resp: Response,
        *,
        min_rows: int,
        expected_models: list[str],
        forbidden_model: str,
    ) -> None:
        rows = ChainRunner.log_rows(resp.json())
        model_names = {str(row.get("model_name")) for row in rows if row.get("model_name")}
        quota_sum = sum(int(row.get("quota") or 0) for row in rows)
        missing = [model for model in expected_models if model not in model_names]
        forbidden_seen = forbidden_model in model_names
        ok = len(rows) >= min_rows and not missing and not forbidden_seen and quota_sum > 0
        detail = (
            f"rows={len(rows)}, quota_sum={quota_sum}, models={sorted(model_names)}"
            if ok
            else f"rows={len(rows)}, missing={missing}, forbidden_seen={forbidden_seen}, quota_sum={quota_sum}, models={sorted(model_names)}"
        )
        self.results.append(
            {
                "name": f"{name}_ledger_coverage",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "ledger covers successful calls and excludes B-blocked invalid model",
                "detail": detail,
            }
        )

    def model_sequence_for_key(self, key_index: int) -> list[str]:
        models = self.selected_models
        sequence: list[str] = []
        for request_index in range(self.args.requests_per_key):
            sequence.append(models[(key_index + request_index) % len(models)])
        return sequence

    def collect_before_state(self) -> list[CKeyRequestPlan]:
        plans: list[CKeyRequestPlan] = []
        for instance in self.instances:
            segment = f"B{instance.index}账前状态"
            self.a_usage_before_by_b[instance.index] = self.request_with_key(
                instance.runner.a_client,
                f"b{instance.index}_a_usage_before",
                segment,
                instance.a_key.auth_key,
                "GET",
                "/v1/dashboard/billing/usage",
                expected="A-side B upstream key usage is queryable before fanout",
            )
            for key_index, c_key in enumerate(instance.c_keys, start=1):
                b_client = instance.runner.require_b_client()
                key_segment = f"B{instance.index}/C-Key{key_index}账前状态"
                before = self.request_with_key(
                    b_client,
                    f"b{instance.index}_ckey{key_index}_usage_before",
                    key_segment,
                    c_key.key,
                    "GET",
                    "/v1/dashboard/billing/usage",
                    expected="B-side C key usage is queryable before fanout",
                )
                models_resp = self.request_with_key(
                    b_client,
                    f"b{instance.index}_ckey{key_index}_models",
                    key_segment,
                    c_key.key,
                    "GET",
                    "/v1/models",
                    expected="C key can discover B-exposed models before fanout",
                )
                self.check_models(
                    f"b{instance.index}_ckey{key_index}_models",
                    key_segment,
                    models_resp,
                    c_key.models,
                )
                plans.append(
                    CKeyRequestPlan(
                        b_index=instance.index,
                        key_index=key_index,
                        key=c_key,
                        model_sequence=self.model_sequence_for_key(key_index),
                        before_usage=before,
                    )
                )
        return plans

    def run_fanout_requests(self, plans: list[CKeyRequestPlan]) -> dict[int, int]:
        expected_success_by_b: dict[int, int] = {instance.index: 0 for instance in self.instances}
        instance_by_index = {instance.index: instance for instance in self.instances}
        tasks: list[tuple[CKeyRequestPlan, int, str]] = []
        for plan in plans:
            expected_success_by_b[plan.b_index] += len(plan.model_sequence)
            for request_index, model in enumerate(plan.model_sequence, start=1):
                tasks.append((plan, request_index, model))

        def call_one(plan: CKeyRequestPlan, request_index: int, model: str) -> tuple[CKeyRequestPlan, int, str, Response]:
            instance = instance_by_index[plan.b_index]
            b_client = instance.runner.require_b_client()
            payload = instance.runner.chat_payload(
                model,
                f"B{plan.b_index} C-key{plan.key_index} fanout request {request_index}. Return ok.",
                max_tokens=self.args.max_tokens,
            )
            resp = b_client.request(
                "POST",
                "/v1/chat/completions",
                json_body=payload,
                api_key=plan.key.key,
                headers={
                    "X-Request-Id": f"codex-mesh-b{plan.b_index}-k{plan.key_index}-r{request_index}-{int(time.time())}"
                },
            )
            return plan, request_index, model, resp

        max_workers = self.args.max_workers or len(tasks)
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = [executor.submit(call_one, plan, request_index, model) for plan, request_index, model in tasks]
            for future in concurrent.futures.as_completed(futures):
                plan, request_index, model, resp = future.result()
                payload = resp.json()
                ok = resp.status == 200 and isinstance(payload, dict) and isinstance(payload.get("choices"), list)
                self.record(
                    f"b{plan.b_index}_ckey{plan.key_index}_req{request_index}",
                    "多B多C多Key同时并发",
                    resp,
                    ok,
                    "ok" if ok else f"bad response: {resp.text()[:220]}",
                    expected=f"C key concurrent request succeeds through B to A, model={model}",
                )
        return expected_success_by_b

    def run_invalid_isolation(self) -> None:
        for instance in self.instances:
            if not instance.c_keys:
                continue
            b_client = instance.runner.require_b_client()
            invalid_model = f"codex-mesh-invalid-b{instance.index}-{int(time.time())}"
            self.invalid_model_by_b[instance.index] = invalid_model
            self.request_with_key(
                b_client,
                f"b{instance.index}_invalid_model_blocked",
                "多B非法模型隔离",
                instance.c_keys[0].key,
                "POST",
                "/v1/chat/completions",
                body=instance.runner.chat_payload(
                    invalid_model,
                    "This invalid model must be blocked by B and not reach A.",
                    max_tokens=16,
                ),
                expect_success=False,
                expected="each B blocks invalid C model request before forwarding to A",
            )

    def collect_after_state(
        self,
        plans: list[CKeyRequestPlan],
        expected_success_by_b: dict[int, int],
    ) -> None:
        instance_by_index = {instance.index: instance for instance in self.instances}
        for plan in plans:
            instance = instance_by_index[plan.b_index]
            b_client = instance.runner.require_b_client()
            key_segment = f"B{plan.b_index}/C-Key{plan.key_index}账后核对"
            plan.after_usage = self.request_with_key(
                b_client,
                f"b{plan.b_index}_ckey{plan.key_index}_usage_after",
                key_segment,
                plan.key.key,
                "GET",
                "/v1/dashboard/billing/usage",
                expected="B-side C key usage is queryable after fanout",
            )
            plan.log_response = self.request_with_key(
                b_client,
                f"b{plan.b_index}_ckey{plan.key_index}_logs",
                key_segment,
                plan.key.key,
                "GET",
                "/api/log/token",
                expected="B-side C key ledger is queryable after fanout",
            )
            self.accounting_check(
                f"b{plan.b_index}_ckey{plan.key_index}",
                key_segment,
                plan.before_usage,
                plan.after_usage,
            )
            self.ledger_check(
                f"b{plan.b_index}_ckey{plan.key_index}",
                key_segment,
                plan.log_response,
                min_rows=len(plan.model_sequence),
                expected_models=sorted(set(plan.model_sequence)),
                forbidden_model=self.invalid_model_by_b.get(plan.b_index, ""),
            )

        for instance in self.instances:
            b_index = instance.index
            segment = f"B{b_index}在A侧上游账后核对"
            self.a_usage_after_by_b[b_index] = self.request_with_key(
                instance.runner.a_client,
                f"b{b_index}_a_usage_after",
                segment,
                instance.a_key.auth_key,
                "GET",
                "/v1/dashboard/billing/usage",
                expected="A-side B upstream key usage is queryable after fanout",
            )
            self.a_logs_by_b[b_index] = self.request_with_key(
                instance.runner.a_client,
                f"b{b_index}_a_logs",
                segment,
                instance.a_key.auth_key,
                "GET",
                "/api/log/token",
                expected="A-side B upstream key ledger is queryable after fanout",
            )
            self.accounting_check(
                f"b{b_index}_a_upstream",
                segment,
                self.a_usage_before_by_b[b_index],
                self.a_usage_after_by_b[b_index],
            )
            self.ledger_check(
                f"b{b_index}_a_upstream",
                segment,
                self.a_logs_by_b[b_index],
                min_rows=expected_success_by_b[b_index],
                expected_models=self.selected_models,
                forbidden_model=self.invalid_model_by_b.get(b_index, ""),
            )

    def run_mesh(self) -> None:
        plans = self.collect_before_state()
        expected_success_by_b = self.run_fanout_requests(plans)
        self.run_invalid_isolation()
        time.sleep(self.args.settle_seconds)
        self.collect_after_state(plans, expected_success_by_b)
        total_expected = sum(expected_success_by_b.values())
        total_actual = sum(
            1
            for result in self.results
            if result["segment"] == "多B多C多Key同时并发" and result["ok"]
        )
        self.record(
            "mesh_success_count",
            "多租户聚合对账",
            None,
            total_actual == total_expected,
            f"actual_success={total_actual}, expected_success={total_expected}, by_b={expected_success_by_b}",
            expected="all planned C-key requests succeed and are available for ledger reconciliation",
            save=False,
        )

    def cleanup(self) -> None:
        for instance in reversed(self.instances):
            try:
                instance.runner.cleanup_a_key()
            except Exception as exc:
                self.warnings.append(f"B{instance.index} A-key cleanup error: {exc}")
            try:
                instance.runner.stop_b_router()
            except Exception as exc:
                self.warnings.append(f"B{instance.index} process cleanup error: {exc}")

    def write_summary(self) -> None:
        passed = sum(1 for result in self.results if result["ok"])
        failed = [result for result in self.results if not result["ok"]]
        total_c_keys = self.args.b_count * self.args.c_per_b * self.args.keys_per_c
        planned_requests = total_c_keys * self.args.requests_per_key
        lines = [
            "# A平台-多B企业-多C员工-多Key并发链路测试",
            "",
            f"- A 平台地址：`{self.a_client.base_url}`",
            f"- B 企业数量：`{self.args.b_count}`",
            f"- 每个 B 的 C 员工数：`{self.args.c_per_b}`",
            f"- 每个 C 的 API Key 数：`{self.args.keys_per_c}`",
            f"- 每个 Key 成功请求数：`{self.args.requests_per_key}`",
            f"- 总 C Key 数：`{total_c_keys}`",
            f"- 计划成功请求数：`{planned_requests}`",
            f"- 最大并发 worker：`{self.args.max_workers or planned_requests}`",
            f"- 测试模型：`{', '.join(self.selected_models)}`",
            f"- A 测试用户 ID：`{self.a_user.get('id')}`",
            f"- 通过/总数：`{passed}/{len(self.results)}`",
            "",
            "## 测试矩阵",
            "",
            "| 链路 | 用例 | HTTP | 耗时秒 | 结果 | 期望 | 观察 |",
            "| --- | --- | ---: | ---: | --- | --- | --- |",
        ]
        for result in self.results:
            mark = "PASS" if result["ok"] else "FAIL"
            lines.append(
                "| {segment} | `{name}` | {status} | {elapsed:.3f} | {mark} | {expected} | {detail} |".format(
                    segment=result["segment"],
                    name=result["name"],
                    status=result["status"],
                    elapsed=result["elapsed"],
                    mark=mark,
                    expected=str(result["expected"]).replace("|", "/"),
                    detail=str(result["detail"]).replace("|", "/").replace("\n", " ")[:260],
                )
            )
        if self.warnings:
            lines += ["", "## 清理/环境警告", ""]
            lines.extend(f"- {warning}" for warning in self.warnings)
        lines += ["", "## 结论", ""]
        if failed:
            lines.append(
                "存在失败项，多 B、多 C、多 Key 并发链路不能判定为通过；需要先处理失败项再扩大给下游客户使用。"
            )
        else:
            lines.append(
                "A 下挂多个 B 企业、每个 B 下挂多个 C 员工、每个 C 多 API Key 的并发链路通过。A 侧按 B 上游 Key 记账，B 侧按 C Key 记账；所有成功请求有用量增量和日志覆盖，非法模型由 B 拦截且未进入 A 侧账本。"
            )
            lines.append(
                "注意：B 给 C 的售价可独立于 A 给 B 的成本价，本测试核对双层账本完整性与请求归属，不要求 A/B 两层金额相等。"
            )
        summary = "\n".join(lines) + "\n"
        (self.run_dir / "summary.md").write_text(summary, encoding="utf-8")
        print(summary)

    def run(self) -> int:
        self.setup_dirs()
        try:
            models = self.select_models()
            binary = self.build_binary()
            self.prepare_instances(binary, models)
            self.run_mesh()
        finally:
            self.cleanup()
        self.write_summary()
        return 1 if any(not result["ok"] for result in self.results) else 0


def parse_args() -> argparse.Namespace:
    default_run_dir = f"/tmp/token-router-b-reseller-c-chain-{time.strftime('%Y%m%d%H%M%S')}-{os.getpid()}"
    parser = argparse.ArgumentParser(description="Run live A->B->C reseller chain tests.")
    parser.add_argument("--a-base-url", default=os.getenv("TOKEN_ROUTER_A_BASE_URL", ""))
    parser.add_argument("--a-username", default=os.getenv("TOKEN_ROUTER_A_USERNAME", ""))
    parser.add_argument("--a-password", default=os.getenv("TOKEN_ROUTER_A_PASSWORD", ""))
    parser.add_argument("--primary-model", default=os.getenv("TOKEN_ROUTER_PRIMARY_MODEL", ""))
    parser.add_argument("--secondary-model", default=os.getenv("TOKEN_ROUTER_SECONDARY_MODEL", ""))
    parser.add_argument("--concurrency", type=int, default=int(os.getenv("TOKEN_ROUTER_CONCURRENCY", "10")))
    parser.add_argument("--b-count", type=int, default=int(os.getenv("TOKEN_ROUTER_B_COUNT", "1")))
    parser.add_argument("--c-per-b", type=int, default=int(os.getenv("TOKEN_ROUTER_C_PER_B", "1")))
    parser.add_argument("--keys-per-c", type=int, default=int(os.getenv("TOKEN_ROUTER_KEYS_PER_C", "1")))
    parser.add_argument("--requests-per-key", type=int, default=int(os.getenv("TOKEN_ROUTER_REQUESTS_PER_KEY", "1")))
    parser.add_argument("--max-workers", type=int, default=int(os.getenv("TOKEN_ROUTER_MAX_WORKERS", "0")))
    parser.add_argument("--max-tokens", type=int, default=int(os.getenv("TOKEN_ROUTER_MAX_TOKENS", "48")))
    parser.add_argument("--a-key-quota", type=int, default=int(os.getenv("TOKEN_ROUTER_A_KEY_QUOTA", "2000000")))
    parser.add_argument("--c-user-quota", type=int, default=int(os.getenv("TOKEN_ROUTER_C_USER_QUOTA", "2000000")))
    parser.add_argument("--c-key-quota", type=int, default=int(os.getenv("TOKEN_ROUTER_C_KEY_QUOTA", "1000000")))
    parser.add_argument("--timeout-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_TIMEOUT_SECONDS", "120")))
    parser.add_argument("--settle-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_SETTLE_SECONDS", "3")))
    parser.add_argument("--startup-timeout-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_STARTUP_TIMEOUT_SECONDS", "60")))
    parser.add_argument("--build-timeout-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_BUILD_TIMEOUT_SECONDS", "180")))
    parser.add_argument("--b-binary", default=os.getenv("TOKEN_ROUTER_B_BINARY", ""))
    parser.add_argument("--repo-root", default=os.getenv("TOKEN_ROUTER_REPO_ROOT", os.getcwd()))
    parser.add_argument("--run-dir", default=os.getenv("TOKEN_ROUTER_RUN_DIR", default_run_dir))
    args = parser.parse_args()
    if not args.a_base_url or not args.a_username or not args.a_password:
        parser.error("--a-base-url, --a-username and --a-password are required")
    if args.concurrency < 1:
        parser.error("--concurrency must be >= 1")
    for attr in ["b_count", "c_per_b", "keys_per_c", "requests_per_key"]:
        if getattr(args, attr) < 1:
            parser.error(f"--{attr.replace('_', '-')} must be >= 1")
    if args.max_workers < 0:
        parser.error("--max-workers must be >= 0")
    args.repo_root = str(Path(args.repo_root).resolve())
    return args


def main() -> int:
    args = parse_args()
    use_mesh = (
        args.b_count > 1
        or args.c_per_b > 1
        or args.keys_per_c > 1
        or args.requests_per_key > 1
    )
    runner: ChainRunner | MultiChainRunner
    runner = MultiChainRunner(args) if use_mesh else ChainRunner(args)
    try:
        return runner.run()
    except Exception as exc:
        print(f"A->B->C chain test failed before complete summary: {exc}", file=sys.stderr)
        try:
            runner.cleanup_a_key()
        finally:
            runner.stop_b_router()
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
