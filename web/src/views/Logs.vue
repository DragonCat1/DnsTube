<template>
  <div class="space-y-4">
    <h2 class="text-lg font-semibold m-0">查询日志</h2>
    <div class="flex flex-wrap gap-2 items-end">
      <a-range-picker
        v-model:value="timeRange"
        show-time
        format="YYYY-MM-DD HH:mm:ss"
        :placeholder="['开始时间', '结束时间']"
        style="width: 360px"
        @change="onFilterChange"
      />
      <a-input
        v-model:value="filterClient"
        allow-clear
        placeholder="客户端 IP"
        style="width: 140px"
        @press-enter="onFilterChange"
      />
      <a-input
        v-model:value="filterQname"
        allow-clear
        placeholder="QNAME 包含"
        style="width: 160px"
        @press-enter="onFilterChange"
      />
      <a-select
        v-model:value="filterQtype"
        allow-clear
        show-search
        placeholder="查询类型"
        :options="dnsQtypeSelectOptions"
        option-filter-prop="label"
        style="width: 160px"
        @change="onFilterChange"
      />
      <a-input
        v-model:value="filterRcode"
        allow-clear
        placeholder="RCODE"
        style="width: 100px"
        @press-enter="onFilterChange"
      />
      <a-select
        v-model:value="filterInstance"
        allow-clear
        placeholder="实例"
        style="width: 160px"
        @change="onFilterChange"
      >
        <a-select-option v-for="i in instances" :key="i.id" :value="i.id">{{
          i.name
        }}</a-select-option>
      </a-select>
      <a-select
        v-model:value="filterForwardGroup"
        allow-clear
        placeholder="转发组"
        style="width: 180px"
        @change="onFilterChange"
      >
        <a-select-option
          v-for="g in upstreamGroups"
          :key="g.id"
          :value="g.id"
          >{{ g.name }}</a-select-option
        >
      </a-select>
      <a-button type="primary" @click="onFilterChange">应用</a-button>
      <a-button danger :disabled="!selection.length" @click="bulkDelete"
        >删除所选</a-button
      >
      <a-button :loading="listLoading" @click="load">刷新</a-button>
    </div>
    <a-table
      :columns="logColumns"
      :data-source="items"
      :loading="listLoading"
      :pagination="false"
      bordered
      row-key="id"
      size="small"
      :scroll="{ x: 1480 }"
      :row-selection="rowSelection"
      :sort-directions="tableSortDirections"
      :row-class-name="logRowClassName"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'created_at'">
          {{ formatDateTime((record as LogRow).created_at) }}
        </template>
        <template v-else-if="column.key === 'cache_hit'">
          {{ (record as LogRow).cache_hit ? '是' : '否' }}
        </template>
        <template v-else-if="column.key === 'forwarded'">
          {{ (record as LogRow).forwarded ? '是' : '否' }}
        </template>
        <template v-else-if="column.key === 'qname'">
          {{ displayFqdn((record as LogRow).qname) }}
        </template>
        <template v-else-if="column.key === 'result_summary'">
          {{ (record as LogRow).result_summary ?? '—' }}
        </template>
        <template v-else-if="column.key === 'instance_name'">
          {{ (record as LogRow).instance_name ?? '—' }}
        </template>
        <template v-else-if="column.key === 'forward_group_name'">
          {{ displayForwardGroupName(record as LogRow) }}
        </template>
      </template>
    </a-table>
    <a-pagination
      v-model:current="page"
      v-model:page-size="pageSize"
      :total="total"
      :show-total="(t: number) => `共 ${t} 条`"
      show-less-items
      @change="load"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { TableColumnType } from 'ant-design-vue'
import type { TableProps } from 'ant-design-vue'
import type { SorterResult } from 'ant-design-vue/es/table/interface'
import type { Dayjs } from 'dayjs'
import * as api from '../api'
import { formatDateTime } from '../datetime'
import { dnsQtypeSelectOptions } from '../dnsQtypes'
import { displayFqdn } from '../dnsfmt'

interface LogRow {
  id: number
  created_at: string
  instance_id: number
  instance_name?: string
  client_ip: string
  qname: string
  qtype: string
  response_code: string
  cache_hit: boolean
  forwarded: boolean
  upstream_addr?: string
  total_ms?: number
  result_summary?: string | null
  forward_group_name?: string
  forward_upstream_group_id?: number | null
}

/** 表头 column.key → 后端 sortBy */
const columnKeyToSortBy: Record<string, string> = {
  id: 'id',
  created_at: 'created_at',
  instance_name: 'instance_id',
  forward_group_name: 'forward_group',
  client_ip: 'client_ip',
  qtype: 'qtype',
  response_code: 'response_code',
  result_summary: 'result_summary',
  cache_hit: 'cache_hit',
  forwarded: 'forwarded',
  upstream_addr: 'upstream_addr',
  total_ms: 'total_ms',
}

