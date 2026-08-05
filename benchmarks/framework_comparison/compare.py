#!/usr/bin/env python3
"""Compare framework construction costs without making LLM or network calls.

The HNO measurement is delegated to the checked-in Go benchmark. Python
frameworks are measured with timeit using the same number of repeated samples.
Allocation units are intentionally not compared across Go and Python.
"""

from __future__ import annotations

import argparse
import gc
import hashlib
import importlib.metadata
import json
import platform
import re
import statistics
import subprocess
import sys
import timeit
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


GO_BENCH_RE = re.compile(
    r"^(BenchmarkAgentCreation(?:WithTools|WithMemory)?|BenchmarkAgentFreshRunLocalDummy)-\d+\s+"
    r"\d+\s+([0-9.]+) ns/op\s+([0-9]+) B/op\s+([0-9]+) allocs/op"
)


def summary(samples: list[float]) -> dict[str, float | int | list[float]]:
    ordered = sorted(samples)
    return {
        "samples": samples,
        "min": ordered[0],
        "median": statistics.median(ordered),
        "max": ordered[-1],
        "mean": statistics.fmean(samples),
    }


def run_hno(repo_root: Path) -> dict[str, Any]:
    command = [
        "go",
        "test",
        "-run=^$",
        "-bench=BenchmarkAgentCreation|BenchmarkAgentFreshRunLocalDummy",
        "-benchmem",
        "-count=10",
        "./pkg/hno/agent/",
    ]
    completed = subprocess.run(
        command,
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
    )
    cpu_match = re.search(r"^cpu:\s+(.+)$", completed.stdout, re.MULTILINE)
    results: dict[str, dict[str, Any]] = {}
    for line in completed.stdout.splitlines():
        match = GO_BENCH_RE.match(line.strip())
        if not match:
            continue
        name, ns, bytes_per_op, allocs = match.groups()
        entry = results.setdefault(
            name,
            {"ns_samples": [], "bytes_per_op": int(bytes_per_op), "allocs_per_op": int(allocs)},
        )
        entry["ns_samples"].append(float(ns))
    expected = {
        "BenchmarkAgentCreation",
        "BenchmarkAgentCreationWithTools",
        "BenchmarkAgentCreationWithMemory",
        "BenchmarkAgentFreshRunLocalDummy",
    }
    missing = expected.difference(results)
    if missing:
        raise RuntimeError(f"HNO benchmark output did not contain: {sorted(missing)}\n{completed.stdout}")
    for entry in results.values():
        entry["ns_op"] = summary(entry.pop("ns_samples"))
    return {
        "command": command,
        "cpu": cpu_match.group(1).strip() if cpu_match else None,
        "benchmarks": results,
    }


def make_agno_model():
    from agno.models.base import Model
    from agno.models.response import ModelResponse

    class DummyModel(Model):
        def __init__(self):
            super().__init__(id="comparison-dummy", provider="comparison")

        def invoke(self, *args, **kwargs):
            return ModelResponse(content="ok")

        def invoke_stream(self, *args, **kwargs):
            yield ModelResponse(content="ok")

        async def ainvoke(self, *args, **kwargs):
            return ModelResponse(content="ok")

        async def ainvoke_stream(self, *args, **kwargs):
            yield ModelResponse(content="ok")

        def _parse_provider_response(self, *args, **kwargs):
            return None

        def _parse_provider_response_delta(self, *args, **kwargs):
            return None

    return DummyModel()


def python_samples(factory, repeat: int, number: int) -> dict[str, Any]:
    for _ in range(100):
        factory()
    timer = timeit.Timer(factory)
    # timeit disables cyclic GC during a sample. This is recorded in the output
    # so the result is not mistaken for a full service memory measurement.
    samples = [value / number * 1_000_000_000 for value in timer.repeat(repeat=repeat, number=number)]
    return {"number_per_sample": number, "gc_disabled_by_timeit": True, "ns_op": summary(samples)}


