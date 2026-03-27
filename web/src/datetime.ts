/** 全站日期时间展示格式（24 小时制） */
export const DATE_TIME_FORMAT = 'YYYY-MM-DD HH:mm:ss'

/**
 * 将 ISO 字符串、时间戳或 Date 格式化为本地时间的 `YYYY-MM-DD HH:mm:ss`。
 */
export function formatDateTime(input: string | number | Date | null | undefined): string {
  if (input == null || input === '') return '—'
  const d = input instanceof Date ? input : new Date(input)
  if (Number.isNaN(d.getTime())) return String(input)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  const s = String(d.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${min}:${s}`
}
