#!/usr/bin/env python3
"""Measure a local HTTP endpoint with bounded client concurrency."""

from __future__ import annotations

import argparse
import json
import statistics
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from time import perf_counter_ns


def request_once(url: str, timeout: float) -> dict[str, int | bool]:
    started = perf_counter_ns()
    try:
        request = urllib.request.Request(url, method="GET")
        with urllib.request.urlopen(request, timeout=timeout) as response:
            response.read()
            status = response.status
        return {
            "duration_ns": perf_counter_ns() - started,
            "status": status,
            "success": 200 <= status < 300,
        }
    except Exception:
        return {
            "duration_ns": perf_counter_ns() - started,
            "status": 0,
            "success": False,
        }


def measure(url: str, runs: int, concurrency: int, timeout: float) -> dict:
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        samples = list(executor.map(lambda _: request_once(url, timeout), range(runs)))
    durations = sorted(sample["duration_ns"] / 1_000_000 for sample in samples)
    success_count = sum(1 for sample in samples if sample["success"])
    p95_index = min(len(durations) - 1, int((len(durations) - 1) * 0.95))
    return {
        "url": url,
        "runs": runs,
        "concurrency": concurrency,
        "mean_ms": statistics.fmean(durations),
        "median_ms": statistics.median(durations),
        "p95_ms": durations[p95_index],
        "min_ms": durations[0],
        "max_ms": durations[-1],
        "success_count": success_count,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://127.0.0.1:8080/health")
    parser.add_argument("--runs", type=int, default=100)
    parser.add_argument("--concurrency", type=int, default=8)
    parser.add_argument("--timeout", type=float, default=10)
    args = parser.parse_args()
    if args.runs < 1 or args.concurrency < 1:
        raise SystemExit("runs and concurrency must be positive")
    print(json.dumps(measure(args.url, args.runs, args.concurrency, args.timeout), indent=2))


if __name__ == "__main__":
    main()
