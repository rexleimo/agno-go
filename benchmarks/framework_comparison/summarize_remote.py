#!/usr/bin/env python3
"""Summarize one concurrent fixed-answer remote benchmark snapshot."""

from __future__ import annotations

import argparse
import json
import statistics
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def load(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def percentile(values: list[float], percentile_rank: float) -> float:
    if len(values) == 1:
        return values[0]
    index = (len(values) - 1) * percentile_rank
    lower = int(index)
    upper = min(lower + 1, len(values) - 1)
    fraction = index - lower
    return values[lower] + (values[upper] - values[lower]) * fraction


def summary(samples: list[dict[str, Any]]) -> dict[str, Any]:
    durations = sorted(float(item["duration_ns"]) / 1_000_000 for item in samples)
    successes = sum(1 for item in samples if item.get("success"))
    return {
        "count": len(durations),
        "mean_ms": statistics.fmean(durations),
        "median_ms": statistics.median(durations),
        "p95_ms": percentile(durations, 0.95),
        "min_ms": durations[0],
        "max_ms": durations[-1],
        "success_count": successes,
        "success_rate": successes / len(durations) if durations else 0.0,
    }


def read_samples(path: Path, framework: str) -> list[dict[str, Any]]:
    data = load(path)
    if framework == "python":
        return data["frameworks"]["agno"]["samples"]
    return data["samples"]


def build_report(directory: Path) -> dict[str, Any]:
    direct = load(directory / "direct_simple_100_concurrent.json")
    hno = load(directory / "hno_simple_100_concurrent.json")
    python = load(directory / "python_simple_100_concurrent.json")
    framework_samples = {
        "direct": direct["samples"],
        "hno": hno["samples"],
        "agno": python["frameworks"]["agno"]["samples"],
        "langgraph": python["frameworks"]["langgraph"]["samples"],
    }
    requested_runs = int(direct["requested_runs"])
    concurrency = int(direct["concurrency"])
    summaries = {name: summary(samples) for name, samples in framework_samples.items()}
    complete = all(item["count"] == requested_runs for item in summaries.values())
    return {
        "report_type": "real_remote_model_fixed_answer_100_concurrent",
        "captured_at_utc": datetime.now(timezone.utc).isoformat(),
        "status": "complete" if complete else "incomplete",
        "provider": "remote",
        "model_service": {
            "endpoint": direct["endpoint"],
            "model": direct["model"],
            "protocol": "OpenAI-compatible",
            "api_key_env": "AGNES_API_KEY",
        },
        "protocol": {
            "scenario": "fixed answer",
            "prompt": "stable deterministic benchmark prefix + Reply with exactly: REMOTE_MODEL_OK",
            "temperature": 0,
            "seed": 42,
            "max_tokens": 128,
            "warmup": direct["warmup"],
            "runs": requested_runs,
            "concurrency": concurrency,
            "measurement": "client wall-clock duration including network, provider queue, and model response",
        },
        "frameworks": {
            name: {"summary": summaries[name], "samples": samples}
            for name, samples in framework_samples.items()
        },
        "limits": [
            "All paths use the same provider, model, prompt, temperature, seed, and output limit.",
            "Warmups are serial; measured requests use bounded concurrency.",
            "The result is a latency snapshot, not a throughput, TTFT, TPS, or cost benchmark.",
            "Provider queueing, rate limits, and network conditions are included in client wall-clock time.",
        ],
    }


def markdown(report: dict[str, Any]) -> str:
    summaries = {name: value["summary"] for name, value in report["frameworks"].items()}
    direct_mean = summaries["direct"]["mean_ms"]
    lines = [
        "# DeepSeek fixed-answer concurrent benchmark",
        "",
        f"Captured: `{report['captured_at_utc']}` | Warmup: `{report['protocol']['warmup']}` | "
        f"Measured runs: `{report['protocol']['runs']}` | Concurrency: `{report['protocol']['concurrency']}`",
        f"Model: `{report['model_service']['model']}` | Endpoint: `{report['model_service']['endpoint']}`",
        "",
        "All paths used the same remote model and fixed-answer prompt. Each path "
        "measured 100 samples with bounded concurrency 8.",
        "",
        "| Path | Mean ms | Median ms | P95 ms | Min-max ms | Success | Relative mean |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for name in ("direct", "hno", "agno", "langgraph"):
        item = summaries[name]
        ratio = direct_mean / item["mean_ms"] if item["mean_ms"] else 0.0
        lines.append(
            f"| {name.title()} | {item['mean_ms']:.2f} | {item['median_ms']:.2f} | "
            f"{item['p95_ms']:.2f} | {item['min_ms']:.2f}-{item['max_ms']:.2f} | "
            f"{item['success_count']}/{item['count']} ({item['success_rate']:.0%}) | {ratio:.2f}x |"
        )
    lines.extend(
        [
            "",
            "`Relative mean` is `Direct API mean / path mean`; it is not a pure framework runtime speedup.",
            "Concurrent client latency includes provider/network/queueing effects and must not be interpreted as RPS.",
            "",
            "## Raw inputs",
            "",
            "- `direct_simple_100_concurrent.json`",
            "- `hno_simple_100_concurrent.json`",
            "- `python_simple_100_concurrent.json`",
            "",
            "## Limits",
            "",
        ]
    )
    lines.extend(f"- {limit}" for limit in report["limits"])
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--directory",
        type=Path,
        default=Path("benchmarks/framework_comparison/results/remote_deepseek_v4_flash"),
    )
    args = parser.parse_args()
    report = build_report(args.directory)
    output_json = args.directory / "latest_100_concurrent.json"
    output_md = args.directory / "latest_100_concurrent.md"
    output_json.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    output_md.write_text(markdown(report), encoding="utf-8")
    print(markdown(report), end="")


if __name__ == "__main__":
    main()
