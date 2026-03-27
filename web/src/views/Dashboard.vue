<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-wrap gap-2 items-center">
      <a-select
        v-model:value="instanceId"
        allow-clear
        placeholder="全部实例"
        style="width: 180px"
        @change="loadAll"
      >
        <a-select-option v-for="i in instances" :key="i.id" :value="i.id">{{
          i.name
        }}</a-select-option>
      </a-select>
      <a-button type="primary" :loading="loading" @click="loadAll"
        >刷新</a-button
      >
    </div>

    <!-- 概览卡片：stretch 等高，避免副标题行数不同时高度不齐 -->
    <a-row :gutter="[16, 16]" align="stretch" class="dns-overview-cards">
      <a-col :xs="24" :sm="12" :lg="6" class="flex flex-col">
        <a-card size="small" class="border-[#303030]! bg-[#141414]! flex-1">
          <div class="text-xs text-[rgba(255,255,255,0.45)]">DNS 实例</div>
          <div
            class="mt-1 text-2xl font-semibold text-[rgba(255,255,255,0.88)] flex items-center gap-2"
          >
            <span aria-hidden="true" class="select-none">🖥️</span>
            <span>{{ overviewSummary?.instance_total ?? '—' }}</span>
          </div>
          <div class="mt-1 text-xs text-[rgba(255,255,255,0.45)]">
            已暂停 {{ overviewSummary?.instance_paused ?? '—' }}
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6" class="flex flex-col">
        <a-card size="small" class="border-[#303030]! bg-[#141414]! flex-1">
          <div class="text-xs text-[rgba(255,255,255,0.45)]">累计查询量</div>
          <div
            class="mt-1 text-2xl font-semibold text-[#177ddc] flex items-center gap-2"
          >
            <span aria-hidden="true" class="select-none">🔍</span>
            <span>{{ overviewSummary?.total_queries_all ?? '—' }}</span>
          </div>
          <div
            class="mt-1 text-xs text-[rgba(255,255,255,0.45)] truncate"
            :title="queryTotalScopeHint"
          >
            {{ queryTotalScopeHint }}
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6" class="flex flex-col">
        <a-card size="small" class="border-[#303030]! bg-[#141414]! flex-1">
          <div class="text-xs text-[rgba(255,255,255,0.45)]">UDP 监听</div>
          <div
            class="mt-1 text-2xl font-semibold flex items-center gap-2"
            :class="
              (overviewSummary?.instances_with_listener_error ?? 0) > 0
                ? 'text-[#ff4d4f]'
                : 'text-[#49aa19]'
            "
          >
            <span aria-hidden="true" class="select-none">🔌</span>
            <span>{{
              overviewSummary == null
                ? '—'
                : overviewSummary.instances_with_listener_error > 0
                  ? `${overviewSummary.instances_with_listener_error} 处异常`
                  : '正常'
            }}</span>
          </div>
          <div class="mt-1 text-xs text-[rgba(255,255,255,0.45)]">
            进程内绑定状态
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6" class="flex flex-col">
        <a-card size="small" class="border-[#303030]! bg-[#141414]! flex-1">
          <div class="text-xs text-[rgba(255,255,255,0.45)]">缓存命中占比</div>
          <div
            class="mt-1 text-2xl font-semibold text-[#faad14] flex items-center gap-2"
          >
            <span aria-hidden="true" class="select-none">🎯</span>
            <span>{{ cacheHitPercent }}</span>
          </div>
          <div class="mt-1 text-xs text-[rgba(255,255,255,0.45)]">
            转发 {{ forwardPercent }}
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 监听异常明细 -->
    <a-alert
      v-if="
        overviewSummary &&
        Object.keys(overviewSummary.listener_errors || {}).length > 0
      "
      type="error"
      show-icon
      class="border-[#434343]! bg-[#1f1315]!"
      message="实例 UDP 绑定失败"
    >
      <template #description>
        <ul class="m-0 list-disc pl-4 space-y-1 text-sm">
          <li v-for="(msg, id) in overviewSummary.listener_errors" :key="id">
            实例 #{{ id }}：{{ msg }}
          </li>
        </ul>
      </template>
    </a-alert>

    <!-- 空数据 -->
    <a-empty
      v-if="
        !summaryLoading &&
        overviewSummary &&
        chartsSummary &&
        chartsSummary.total_queries === 0
      "
      class="rounded border border-dashed border-[#434343] bg-[#141414]/50 py-12"
      :image="Empty.PRESENTED_IMAGE_SIMPLE"
    >
      <template #description>
        <div class="text-[rgba(255,255,255,0.65)]">
          <template v-if="(overviewSummary.instance_total ?? 0) === 0">
            尚未创建 DNS 实例。请先
            <router-link to="/instances" class="text-[#177ddc] hover:underline"
              >添加实例</router-link
            >
            并配置监听。
          </template>
          <template v-else>
            当前时间范围内暂无查询日志。可对实例发起解析请求后
            <router-link to="/logs" class="text-[#177ddc] hover:underline"
              >查看查询日志</router-link
            >
            ，或切换页面底部「查询趋势」中的统计维度。
          </template>
        </div>
      </template>
    </a-empty>

    <!-- 查询趋势：日/周/月/年收口到同一卡片（置底） -->
    <a-card class="border-[#303030]! bg-[#141414]!">
      <template #title>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-base font-semibold">查询趋势</span>
          <div class="flex flex-wrap items-center gap-2">
            <a-radio-group
              v-model:value="range"
              button-style="solid"
              size="small"
              class="dns-dashboard-range"
              @change="loadAll"
            >
              <a-radio-button
                v-for="r in rangeMeta"
                :key="r.key"
                :value="r.key"
              >
                {{ r.label }}
              </a-radio-button>
            </a-radio-group>
            <a-input
              v-model:value="clientIpInput"
              allow-clear
              placeholder="客户端 IP（可选）"
              style="width: 200px"
              size="small"
              @press-enter="applyClientIpFilter"
            />
            <a-button size="small" @click="applyClientIpFilter" class="text-xs"
              >应用</a-button
            >
          </div>
        </div>
      </template>
      <div class="text-sm text-[rgba(255,255,255,0.45)] mb-4">
        当前维度：<span class="text-[rgba(255,255,255,0.75)]"
          >{{ currentRangeLabel }}范围</span
        >
        <template v-if="clientIpTrimmed">
          ；客户端 IP：<span class="text-[rgba(255,255,255,0.75)]">{{
            clientIpTrimmed
          }}</span>
        </template>
        （与下方分布图所选时间范围一致）
      </div>
      <div class="mb-4">
        <div class="text-[rgba(255,255,255,0.65)] mb-2">查询请求数</div>
        <div class="text-3xl font-bold text-[#177ddc]">
          {{ trendStats?.total ?? '—' }}
        </div>
      </div>
      <a-row :gutter="16">
        <a-col :xs="24" :lg="16">
          <div class="text-sm text-[rgba(255,255,255,0.65)] mb-2">趋势</div>
          <EchartBox
            :option="lineOption(trendStats?.series ?? [])"
            :height="320"
          />
        </a-col>
        <a-col :xs="24" :lg="8">
          <div class="text-sm text-[rgba(255,255,255,0.65)] mb-2">
            客户端排行
          </div>
          <EchartBox
            :option="barOption(trendStats?.topClients ?? [])"
            :height="320"
          />
        </a-col>
      </a-row>
    </a-card>

    <!-- 分布图表（有数据时） -->
    <template v-if="chartsSummary && chartsSummary.total_queries > 0">
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :lg="12">
          <a-card title="TOP 域名" class="border-[#303030]! bg-[#141414]!">
            <EchartBox :option="topQnameOption" :height="280" />
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card
            title="查询类型 QTYPE"
            class="border-[#303030]! bg-[#141414]!"
          >
            <EchartBox :option="qtypePieOption" :height="280" />
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card title="应答码 RCODE" class="border-[#303030]! bg-[#141414]!">
            <EchartBox :option="rcodePieOption" :height="280" />
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card
            title="缓存 / 转发 / 其它"
            class="border-[#303030]! bg-[#141414]!"
          >
            <EchartBox :option="cacheForwardOption" :height="280" />
          </a-card>
        </a-col>
      </a-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Empty } from 'ant-design-vue'
