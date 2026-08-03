---
name: web-research
description: Research a topic on the web and return a structured summary with cited sources.
---

# Web Research

## When to use
Use this skill when the user asks to research, investigate, or find information about a topic on the web.

## Steps
1. Formulate 2-3 search queries covering different angles of the topic.
2. Fetch and read the top results for each query.
3. Extract key facts, numbers, and claims with their source URLs.
4. Write a structured summary: overview, key findings, sources.

## Output format
Return the summary as markdown with a "## Sources" section listing URLs.

## Notes
- Prefer primary sources (official docs, papers) over secondary articles.
- If results conflict, note the disagreement explicitly.
