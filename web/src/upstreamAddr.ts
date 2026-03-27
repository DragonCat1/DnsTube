/**
 * 与后端 `internal/api/upstream_addr.go` 一致：仅接受 IPv4 或 IPv6（可带方括号）。
 * 返回 null 表示不合法；合法时返回 trim 后的地址（IPv6 无方括号，与入库一致由服务端再规范化）。
 */
export function normalizeUpstreamAddr(input: string): string | null {
  const t = input.trim();
  if (!t) return null;
  if (t.startsWith("[")) {
    const end = t.lastIndexOf("]");
    if (end <= 1) return null;
    const inner = t.slice(1, end);
    if (!inner.includes(":")) return null;
    try {
      new URL(`http://[${inner}]`);
      return inner;
    } catch {
      return null;
    }
  }
  if (!t.includes(":")) {
    return isIPv4DottedDecimal(t) ? t : null;
  }
  try {
    new URL(`http://[${t}]`);
    return t;
  } catch {
    return null;
  }
}

function isIPv4DottedDecimal(s: string): boolean {
  const parts = s.split(".");
  if (parts.length !== 4) return false;
  for (const p of parts) {
    if (!/^\d{1,3}$/.test(p)) return false;
    const n = Number(p);
    if (n < 0 || n > 255 || String(n) !== p) return false;
  }
  return true;
}

export function isUpstreamIpV4OrV6(input: string): boolean {
  return normalizeUpstreamAddr(input) !== null;
}
