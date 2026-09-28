#!/usr/bin/env bash
# 切片 30 D5：R17 取消归一化族的合成负载协议（S24-STD-2 的收口口径）。
#
# 为什么负载是交付物而不是「多跑几次」：加固前的三行把「取消时图仍在派发」建立在**读一个
# 瞬时值**上，负载（M1：14 workers / 8 核）会把观测窗口拉到失效——实测原始夹具
# 定向三行 9/22 轮红 + 同二进制 A/B 2/8 轮红 + 整包 -count=5 -race 2/3 轮红，
# 而串行门禁下全绿。只跑串行门禁的「绿」证明不了加固，所以这一条落在仓库里可复跑
# （S17-STD-5 / S18-STD-6：判定脚本留在 /tmp 时别人无法复现结论）。
#
# 用法：
#   bash scripts/mutation/p1r17-load-rounds.sh                 # 14 workers × 10 轮，跑 TestP1R17_ 族
#   bash scripts/mutation/p1r17-load-rounds.sh 14 16 'TestP1R17_'
#   R17_LOAD_OUT=/tmp/r17-load bash scripts/mutation/p1r17-load-rounds.sh 14 10 'TestP1R17Hard'
#
# 协议：起 WORKERS 个 `yes > /dev/null` 压载循环 → 每轮跑一次契约绑定的场景命令
#   （go test ./pkg/hno/graph -run <RUN> -count=1 -race -v）→ 每轮把 `sysctl -n vm.loadavg`
#   与本轮原始输出文件路径写进同一份轮次账 → 收尾杀压载。
# 任一红即 exit 1（跨轮 0 容忍）；判红轮次的原始文件路径逐条列账，失败分型靠文件不靠记忆。
#
# 写在代码里的两条禁令：不许重试（红就是红）、不许在这里放宽阈值（阈值 = 0 红）。
set -uo pipefail

WORKERS="${1:-14}"
ROUNDS="${2:-10}"
RUN_PATTERN="${3:-TestP1R17_}"
PKG="./pkg/hno/graph"
OUT_DIR="${R17_LOAD_OUT:-/tmp/r17-load}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
STAMP="$(date +%Y%m%d-%H%M%S)"

mkdir -p "$OUT_DIR"
SUMMARY="$OUT_DIR/rounds.$STAMP.txt"

cd "$REPO_ROOT" || exit 2

# 编译成本挪到加压之前，但每轮跑的仍是契约逐字写下的那条命令（不换预编译二进制，
# 否则「轮次里到底执行了什么」要和契约对不上）。
go test "$PKG" -run "$RUN_PATTERN" -count=1 -race -v >/dev/null 2>&1
WARM=$?

PIDS=""
cleanup() {
  for p in $PIDS; do
    kill "$p" 2>/dev/null
  done
  wait 2>/dev/null
  PIDS=""
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 "$WORKERS"); do
  yes > /dev/null &
  PIDS="$PIDS $!"
done

echo "load protocol: workers=$WORKERS rounds=$ROUNDS run='$RUN_PATTERN' pkg=$PKG warmup_exit=$WARM" | tee "$SUMMARY"
echo "idle loadavg $(sysctl -n vm.loadavg)" | tee -a "$SUMMARY"

reds=0
for r in $(seq 1 "$ROUNDS"); do
  f="$OUT_DIR/round.$STAMP.$r.txt"
  before="$(sysctl -n vm.loadavg)"
  go test "$PKG" -run "$RUN_PATTERN" -count=1 -race -v >"$f" 2>&1
  code=$?
  after="$(sysctl -n vm.loadavg)"
  failnames="$(grep -E '^[[:space:]]*--- FAIL:' "$f" | sed 's/^[[:space:]]*//' | tr '\n' ';')"
  passrows="$(grep -cE '^[[:space:]]*--- PASS:' "$f")"
  if [ "$code" -ne 0 ]; then reds=$((reds + 1)); fi
  echo "round=$r exit=$code pass_rows=$passrows loadavg_before=$before loadavg_after=$after fail=[$failnames] raw=$f" | tee -a "$SUMMARY"
done

cleanup

if [ "$reds" -ne 0 ]; then
  echo "VERDICT red_rounds=$reds/$ROUNDS —— 跨轮 0 容忍：任一红即本片未收口（禁止重试、禁止放宽判据）" | tee -a "$SUMMARY"
  exit 1
fi
echo "VERDICT red_rounds=0/$ROUNDS —— 负载协议全绿，账本 $SUMMARY" | tee -a "$SUMMARY"
exit 0
