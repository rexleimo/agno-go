#!/usr/bin/env node
// 证据指针对账：把 docs/design 里引用的 harness UUID 与 .rex-harness/ 实际存在的对象逐一对账。
// 用法：node scripts/evidence-audit.mjs [--json]
// exit 1 = 存在任何角色都找不到的引用（无法核验的断言）。
import { readFileSync, readdirSync, existsSync, statSync } from "node:fs";
import { join } from "node:path";

const repo = process.cwd();
const docsDir = join(repo, "docs", "design");
const harness = join(repo, ".rex-harness");
const UUID = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/g;

const listFiles = (dir) =>
  existsSync(dir)
    ? readdirSync(dir).filter((f) => {
        try {
          return statSync(join(dir, f)).isFile();
        } catch {
          return false;
        }
      })
    : [];

const own = (dir, ext) => new Set(listFiles(dir).filter((f) => f.endsWith(ext)).map((f) => f.slice(0, -ext.length)));
const receipts = own(join(harness, "receipts"), ".json");
const activations = own(join(harness, "activations"), ".json");
const evidence = own(join(harness, "evidence"), ".ndjson");

// 兜底：出现在任一 harness 文件正文里的 id（前缀写错命名空间时仍能追回对象）。
let blob = "";
for (const sub of ["receipts", "activations", "evidence"]) {
  for (const f of listFiles(join(harness, sub))) blob += readFileSync(join(harness, sub, f), "utf8");
}

const rows = [];
for (const f of listFiles(docsDir)) {
  if (!/\.(json|md)$/.test(f)) continue;
  const text = readFileSync(join(docsDir, f), "utf8");
  for (const id of new Set(text.match(UUID) ?? [])) {
    const where = receipts.has(id) ? "receipt" : activations.has(id) ? "activation" : evidence.has(id) ? "evidence" : blob.includes(id) ? "inside-harness" : "NOWHERE";
    if (where !== "receipt") rows.push({ file: f, id, where });
  }
}

const groupBy = (key) => rows.reduce((m, r) => ((m[r[key]] = (m[r[key]] ?? 0) + 1), m), {});
const nowhere = rows.filter((r) => r.where === "NOWHERE");

if (process.argv.includes("--json")) {
  console.log(JSON.stringify({ byKind: groupBy("where"), nowhere }, null, 2));
} else {
  console.log("non-receipt harness references in docs/design:");
  for (const [kind, n] of Object.entries(groupBy("where"))) console.log(`  ${kind.padEnd(16)} ${n}`);
  console.log(`  total unresolved-as-receipt ${rows.length}, of which unverifiable ${nowhere.length}`);
  for (const r of nowhere) console.log(`  NOWHERE  ${r.file}  ${r.id}`);
}
process.exit(nowhere.length ? 1 : 0);