import type { EChartsOption } from 'echarts'
import EchartBox from '../components/EchartBox.vue'
import * as api from '../api'
import type { DashboardSummaryDTO } from '../api'
import { formatDateTime } from '../datetime'
import { displayFqdn } from '../dnsfmt'

const axisMuted = '#737373'
const tooltipBg = '#1f1f1f'

const rangeMeta = [
  { key: 'day' as const, label: '日' },
  { key: 'week' as const, label: '周' },
  { key: 'month' as const, label: '月' },
  { key: 'year' as const, label: '年' },
]

type RangeKey = (typeof rangeMeta)[number]['key']

interface BlockStats {
  total: number
  series: { bucket: string; count: number }[]
  topClients: { client_ip: string; count: number }[]
}

const instanceId = ref<number | undefined>()
const instances = ref<{ id: number; name: string }[]>([])
const loading = ref(false)
/** 日/周/月/年 */
const range = ref<RangeKey>('day')
/** 与请求一致的客户端 IP 筛选（子串匹配，与查询日志一致） */
const clientIp = ref('')
const clientIpInput = ref('')
/** 顶部概览卡片：不含 client_ip，不受 IP 筛选影响 */
const overviewSummary = ref<DashboardSummaryDTO | null>(null)
/** 趋势与分布图：含 client_ip 筛选（无筛选时与 overview 同源） */
const chartsSummary = ref<DashboardSummaryDTO | null>(null)
const summaryLoading = ref(false)
const trendStats = ref<BlockStats | null>(null)

