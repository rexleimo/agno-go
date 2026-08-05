#!/usr/bin/env python3
"""Run a fair local HNO, Agno, and LangGraph overhead matrix."""

from __future__ import annotations

import argparse
import json
import os
import platform
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from statistics import fmean, median
from typing import Any

try:
    import psutil
except ImportError as exc:  # pragma: no cover - exercised by the CLI environment
    raise SystemExit("psutil is required; run this script with uv --with psutil") from exc


FRAMEWORKS = ("hno", "agno", "langgraph")


def percentile(values: list[float], rank: float) -> float:
    if len(values) == 1:
        return values[0]
    position = (len(values) - 1) * rank
    lower = int(position)
    upper = min(lower + 1, len(values) - 1)
    fraction = position - lower
    return values[lower] + (values[upper] - values[lower]) * fraction


def summarize(samples: list[dict[str, Any]], measured_elapsed_ns: int) -> dict[str, Any]:
    durations = sorted(float(sample["duration_ns"]) / 1_000_000 for sample in samples)
    success_count = sum(1 for sample in samples if sample.get("success"))
    elapsed_s = measured_elapsed_ns / 1_000_000_000
    return {
        "count": len(samples),
        "mean_ms": fmean(durations),
        "median_ms": median(durations),
        "p95_ms": percentile(durations, 0.95),
        "min_ms": durations[0],
        "max_ms": durations[-1],
        "success_count": success_count,
        "success_rate": success_count / len(samples) if samples else 0.0,
        "measured_elapsed_ms": measured_elapsed_ns / 1_000_000,
        "measured_rps": len(samples) / elapsed_s if elapsed_s else 0.0,
    }


def reserve_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def wait_for_stub(process: subprocess.Popen[str], endpoint: str) -> None:
    deadline = time.monotonic() + 15
    health_url = endpoint + "/health"
    while time.monotonic() < deadline:
        if process.poll() is not None:
            stderr = process.stderr.read() if process.stderr else ""
            raise RuntimeError(f"stub exited with {process.returncode}: {stderr[-2000:]}")
        try:
            with urllib.request.urlopen(health_url, timeout=1) as response:
                if response.status == 200:
                    return
        except OSError:
            time.sleep(0.1)
    raise TimeoutError(f"stub did not become ready: {health_url}")


def sample_process(
    command: list[str],
    cwd: Path,
    env: dict[str, str],
    raw_path: Path,
) -> dict[str, Any]:
    started = time.perf_counter()
    stdout_handle = raw_path.open("w", encoding="utf-8")
    process = subprocess.Popen(
        command,
        cwd=cwd,
        env=env,
        stdout=stdout_handle,
        stderr=subprocess.PIPE,
        text=True,
    )
    monitored = psutil.Process(process.pid)
    peak_rss = 0
    cpu_seconds = 0.0

    def process_tree() -> list[psutil.Process]:
        try:
            return [monitored, *monitored.children(recursive=True)]
        except psutil.Error:
            return [monitored]

    def sample_resources() -> None:
        nonlocal peak_rss, cpu_seconds
        current_rss = 0
        current_cpu = 0.0
        for item in process_tree():
            try:
                current_rss += item.memory_info().rss
                times = item.cpu_times()
                current_cpu += times.user + times.system
            except psutil.Error:
                continue
        peak_rss = max(peak_rss, current_rss)
        cpu_seconds = max(cpu_seconds, current_cpu)

    try:
        sample_resources()
    except psutil.Error:
        pass

    while True:
        sample_resources()
        if process.poll() is not None:
            break
        time.sleep(0.01)

    _, stderr = process.communicate()
    stdout_handle.close()
    wall_seconds = time.perf_counter() - started
    if process.returncode != 0:
        raise RuntimeError(
            f"benchmark failed ({process.returncode}): {' '.join(command)}\n{stderr[-4000:]}"
        )
    return {
        "exit_code": process.returncode,
        "wall_time_s": wall_seconds,
        "peak_rss_mb": peak_rss / 1024 / 1024,
        "cpu_time_s": cpu_seconds,
        "average_cpu_percent": cpu_seconds / wall_seconds * 100 if wall_seconds else 0.0,
    }


