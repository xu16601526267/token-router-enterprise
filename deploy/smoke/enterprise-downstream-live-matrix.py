#!/usr/bin/env python3
"""Live downstream customer matrix test for Token Router.

The script logs in with a test user, creates temporary API keys, exercises
C-side direct usage and B-side relay-station usage, then deletes the keys.
It uses only Python stdlib and never writes full credentials or full API keys
to the evidence report.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import http.cookiejar
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


TOKEN_STATUS_DISABLED = 2


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
class TempKey:
    label: str
    token_id: int
    name: str
    key: str
    models: list[str]

    @property
    def masked(self) -> str:
        if len(self.key) <= 10:
            return "***"
        return f"{self.key[:5]}...{self.key[-4:]}"


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


class MatrixRunner:
    def __init__(self, args: argparse.Namespace) -> None:
        self.args = args
        self.client = Client(args.base_url, args.timeout_seconds)
        self.run_dir = Path(args.run_dir).resolve()
        self.responses_dir = self.run_dir / "responses"
        self.payloads_dir = self.run_dir / "payloads"
        self.temp_keys: list[TempKey] = []
        self.results: list[dict[str, Any]] = []
        self.warnings: list[str] = []

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
            json.dumps(meta, indent=2, ensure_ascii=False), encoding="utf-8"
        )

    def record(
        self,
        name: str,
        segment: str,
        resp: Response,
        ok: bool,
        detail: str,
        *,
        expected: str,
        save: bool = True,
    ) -> None:
        if save:
            self.save_response(name, resp)
        self.results.append(
            {
                "name": name,
                "segment": segment,
                "status": resp.status,
                "elapsed": resp.elapsed,
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
    def usage_total(payload: Any) -> float | None:
        if isinstance(payload, dict) and isinstance(payload.get("total_usage"), (int, float)):
            return float(payload["total_usage"])
        return None

    @staticmethod
    def hard_limit(payload: Any) -> float | None:
        if isinstance(payload, dict) and isinstance(payload.get("hard_limit_usd"), (int, float)):
            return float(payload["hard_limit_usd"])
        return None

    def user_get_models(self) -> list[str]:
        resp = self.client.request("GET", "/api/user/models", user_auth=True)
        payload = resp.json()
        if resp.status != 200 or not isinstance(payload, dict) or not payload.get("success"):
            raise RuntimeError(f"model list failed: status={resp.status}, body={resp.text()[:300]}")
        models = payload.get("data") or []
        if not isinstance(models, list) or not models:
            raise RuntimeError("test user has no usable models")
        return [str(model) for model in models]

    def create_key(self, label: str, models: list[str], remain_quota: int) -> TempKey:
        name = f"codex-{label}-matrix-{int(time.time())}-{len(self.temp_keys) + 1}"
        payload = {
            "name": name,
            "expired_time": int(time.time()) + 86400,
            "remain_quota": remain_quota,
            "unlimited_quota": False,
            "model_limits_enabled": True,
            "model_limits": ",".join(models),
            "allow_ips": "",
            "group": "default",
            "cross_group_retry": False,
        }
        resp = self.client.request("POST", "/api/token/", json_body=payload, user_auth=True)
        data = resp.json()
        if resp.status != 200 or not isinstance(data, dict) or not data.get("success"):
            raise RuntimeError(f"create key {label} failed: status={resp.status}, body={resp.text()[:300]}")

        token_id = self.find_token_id(name)
        key_resp = self.client.request("POST", f"/api/token/{token_id}/key", user_auth=True)
        key_payload = key_resp.json()
        key = ((key_payload or {}).get("data") or {}).get("key")
        if key_resp.status != 200 or not key:
            raise RuntimeError(f"fetch key {label} failed: status={key_resp.status}, body={key_resp.text()[:300]}")
        temp_key = TempKey(label=label, token_id=token_id, name=name, key=str(key), models=models)
        self.temp_keys.append(temp_key)
        return temp_key

    def find_token_id(self, name: str) -> int:
        resp = self.client.request("GET", "/api/token/?p=1&size=100", user_auth=True)
        payload = resp.json()
        data = payload.get("data") if isinstance(payload, dict) else None
        items = []
        if isinstance(data, dict):
            items = data.get("items") or []
        elif isinstance(data, list):
            items = data
        for item in items:
            if isinstance(item, dict) and item.get("name") == name:
                return int(item["id"])
        raise RuntimeError(f"created key not found in token list: {name}")

    def disable_key(self, temp_key: TempKey) -> None:
        resp = self.client.request(
            "PUT",
            "/api/token/?status_only=true",
            json_body={"id": temp_key.token_id, "status": TOKEN_STATUS_DISABLED},
            user_auth=True,
        )
        payload = resp.json()
        if resp.status != 200 or not isinstance(payload, dict) or not payload.get("success"):
            raise RuntimeError(f"disable key failed: status={resp.status}, body={resp.text()[:300]}")

    def delete_keys(self) -> None:
        for temp_key in list(reversed(self.temp_keys)):
            resp = self.client.request("DELETE", f"/api/token/{temp_key.token_id}", user_auth=True)
            payload = resp.json()
            if resp.status != 200 or not isinstance(payload, dict) or not payload.get("success"):
                self.warnings.append(
                    f"temporary key cleanup failed: label={temp_key.label}, id={temp_key.token_id}, status={resp.status}"
                )

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

    def request_with_key(
        self,
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
        resp = self.client.request(method, path, json_body=body, api_key=key, headers=headers)
        payload = resp.json()
        if expect_success:
            ok = 200 <= resp.status < 300 and not self.has_business_error(payload)
            detail = "ok" if ok else f"unexpected failure: {resp.text()[:240]}"
        else:
            ok = resp.status >= 400 or self.has_business_error(payload)
            detail = "rejected as expected" if ok else "unexpectedly accepted"
        self.record(name, segment, resp, ok, detail, expected=expected)
        return resp

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
        detail = (
            f"SSE stream ok ({content_type})"
            if ok
            else f"stream shape invalid ({content_type}): {text[:240]}"
        )
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
        detail = (
            f"listed {len(listed)} models"
            if ok
            else f"missing models={missing}, listed_sample={sorted(listed)[:8]}"
        )
        self.results.append(
            {
                "name": f"{name}_model_scope",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "allowed models visible through /v1/models",
                "detail": detail,
            }
        )

    def logs_for_key(self, name: str, segment: str, temp_key: TempKey, min_rows: int, expected_models: list[str]) -> None:
        resp = self.request_with_key(
            name,
            segment,
            temp_key.key,
            "GET",
            "/api/log/token",
            expected="token logs are queryable by downstream key",
        )
        payload = resp.json()
        rows = []
        if isinstance(payload, dict) and isinstance(payload.get("data"), list):
            rows = payload["data"]
        model_names = {str(row.get("model_name")) for row in rows if isinstance(row, dict)}
        quota_sum = sum(int(row.get("quota") or 0) for row in rows if isinstance(row, dict))
        missing_models = [model for model in expected_models if model not in model_names]
        ok = len(rows) >= min_rows and not missing_models
        detail = (
            f"rows={len(rows)}, quota_sum={quota_sum}, models={sorted(model_names)}"
            if ok
            else f"rows={len(rows)}, missing_models={missing_models}, quota_sum={quota_sum}"
        )
        self.results.append(
            {
                "name": f"{name}_ledger_coverage",
                "segment": segment,
                "status": resp.status,
                "elapsed": 0,
                "ok": ok,
                "expected": "logs cover tested requests and models",
                "detail": detail,
            }
        )

    def accounting_check(
        self,
        name: str,
        segment: str,
        before_resp: Response,
        after_resp: Response,
        *,
        strict: bool,
    ) -> None:
        before = self.usage_total(before_resp.json())
        after = self.usage_total(after_resp.json())
        ok = isinstance(before, float) and isinstance(after, float) and after > before
        detail = f"before={before}, after={after}"
        self.results.append(
            {
                "name": f"{name}_usage_delta",
                "segment": segment,
                "status": after_resp.status,
                "elapsed": 0,
                "ok": ok if strict else bool(isinstance(before, float) and isinstance(after, float)),
                "expected": "usage increases after billable requests",
                "detail": detail,
            }
        )

    def run_c_side(self, temp_key: TempKey, model: str) -> None:
        segment = "C端直连"
        before_usage = self.request_with_key(
            "c_usage_before",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="C-side can query usage before requests",
        )
        before_sub = self.request_with_key(
            "c_subscription_before",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/subscription",
            expected="C-side can query key balance/limit",
        )
        model_resp = self.request_with_key(
            "c_models",
            segment,
            temp_key.key,
            "GET",
            "/v1/models",
            expected="C-side sees allowed models",
        )
        self.check_models("c_models", segment, model_resp, [model])

        chat_resp = self.request_with_key(
            "c_chat_single",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "Return exactly: c-direct-ok", max_tokens=48),
            expected="C-side non-stream chat succeeds",
            headers={"X-Request-Id": f"codex-c-single-{int(time.time())}"},
        )
        self.check_chat_json("c_chat_single", segment, chat_resp)

        cn_resp = self.request_with_key(
            "c_chat_chinese",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "用中文回答：中转站计费对账正常。", max_tokens=64),
            expected="C-side Chinese prompt succeeds",
        )
        self.check_chat_json("c_chat_chinese", segment, cn_resp)

        stream_resp = self.request_with_key(
            "c_chat_stream",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "Stream a short response.", max_tokens=64, stream=True),
            expected="C-side stream chat succeeds",
        )
        self.check_stream("c_chat_stream", segment, stream_resp)

        def concurrent_call(index: int) -> tuple[int, Response]:
            payload = self.chat_payload(
                model,
                f"C-side direct concurrent request {index}. Return ok-{index}.",
                max_tokens=self.args.max_tokens,
            )
            resp = self.client.request(
                "POST",
                "/v1/chat/completions",
                json_body=payload,
                api_key=temp_key.key,
                headers={"X-Request-Id": f"codex-c-concurrent-{int(time.time())}-{index}"},
            )
            return index, resp

        with concurrent.futures.ThreadPoolExecutor(max_workers=self.args.c_concurrency) as executor:
            futures = [executor.submit(concurrent_call, i) for i in range(1, self.args.c_concurrency + 1)]
            for future in concurrent.futures.as_completed(futures):
                index, resp = future.result()
                name = f"c_concurrent_{index}"
                payload = resp.json()
                ok = resp.status == 200 and isinstance(payload, dict) and isinstance(payload.get("choices"), list)
                detail = "ok" if ok else f"bad concurrent response: {resp.text()[:200]}"
                self.record(
                    name,
                    segment,
                    resp,
                    ok,
                    detail,
                    expected="C-side concurrent OpenAI-compatible request succeeds",
                )

        self.request_with_key(
            "c_invalid_model",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(
                f"codex-invalid-model-{int(time.time())}",
                "This model must be denied.",
                max_tokens=16,
            ),
            expect_success=False,
            expected="C-side model whitelist denies invalid model",
        )

        time.sleep(self.args.settle_seconds)
        after_usage = self.request_with_key(
            "c_usage_after",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="C-side can query usage after requests",
        )
        after_sub = self.request_with_key(
            "c_subscription_after",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/subscription",
            expected="C-side can query balance after requests",
        )
        self.accounting_check("c", segment, before_usage, after_usage, strict=True)
        self.results.append(
            {
                "name": "c_balance_limit_present",
                "segment": segment,
                "status": after_sub.status,
                "elapsed": 0,
                "ok": self.hard_limit(before_sub.json()) is not None and self.hard_limit(after_sub.json()) is not None,
                "expected": "subscription response exposes hard_limit_usd before and after",
                "detail": f"before={self.hard_limit(before_sub.json())}, after={self.hard_limit(after_sub.json())}",
            }
        )
        self.logs_for_key("c_logs", segment, temp_key, min_rows=self.args.c_concurrency + 4, expected_models=[model])

    def run_b_side(self, temp_key: TempKey, models: list[str]) -> None:
        segment = "B端二级中转"
        primary = models[0]
        secondary = models[1] if len(models) > 1 else models[0]
        before_usage = self.request_with_key(
            "b_usage_before",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="B-side relay can query upstream usage before requests",
        )
        model_resp = self.request_with_key(
            "b_models",
            segment,
            temp_key.key,
            "GET",
            "/v1/models",
            expected="B-side relay can discover upstream model list",
        )
        self.check_models("b_models", segment, model_resp, sorted(set(models)))

        headers = {
            "X-Request-Id": f"codex-b-relay-{int(time.time())}",
            "X-Session-Id": f"codex-b-session-{int(time.time())}",
        }
        chat_resp = self.request_with_key(
            "b_chat_primary",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(primary, "Relay station primary model check. Return b-primary-ok.", max_tokens=48),
            expected="B-side primary model non-stream chat succeeds",
            headers=headers,
        )
        self.check_chat_json("b_chat_primary", segment, chat_resp)

        if secondary != primary:
            second_resp = self.request_with_key(
                "b_chat_secondary",
                segment,
                temp_key.key,
                "POST",
                "/v1/chat/completions",
                body=self.chat_payload(secondary, "Relay station secondary model check. Return b-secondary-ok.", max_tokens=48),
                expected="B-side secondary model non-stream chat succeeds",
            )
            self.check_chat_json("b_chat_secondary", segment, second_resp)

        stream_resp = self.request_with_key(
            "b_chat_stream",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(primary, "Relay station stream check.", max_tokens=64, stream=True),
            expected="B-side stream chat succeeds",
        )
        self.check_stream("b_chat_stream", segment, stream_resp)

        def concurrent_call(index: int) -> tuple[int, Response]:
            payload = self.chat_payload(
                primary,
                f"Concurrent relay request {index}. Return ok-{index}.",
                max_tokens=self.args.max_tokens,
            )
            resp = self.client.request(
                "POST",
                "/v1/chat/completions",
                json_body=payload,
                api_key=temp_key.key,
                headers={"X-Request-Id": f"codex-b-concurrent-{int(time.time())}-{index}"},
            )
            return index, resp

        with concurrent.futures.ThreadPoolExecutor(max_workers=self.args.concurrency) as executor:
            futures = [executor.submit(concurrent_call, i) for i in range(1, self.args.concurrency + 1)]
            for future in concurrent.futures.as_completed(futures):
                index, resp = future.result()
                name = f"b_concurrent_{index}"
                payload = resp.json()
                ok = resp.status == 200 and isinstance(payload, dict) and isinstance(payload.get("choices"), list)
                detail = "ok" if ok else f"bad concurrent response: {resp.text()[:200]}"
                self.record(
                    name,
                    segment,
                    resp,
                    ok,
                    detail,
                    expected="B-side concurrent OpenAI-compatible request succeeds",
                )

        self.request_with_key(
            "b_invalid_model",
            segment,
            temp_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(
                f"codex-b-invalid-model-{int(time.time())}",
                "This model must be denied.",
                max_tokens=16,
            ),
            expect_success=False,
            expected="B-side model whitelist denies invalid model",
        )

        time.sleep(self.args.settle_seconds)
        after_usage = self.request_with_key(
            "b_usage_after",
            segment,
            temp_key.key,
            "GET",
            "/v1/dashboard/billing/usage",
            expected="B-side relay can query upstream usage after requests",
        )
        self.accounting_check("b", segment, before_usage, after_usage, strict=True)
        expected_models = [primary] + ([secondary] if secondary != primary else [])
        self.logs_for_key(
            "b_logs",
            segment,
            temp_key,
            min_rows=self.args.concurrency + 4,
            expected_models=expected_models,
        )

    def run_negative_cases(self, low_key: TempKey, disabled_key: TempKey, model: str) -> None:
        segment = "拒绝链路"
        self.request_with_key(
            "negative_low_quota",
            segment,
            low_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "This request should exceed a tiny key quota.", max_tokens=128),
            expect_success=False,
            expected="low-quota key is rejected before unsafe overspend",
        )
        self.disable_key(disabled_key)
        self.request_with_key(
            "negative_disabled_key",
            segment,
            disabled_key.key,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "Disabled key should be rejected.", max_tokens=16),
            expect_success=False,
            expected="disabled key is rejected",
        )
        mutated = disabled_key.key[:-1] + ("A" if disabled_key.key[-1] != "A" else "B")
        self.request_with_key(
            "negative_invalid_key",
            segment,
            mutated,
            "POST",
            "/v1/chat/completions",
            body=self.chat_payload(model, "Invalid key should be rejected.", max_tokens=16),
            expect_success=False,
            expected="invalid key is rejected",
        )
        resp = self.client.request(
            "POST",
            "/v1/chat/completions",
            json_body=self.chat_payload(model, "Missing key should be rejected.", max_tokens=16),
        )
        ok = resp.status >= 400 or self.has_business_error(resp.json())
        self.record(
            "negative_missing_key",
            segment,
            resp,
            ok,
            "rejected as expected" if ok else "unexpectedly accepted",
            expected="missing key is rejected",
        )

    def write_summary(self, user: dict[str, Any], models: list[str]) -> None:
        passed = sum(1 for result in self.results if result["ok"])
        failed = [result for result in self.results if not result["ok"]]
        lines = [
            "# 下游客户真实环境矩阵测试",
            "",
            f"- 服务地址：`{self.client.base_url}`",
            f"- 测试用户 ID：`{user.get('id')}`",
            f"- 可用模型：`{', '.join(models)}`",
            f"- C 端并发数：`{self.args.c_concurrency}`",
            f"- B 端并发数：`{self.args.concurrency}`",
            f"- 临时 Key：`{', '.join(key.label + ':' + key.masked for key in self.temp_keys)}`",
            f"- 通过/总数：`{passed}/{len(self.results)}`",
            "",
            "## 测试矩阵",
            "",
            "| 客户/链路 | 用例 | HTTP | 耗时秒 | 结果 | 期望 | 观察 |",
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
            lines.append("存在失败项，不能判定真实下游链路完全通过。")
        else:
            lines.append(
                "C 端直连、B 端二级中转、并发、流式、长输出/中文、模型白名单、低额度、禁用 Key、错误 Key、缺失 Key、用量增长和日志覆盖均通过。"
            )
        summary = "\n".join(lines) + "\n"
        (self.run_dir / "summary.md").write_text(summary, encoding="utf-8")
        print(summary)

    def run(self) -> int:
        self.setup_dirs()
        user = self.client.login(self.args.username, self.args.password)
        models = self.user_get_models()
        primary = self.args.primary_model or models[0]
        secondary = self.args.secondary_model or (models[1] if len(models) > 1 else primary)
        selected_models = [model for model in [primary, secondary] if model in models]
        if primary not in models:
            raise RuntimeError(f"primary model {primary!r} is not usable by test user; usable={models}")
        if not selected_models:
            selected_models = [primary]

        c_key = self.create_key("c", [primary], self.args.normal_quota)
        b_key = self.create_key("b", selected_models, self.args.normal_quota * 2)
        low_key = self.create_key("low", [primary], self.args.low_quota)
        disabled_key = self.create_key("disabled", [primary], self.args.normal_quota)

        try:
            self.run_c_side(c_key, primary)
            self.run_b_side(b_key, selected_models)
            self.run_negative_cases(low_key, disabled_key, primary)
        finally:
            self.delete_keys()

        self.write_summary(user, models)
        return 1 if any(not result["ok"] for result in self.results) else 0


def parse_args() -> argparse.Namespace:
    default_run_dir = f"/tmp/token-router-downstream-matrix-{time.strftime('%Y%m%d%H%M%S')}-{os.getpid()}"
    parser = argparse.ArgumentParser(description="Run live downstream customer matrix tests.")
    parser.add_argument("--base-url", default=os.getenv("TOKEN_ROUTER_BASE_URL", ""))
    parser.add_argument("--username", default=os.getenv("TOKEN_ROUTER_USERNAME", ""))
    parser.add_argument("--password", default=os.getenv("TOKEN_ROUTER_PASSWORD", ""))
    parser.add_argument("--primary-model", default=os.getenv("TOKEN_ROUTER_PRIMARY_MODEL", ""))
    parser.add_argument("--secondary-model", default=os.getenv("TOKEN_ROUTER_SECONDARY_MODEL", ""))
    parser.add_argument("--concurrency", type=int, default=int(os.getenv("TOKEN_ROUTER_CONCURRENCY", "8")))
    parser.add_argument("--c-concurrency", type=int, default=int(os.getenv("TOKEN_ROUTER_C_CONCURRENCY", "0")))
    parser.add_argument("--max-tokens", type=int, default=int(os.getenv("TOKEN_ROUTER_MAX_TOKENS", "48")))
    parser.add_argument("--normal-quota", type=int, default=int(os.getenv("TOKEN_ROUTER_NORMAL_QUOTA", "800000")))
    parser.add_argument("--low-quota", type=int, default=int(os.getenv("TOKEN_ROUTER_LOW_QUOTA", "1")))
    parser.add_argument("--timeout-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_TIMEOUT_SECONDS", "120")))
    parser.add_argument("--settle-seconds", type=int, default=int(os.getenv("TOKEN_ROUTER_SETTLE_SECONDS", "3")))
    parser.add_argument("--run-dir", default=os.getenv("TOKEN_ROUTER_RUN_DIR", default_run_dir))
    args = parser.parse_args()
    if not args.base_url or not args.username or not args.password:
        parser.error("--base-url, --username and --password are required, or set TOKEN_ROUTER_BASE_URL/TOKEN_ROUTER_USERNAME/TOKEN_ROUTER_PASSWORD")
    if args.concurrency < 1:
        parser.error("--concurrency must be >= 1")
    if args.c_concurrency == 0:
        args.c_concurrency = args.concurrency
    if args.c_concurrency < 1:
        parser.error("--c-concurrency must be >= 1")
    return args


def main() -> int:
    args = parse_args()
    runner = MatrixRunner(args)
    try:
        return runner.run()
    except Exception as exc:
        print(f"matrix test failed before summary: {exc}", file=sys.stderr)
        try:
            runner.delete_keys()
        finally:
            return 1


if __name__ == "__main__":
    raise SystemExit(main())
