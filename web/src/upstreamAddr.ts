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

/** 上游协议字面量；与后端 internal/api/upstream_addr.go 常量保持同步。 */
export type UpstreamProtocol = "udp" | "dot" | "doh";

/** 协议对应的常用默认端口。 */
export function defaultPortForProtocol(p: UpstreamProtocol): number {
  switch (p) {
    case "dot":
      return 853;
    case "doh":
      return 443;
    default:
      return 53;
  }
}

/** 协议对应的中文展示名（表单与列表共用）。 */
export function protocolLabel(p: UpstreamProtocol | string): string {
  switch (p) {
    case "dot":
      return "DoT";
    case "doh":
      return "DoH";
    case "udp":
    default:
      return "UDP";
  }
}

/**
 * 归一 DoH 端点路径：
 * - 空 / "/" 视为缺省 "/dns-query"。
 * - 必须以 "/" 开头，且不含查询字符串或片段。
 * - 不合法返回 null。
 */
export function normalizeDoHPath(input: string): string | null {
  const t = input.trim();
  if (!t || t === "/") return "/dns-query";
  if (!t.startsWith("/")) return null;
  if (t.includes("?") || t.includes("#")) return null;
  return t;
}

/**
 * 归一 TLS SNI（可选）：空字符串视为不填，返回 ""；不合法返回 null。
 * 校验规则：1-253 字符；每段 1-63 字符；字母/数字/连字符；连字符不在首尾。
 * 也允许 IP 字面量直接做 SNI（与后端一致）。
 */
export function normalizeTLSServerName(input: string): string | null {
  const t = input.trim();
  if (!t) return "";
  if (t.length > 253) return null;
  if (isUpstreamIpV4OrV6(t)) return normalizeUpstreamAddr(t);
  const labels = t.replace(/\.$/, "").split(".");
  for (const label of labels) {
    if (!label || label.length > 63) return null;
    if (label.startsWith("-") || label.endsWith("-")) return null;
    if (!/^[A-Za-z0-9-]+$/.test(label)) return null;
  }
  return t;
}
