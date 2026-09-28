# 切片 21 REFACTOR（票面 §4 B6 第 2 项：重复节点名构建期拒绝）

工作项 `work-p1-graph-r21-duplicate-node-name` / activation `2d96c915-5fc4-48c2-baa6-b27b7c27b29e` / 阶段 refactor。
契约：`docs/design/v3-test-scope-p1-graph-slice21.json`（D1–D5）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付的实现已经是本判据的最小形状（一个非导出账本字段、一处记账、一个末位检查函数），
没有可合并的重复、没有可下沉的抽象；REFACTOR 阶段零代码改动。
最终字节：`pkg/hno/graph/graph.go = b7790b9f0eab673d6b89ad8fdf1008652ea6b468`（614 行，
GREEN 时同值 —— 事故恢复后按逐字节哈希复核，见 GREEN 文档 §5）。
最终门禁复测：绑定命令 exit 0（`receipt:bd24a87d-73fa-4678-accb-04e8904529b9`），
非测试 LOC 872（≤1500）、导出符号 24、`sync.` 计数 0，`go vet`/`gofmt -l` 干净。

## 2. 完整变异矩阵（`scripts/mutation/p1r21-duplicate-node.mjs`，仓库内可复现）

执行体沿用 p1r19 的骨架（锚唯一性预检、逐字节恢复自检、判红只认 `-v` Test 名）。
命令 A = 绑定场景命令（`TestP1R21_`，`-race -v`），命令 B = `TestP1G_`（既有判据的邻近命令）。
基线：`graph.go = b7790b9f…`；每条变异后 `restored=true`、`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | 检查整段失效（末位改回环检查） | exit 1，杀红 D1/D2/D5 | exit 0 | **杀红**（D1/D2/D5 的存在性牙齿） |
| m2 | 名字槽位传空串（文案不点名） | exit 1，杀红 D1/D5 | exit 0 | **杀红**（可定位性牙齿） |
| m3 | 记账退化成「每次注册都记」 | exit 1，杀红 D3/D4/D5 | exit 1，9 个 `TestP1G_` 判红 | **杀红**（反向对照的钉） |
| m4 | 检查挪到入口检查之前（次序变异） | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限，不算杀红** |

m2 的第一版锚把字符串字面量顶破（无格式指令的常量文案），`go test` 内置的 vet printf 检查先于测试
拦下它 —— 那不是测试杀红，不作数；已重做为「保持 `%q` 指令、传空名」，杀红来自 D1/D5 的测试名。

### 2.1 m4 等价性的读法（观察上限，契约 observabilityLimit 的实测面）

D1–D5 的每张图除重复名外全部成立（端点齐全、可达、无环），所以检查放在 `Validate` 的哪个位置
它们都绿；既有 `TestP1G_` 也没有一行钉「重复名必须晚于端点/可达性报出」。
因此**「末位」今天是 `Validate` 文档固定的设计声明，不是测试事实**。要让次序成为判据，需要一张
「既重复名又有其他非法声明」的图并断言报的是重复名 —— 本片契约没有这一行，按契约不追加；
若日后调用方依赖报错次序，那是一次新的 `rex-test-design`（新 RED），不是本片的补丁。

## 3. 契约完成判据逐条对账

1. D1–D5 每行一个可独立失败的 Test，RED 观察到「今天静默覆盖」形状（无编译/环境失败）——
   `receipt:5964bbf8…`（D1/D2/D5 红）+ D3/D4 反向对照红不了的形状在 m3 下实测可红。✅
2. 牙齿由变异证明：m1/m2/m3 杀红、m4 等价登记（§2）。✅
3. 既有 9 份测试文件零改动，逐文件 `git hash-object --no-filters` 复核一致（GREEN 文档 §3）。✅
4. 票面 §7 门禁全 exit 0：build / test / -race / vet / gofmt（GREEN 文档 §3 表）。✅
5. 结构判据不放宽：`sync.`=0、非测试 LOC 872≤1500、导出符号=24。✅
6. 邻近回归不红：agent / runner / session-contract 三包 `ok`（`receipt:06927728…`）。✅
7. 副本实验（M2）数字在最终字节上重跑并取新回执：绑定命令 `-race` exit 0
   （`receipt:aaea7171…`、`receipt:bd24a87d…`、`receipt:d47899e6…` count=30），无 DATA RACE。✅
8. 写回完成：票面 §4 B6 行 + 新增 §17、母约 §3.5 第 2 项标注已交付且第 6 项收口
   （§3.5 全项闭合）、状态文档 §2/§3/§4 改写并登记事故。✅

## 4. 审查

review 结论见 `docs/design/v3-p1-graph-r21-review-verdict.json`（作者自查，独立性披露同前序切片）。
