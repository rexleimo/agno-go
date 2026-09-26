// 校验 docs/design/v3-delivery-ticket.json 是否符合 rex-harness 的 Delivery Ticket schema。
// 权威校验逻辑来自 harness 自身，此处只做调用与人类可读输出，避免手写校验漂移。
//
// 用法：
//   node scripts/validate-delivery-ticket.mjs
//   node scripts/validate-delivery-ticket.mjs path/to/ticket.json
//
// 环境变量：
//   REX_HARNESS_ENTRY  可选。指向 harness 的 planning-artifact.mjs 绝对路径。
//                     未设置时按以下顺序探测：$REX_HOME → ~/.rexcil/harness-cli/rex-harness
//
// 退出码：0 = 合规；1 = 不合规（stderr 打印 harness 原始错误）
import { existsSync, readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, resolve } from 'node:path';

const RELATIVE_ENTRY = 'src/domain/planning-artifact.mjs';

// 按优先级探测 harness 的 planning-artifact.mjs；找不到则返回 null。
// 探测顺序：显式环境变量 → REX_HOME → 默认安装位置。
function resolveHarnessEntry() {
  const candidates = [];

  if (process.env.REX_HARNESS_ENTRY) {
    candidates.push(process.env.REX_HARNESS_ENTRY);
  }
  if (process.env.REX_HOME) {
    candidates.push(join(process.env.REX_HOME, RELATIVE_ENTRY));
  }
  candidates.push(join(homedir(), '.rexcil', 'harness-cli', 'rex-harness', RELATIVE_ENTRY));

  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return candidate;
    }
  }
  return null;
}

const ticketPath = resolve(process.argv[2] || 'docs/design/v3-delivery-ticket.json');
const harnessEntry = resolveHarnessEntry();

if (!harnessEntry) {
  console.error('无法定位 harness 的 planning-artifact.mjs。');
  console.error('请设置 REX_HARNESS_ENTRY 指向该文件的绝对路径。');
  process.exit(1);
}

let normalizePlanningArtifact;
try {
  ({ normalizePlanningArtifact } = await import(harnessEntry));
} catch (err) {
  console.error(`无法加载 harness schema 校验器：${harnessEntry}`);
  console.error('请确认 rex-harness 已安装，或设置 REX_HARNESS_ENTRY。');
  console.error(String(err));
  process.exit(1);
}

let raw;
try {
  raw = JSON.parse(readFileSync(ticketPath, 'utf8'));
} catch (err) {
  console.error(`无法解析交付票 JSON：${ticketPath}`);
  console.error(String(err));
  process.exit(1);
}

try {
  const t = normalizePlanningArtifact(raw);
  console.log(`VALID  ${ticketPath}`);
  console.log(`  kind             ${t.kind}`);
  console.log(`  status           ${t.status}`);
  console.log(`  objective        ${t.objective.length} chars`);
  console.log(`  decisionTicket   ${t.decisionTicketRef}`);
  console.log(`  workItems        ${t.workItems.length}`);
  console.log(`  completionClaim  ${t.completionClaim}`);
  console.log(`  frontier.ready   ${t.frontier.ready.length}  blocked ${t.frontier.blocked.length}`);
  console.log(`  parallelGroups   ${t.parallelGroups.length}`);
  for (const w of t.workItems) {
    console.log(`    ${w.id.padEnd(36)} deps=[${w.dependsOn.join(', ')}]`);
  }
  process.exit(0);
} catch (err) {
  console.error(`INVALID  ${ticketPath}`);
  console.error(String(err && err.message ? err.message : err));
  process.exit(1);
}