const currentRangeLabel = computed(() => {
  const m = rangeMeta.find((x) => x.key === range.value)
  return m?.label ?? ''
})

const clientIpTrimmed = computed(() => clientIp.value.trim())

/** 累计查询量与「日/周/月/年」无关；仅说明实例筛选范围（不受客户端 IP 筛选影响） */
const queryTotalScopeHint = computed(() =>
  instanceId.value != null ? '仅当前所选实例' : '全部实例',
)

function applyClientIpFilter() {
  clientIp.value = clientIpInput.value.trim()
  loadAll()
}

const cacheHitPercent = computed(() => {
  const cf = overviewSummary.value?.cache_forward
  if (!cf || cf.total <= 0) return '—'
  return `${((cf.cache_hits / cf.total) * 100).toFixed(1)}%`
})

const forwardPercent = computed(() => {
  const cf = overviewSummary.value?.cache_forward
  if (!cf || cf.total <= 0) return '—'
  return `${((cf.forwarded / cf.total) * 100).toFixed(1)}%`
})

function pieOption(
  title: string,
  items: { name: string; value: number }[],
  colors: string[],
): EChartsOption {
  return {
    backgroundColor: 'transparent',
    textStyle: { color: axisMuted },
    tooltip: {
      trigger: 'item',
      backgroundColor: tooltipBg,
      borderColor: '#424242',
      textStyle: { color: '#fff' },
    },
    legend: {
      type: 'scroll',
      bottom: 0,
      textStyle: { color: axisMuted },
    },
    series: [
      {
        name: title,
        type: 'pie',
        radius: ['42%', '68%'],
        avoidLabelOverlap: true,
        itemStyle: { borderColor: '#141414', borderWidth: 1 },
        label: { color: '#d4d4d4' },
        data: items.map((d, i) => ({
          ...d,
          itemStyle: { color: colors[i % colors.length] },
        })),
      },
    ],
  }
}

const pieColors = [
  '#177ddc',
  '#49aa19',
  '#faad14',
  '#d4380d',
  '#722ed1',
  '#13c2c2',
  '#eb2f96',
]

const qtypePieOption = computed<EChartsOption>(() => {
  const rows = chartsSummary.value?.qtype_distribution ?? []
  return pieOption(
    'QTYPE',
    rows.map((r) => ({ name: r.qtype || '—', value: Number(r.count) })),
    pieColors,
  )
})

const rcodePieOption = computed<EChartsOption>(() => {
  const rows = chartsSummary.value?.rcode_distribution ?? []
  return pieOption(
    'RCODE',
    rows.map((r) => ({ name: r.response_code || '—', value: Number(r.count) })),
    pieColors,
  )
})

