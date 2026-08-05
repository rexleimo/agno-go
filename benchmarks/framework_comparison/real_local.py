#!/usr/bin/env python3
"""Run OpenAI-compatible model scenarios through Agno and LangGraph.

The endpoint must expose the OpenAI-compatible /v1 API. The script records
client wall-clock latency, not server RSS.
"""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import os
import statistics
import time
import uuid
from pathlib import Path
from typing import Annotated, Any, TypedDict

from agno.agent import Agent
from agno.models.openai import OpenAIChat
from agno.models.message import Message
from langchain_core.messages import HumanMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import END, START, StateGraph
from langgraph.prebuilt import ToolNode, tools_condition
from langgraph.graph.message import add_messages
from langchain_core.tools import tool


class MessageState(TypedDict):
    messages: Annotated[list[Any], add_messages]


CACHE_PREFIX = (
    "You are a deterministic benchmark assistant. Follow the user instruction exactly. "
    "Keep the answer concise and do not add explanations. "
) * 80


def load_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip().strip('"\'')
    return values


def base_url(endpoint: str) -> str:
    value = endpoint.rstrip("/")
    return value if value.endswith("/v1") else value + "/v1"


def prompt(scenario: str) -> str:
    prefix = CACHE_PREFIX + "\n" if os.getenv("AGNES_API_KEY") else "/no_think\n"
    if scenario == "simple":
        expected = "REMOTE_MODEL_OK" if os.getenv("AGNES_API_KEY") else "LOCAL_MODEL_OK"
        return prefix + f"Reply with exactly: {expected}"
    if scenario == "tool":
        return prefix + "Use the add tool to calculate 25 + 17. After the tool returns, reply with exactly: RESULT_42"
    if scenario == "memory":
        return prefix + "Remember this code exactly: BLUE-42. Reply with ACK only."
    raise ValueError(f"unsupported scenario: {scenario}")


def success(scenario: str, content: str) -> bool:
    text = content.upper()
    if scenario == "simple":
        expected = "REMOTE_MODEL_OK" if os.getenv("AGNES_API_KEY") else "LOCAL_MODEL_OK"
        return expected in text
    if scenario == "memory":
        return "BLUE-42" in text
    return "RESULT_42" in text or "42" in text


def local_agno_model(model_id: str, endpoint: str) -> OpenAIChat:
    api_key = os.getenv("AGNES_API_KEY", "local")
    options = dict(
        id=model_id,
        api_key=api_key,
        base_url=base_url(endpoint),
        temperature=0,
        seed=42,
        max_tokens=128,
    )
    if api_key == "local":
        options["extra_body"] = {"chat_template_kwargs": {"enable_thinking": False}}
    return OpenAIChat(**options)


def local_langchain_model(model_id: str, endpoint: str) -> ChatOpenAI:
    api_key = os.getenv("AGNES_API_KEY", "local")
    options = dict(
        model=model_id,
        api_key=api_key,
        base_url=base_url(endpoint),
        temperature=0,
        seed=42,
        max_tokens=128,
    )
    if api_key == "local":
        options["extra_body"] = {"chat_template_kwargs": {"enable_thinking": False}}
    return ChatOpenAI(**options)


def add(a: int, b: int) -> int:
    """Add two integers."""
    return a + b


@tool
def langchain_add(a: int, b: int) -> int:
    """Add two integers."""
    return a + b


def create_langgraph(model_id: str, endpoint: str, scenario: str):
    llm = local_langchain_model(model_id, endpoint)
    if scenario == "tool":
        llm = llm.bind_tools([langchain_add])

    def call_model(state: MessageState):
        return {"messages": [llm.invoke(state["messages"])]}

    graph = StateGraph(MessageState)
    graph.add_node("model", call_model)
    graph.add_edge(START, "model")
    if scenario == "tool":
        graph.add_node("tools", ToolNode([langchain_add]))
        graph.add_conditional_edges("model", tools_condition)
        graph.add_edge("tools", "model")
    else:
        graph.add_edge("model", END)
    return graph.compile()


def run_agno(model_id: str, endpoint: str, scenario: str) -> str:
    model = local_agno_model(model_id, endpoint)
    agent = Agent(
        model=model,
        name="local-llama-benchmark",
        tools=[add] if scenario == "tool" else None,
        telemetry=False,
        markdown=False,
        session_id=f"benchmark-{uuid.uuid4()}",
    )
    if scenario == "memory":
        prefix = CACHE_PREFIX + "\n" if os.getenv("AGNES_API_KEY") else "/no_think\n"
        first = agent.run(prompt("memory"))
        second_input = first.messages + [
            Message(role="user", content=prefix + "What code did I ask you to remember? Reply with the code only.")
        ]
        result = agent.run(second_input)
    else:
        result = agent.run(prompt(scenario))
    return str(result.content or "")


