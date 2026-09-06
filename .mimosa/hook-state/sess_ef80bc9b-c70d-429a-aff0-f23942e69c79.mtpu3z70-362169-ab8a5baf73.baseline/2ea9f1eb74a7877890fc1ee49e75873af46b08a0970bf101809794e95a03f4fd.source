#!/usr/bin/env bash
# check-arch-sync.sh — 架构文档同步检查
#
# 维护机制（P2，2026-08-08 建立）：
#   ARCHITECTURE.md §2 是模块结构的唯一事实来源。每次新增/删除 internal/ 包，
#   都应同步更新该章节。本脚本对比实际包与文档提及，防止文档漂移：
#     - 实际存在但文档完全未提及的包 → 失败（新增包必须补文档）
#     - 文档提及但实际不存在的包 → 警告（可能是已删除的包未清理）
#
# 用法: scripts/check-arch-sync.sh [--strict]
#   --strict  把"文档提及但实际不存在"也视为失败（默认仅警告）
# 退出码: 0 = 同步正常; 1 = 存在未记录的新包（CI 应失败）

set -uo pipefail
cd "$(dirname "$0")/.."

DOC="docs/ARCHITECTURE.md"
[ -f "$DOC" ] || { echo "❌ $DOC 不存在"; exit 1; }

STRICT=0
[ "${1:-}" = "--strict" ] && STRICT=1

# 实际一级包（含 .go 文件的 internal/* 目录）
actual=$(find internal -maxdepth 1 -type d -name '[a-z]*' \
  | while read -r d; do
      if find "$d" -maxdepth 1 -name '*.go' | grep -q .; then
        basename "$d"
      fi
    done | sort -u)

missing=0
for pkg in $actual; do
  if ! grep -qE "(^|[^a-z])${pkg}(/|[^a-z]|$)" "$DOC"; then
    echo "❌ internal/${pkg} 已存在但 docs/ARCHITECTURE.md §2 未提及 —— 请更新模块结构章节"
    missing=$((missing+1))
  fi
done

stale=0
# 文档中出现的 internal/xxx 引用（宽松提取，用于 stale 警告）
doc_refs=$(grep -oE 'internal/[a-z][a-z0-9]*' "$DOC" | sed 's|internal/||' | sort -u)
for pkg in $doc_refs; do
  if [ ! -d "internal/$pkg" ]; then
    echo "⚠️  docs/ARCHITECTURE.md 提到 internal/${pkg} 但目录已不存在 —— 建议清理文档"
    stale=$((stale+1))
  fi
done

if [ "$missing" -gt 0 ]; then
  echo "❌ check-arch-sync: $missing 个新包未同步到架构文档（新增 internal 包必须更新 ARCHITECTURE.md §2）"
  exit 1
fi
if [ "$STRICT" = "1" ] && [ "$stale" -gt 0 ]; then
  echo "❌ check-arch-sync --strict: $stale 个文档引用已失效"
  exit 1
fi
echo "✅ check-arch-sync: $DOC 与实际 internal/ 包同步（$([ "$stale" -gt 0 ] && echo "$stale 个过期引用警告" || echo "无过期引用")）"
exit 0
