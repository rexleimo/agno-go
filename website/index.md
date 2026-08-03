---
layout: home

hero:
  name: "HNO"
  text: "Go-Native Multi-Agent Framework"
  tagline: "Type-safe agents, teams and workflows. 17 providers, one abstraction. Zero cgo."
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View on GitHub
      link: https://github.com/rexleimo/agno-go

features:
  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2L3 14h7l-1 8 10-12h-7l1-8z"/></svg>'
    title: Extreme Performance
    details: Go-native runtime. Agent instantiation in ~180ns at ~1.2KB per agent — orders of magnitude faster than Python counterparts.

  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="9" rx="1"/><rect x="14" y="3" width="7" height="5" rx="1"/><rect x="14" y="12" width="7" height="9" rx="1"/><rect x="3" y="16" width="7" height="5" rx="1"/></svg>'
    title: One Abstraction, 17 Providers
    details: OpenAI, Anthropic, Gemini, DeepSeek, GLM and more behind a single Model interface. Add a provider in under 200 lines.

  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/></svg>'
    title: Shared Orchestration Kernel
    details: Agents, teams and workflows run on one loop kernel. New orchestrators (supervisors, routers) compose in tens of lines.

  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19V5m0 14a2 2 0 0 0 2 2h14M4 19a2 2 0 0 1 2-2h14M6 9h12M6 13h8"/></svg>'
    title: Observability Built In
    details: OpenTelemetry spans with GenAI semantic conventions, cost estimation, retries, rate limiting and circuit breakers — zero config, zero cost when unused.

  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2l2.4 7.4H22l-6 4.6 2.3 7.4-6.3-4.5-6.3 4.5L8 14 2 9.4h7.6z"/></svg>'
    title: Skills, MCP, Memory
    details: Agent Skills with progressive disclosure, first-class MCP server bridging, and pluggable memory and session storage.

  - icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4"/><path d="M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6z"/></svg>'
    title: Verified Protocols
    details: Chat, Anthropic, Gemini and Responses protocols verified against local real model inference — not just mocks.
---