/** 服务端排序：不在浏览器里重排，仅用于表头可点 */
function serverOnlySorter(_a: LogRow, _b: LogRow) {
  return 0
}

const sortBy = ref('id')
const sortOrder = ref<'asc' | 'desc'>('desc')

/** 随当前升降序切换周期，避免 antd 在「当前顺序」处于数组末尾时下一次点击先变成取消排序 */
const tableSortDirections = computed(() =>
  sortOrder.value === 'asc'
    ? (['ascend', 'descend'] as const)
    : (['descend', 'ascend'] as const),
)

function sortOrderForColumn(
  columnKey: string,
): 'ascend' | 'descend' | undefined {
  const api = columnKeyToSortBy[columnKey]
  if (!api || sortBy.value !== api) return undefined
  return sortOrder.value === 'asc' ? 'ascend' : 'descend'
}

const logColumns = computed<TableColumnType<LogRow>[]>(() => {
  const o = sortOrderForColumn
  return [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 90,
      sorter: serverOnlySorter,
      sortOrder: o('id'),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 176,
      sorter: serverOnlySorter,
      sortOrder: o('created_at'),
    },
    {
      title: '实例',
      dataIndex: 'instance_name',
      key: 'instance_name',
      width: 100,
      ellipsis: true,
      sorter: serverOnlySorter,
      sortOrder: o('instance_name'),
    },
    {
      title: '转发组',
      dataIndex: 'forward_group_name',
      key: 'forward_group_name',
      width: 120,
      ellipsis: true,
      sorter: serverOnlySorter,
      sortOrder: o('forward_group_name'),
    },
    {
      title: '客户端',
      dataIndex: 'client_ip',
      key: 'client_ip',
      width: 130,
      sorter: serverOnlySorter,
      sortOrder: o('client_ip'),
    },
    {
      title: 'QNAME',
      dataIndex: 'qname',
      key: 'qname',
      width: 200,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'qtype',
      key: 'qtype',
      width: 70,
      sorter: serverOnlySorter,
      sortOrder: o('qtype'),
    },
    {
      title: 'RCODE',
      dataIndex: 'response_code',
      key: 'response_code',
      width: 90,
      sorter: serverOnlySorter,
      sortOrder: o('response_code'),
    },
    {
      title: '结果',
      dataIndex: 'result_summary',
      key: 'result_summary',
      ellipsis: {
        showTitle: true,
      },
      sorter: serverOnlySorter,
      sortOrder: o('result_summary'),
    },
    {
      title: '缓存',
      key: 'cache_hit',
      width: 64,
      sorter: serverOnlySorter,
      sortOrder: o('cache_hit'),
    },
    {
      title: '转发',
      key: 'forwarded',
      width: 64,
      sorter: serverOnlySorter,
      sortOrder: o('forwarded'),
    },
    {
      title: '上游',
      dataIndex: 'upstream_addr',
      key: 'upstream_addr',
      width: 130,
      sorter: serverOnlySorter,
      sortOrder: o('upstream_addr'),
    },
    {
      title: '总耗时ms',
      dataIndex: 'total_ms',
      key: 'total_ms',
      width: 96,
      sorter: serverOnlySorter,
      sortOrder: o('total_ms'),
    },
  ]
})

const instances = ref<{ id: number; name: string }[]>([])
const upstreamGroups = ref<{ id: number; name: string }[]>([])
const filterInstance = ref<number>()
const filterForwardGroup = ref<number>()
const filterClient = ref('')
const filterQname = ref('')
const filterQtype = ref<string | undefined>(undefined)
const filterRcode = ref('')
const timeRange = ref<[Dayjs, Dayjs] | null>(null)

const items = ref<LogRow[]>([])
const listLoading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const selection = ref<LogRow[]>([])
const selectedRowKeys = ref<number[]>([])

const rowSelection = computed<TableProps['rowSelection']>(() => ({
  selectedRowKeys: selectedRowKeys.value,
  onChange: (keys: readonly (string | number)[], rows: LogRow[]) => {
    selectedRowKeys.value = keys.map(Number)
    selection.value = rows
  },
}))

