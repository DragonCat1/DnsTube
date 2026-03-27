import axios from 'axios'

/** 与后端 `internal/api/errcode.go` 对齐，用于无 message 字段时的兜底（优先使用接口返回的 message） */
export const apiErrorMessages: Record<string, string> = {
  INVALID_JSON: '请求体 JSON 无效',
  BAD_ID: '路径中的 ID 无效',
  NOT_FOUND: '资源不存在',
  DUPLICATE_KEY: '与已有数据冲突（唯一约束）',
  DATABASE_ERROR: '数据库错误',
  BAD_REQUEST: '请求参数无效',
  MISSING_BEARER_TOKEN: '缺少 Bearer Token',
  INVALID_TOKEN: 'Token 无效或已过期',
  NO_USER_CONTEXT: '未找到当前用户',
  INVALID_CREDENTIALS: '用户名或密码错误',
  TOKEN_SIGN_FAILED: '签发 Token 失败',
  PASSWORD_HASH_FAILED: '密码处理失败',
  INSTANCE_NAME_OR_PORT_REQUIRED: '实例名称与监听端口为必填',
  FORWARD_RULE_FIELDS_REQUIRED: '转发规则需填写域名正则与模式',
  PATTERN_URL_FETCH_FAILED: '拉取规则文件失败',
  FORWARD_RULE_MODE_INVALID: '转发模式须为 sequential 或 parallel',
  UPSTREAM_GROUP_NAME_REQUIRED: '转发组名称为必填',
  UPSTREAM_SERVER_ADDRESS_REQUIRED: '上游服务器地址为必填',
  UPSTREAM_SERVER_IP_INVALID: '上游服务器地址须为合法 IPv4 或 IPv6',
  DNS_RECORD_IPV4_INVALID: 'A 记录正文须为合法 IPv4 地址',
  DNS_RECORD_IPV6_INVALID: 'AAAA 记录正文须为合法 IPv6 地址',
  DNS_ENGINE_RELOAD_FAILED: 'DNS 引擎重载配置失败',
  DNS_UDP_BIND_FAILED: 'UDP 监听地址绑定失败（端口可能被占用）',
}

/** 后端统一 JSON：对象接口 */
export type ApiObjEnvelope = {
  success: boolean
  code: number
  message: string
  data?: unknown
  /** 旧版兼容 */
  error?: { code?: string; message?: string }
}

/** 后端统一 JSON：列表接口 */
export type ApiListEnvelope = {
  success: boolean
  code: number
  message: string
  data: unknown[]
  page: number
  total: number
}

export function apiErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    const d = err.response?.data as ApiObjEnvelope | undefined
    if (d?.message) return d.message
    if (d?.error?.message) return d.error.message
    if (d?.error?.code && apiErrorMessages[d.error.code]) {
      return apiErrorMessages[d.error.code]
    }
  }
  if (err instanceof Error && err.message && err.message !== 'Request failed with status code') {
    return err.message
  }
  return fallback
}