const topQnameOption = computed<EChartsOption>(() => {
  const rows = [...(chartsSummary.value?.top_qnames ?? [])].reverse()
  const labels = rows.map((r) => displayFqdn(r.qname))
  const values = rows.map((r) => Number(r.count))
  return {
    backgroundColor: 'transparent',
    textStyle: { color: axisMuted },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: tooltipBg,
      borderColor: '#424242',
      textStyle: { color: '#fff' },
    },
    grid: { left: 120, right: 24, top: 16, bottom: 16 },
    xAxis: {
      type: 'value',
      axisLabel: { color: axisMuted },
      splitLine: { lineStyle: { color: '#303030' } },
    },
    yAxis: {
      type: 'category',
      data: labels,
      axisLabel: { color: axisMuted },
      axisLine: { lineStyle: { color: '#424242' } },
    },
    series: [
      {
        type: 'bar',
        data: values,
        itemStyle: { color: '#177ddc' },
      },
    ],
  }
})

const cacheForwardOption = computed<EChartsOption>(() => {
  const cf = chartsSummary.value?.cache_forward
  const parts = cf
    ? [
        { name: '缓存命中', value: cf.cache_hits },
        { name: '转发上游', value: cf.forwarded },
        { name: '其它', value: cf.other },
      ]
    : []
  return pieOption('占比', parts, ['#49aa19', '#177ddc', '#737373'])
})

function lineOption(
  series: { bucket: string; count: number }[],
): EChartsOption {
  return {
    backgroundColor: 'transparent',
    textStyle: { color: axisMuted },
    tooltip: {
      trigger: 'axis',
      backgroundColor: tooltipBg,
      borderColor: '#424242',
      textStyle: { color: '#fff' },
    },
    grid: { left: 48, right: 24, top: 24, bottom: 32 },
    xAxis: {
      type: 'category',
      data: series.map((s) => s.bucket),
      axisLabel: { color: axisMuted },
      axisLine: { lineStyle: { color: '#424242' } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: axisMuted },
      splitLine: { lineStyle: { color: '#303030' } },
    },
    series: [
      {
        type: 'line',
        data: series.map((s) => s.count),
        smooth: true,
        itemStyle: { color: '#177ddc' },
        lineStyle: { color: '#177ddc' },
      },
    ],
  }
}

function barOption(
  topClients: { client_ip: string; count: number }[],
): EChartsOption {
  return {
    backgroundColor: 'transparent',
    textStyle: { color: axisMuted },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: tooltipBg,
      borderColor: '#424242',
      textStyle: { color: '#fff' },
    },
    grid: { left: 100, right: 24, top: 24, bottom: 24 },
    xAxis: {
      type: 'value',
      axisLabel: { color: axisMuted },
      splitLine: { lineStyle: { color: '#303030' } },
    },
    yAxis: {
      type: 'category',
      data: topClients.map((t) => t.client_ip).reverse(),
      axisLabel: { color: axisMuted },
      axisLine: { lineStyle: { color: '#424242' } },
    },
    series: [
      {
        type: 'bar',
        data: topClients.map((t) => t.count).reverse(),
        itemStyle: { color: '#49aa19' },
      },
    ],
  }
}

async function loadAll() {
  loading.value = true
  summaryLoading.value = true
  try {
    const cip = clientIpTrimmed.value || undefined
    const [overview, trend, chartsFiltered] = await Promise.all([
      api.dashboardSummary(range.value, instanceId.value, 10),
      api.dashboardStats(range.value, instanceId.value, cip),
      cip
        ? api.dashboardSummary(range.value, instanceId.value, 10, cip)
        : Promise.resolve(null as DashboardSummaryDTO | null),
    ])
    overviewSummary.value = overview
    chartsSummary.value = chartsFiltered ?? overview
    const d = trend as {
      total_queries: number
      series: { bucket: string; count: number }[]
      top_clients: { client_ip: string; count: number }[]
    }
    trendStats.value = {
      total: d.total_queries,
      series: (d.series || []).map((s) => ({
        bucket: formatDateTime(s.bucket),
        count: Number(s.count),
      })),
      topClients: d.top_clients || [],
    }
  } finally {
    loading.value = false
    summaryLoading.value = false
  }
}

onMounted(async () => {
  instances.value = (await api.listInstances()) as {
    id: number
    name: string
  }[]
  await loadAll()
})
</script>

<style scoped>
.dns-dashboard-range :deep(.ant-radio-button-wrapper) {
  padding-inline: 12px;
}

/* 与 align="stretch" 配合，让 Card 根节点真正铺满列高 */
.dns-overview-cards :deep(.ant-card) {
  height: 100%;
}
</style>
