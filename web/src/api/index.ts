import client from './client'

export async function login(username: string, password: string) {
  const { data } = await client.post('/api/v1/auth/login', {
    username,
    password,
  })
  return data as { token: string; username: string }
}

export async function me() {
  const { data } = await client.get('/api/v1/auth/me')
  return data as { id: number; username: string }
}

export async function patchProfile(body: {
  username?: string
  password?: string
}) {
  await client.patch('/api/v1/auth/profile', body)
}

export type SystemSettingsDTO = {
  telegram_enabled: boolean
  telegram_chat_id: string
  telegram_bot_token_set: boolean
  /** 推送级别：debug/info/warn/error 为系统日志；audit 为审计事件 */
  telegram_notify_levels: string[]
}

export async function getSystemSettings() {
  const { data } = await client.get('/api/v1/settings/system')
  return data as SystemSettingsDTO
}

export async function patchSystemSettings(body: Record<string, unknown>) {
  await client.patch('/api/v1/settings/system', body)
}

export async function postTelegramTest() {
  await client.post('/api/v1/settings/system/telegram-test')
}

type ListPayload<T = unknown> = { items: T[]; page: number; total: number }

export async function listInstances() {
  const { data } = await client.get('/api/v1/instances')
  const d = data as ListPayload
  return d.items
}

export async function createInstance(body: Record<string, unknown>) {
  const { data } = await client.post('/api/v1/instances', body)
  return data
}

export async function patchInstance(id: number, body: Record<string, unknown>) {
  await client.patch(`/api/v1/instances/${id}`, body)
}

export async function deleteInstance(id: number) {
  await client.delete(`/api/v1/instances/${id}`)
}

export async function listRecords(instanceId: number) {
  const { data } = await client.get(`/api/v1/instances/${instanceId}/records`)
  const d = data as ListPayload
  return d.items
}

export async function createRecord(
  instanceId: number,
  body: Record<string, unknown>,
) {
  await client.post(`/api/v1/instances/${instanceId}/records`, body)
}

export async function patchRecord(id: number, body: Record<string, unknown>) {
  await client.patch(`/api/v1/records/${id}`, body)
}

export async function deleteRecord(id: number) {
  await client.delete(`/api/v1/records/${id}`)
}

export async function listUpstreamGroups() {
  const { data } = await client.get('/api/v1/upstream-groups')
  const d = data as ListPayload
  return d.items
}

export async function listGroupServers(groupId: number) {
  const { data } = await client.get(
    `/api/v1/upstream-groups/${groupId}/servers`,
  )
  const d = data as ListPayload
  return d.items
}

export async function createGroup(name: string) {
  await client.post('/api/v1/upstream-groups', { name })
}

export async function patchUpstreamGroup(id: number, body: { name: string }) {
  await client.patch(`/api/v1/upstream-groups/${id}`, body)
}

export async function deleteGroup(id: number) {
  await client.delete(`/api/v1/upstream-groups/${id}`)
}

export async function addServer(
  groupId: number,
  body: { address: string; port: number; sort_order: number },
) {
  await client.post(`/api/v1/upstream-groups/${groupId}/servers`, body)
}

export async function deleteServer(id: number) {
  await client.delete(`/api/v1/upstream-servers/${id}`)
}

export async function listForwardRules(instanceId: number) {
  const { data } = await client.get(
    `/api/v1/instances/${instanceId}/forward-rules`,
  )
  const d = data as ListPayload
  return d.items
}

export async function createForwardRule(
  instanceId: number,
  body: Record<string, unknown>,
) {
  const { data } = await client.post(
    `/api/v1/instances/${instanceId}/forward-rules`,
    body,
  )
  return data as { id: number }
}

/** 手动重新拉取 URL 规则文件并更新缓存 */
export async function postRefreshForwardRulePattern(ruleId: number) {
  const { data } = await client.post(
    `/api/v1/forward-rules/${ruleId}/refresh-pattern`,
  )
  return data as {
    pattern_rule_count: number
    pattern_fetched_at: string
  }
}

/** 获取规则条目列表（正则列表 / AutoProxy 解析结果等） */
export async function getForwardRulePatternEntries(ruleId: number) {
  const { data } = await client.get(
    `/api/v1/forward-rules/${ruleId}/pattern-entries`,
  )
  return (data as { entries: string[] }).entries
}

export async function patchForwardRule(
  id: number,
  body: Record<string, unknown>,
) {
  await client.patch(`/api/v1/forward-rules/${id}`, body)
}

export async function patchForwardRuleDisabled(id: number, disabled: boolean) {
  await client.patch(`/api/v1/forward-rules/${id}/disabled`, { disabled })
}

export async function deleteForwardRule(id: number) {
  await client.delete(`/api/v1/forward-rules/${id}`)
}

export async function listQueryLogs(
  params: Record<string, string | number | undefined>,
) {
  const { data } = await client.get('/api/v1/query-logs', { params })
  const d = data as ListPayload
  return { items: d.items, total: d.total, page: d.page }
}

export async function listCacheEntries(
  params: Record<string, string | number | undefined>,
) {
  const { data } = await client.get('/api/v1/cache-entries', { params })
  const d = data as ListPayload
  return { items: d.items, total: d.total, page: d.page }
}

export async function deleteQueryLogs(ids: number[]) {
  await client.post('/api/v1/query-logs/delete', { ids })
}

export async function listSystemLogs(
  params: Record<string, string | number | undefined>,
) {
  const { data } = await client.get('/api/v1/system-logs', { params })
  const d = data as ListPayload
  return { items: d.items, total: d.total, page: d.page }
}

export async function deleteSystemLogs(ids: number[]) {
  await client.post('/api/v1/system-logs/delete', { ids })
}

export async function dashboardStats(
  range: string,
  instance_id?: number,
  client_ip?: string,
) {
  const { data } = await client.get('/api/v1/dashboard/stats', {
    params: { range, instance_id, client_ip },
  })
  return data
}

/** GET /api/v1/dashboard/summary 聚合概览（与 stats 共用 range / instance_id） */
export type DashboardSummaryDTO = {
  range: string
  from: string
  to: string
  /** 当前 range 时间窗内查询量（与图表、分布一致） */
  total_queries: number
  /** 查询日志累计条数（不限时间；受 instance_id 筛选） */
  total_queries_all: number
  instance_total: number
  instance_paused: number
  instances_with_listener_error: number
  listener_errors: Record<string, string>
  top_qnames: { qname: string; count: number }[]
  qtype_distribution: { qtype: string; count: number }[]
  rcode_distribution: { response_code: string; count: number }[]
  cache_forward: {
    total: number
    cache_hits: number
    forwarded: number
    other: number
  }
}

export async function dashboardSummary(
  range: string,
  instance_id?: number,
  top_n?: number,
  client_ip?: string,
) {
  const { data } = await client.get('/api/v1/dashboard/summary', {
    params: { range, instance_id, top_n, client_ip },
  })
  return data as DashboardSummaryDTO
}
