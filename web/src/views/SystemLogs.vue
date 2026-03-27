<template>
  <div class="space-y-4">
    <h2 class="text-lg font-semibold m-0">系统日志</h2>
    <div class="flex flex-wrap gap-2 items-end">
      <a-range-picker
        v-model:value="timeRange"
        show-time
        format="YYYY-MM-DD HH:mm:ss"
        :placeholder="['开始时间', '结束时间']"
        style="width: 360px"
        @change="onFilterChange"
      />
      <a-select
        v-model:value="filterKind"
        allow-clear
        placeholder="类型"
        style="width: 120px"
        @change="onFilterChange"
      >
        <a-select-option value="system">系统</a-select-option>
        <a-select-option value="audit">审计</a-select-option>
      </a-select>
      <a-select
        v-model:value="filterLevel"
        allow-clear
        placeholder="级别"
        style="width: 120px"
        @change="onFilterChange"
      >
        <a-select-option value="debug">debug</a-select-option>
        <a-select-option value="info">info</a-select-option>
        <a-select-option value="warn">warn</a-select-option>
        <a-select-option value="error">error</a-select-option>
      </a-select>
      <a-input
        v-model:value="filterEvent"
        allow-clear
        placeholder="事件（模糊）"
        style="width: 140px"
        @press-enter="onFilterChange"
      />
      <a-input
        v-model:value="filterMessage"
        allow-clear
        placeholder="消息包含"
        style="width: 160px"
        @press-enter="onFilterChange"
      />
      <a-input
        v-model:value="filterUsername"
        allow-clear
        placeholder="用户名"
        style="width: 120px"
        @press-enter="onFilterChange"
      />
      <a-input
        v-model:value="filterClient"
        allow-clear
        placeholder="客户端 IP"
        style="width: 130px"
        @press-enter="onFilterChange"
      />
      <a-button type="primary" @click="onFilterChange">应用</a-button>
      <a-button danger :disabled="!selection.length" @click="bulkDelete"
        >删除所选</a-button
      >
      <a-button @click="load">刷新</a-button>
    </div>
    <a-table
      :columns="logColumns"
      :data-source="items"
      :pagination="false"
      bordered
      row-key="id"
      size="middle"
      :scroll="{ x: 1320 }"
      :row-selection="rowSelection"
      :sort-directions="tableSortDirections"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'created_at'">
          {{ formatDateTime((record as SLRow).created_at) }}
        </template>
        <template v-else-if="column.key === 'event'">
          {{ (record as SLRow).event ?? '—' }}
        </template>
        <template v-else-if="column.key === 'username'">
          {{ (record as SLRow).username ?? '—' }}
        </template>
        <template v-else-if="column.key === 'client_ip'">
          {{ (record as SLRow).client_ip ?? '—' }}
        </template>
        <template v-else-if="column.key === 'meta'">
          <span class="text-[rgba(255,255,255,0.65)]">{{
            formatMeta((record as SLRow).meta)
          }}</span>
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

interface SLRow {
  id: number
  created_at: string
  kind: string
  level: string
  event?: string | null
  message: string
  username?: string | null
  client_ip?: string | null
  meta?: unknown
}

const columnKeyToSortBy: Record<string, string> = {
  id: 'id',
  created_at: 'created_at',
  kind: 'kind',
  level: 'level',
  event: 'event',
  message: 'message',
  username: 'username',
  client_ip: 'client_ip',
}

function serverOnlySorter(_a: SLRow, _b: SLRow) {
  return 0
}

const sortBy = ref('id')
const sortOrder = ref<'asc' | 'desc'>('desc')

const tableSortDirections = computed(() =>
  sortOrder.value === 'asc'
    ? (['ascend', 'descend'] as const)
    : (['descend', 'ascend'] as const),
)

function sortOrderForColumn(
  columnKey: string,
): 'ascend' | 'descend' | undefined {
  const apiField = columnKeyToSortBy[columnKey]
  if (!apiField || sortBy.value !== apiField) return undefined
  return sortOrder.value === 'asc' ? 'ascend' : 'descend'
}

const logColumns = computed<TableColumnType<SLRow>[]>(() => {
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
      title: '类型',
      dataIndex: 'kind',
      key: 'kind',
      width: 90,
      sorter: serverOnlySorter,
      sortOrder: o('kind'),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      sorter: serverOnlySorter,
      sortOrder: o('level'),
    },
    {
      title: '事件',
      dataIndex: 'event',
      key: 'event',
      width: 160,
      ellipsis: true,
      sorter: serverOnlySorter,
      sortOrder: o('event'),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: {
        showTitle: true,
      },
      sorter: serverOnlySorter,
      sortOrder: o('message'),
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 120,
      ellipsis: true,
      sorter: serverOnlySorter,
      sortOrder: o('username'),
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
      title: '附加',
      key: 'meta',
      ellipsis: {
        showTitle: true,
      },
    },
  ]
})

const filterKind = ref<string>()
const filterLevel = ref<string>()
const filterEvent = ref('')
const filterMessage = ref('')
const filterUsername = ref('')
const filterClient = ref('')
const timeRange = ref<[Dayjs, Dayjs] | null>(null)

const items = ref<SLRow[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const selection = ref<SLRow[]>([])
const selectedRowKeys = ref<number[]>([])

const rowSelection = computed<TableProps['rowSelection']>(() => ({
  selectedRowKeys: selectedRowKeys.value,
  onChange: (keys: readonly (string | number)[], rows: SLRow[]) => {
    selectedRowKeys.value = keys.map(Number)
    selection.value = rows
  },
}))

function formatMeta(m: unknown) {
  if (m == null) return '—'
  if (typeof m === 'string') return m || '—'
  try {
    return JSON.stringify(m)
  } catch {
    return String(m)
  }
}

function buildParams(): Record<string, string | number | undefined> {
  const params: Record<string, string | number | undefined> = {
    limit: pageSize.value,
    page: page.value,
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
  }
  if (filterKind.value) params.kind = filterKind.value
  if (filterLevel.value) params.level = filterLevel.value
  const ev = filterEvent.value.trim()
  if (ev) params.event = ev
  const msg = filterMessage.value.trim()
  if (msg) params.message = msg
  const u = filterUsername.value.trim()
  if (u) params.username = u
  const c = filterClient.value.trim()
  if (c) params.client_ip = c
  if (timeRange.value?.[0] && timeRange.value[1]) {
    params.from = timeRange.value[0].toISOString()
    params.to = timeRange.value[1].toISOString()
  }
  return params
}

async function load() {
  const d = await api.listSystemLogs(buildParams())
  items.value = d.items as SLRow[]
  total.value = d.total
}

function onFilterChange() {
  page.value = 1
  load()
}

function resolveSorterColumnKey(s: SorterResult<SLRow>): string {
  const raw = s.columnKey ?? s.field
  if (raw == null) return ''
  return Array.isArray(raw) ? String(raw[0]) : String(raw)
}

function onTableChange(
  _pagination: unknown,
  _filters: unknown,
  sorter: SorterResult<SLRow> | SorterResult<SLRow>[],
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
    content: `删除 ${selection.value.length} 条？`,
    async onOk() {
      await api.deleteSystemLogs(selection.value.map((x) => x.id))
      selection.value = []
      selectedRowKeys.value = []
      await load()
      message.success('已删除')
    },
  })
}

onMounted(() => load())
</script>
