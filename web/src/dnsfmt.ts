/** FQDN 展示：去掉 DNS 文本形式末尾的根点（如 `example.com.` → `example.com`） */
export function displayFqdn(name: string): string {
  return name.endsWith(".") ? name.slice(0, -1) : name;
}
