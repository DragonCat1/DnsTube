<template>
  <div ref="root" class="dns-echart-box w-full min-w-0" :style="{ height: `${height}px` }" />
</template>

<script setup lang="ts">
/**
 * 使用完整 echarts 包 + DOM 初始化，避免 vue-echarts 自定义元素 / 按需注册在部分环境下的兼容问题。
 */
import * as echarts from 'echarts'
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'

const props = withDefaults(
  defineProps<{
    option: echarts.EChartsOption
    height?: number
  }>(),
  { height: 320 }
)

const root = ref<HTMLDivElement | null>(null)
const chart = shallowRef<echarts.ECharts | null>(null)
let resizeObs: ResizeObserver | null = null

function paint() {
  const c = chart.value
  if (!c) return
  c.setOption(props.option, { notMerge: true })
}

onMounted(() => {
  const el = root.value
  if (!el) return
  chart.value = echarts.init(el)
  paint()
  resizeObs = new ResizeObserver(() => chart.value?.resize())
  resizeObs.observe(el)
})

watch(
  () => props.option,
  () => paint(),
  { deep: true }
)

onUnmounted(() => {
  resizeObs?.disconnect()
  resizeObs = null
  chart.value?.dispose()
  chart.value = null
})
</script>
