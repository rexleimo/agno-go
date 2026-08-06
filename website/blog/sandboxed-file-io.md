---
title: "Agent File Tools Need Root-Bound Sandboxes, Not Path Prefix Checks"
description: "How HNO confines agent file reads and writes with explicit capabilities, root-bound Go filesystem handles, bounded I/O, and a separate deployment-isolation layer."
date: 2026-08-06
lastUpdated: 2026-08-06
author: HNO Team
category: Security engineering
tags:
  - AI agents
  - agent security
  - sandbox
  - file I/O
  - Go
head:
  - - meta
    - name: keywords
      content: "AI agent file sandbox, secure agent file I/O, Go os.Root, agent tool security, pathname confinement"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "Agent File Tools Need Root-Bound Sandboxes, Not Path Prefix Checks"
  - - meta
    - property: og:description
      content: "How HNO confines agent file tools with explicit roots and root-bound filesystem operations."
  - - meta
    - property: article:published_time
      content: "2026-08-06T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/blog/sandboxed-file-io
---

# Agent File Tools Need Root-Bound Sandboxes, Not Path Prefix Checks

An Agent that can read a report, inspect a template, or create an artifact is
useful. An Agent that can resolve arbitrary host paths is a security boundary
waiting to fail.

The distinction is easy to miss because both versions expose functions with
nearly identical names: `read_file`, `write_file`, and `create_file`. The
important question is not the function name. It is: **what does the filesystem
actually authorize after an untrusted model has supplied a path?**

HNO's new sandboxed file-I/O design answers that with explicit read and write
capabilities, root-relative agent paths, and Go's root-bound filesystem handles.

<img src="/diagrams/sandboxed-file-io.svg" alt="HNO sandboxed file I/O architecture: agent tool calls pass through capability policy and root-bound os.Root handles to separate read and write roots." />

## The trap in a path-prefix allowlist

A first implementation often looks like this:

1. make an untrusted path absolute;
2. compare it with an allowed directory;
3. call `os.ReadFile` or `os.WriteFile` on that path.

That catches many obvious `../` cases. It is still not enough for an Agent
runtime. A symbolic link can change what the checked name resolves to, and the
filesystem state can change between validation and the unrestricted I/O call.
Windows also introduces drive-relative, UNC, device, and alternate-data-stream
forms that a portable string check must account for.

The lesson is not that input validation is useless. It is that validation should
provide early rejection and clear errors, while the actual filesystem primitive
must enforce the boundary.

## The architecture: capability first, handle-bound enforcement second

HNO creates a sandbox with explicit roots:

```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./inputs", "./templates"),
    file.WithWriteRoot("./workspace"),
    file.WithMaxReadBytes(10<<20),
    file.WithMaxWriteBytes(5<<20),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()

readWriteTools := file.NewWithSandbox(sandbox)
generationTools := filegen.NewWithSandbox(sandbox)
```

The flow has three layers:

1. **Toolkits** expose familiar Agent tools. `file.FileTools` handles reading,
   listing, metadata, PPTX extraction, writing, and deletion. `filegen` handles
   artifact creation.
2. **The sandbox policy** decides whether a requested operation has read or
   write capability, rejects unsafe root-relative names, applies byte limits,
   controls overwrites, and can emit audit entries.
3. **`os.Root` handles** perform the real `Open`, `Stat`, `OpenFile`, `Mkdir`,
   `OpenRoot`, and `Remove` operations. A resolved path cannot leave the
   directory tree represented by the handle, including by following a symbolic
   link outside that tree.

The path supplied by an Agent is not a host path. It is a name relative to a
configured root, such as `reports/summary.md`.

## Read and write are different grants

A read root does not silently create write access, and a write root does not
silently create read access. That makes it practical to assemble smaller Agent
capabilities:

- a reviewer can read `./inputs` and `./templates` but cannot change them;
- a generator can write only to `./workspace`;
- a run can use a private per-tenant workspace without exposing the host
  checkout or home directory;
- a sandbox with no roots denies all operations by default.

Multiple read roots are checked in configuration order. The first one containing
the requested relative name is used. There is exactly one write root, which
keeps generated artifacts in a single intentional destination.

## Generation needs an additional overwrite decision

`filegen.create_file` has an `overwrite` tool argument because creating a new
artifact and replacing an existing one are materially different actions.

The sandbox requires two independent approvals before replacement:

- the tool call says `overwrite: true`; and
- the host configured `file.WithAllowOverwrite(true)`.

This prevents a model from replacing a file simply because the application has
the technical ability to do so. New files are created exclusively by default.
For Go 1.24 compatibility, an authorized replacement uses a root-bound
truncate-and-write operation rather than an atomic rename; critical publication
flows should generate and validate a new artifact before a higher-level,
controlled release step promotes it.

## What the sandbox does not replace

This feature confines **file-tool path resolution**. It does not turn arbitrary
code execution into a safe operation. It does not restrict outbound network
access, CPU, memory, processes, mounts, or privileged resources that already
exist inside a granted root.

For an untrusted or multi-tenant workload, use the file sandbox as one layer in
a defense-in-depth design:

- a private volume or directory tree per run or tenant;
- least-privilege service accounts and filesystem ACLs;
- containers, VMs, or a dedicated execution sandbox for processes;
- memory, CPU, process, and disk quotas;
- egress controls and durable audit logging.

This boundary statement matters as much as the implementation. A path sandbox
is strong at preventing pathname escape when given a careful root. It is not an
excuse to mount a sensitive host filesystem into an untrusted execution
environment.

## A testable security contract

The regression suite covers the operational contract rather than only happy
paths: fail-closed startup, separated capabilities, path traversal, absolute
and Windows special paths, symbolic-link escape attempts, final-link targets,
read/write size limits, parent creation, audit events, and the `filegen`
overwrite rule.

Run the focused checks with:

```bash
go test ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/... -v
go vet ./pkg/hno/tools/file/... ./pkg/hno/tools/filegen/...
```

For the complete API contract, deployment guidance, and the full architecture
walkthrough, see the [Sandboxed File I/O guide](/guide/sandboxed-file-io).