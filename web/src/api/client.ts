import axios from 'axios'
import router from '../router'
import { useAuthStore } from '../stores/auth'
import type { ApiObjEnvelope } from './errors'

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '',
  timeout: 30000,
})

client.interceptors.request.use((config) => {
  const t = localStorage.getItem('dnstube_token')
  if (t) {
    config.headers.Authorization = `Bearer ${t}`
  }
  return config
})

function isFailedEnvelope(body: unknown): body is ApiObjEnvelope {
  if (!body || typeof body !== 'object') return false
  const o = body as ApiObjEnvelope
  if (o.success === false) return true
  if (typeof o.code === 'number' && o.code !== 0) return true
  return false
}

client.interceptors.response.use(
  (response) => {
    const body = response.data as ApiObjEnvelope | unknown
    if (body && typeof body === 'object' && body !== null && 'success' in body && isFailedEnvelope(body)) {
      const env = body as ApiObjEnvelope
      const msg = env.message || env.error?.message || env.error?.code || '请求失败'
      const e = new Error(msg)
      ;(e as Error & { code?: number }).code = typeof env.code === 'number' ? env.code : undefined
      return Promise.reject(e)
    }
    if (
      body &&
      typeof body === 'object' &&
      body !== null &&
      'success' in body &&
      (body as ApiObjEnvelope).success === true &&
      typeof (body as ApiObjEnvelope).code === 'number' &&
      (body as ApiObjEnvelope).code === 0 &&
      'data' in body
    ) {
      const env = body as ApiObjEnvelope & { page?: number; total?: number; data?: unknown }
      if (
        Array.isArray(env.data) &&
        typeof env.page === 'number' &&
        typeof env.total === 'number'
      ) {
        return {
          ...response,
          data: {
            items: env.data,
            page: env.page,
            total: env.total,
          },
        }
      }
      return { ...response, data: env.data }
    }
    return response
  },
  (err) => {
    // 401 须优先处理：服务端常带 JSON 失败 envelope，若先走 isFailedEnvelope 会漏掉清 token / 跳转
    if (err.response?.status === 401) {
      useAuthStore().logout()
      if (router.currentRoute.value.name !== 'login') {
        router.replace({ name: 'login' })
      }
      return Promise.reject(err)
    }
    const data = err.response?.data as ApiObjEnvelope | undefined
    if (data && isFailedEnvelope(data)) {
      const msg = data.message || data.error?.message || data.error?.code || err.message
      const e = new Error(msg)
      ;(e as Error & { code?: number }).code = typeof data.code === 'number' ? data.code : undefined
      return Promise.reject(e)
    }
    return Promise.reject(err)
  },
)

export default client
