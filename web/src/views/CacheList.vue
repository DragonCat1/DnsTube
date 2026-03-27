<template>
  <div class="space-y-4">
    <h2 class="text-lg font-semibold m-0">缓存列表</h2>
    <p class="text-sm text-[rgba(255,255,255,0.45)]">
      当前进程内存中的 DNS 应答缓存（LRU），重启或过期后消失。
    </p>
    <div class="flex flex-wrap gap-2 items-end">
      <a-select
        v-model:value="filterInstance"
        allow-clear
        placeholder="实例"
        style="width: 180px"
        @change="onFilterChange"
      >
        <a-select-option v-for="i in instances" :key="i.id" :value="i.id">{{
          i.name
        }}</a-select-option>
      </a-select>
      <a-input
        v-model:value="filterQname"
        allow-clear
        placeholder="QNAME 包含"
        style="width: 180px"
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
      <a-button type="primary" @click="onFilterChange">应用</a-button>
      <a-button @click="load">刷新</a-button>
    </div>
    <a-table
      :columns="columns"
      :data-source="items"
      :pagination="false"
      bordered
      row-key="rowKey"
      size="small"
      :sort-directions="tableSortDirections"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'cached_at'">
          {{ formatDateTime((record as CacheRow).cached_at) }}
        </template>
        <template v-else-if="column.key === 'expires_at'">
          {{ formatDateTime((record as CacheRow).expires_at) }}
        </template>
        <template v-else-if="column.key === 'qname'">
          {{ displayFqdn((record as CacheRow).qname) }}
        </template>
        <template v-else-if="column.key === 'instance_id'">
          {{ instanceName((record as CacheRow).instance_id) }}
        </template>
        <template v-else-if="column.key === 'result_summary'">
          {{ (record as CacheRow).result_summary ?? '—' }}
        </template>
        <template v-else-if="column.key === 'ttl_seconds'">
          {{ formatTTL((record as CacheRow).ttl_seconds) }}
        </template>
      </template>
    </a-table>
    <div class="text-sm text-[rgba(255,255,255,0.45)]">共 {{ total }} 条</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { TableColumnType } from 'ant-design-vue'
import type { SorterResult } from 'ant-design-vue/es/table/interface'
import * as api from '../api'
import { formatDateTime } from '../datetime'
import { dnsQtypeSelectOptions } from '../dnsQtypes'
import { displayFqdn } from '../dnsfmt'

interface CacheRow {
  instance_id: number
  qname: string
  qtype: string
  ttl_seconds?: number
  cached_at?: string
  expires_at: string
  result_summary?: string
  rowKey: string
}

const columnKeyToSortBy: Record<string, string> = {
  instance_id: 'instance_id',
  qname: 'qname',
  qtype: 'qtype',
  ttl_seconds: 'ttl_seconds',
  cached_at: 'cached_at',
  result_summary: 'result_summary',
  expires_at: 'expires_at',
}

function serverOnlySorter(_a: CacheRow, _b: CacheRow) {
  return 0
}

function formatTTL(sec: number | undefined) {
  if (sec == null || Number.isNaN(sec)) return '—'
  return `${sec}s`
}

const sortBy = ref('expires_at')
const sortOrder = ref<'asc' | 'desc'>('asc')

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

const columns = computed<TableColumnType<CacheRow>[]>(() => {
  const o = sortOrderForColumn
  return [
    {
      title: '实例',
      key: 'instance_id',
      width: 160,
      sorter: serverOnlySorter,
      sortOrder: o('instance_id'),
    },
    {
      title: 'QNAME',
      dataIndex: 'qname',
      key: 'qname',
      ellipsis: true,
      sorter: serverOnlySorter,
      sortOrder: o('qname'),
    },
    {
      title: '类型',
      dataIndex: 'qtype',
      key: 'qtype',
      width: 90,
      sorter: serverOnlySorter,
      sortOrder: o('qtype'),
    },
    {
      title: 'TTL',
      dataIndex: 'ttl_seconds',
      key: 'ttl_seconds',
      width: 88,
      sorter: serverOnlySorter,
      sortOrder: o('ttl_seconds'),
    },
    {
      title: '结果',
      dataIndex: 'result_summary',
      key: 'result_summary',
      ellipsis: {
        showTitle: true,
      },
      width: 220,
      sorter: serverOnlySorter,
      sortOrder: o('result_summary'),
    },
    {
      title: '缓存时间',
      key: 'cached_at',
      width: 190,
      sorter: serverOnlySorter,
      sortOrder: o('cached_at'),
    },
    {
      title: '过期时间',
      key: 'expires_at',
      width: 190,
      sorter: serverOnlySorter,
      sortOrder: o('expires_at'),
    },
  ]
})

const instances = ref<{ id: number; name: string }[]>([])
const items = ref<CacheRow[]>([])
const total = ref(0)
const filterInstance = ref<number>()
const filterQname = ref('')
const filterQtype = ref<string | undefined>(undefined)

function instanceName(id: number) {
  const i = instances.value.find((x) => x.id === id)
  return i ? `${i.name} (#${id})` : String(id)
}

function buildParams(): Record<string, string | number | undefined> {
  const p: Record<string, string | number | undefined> = {
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
    limit: 0,
    page: 1,
  }
  if (filterInstance.value) p.instance_id = filterInstance.value
  const qn = filterQname.value.trim()
  if (qn) p.qname = qn
  const qt = filterQtype.value?.trim()
  if (qt) p.qtype = qt
  return p
}

async function load() {
  const d = await api.listCacheEntries(buildParams())
  const raw = d.items as {
    instance_id: number
    qname: string
    qtype: string
    ttl_seconds?: number
    cached_at?: string
    expires_at: string
    result_summary?: string
  }[]
  items.value = raw.map((r, i) => ({
    ...r,
    rowKey: `${r.instance_id}-${i}-${r.qname}-${r.qtype}`,
  }))
  total.value = d.total
}

function onFilterChange() {
  load()
}

function resolveSorterColumnKey(s: SorterResult<CacheRow>): string {
  const raw = s.columnKey ?? s.field
  if (raw == null) return ''
  return Array.isArray(raw) ? String(raw[0]) : String(raw)
}

function onTableChange(
  _pagination: unknown,
  _filters: unknown,
  sorter: SorterResult<CacheRow> | SorterResult<CacheRow>[],
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
    sortBy.value = 'expires_at'
    sortOrder.value = 'asc'
  }
  load()
}

onMounted(async () => {
  instances.value = (await api.listInstances()) as {
    id: number
    name: string
  }[]
  await load()
})
</script>