def run_python(repeat: int, number: int) -> dict[str, Any]:
    from agno.agent import Agent
    from langgraph.graph import END, START, StateGraph
    from typing_extensions import TypedDict

    model = make_agno_model()

    def make_agno_agent():
        return Agent(model=model, name="comparison-agent", telemetry=False)

    def add(a: int, b: int) -> int:
        return a + b

    def make_agno_agent_with_tool():
        return Agent(model=model, name="comparison-agent", tools=[add], telemetry=False)

    def fresh_agno_run():
        return Agent(model=model, name="comparison-agent", telemetry=False).run("ping")

    class State(TypedDict):
        value: str

    def respond(state: State) -> dict[str, str]:
        return {"value": state["value"]}

    def compile_langgraph():
        graph = StateGraph(State)
        graph.add_node("respond", respond)
        graph.add_edge(START, "respond")
        graph.add_edge("respond", END)
        return graph.compile()

    gc.disable()
    try:
        result = {
            "agno_agent": python_samples(make_agno_agent, repeat, number),
            "agno_agent_with_tool": python_samples(make_agno_agent_with_tool, repeat, number),
            "agno_fresh_run_local_dummy": python_samples(fresh_agno_run, repeat, max(100, number // 10)),
            "langgraph_minimal_compile": python_samples(compile_langgraph, repeat, max(100, number // 10)),
        }
    finally:
        gc.enable()
    return {
        "python": platform.python_version(),
        "packages": {
            "agno": importlib.metadata.version("agno"),
            "langgraph": importlib.metadata.version("langgraph"),
        },
        "benchmarks": result,
    }


def run_command(command: list[str], cwd: Path) -> str:
    return subprocess.run(command, cwd=cwd, check=True, capture_output=True, text=True).stdout.strip()


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build_report(repo_root: Path, hno: dict[str, Any], python: dict[str, Any], repeat: int, number: int) -> dict[str, Any]:
    status = run_command(["git", "status", "--short"], repo_root)
    benchmark_sources = [
        repo_root / "pkg/hno/agent/agent_bench_test.go",
        repo_root / "benchmarks/framework_comparison/compare.py",
        repo_root / "benchmarks/framework_comparison/requirements.txt",
    ]
    return {
        "recorded_at_utc": datetime.now(timezone.utc).isoformat(),
        "git_revision": run_command(["git", "rev-parse", "HEAD"], repo_root),
        "worktree_dirty": bool(status),
        "worktree_status": status,
        "benchmark_source_sha256": {str(path.relative_to(repo_root)): sha256_file(path) for path in benchmark_sources},
        "platform": {
            "system": platform.system(),
            "release": platform.release(),
            "machine": platform.machine(),
            "processor": hno.get("cpu") or platform.processor(),
        },
        "go_version": run_command(["go", "version"], repo_root),
        "python": python["python"],
        "packages": python["packages"],
        "python_repeat": repeat,
        "python_number": number,
        "hno": hno,
        "python_frameworks": python["benchmarks"],
        "limits": [
            "All frameworks use local dummy objects; no LLM, network, database, or token generation is measured.",
            "HNO agent creation is compared only with Agno Agent creation.",
            "LangGraph compiles a minimal StateGraph and is not presented as an Agent-construction equivalent.",
            "Go B/op and Python timeit results are different measurement systems and are not compared as memory.",
        ],
    }


def markdown(report: dict[str, Any]) -> str:
    hno = report["hno"]["benchmarks"]
    py = report["python_frameworks"]
    lines = [
        "# Framework comparison snapshot",
        "",
        f"Recorded: `{report['recorded_at_utc']}`",
        f"Revision: `{report['git_revision'][:12]}`",
        f"Worktree: `{'dirty' if report['worktree_dirty'] else 'clean'}` (source hashes are in `latest.json`)",
        f"Platform: `{report['platform']['system']} {report['platform']['machine']}`",
        f"CPU: `{report['platform']['processor']}`",
        f"Go: `{report['go_version']}`",
        f"Python: `{report['python']}`",
        f"Packages: `agno=={report['packages']['agno']}`, `langgraph=={report['packages']['langgraph']}`",
        "",
        "All measurements use local dummy objects. No LLM, network, database, or token generation is involved.",
        "",
        "## HNO and Agno: minimal Agent construction",
        "",
        "| Operation | Framework | Mean ns/op | Median ns/op | Min-max ns/op |",
        "| --- | --- | ---: | ---: | ---: |",
    ]
    pairs = [
        ("Agent creation", "HNO", hno["BenchmarkAgentCreation"]["ns_op"]),
        ("Agent creation", "Agno", py["agno_agent"]["ns_op"]),
        ("Agent creation with one tool", "HNO", hno["BenchmarkAgentCreationWithTools"]["ns_op"]),
        ("Agent creation with one tool", "Agno", py["agno_agent_with_tool"]["ns_op"]),
    ]
    for operation, framework, stats in pairs:
        lines.append(
            f"| {operation} | {framework} | {stats['mean']:.1f} | {stats['median']:.1f} | "
            f"{stats['min']:.1f}-{stats['max']:.1f} |"
        )
    simple_ratio = py["agno_agent"]["ns_op"]["mean"] / hno["BenchmarkAgentCreation"]["ns_op"]["mean"]
    tool_ratio = py["agno_agent_with_tool"]["ns_op"]["mean"] / hno["BenchmarkAgentCreationWithTools"]["ns_op"]["mean"]
    run_ratio = py["agno_fresh_run_local_dummy"]["ns_op"]["mean"] / hno["BenchmarkAgentFreshRunLocalDummy"]["ns_op"]["mean"]
    lines.extend(
        [
            "",
            f"Within this construction-only workload, the Agno/HNO mean ratio is "
            f"**{simple_ratio:.1f}x** for the minimal Agent and **{tool_ratio:.1f}x** with one tool. "
            "This ratio is not an end-to-end agent or service speedup.",
        ]
    )
    lines.extend(
        [
            "",
            "## HNO and Agno: fresh local dummy run",
            "",
            "This operation constructs a fresh Agent and performs one `ping` run with a fixed local response.",
            "",
            "| Framework | Mean ns/op | Median ns/op | Min-max ns/op |",
            "| --- | ---: | ---: | ---: |",
            f"| HNO | {hno['BenchmarkAgentFreshRunLocalDummy']['ns_op']['mean']:.1f} | "
            f"{hno['BenchmarkAgentFreshRunLocalDummy']['ns_op']['median']:.1f} | "
            f"{hno['BenchmarkAgentFreshRunLocalDummy']['ns_op']['min']:.1f}-"
            f"{hno['BenchmarkAgentFreshRunLocalDummy']['ns_op']['max']:.1f} |",
            f"| Agno | {py['agno_fresh_run_local_dummy']['ns_op']['mean']:.1f} | "
            f"{py['agno_fresh_run_local_dummy']['ns_op']['median']:.1f} | "
            f"{py['agno_fresh_run_local_dummy']['ns_op']['min']:.1f}-"
            f"{py['agno_fresh_run_local_dummy']['ns_op']['max']:.1f} |",
            "",
            f"Within this fixed local run, the Agno/HNO mean ratio is **{run_ratio:.1f}x**. "
            "It excludes real model, network, database, and token-generation latency.",
            "",
            "HNO's Go `B/op` and Python's `timeit` time are not memory equivalents. The HNO benchmark uses the checked-in `MockModel`; Agno uses a local `Model` subclass with the same no-network intent.",
            "",
            "## LangGraph: separate operation",
            "",
            "LangGraph is a graph/orchestration library rather than a direct Agent object equivalent. Its result below is therefore reported separately:",
            "",
            f"- Minimal `StateGraph` build + compile median: **{py['langgraph_minimal_compile']['ns_op']['median']:.1f} ns/op**",
            f"- Observed range: **{py['langgraph_minimal_compile']['ns_op']['min']:.1f}-{py['langgraph_minimal_compile']['ns_op']['max']:.1f} ns/op**",
            "",
            "Do not turn these values into a universal framework ranking. The next fair comparison would need the same application semantics, tool loop, state, and output contract in every framework.",
            "",
            "## Reproduce",
            "",
            "```bash",
            "uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' python benchmarks/framework_comparison/compare.py",
            "```",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repeat", type=int, default=10)
    parser.add_argument("--number", type=int, default=100)
    parser.add_argument("--output-dir", type=Path, default=Path("benchmarks/framework_comparison/results"))
    args = parser.parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    hno = run_hno(repo_root)
    python = run_python(args.repeat, args.number)
    report = build_report(repo_root, hno, python, args.repeat, args.number)
    output_dir = (repo_root / args.output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    (output_dir / "latest.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    (output_dir / "latest.md").write_text(markdown(report), encoding="utf-8")
    print(markdown(report))


if __name__ == "__main__":
    main()