def run_langgraph(model_id: str, endpoint: str, scenario: str) -> str:
    graph = create_langgraph(model_id, endpoint, scenario)
    result = graph.invoke({"messages": [HumanMessage(content=prompt(scenario))]})
    if scenario == "memory":
        prefix = CACHE_PREFIX + "\n" if os.getenv("AGNES_API_KEY") else "/no_think\n"
        result = graph.invoke(
            {
                "messages": result["messages"]
                + [HumanMessage(content=prefix + "What code did I ask you to remember? Reply with the code only.")]
            }
        )
    return str(result["messages"][-1].content or "")


def measure(factory, scenario: str, warmup: int, runs: int, concurrency: int) -> dict[str, Any]:
    if warmup < 0 or runs < 1 or concurrency < 1:
        raise ValueError("warmup must be non-negative, runs positive, and concurrency positive")
    for _ in range(warmup):
        factory()

    def measure_one(_: int) -> dict[str, Any]:
        started = time.perf_counter_ns()
        try:
            content = factory()
            error = None
        except Exception as exc:  # noqa: BLE001 - preserve provider failures in the report
            content = ""
            error = f"{type(exc).__name__}: {exc}"
        duration_ns = time.perf_counter_ns() - started
        return {
            "duration_ns": duration_ns,
            "content": content[:500],
            "success": error is None and success(scenario, content),
            **({"error": error} if error else {}),
        }

    measured_started = time.perf_counter_ns()
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        samples = list(executor.map(measure_one, range(runs)))
    measured_elapsed_ns = time.perf_counter_ns() - measured_started
    durations = [item["duration_ns"] for item in samples]
    return {
        "warmup": warmup,
        "runs": runs,
        "concurrency": concurrency,
        "measured_elapsed_ns": measured_elapsed_ns,
        "samples": samples,
        "mean_ns": statistics.fmean(durations),
        "median_ns": statistics.median(durations),
        "p95_ns": statistics.quantiles(durations, n=20, method="inclusive")[18] if len(durations) >= 2 else durations[0],
        "min_ns": min(durations),
        "max_ns": max(durations),
        "success_count": sum(1 for item in samples if item["success"]),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", type=Path, help="optional local env file; secrets are not printed")
    parser.add_argument("--endpoint")
    parser.add_argument("--model")
    parser.add_argument("--scenario", choices=["simple", "tool", "memory"], default="simple")
    parser.add_argument("--framework", choices=["all", "agno", "langgraph"], default="all")
    parser.add_argument("--warmup", type=int, default=3)
    parser.add_argument("--runs", type=int, default=100)
    parser.add_argument("--concurrency", type=int, default=8)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    config = load_env(args.config) if args.config else {}
    for key, value in config.items():
        os.environ.setdefault(key, value)
    args.endpoint = args.endpoint or config.get("AGNES_BASE_URL") or "http://127.0.0.1:8081"
    args.model = args.model or config.get("AGNES_MODEL") or r"models\Qwen3-4B-Q8_0.gguf"

    frameworks = {}
    if args.framework in {"all", "agno"}:
        frameworks["agno"] = measure(
            lambda: run_agno(args.model, args.endpoint, args.scenario),
            args.scenario,
            args.warmup,
            args.runs,
            args.concurrency,
        )
    if args.framework in {"all", "langgraph"}:
        frameworks["langgraph"] = measure(
            lambda: run_langgraph(args.model, args.endpoint, args.scenario),
            args.scenario,
            args.warmup,
            args.runs,
            args.concurrency,
        )

    result = {
        "endpoint": args.endpoint,
        "model": args.model,
        "scenario": args.scenario,
        "framework": args.framework,
        "lifecycle": "fresh operation",
        "warmup": args.warmup,
        "runs": args.runs,
        "concurrency": args.concurrency,
        "frameworks": frameworks,
        "limits": [
            "All requests use the same OpenAI-compatible endpoint and model.",
            "Measurements are client wall-clock time and include framework request preparation.",
            "The model may be local or remote; provider and network latency are included.",
            "Server-side model load, CPU, and RSS are not attributed to an individual framework.",
        ],
    }
    encoded = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(encoded, encoding="utf-8")
    print(encoded, end="")


if __name__ == "__main__":
    main()
