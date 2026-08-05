#!/usr/bin/env python3
"""Run a serial fixed-answer baseline against an OpenAI-compatible API."""

from __future__ import annotations

import argparse
import json
import os
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from time import perf_counter_ns

import httpx


CACHE_PREFIX = (
    "You are a deterministic benchmark assistant. Follow the user instruction exactly. "
    "Keep the answer concise and do not add explanations. "
) * 80
PROMPT = CACHE_PREFIX + "\nReply with exactly: REMOTE_MODEL_OK"
EXPECTED = "REMOTE_MODEL_OK"


def load_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip().strip('"')
    return values


def endpoint(base_url: str) -> str:
    base = base_url.rstrip("/")
    if not base.endswith("/v1"):
        base += "/v1"
    return base + "/chat/completions"


def run_once(client: httpx.Client, url: str, model: str) -> dict:
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": PROMPT}],
        "temperature": 0,
        "seed": 42,
        "max_tokens": 128,
    }
    started = perf_counter_ns()
    try:
        response = client.post(url, json=payload)
        duration_ns = perf_counter_ns() - started
        body = response.json()
        choice = (body.get("choices") or [{}])[0]
        message = choice.get("message") or {}
        content = (message.get("content") or "").strip()
        usage = body.get("usage") or {}
        return {
            "duration_ns": duration_ns,
            "status_code": response.status_code,
            "content": content,
            "success": response.is_success and EXPECTED in content,
            "prompt_tokens": usage.get("prompt_tokens"),
            "completion_tokens": usage.get("completion_tokens"),
            "prompt_cache_hit_tokens": usage.get("prompt_cache_hit_tokens"),
            "prompt_cache_miss_tokens": usage.get("prompt_cache_miss_tokens"),
            "error": body.get("error") if not response.is_success else None,
        }
    except Exception as exc:  # Keep failed samples visible in the report.
        return {
            "duration_ns": perf_counter_ns() - started,
            "status_code": None,
            "content": "",
            "success": False,
            "error": f"{type(exc).__name__}: {exc}",
        }


def run_concurrent(
    client: httpx.Client,
    url: str,
    model: str,
    runs: int,
    concurrency: int,
    interval: float,
) -> list[dict]:
    """Run measured requests in bounded concurrent batches.

    Batching keeps the request count exact while allowing a small pause between
    waves so a benchmark does not accidentally become an unbounded burst.
    """
    samples: list[dict] = []
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        for start in range(0, runs, concurrency):
            batch_size = min(concurrency, runs - start)
            futures = [executor.submit(run_once, client, url, model) for _ in range(batch_size)]
            samples.extend(future.result() for future in futures)
            if start + batch_size < runs and interval:
                time.sleep(interval)
    return samples


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default="benchmarks/framework_comparison/remote_model.local.env")
    parser.add_argument("--warmup", type=int, default=3)
    parser.add_argument("--runs", type=int, default=100)
    parser.add_argument("--concurrency", type=int, default=8)
    parser.add_argument("--interval", type=float, default=0.2)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    values = load_env(Path(args.config))
    api_key = values.get("AGNES_API_KEY", "")
    base_url = values.get("AGNES_BASE_URL", "")
    model = values.get("AGNES_MODEL", "")
    if not api_key or not base_url or not model:
        raise SystemExit("remote model config is incomplete")
    if args.runs < 1 or args.warmup < 0 or args.concurrency < 1:
        raise SystemExit("runs must be positive, warmup non-negative, and concurrency positive")

    url = endpoint(base_url)
    headers = {"Authorization": f"Bearer {api_key}"}
    with httpx.Client(headers=headers, timeout=120.0) as client:
        for _ in range(args.warmup):
            run_once(client, url, model)
            if args.interval:
                time.sleep(args.interval)
        samples = run_concurrent(client, url, model, args.runs, args.concurrency, args.interval)

    report = {
        "framework": "direct-api",
        "scenario": "simple",
        "provider": "remote",
        "endpoint": base_url,
        "model": model,
        "warmup": args.warmup,
        "requested_runs": args.runs,
        "completed_runs": len(samples),
        "concurrency": args.concurrency,
        "interval_seconds": args.interval,
        "prompt": PROMPT,
        "api_key_stored": False,
        "samples": samples,
    }
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"completed_runs={len(samples)} requested_runs={args.runs} output={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