function buildParams(): Record<string, string | number | undefined> {
  const params: Record<string, string | number | undefined> = {
    limit: pageSize.value,
    page: page.value,
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
  }
  if (filterInstance.value) params.instance_id = filterInstance.value
  if (filterForwardGroup.value)
    params.forward_upstream_group_id = filterForwardGroup.value
  const c = filterClient.value.trim()
  if (c) params.client_ip = c
  const qn = filterQname.value.trim()
  if (qn) params.qname = qn
  const qt = filterQtype.value?.trim()
  if (qt) params.qtype = qt
  const rc = filterRcode.value.trim()
  if (rc) params.response_code = rc
  if (timeRange.value?.[0] && timeRange.value[1]) {
    params.from = timeRange.value[0].toISOString()
    params.to = timeRange.value[1].toISOString()
  }
  return params
}

let activeLoadSeq = 0
let filterDebounceTimer: ReturnType<typeof setTimeout> | null = null

async function load() {
  const seq = ++activeLoadSeq
  listLoading.value = true
  try {
    const d = await api.listQueryLogs(buildParams())
    // 仅采纳最后一次请求结果，避免快速筛选导致数据回跳。
    if (seq !== activeLoadSeq) return
    items.value = d.items as LogRow[]
    total.value = d.total
  } finally {
    if (seq === activeLoadSeq) {
      listLoading.value = false
    }
  }
}

function onFilterChange() {
  page.value = 1
  if (filterDebounceTimer) {
    clearTimeout(filterDebounceTimer)
  }
  filterDebounceTimer = setTimeout(() => {
    load()
  }, 300)
}

function resolveSorterColumnKey(s: SorterResult<LogRow>): string {
  const raw = s.columnKey ?? s.field
  if (raw == null) return ''
  return Array.isArray(raw) ? String(raw[0]) : String(raw)
}

function onTableChange(
  _pagination: unknown,
  _filters: unknown,
  sorter: SorterResult<LogRow> | SorterResult<LogRow>[],
) {
  const s = Array.isArray(sorter) ? sorter[0] : sorter
  const key = resolveSorterColumnKey(s)
  if (!key) return
  const apiField = columnKeyToSortBy[key]
  if (!apiField) return
  if (s.order === 'ascend') {
    sortOrder.value = 'asc'
    sortBy.value = apiField
  } else if (s.order === 'descend') {
    sortOrder.value = 'desc'
    sortBy.value = apiField
  } else {
    sortBy.value = 'created_at'
    sortOrder.value = 'desc'
  }
  page.value = 1
  load()
}

function bulkDelete() {
  Modal.confirm({
    title: '确认',
    content: `确定删除已选中的 ${selection.value.length} 条查询日志吗？该操作不可恢复。`,
    async onOk() {
      await api.deleteQueryLogs(selection.value.map((x) => x.id))
      selection.value = []
      selectedRowKeys.value = []
      await load()
      message.success('已删除')
    },
  })
}

function displayForwardGroupName(record: LogRow): string {
  if (record.forward_group_name) {
    return `${record.forward_group_name} (#${record.forward_upstream_group_id})`
  }
  if (record.cache_hit) {
    return '/*缓存*/'
  }
  return '/*静态记录*/'
}

/** RCODE：除 NOERROR / NXDOMAIN 外视为结果错误（SERVFAIL 等） */
function isQueryLogErrorRow(row: LogRow): boolean {
  const rc = (row.response_code || '').toUpperCase()
  if (!rc) return false
  return rc !== 'NOERROR' && rc !== 'NXDOMAIN'
}

/** 无记录：NXDOMAIN，或 NOERROR 但结果摘要为空（与后端 AnswerSummaryForLog「—」一致） */
function isQueryLogNoRecordRow(row: LogRow): boolean {
  if (row.cache_hit) return false
  const rc = (row.response_code || '').toUpperCase()
  if (rc === 'NXDOMAIN') return true
  const rs = row.result_summary?.trim()
  if (rc === 'NOERROR' && (rs === '—' || rs === '' || rs == null)) return true
  return false
}

function logRowClassName(record: LogRow) {
  if (record.cache_hit) return 'dns-query-log-row--cache'
  if (isQueryLogErrorRow(record)) return 'dns-query-log-row--error'
  if (isQueryLogNoRecordRow(record)) return 'dns-query-log-row--norecord'
  return ''
}

onMounted(async () => {
  instances.value = (await api.listInstances()) as {
    id: number
    name: string
  }[]
  upstreamGroups.value = (await api.listUpstreamGroups()) as {
    id: number
    name: string
  }[]
  await load()
})
</script>

<style scoped>
:deep(.dns-query-log-row--cache > td) {
  background-color: rgba(34, 197, 94, 0.12) !important;
}
:deep(.dns-query-log-row--error > td) {
  background-color: rgba(220, 38, 38, 0.18) !important;
}
:deep(.dns-query-log-row--norecord > td) {
  background-color: rgba(234, 179, 8, 0.16) !important;
}
</style>
