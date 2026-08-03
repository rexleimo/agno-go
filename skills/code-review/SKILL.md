---
name: code-review
description: Review Go code changes for correctness, security, and style; report findings by severity.
---

# Code Review

## When to use
Use this skill when the user asks to review code, a diff, or a pull request.

## Steps
1. Understand the change: what is the intent, what files are touched.
2. Check for: logic errors, race conditions, error handling, security issues (injection, secrets, path traversal).
3. Check project conventions: error wrapping (`fmt.Errorf("ctx: %w", err)`), exported symbols documented, table-driven tests.
4. Verify test coverage of the changed behavior.

## Output format
Report findings grouped by severity:
- **Critical**: must fix before merge (bugs, security).
- **Warning**: should fix (error handling, edge cases).
- **Nit**: style, naming, minor improvements.

End with a verdict: approve / request changes.