def load_framework_samples(path: Path, framework: str) -> tuple[list[dict[str, Any]], int]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if framework == "hno":
        samples = data["samples"]
        elapsed_ns = int(data["measured_elapsed_ns"])
    else:
        result = data["frameworks"][framework]
        samples = result["samples"]
        elapsed_ns = int(result["measured_elapsed_ns"])
    return samples, elapsed_ns


def build_hno(repo_root: Path, output_dir: Path) -> Path:
    binary = output_dir / ("hno_matrix_runner.exe" if os.name == "nt" else "hno_matrix_runner")
    subprocess.run(
        ["go", "build", "-o", str(binary), "./benchmarks/framework_comparison/hno_local_runner"],
        cwd=repo_root,
        check=True,
    )
    return binary


def markdown(report: dict[str, Any]) -> str:
    lines = [
        "# Local fixed-model overhead matrix",
        "",
        f"Captured: `{report['captured_at_utc']}`",
        f"Model: `{report['protocol']['model']}` | Stub delay: `{report['protocol']['stub_delay_ms']} ms`",
        f"Measured runs: `{report['protocol']['runs']}` per framework/concurrency | Warmup: `{report['protocol']['warmup']}`",
        f"Lifecycle: `{report['protocol']['lifecycle']}`",
        f"Environment: `{report['environment']['system']} {report['environment']['machine']}` | "
        f"Python `{report['environment']['python']}` | `{report['environment']['go']}`",
        "",
        "All frameworks used the same local OpenAI-compatible stub and returned the same fixed response. "
        "The measured latency excludes warmup but includes each framework's request preparation and HTTP client path.",
        "",
    ]
    for concurrency in report["protocol"]["concurrencies"]:
        lines.extend(
            [
                f"## Concurrency {concurrency}",
                "",
                "| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |",
                "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
            ]
        )
        for framework in FRAMEWORKS:
            item = report["matrix"][str(concurrency)][framework]
            summary = item["summary"]
            resources = item["resources"]
            lines.append(
                f"| {framework.upper() if framework == 'hno' else framework.title()} | "
                f"{summary['mean_ms']:.3f} | {summary['median_ms']:.3f} | {summary['p95_ms']:.3f} | "
                f"{summary['measured_rps']:.2f} | {summary['success_count']}/{summary['count']} | "
                f"{resources['peak_rss_mb']:.1f} | {resources['cpu_time_s']:.3f} | "
                f"{resources['wall_time_s']:.3f} |"
            )
        lines.append("")

    lines.extend(
        [
            "## Resource metric definitions",
            "",
            "- `Peak RSS MB` is the maximum resident set observed for the framework process tree, including interpreter/runtime startup.",
            "- `CPU s` is user plus system CPU time observed for that process tree.",
            "- `Process wall s` includes process startup, warmups, and measured requests; `Measured RPS` uses only the measured batch elapsed time.",
            "- The stub is shared infrastructure and is not attributed to any framework.",
            "",
            "## Limits",
            "",
        ]
    )
    lines.extend(f"- {limit}" for limit in report["limits"])
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=100)
    parser.add_argument("--warmup", type=int, default=5)
    parser.add_argument("--concurrencies", default="1,8,32")
    parser.add_argument("--delay-ms", type=float, default=1)
    parser.add_argument("--port", type=int, default=18081)
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path("benchmarks/framework_comparison/results/local_stub_matrix"),
    )
    args = parser.parse_args()
    if args.runs < 1 or args.warmup < 0:
        raise SystemExit("runs must be positive and warmup must be non-negative")
    concurrencies = [int(value) for value in args.concurrencies.split(",") if value.strip()]
    if not concurrencies or any(value < 1 for value in concurrencies):
        raise SystemExit("concurrencies must contain positive integers")

    repo_root = Path(__file__).resolve().parents[2]
    output_dir = args.output_dir if args.output_dir.is_absolute() else repo_root / args.output_dir
    output_dir.mkdir(parents=True, exist_ok=True)
    endpoint = f"http://127.0.0.1:{args.port}"
    stub_script = Path(__file__).with_name("local_openai_stub.py")
    python_runner = Path(__file__).with_name("real_local.py")
    base_env = os.environ.copy()
    for key in ("AGNES_API_KEY", "AGNES_BASE_URL", "AGNES_MODEL"):
        base_env.pop(key, None)
    base_env["PYTHONUNBUFFERED"] = "1"

    with tempfile.TemporaryDirectory(prefix="hno-local-matrix-") as temp_dir:
        binary = build_hno(repo_root, Path(temp_dir))
        stub = subprocess.Popen(
            [
                sys.executable,
                str(stub_script),
                "--host",
                "127.0.0.1",
                "--port",
                str(args.port),
                "--model",
                "stub-model",
                "--delay-ms",
                str(args.delay_ms),
            ],
            cwd=repo_root,
            env=base_env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            wait_for_stub(stub, endpoint)
            matrix: dict[str, dict[str, Any]] = {}
            raw_files: dict[str, dict[str, str]] = {}
            for concurrency in concurrencies:
                matrix[str(concurrency)] = {}
                raw_files[str(concurrency)] = {}
                for framework in FRAMEWORKS:
                    raw_path = output_dir / f"{framework}_simple_c{concurrency}.json"
                    if framework == "hno":
                        command = [
                            str(binary),
                            "-endpoint",
                            endpoint,
                            "-model",
                            "stub-model",
                            "-scenario",
                            "simple",
                            "-warmup",
                            str(args.warmup),
                            "-runs",
                            str(args.runs),
                            "-concurrency",
                            str(concurrency),
                            "-lifecycle",
                            "fresh",
                        ]
                    else:
                        command = [
                            sys.executable,
                            str(python_runner),
                            "--endpoint",
                            endpoint,
                            "--model",
                            "stub-model",
                            "--scenario",
                            "simple",
                            "--framework",
                            framework,
                            "--warmup",
                            str(args.warmup),
                            "--runs",
                            str(args.runs),
                            "--concurrency",
                            str(concurrency),
                        ]
                    resources = sample_process(command, repo_root, base_env, raw_path)
                    samples, elapsed_ns = load_framework_samples(raw_path, framework)
                    matrix[str(concurrency)][framework] = {
                        "summary": summarize(samples, elapsed_ns),
                        "resources": resources,
                    }
                    raw_files[str(concurrency)][framework] = str(raw_path.relative_to(repo_root))
        finally:
            stub.terminate()
            try:
                stub.wait(timeout=5)
            except subprocess.TimeoutExpired:
                stub.kill()
                stub.wait()

    report = {
        "report_type": "local_fixed_model_overhead_matrix",
        "captured_at_utc": datetime.now(timezone.utc).isoformat(),
        "status": "complete",
        "protocol": {
            "frameworks": list(FRAMEWORKS),
            "model": "stub-model",
            "stub_response": "LOCAL_MODEL_OK",
            "stub_delay_ms": args.delay_ms,
            "warmup": args.warmup,
            "runs": args.runs,
            "concurrencies": concurrencies,
            "measurement": "per-request client wall-clock samples plus process RSS and CPU telemetry",
            "lifecycle": "fresh operation: client/model/Agent/Graph setup is included per operation",
        },
        "environment": {
            "system": platform.system(),
            "release": platform.release(),
            "machine": platform.machine(),
            "processor": platform.processor(),
            "python": platform.python_version(),
            "go": subprocess.run(["go", "version"], cwd=repo_root, check=True, capture_output=True, text=True).stdout.strip(),
        },
        "matrix": matrix,
        "raw_files": raw_files,
        "limits": [
            "The local stub removes remote provider variance but is not a model-quality or provider benchmark.",
            "The same HTTP endpoint and fixed response are used by every framework.",
            "Peak RSS includes runtime and import overhead; it is not an allocation-per-request measurement.",
            "Agno and LangGraph are run in separate processes so their RSS and CPU values are not combined.",
            "This measures the simple one-request scenario; tool loops, memory, streaming, and teams need separate rows.",
        ],
    }
    output_json = output_dir / "latest.json"
    output_md = output_dir / "latest.md"
    output_json.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    output_md.write_text(markdown(report), encoding="utf-8")
    print(markdown(report), end="")


if __name__ == "__main__":
    main()
