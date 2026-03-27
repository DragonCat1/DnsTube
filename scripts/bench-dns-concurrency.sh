#!/usr/bin/env bash
# 并发 DNS 压测：依次对 DnsTube、dnsmasq（或任意 UDP DNS）运行 dnsbench，便于对比输出。
#
# 用法示例：
#   ./scripts/bench-dns-concurrency.sh --dnstube 127.0.0.1:5353 --dnsmasq 127.0.0.1:53 -- \
#     -c 500 -n 100000 -q example.com -warmup 1000
#
# 仅测一个目标：
#   ./scripts/bench-dns-concurrency.sh --dnsmasq 127.0.0.1:53 -- -c 200 -n 20000
#
# 直接透传（等价于手动运行 dnsbench）：
#   ./scripts/bench-dns-concurrency.sh -- -addr 127.0.0.1:53 -c 100 -n 5000
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DNSBENCH="${DNSBENCH:-$REPO_ROOT/dnsbench}"

if [[ ! -x "$DNSBENCH" ]]; then
  (cd "$REPO_ROOT" && go build -trimpath -o dnsbench ./cmd/dnsbench)
  DNSBENCH="$REPO_ROOT/dnsbench"
fi

DNSTUBE_ADDR=""
DNSMASQ_ADDR=""
PASSTHRU=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dnstube)
      DNSTUBE_ADDR="${2:-}"
      if [[ -z "$DNSTUBE_ADDR" ]]; then echo "missing value for --dnstube" >&2; exit 2; fi
      shift 2
      ;;
    --dnsmasq)
      DNSMASQ_ADDR="${2:-}"
      if [[ -z "$DNSMASQ_ADDR" ]]; then echo "missing value for --dnsmasq" >&2; exit 2; fi
      shift 2
      ;;
    --)
      shift
      PASSTHRU=("$@")
      break
      ;;
    -h|--help)
      sed -n '2,14p' "$0" | cat
      exit 0
      ;;
    *)
      PASSTHRU+=("$1")
      shift
      ;;
  esac
done

ec=0

run_one() {
  local title="$1"
  local addr="$2"
  shift 2
  echo ""
  echo "################################################################################"
  echo "# ${title}  (${addr})"
  echo "################################################################################"
  if "$DNSBENCH" -addr "$addr" "$@"; then
    :
  else
    ec=1
  fi
}

if [[ ${#PASSTHRU[@]} -gt 0 && -z "$DNSTUBE_ADDR" && -z "$DNSMASQ_ADDR" ]]; then
  exec "$DNSBENCH" "${PASSTHRU[@]}"
fi

if [[ -n "$DNSTUBE_ADDR" ]]; then
  run_one "DnsTube" "$DNSTUBE_ADDR" "${PASSTHRU[@]}"
fi
if [[ -n "$DNSMASQ_ADDR" ]]; then
  run_one "dnsmasq" "$DNSMASQ_ADDR" "${PASSTHRU[@]}"
fi

if [[ -z "$DNSTUBE_ADDR" && -z "$DNSMASQ_ADDR" ]]; then
  echo "请指定至少一个目标，或使用 -- 直接传入 dnsbench 参数。" >&2
  echo "示例: $0 --dnstube 127.0.0.1:5353 --dnsmasq 127.0.0.1:53 -- -c 200 -n 20000 -q example.com" >&2
  exit 2
fi

exit "$ec"
