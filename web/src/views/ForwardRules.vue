<template>
  <div class="space-y-4">
    <div class="flex flex-wrap justify-between items-center gap-2">
      <h2 class="text-lg font-semibold m-0">转发规则</h2>
      <div class="flex flex-wrap gap-2 items-center">
        <a-select
          v-model:value="instanceId"
          allow-clear
          placeholder="选择实例"
          style="width: 220px"
          @change="load"
        >
          <a-select-option v-for="i in instances" :key="i.id" :value="i.id">{{
            i.name
          }}</a-select-option>
        </a-select>
        <a-button :disabled="!instanceId" :loading="listLoading" @click="load"
          >刷新</a-button
        >
        <a-button type="primary" :disabled="!instanceId" @click="openCreate"
          >添加规则</a-button
        >
      </div>
    </div>
    <a-table
      v-if="instanceId"
      :columns="ruleColumns"
      :data-source="rules"
      :pagination="false"
      bordered
      row-key="id"
      size="middle"
      :row-class-name="ruleRowClassName"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'pattern'">
          <div
            v-if="(record as RuleRow).pattern_source === 'url'"
            class="max-w-md"
          >
            <div class="truncate text-xs text-[rgba(255,255,255,0.45)]">
              {{
                formatLabels[
                  String((record as RuleRow).pattern_format || '')
                ] || (record as RuleRow).pattern_format
              }}
            </div>
            <div
              class="truncate text-sm"
              :title="String((record as RuleRow).pattern_url)"
            >
              {{ (record as RuleRow).pattern_url }}
            </div>
          </div>
          <div v-else class="max-w-md">
            <div class="truncate text-xs text-[rgba(255,255,255,0.45)]">
              {{
                formatLabels[
                  String((record as RuleRow).pattern_format || '')
                ] || (record as RuleRow).pattern_format
              }}
            </div>
            <div
              class="truncate text-sm font-mono"
              :title="previewInlinePattern(record as RuleRow, 500)"
            >
              {{ previewInlinePattern(record as RuleRow, 120) }}
            </div>
          </div>
        </template>
        <template v-else-if="column.key === 'target_group_id'">
          <span
            class="block max-w-full truncate text-sm text-[rgba(255,255,255,0.88)]"
            :title="forwardGroupLabel((record as RuleRow).target_group_id)"
            >{{ forwardGroupLabel((record as RuleRow).target_group_id) }}</span
          >
        </template>
        <template v-else-if="column.key === 'mode'">
          <span>{{ formatModeLabel((record as RuleRow).mode) }}</span>
        </template>
        <template v-else-if="column.key === 'pattern_rule_count'">
          <a-button
            v-if="patternEntriesClickable(record as RuleRow)"
            type="link"
            size="small"
            class="p-0 h-auto"
            @click="openPatternEntries(record as RuleRow)"
          >
            {{ (record as RuleRow).pattern_rule_count }}
          </a-button>
          <span v-else>{{ patternCountDisplay(record as RuleRow) }}</span>
        </template>
        <template v-else-if="column.key === 'pattern_fetched_at'">
          <span v-if="(record as RuleRow).pattern_fetched_at">{{
            formatDateTime((record as RuleRow).pattern_fetched_at)
          }}</span>
          <span v-else>—</span>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button
              type="text"
              size="small"
              class="inline-flex items-center"
              @click="toggleRuleDisabled(record as RuleRow)"
            >
              <PauseOutlined v-if="!(record as RuleRow).disabled" />
              <CaretRightOutlined v-else />
              <span>{{ (record as RuleRow).disabled ? '启用' : '禁用' }}</span>
            </a-button>
            <a-button
              v-if="(record as RuleRow).pattern_source === 'url'"
              type="text"
              size="small"
              class="inline-flex items-center"
              :loading="refreshingId === (record as RuleRow).id"
              @click="refreshPattern(record as RuleRow)"
            >
              <ReloadOutlined />
              <span>重新拉取</span>
            </a-button>
            <a-button
              type="text"
              size="small"
              class="inline-flex items-center"
              @click="edit(record as Record<string, unknown>)"
            >
              <EditOutlined />
              <span>编辑</span>
            </a-button>
            <a-button
              type="text"
              size="small"
              danger
              class="inline-flex items-center"
              @click="remove(record as Record<string, unknown>)"
            >
              <DeleteOutlined />
              <span>删除</span>
            </a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="dlg"
      :title="editId ? '编辑规则' : '新建规则'"
      width="min(640px, 92vw)"
      destroy-on-close
      wrap-class-name="dns-modal"
    >
      <a-form
        ref="ruleFormRef"
        :model="form"
        :rules="dynamicRules"
        layout="vertical"
      >
        <a-form-item label="匹配方式" name="pattern_source">
          <a-radio-group v-model:value="form.pattern_source">
            <a-radio value="inline">多行文本</a-radio>
            <a-radio value="url">URL</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="文件格式" name="pattern_format">
          <a-select
            v-model:value="form.pattern_format"
            class="w-full"
            placeholder="选择格式"
            :options="patternFormatOptions"
          />
        </a-form-item>
        <template v-if="form.pattern_source === 'inline'">
          <a-form-item label="规则内容" name="name_pattern">
            <a-textarea
              v-model:value="form.name_pattern"
              :rows="14"
              class="font-mono text-sm"
              placeholder="按所选格式填写：每行一条正则、AutoProxy 文本，或 Base64（可含 gzip，如 gfwlist）"
              allow-clear
            />
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item label="规则文件 URL" name="pattern_url">
            <a-input
              v-model:value="form.pattern_url"
              placeholder="https://…"
              allow-clear
            />
          </a-form-item>
        </template>
        <a-form-item label="转发组" name="target_group_id">
          <a-select
            v-model:value="form.target_group_id"
            show-search
            allow-clear
            class="w-full"
            placeholder="搜索组名或 ID"
            :options="upstreamGroupSelectOptions"
            :filter-option="filterUpstreamGroupOption"
          />
        </a-form-item>
        <a-form-item label="模式" name="mode">
          <a-select
            v-model:value="form.mode"
            class="w-full"
            placeholder="选择转发模式"
          >
            <a-select-option value="sequential">顺序</a-select-option>
            <a-select-option value="parallel">并行</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="优先级" name="priority">
          <a-input-number
            v-model:value="form.priority"
            :min="0"
            class="w-full"
            placeholder="数字越小越先匹配"
          />
        </a-form-item>
        <a-form-item label="启用规则" name="enabled">
          <a-switch v-model:checked="form.enabled" />
          <span class="ml-2 text-xs text-[rgba(255,255,255,0.45)]"
            >关闭后仅保存配置，解析时不会匹配该规则</span
          >
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button type="button" @click="dlg = false">取消</a-button>
        <a-button type="primary" :loading="saveLoading" @click="submitRule"
          >保存</a-button
        >
      </template>
    </a-modal>

    <a-modal
      v-model:open="entriesDlg"
      title="规则条目"
      width="min(560px, 92vw)"
      :footer="null"
      destroy-on-close
      wrap-class-name="dns-modal"
    >
      <a-spin :spinning="entriesLoading">
        <div
          class="font-mono text-sm max-h-[60vh] overflow-y-auto whitespace-pre-wrap break-all rounded border border-[rgba(255,255,255,0.12)] p-2"
        >
          <template v-if="!entriesLoading && entriesLines.length === 0"
            >无条目</template
          >
          <template v-else>
            <div v-for="(line, i) in entriesLines" :key="i" class="flex gap-2">
              <span class="select-none text-[rgba(255,255,255,0.45)]">{{
                i + 1
              }}</span>
              <span class="flex-1">{{ line }}</span>
            </div>
          </template>
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import {
  CaretRightOutlined,
  DeleteOutlined,
  EditOutlined,
  PauseOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue'
import { ref, reactive, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import type { TableColumnType } from 'ant-design-vue'
import * as api from '../api'
import { apiErrorMessage } from '../api/errors'
import { formatDateTime } from '../datetime'

const formatLabels: Record<string, string> = {
  regex_list: '正则列表',
  autoproxy: 'AutoProxy',
  autoproxy_base64: 'Base64 / gfwlist',
}

/** 多行文本与 URL 共用的三种格式 */
const patternFormatOptions = [
  { label: '正则列表（每行一条正则）', value: 'regex_list' },
  { label: 'AutoProxy 文本', value: 'autoproxy' },
  { label: 'Base64（可含 gzip，如 gfwlist）', value: 'autoproxy_base64' },
]

interface RuleRow {
  id: number
  name_pattern: string
  pattern_source?: string
  pattern_url?: string | null
  pattern_format?: string
  target_group_id: number
  mode: string
  priority: number
  /** 为 true 时不参与匹配 */
  disabled?: boolean
  /** 非缓存查询中该规则被匹配并用于转发的次数 */
  hit_count?: number
  /** URL 源：从文件中解析出的规则条数 */
  pattern_rule_count?: number
  /** URL 源：上次成功拉取时间 */
  pattern_fetched_at?: string | null
}

const dynamicRules = computed<Record<string, Rule[]>>(() => {
  const base: Record<string, Rule[]> = {
    target_group_id: [
      {
        required: true,
        validator: async (_rule, value: unknown) => {
          const n = value === null || value === undefined ? NaN : Number(value)
          if (!Number.isFinite(n) || n < 1) {
            return Promise.reject(new Error('请选择转发组'))
          }
          return Promise.resolve()
        },
        trigger: 'change',
      },
    ],
    mode: [{ required: true, message: '请选择模式', trigger: 'change' }],
    priority: [
      {
        required: true,
        validator: async (_rule, value: unknown) => {
          if (value === null || value === undefined) {
            return Promise.reject(new Error('请输入优先级'))
          }
          const n = Number(value)
          if (!Number.isFinite(n) || n < 0) {
            return Promise.reject(new Error('优先级须为 ≥0 的整数'))
          }
          return Promise.resolve()
        },
        trigger: 'change',
      },
    ],
  }
  base.pattern_format = [
    { required: true, message: '请选择格式', trigger: 'change' },
  ]
  if (form.pattern_source === 'inline') {
    base.name_pattern = [
      {
        required: true,
        validator: async (_rule, value: unknown) => {
          const s = typeof value === 'string' ? value.trim() : ''
          if (!s) {
            return Promise.reject(new Error('请输入匹配内容'))
          }
          return Promise.resolve()
        },
        trigger: 'blur',
      },
    ]
  } else {
    base.pattern_url = [
      {
        required: true,
        validator: async (_rule, value: unknown) => {
          const s = typeof value === 'string' ? value.trim() : ''
          if (!s) {
            return Promise.reject(new Error('请输入 URL'))
          }
          return Promise.resolve()
        },
        trigger: 'blur',
      },
    ]
  }
  return base
})

const ruleColumns: TableColumnType<RuleRow>[] = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '匹配', key: 'pattern' },
  {
    title: '转发组',
    dataIndex: 'target_group_id',
    key: 'target_group_id',
    width: 110,
    ellipsis: true,
  },
  { title: '模式', dataIndex: 'mode', key: 'mode', width: 50 },
  { title: '优先级', dataIndex: 'priority', key: 'priority', width: 60 },
  {
    title: '规则条数',
    key: 'pattern_rule_count',
    width: 96,
  },
  {
    title: '上次更新',
    key: 'pattern_fetched_at',
    width: 160,
  },
  {
    title: '命中次数',
    dataIndex: 'hit_count',
    key: 'hit_count',
    width: 80,
  },
  { title: '操作', key: 'action', width: 334 },
]

const instances = ref<{ id: number; name: string }[]>([])
const instanceId = ref<number>()
const listLoading = ref(false)
const rules = ref<Record<string, unknown>[]>([])
const dlg = ref(false)
const saveLoading = ref(false)
/** 正在执行「重新拉取」的规则 id */
const refreshingId = ref<number | null>(null)
const entriesDlg = ref(false)
const entriesLoading = ref(false)
const entriesLines = ref<string[]>([])
const ruleFormRef = ref<FormInstance>()
const editId = ref<number | null>(null)
const form = reactive({
  pattern_source: 'inline' as 'inline' | 'url',
  name_pattern: '',
  pattern_url: '',
  pattern_format: 'regex_list' as string,
  target_group_id: undefined as number | undefined,
  mode: 'parallel' as 'sequential' | 'parallel',
  priority: 100,
  /** 与后端 disabled 相反 */
  enabled: true,
})

const upstreamGroups = ref<{ id: number; name: string }[]>([])

const upstreamGroupSelectOptions = computed(() =>
  upstreamGroups.value.map((g) => ({
    value: g.id,
    label: `${g.name} (#${g.id})`,
  })),
)

function filterUpstreamGroupOption(input: string, option: { label?: string }) {
  return (option.label || '')
    .toLowerCase()
    .includes(String(input).toLowerCase())
}

const modeLabels: Record<string, string> = {
  sequential: '顺序',
  parallel: '并行',
}

function formatModeLabel(mode: string | undefined) {
  const m = String(mode ?? '')
  return modeLabels[m] ?? m
}

/** 与实例列表「默认转发组」展示一致：名称 (#id) */
function forwardGroupLabel(gid: number | undefined) {
  if (gid == null || !Number.isFinite(gid)) return '—'
  const g = upstreamGroups.value.find((x) => x.id === gid)
  return g ? `${g.name} (#${g.id})` : `#${gid}`
}

async function load() {
  if (!instanceId.value) {
    rules.value = []
    return
  }
  listLoading.value = true
  try {
    rules.value = (await api.listForwardRules(instanceId.value)) as Record<
      string,
      unknown
    >[]
  } finally {
    listLoading.value = false
  }
}

function previewInlinePattern(row: RuleRow, count: number) {
  const t = String(row.name_pattern || '')
  if (t.length <= count) return t
  return `${t.slice(0, count)}…`
}

function patternEntriesClickable(row: RuleRow) {
  const n = row.pattern_rule_count ?? 0
  return n > 0
}

function patternCountDisplay(row: RuleRow) {
  const n = row.pattern_rule_count
  if (n != null && n > 0) return n
  return '—'
}

async function openPatternEntries(row: RuleRow) {
  entriesDlg.value = true
  entriesLoading.value = true
  entriesLines.value = []
  try {
    entriesLines.value = await api.getForwardRulePatternEntries(row.id)
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '加载条目失败'))
    entriesDlg.value = false
  } finally {
    entriesLoading.value = false
  }
}

function openCreate() {
  editId.value = null
  Object.assign(form, {
    pattern_source: 'inline',
    name_pattern: '',
    pattern_url: '',
    pattern_format: 'regex_list',
    target_group_id: undefined,
    mode: 'parallel',
    priority: 100,
    enabled: true,
  })
  dlg.value = true
}

function edit(row: Record<string, unknown>) {
  editId.value = row.id as number
  const src = (row.pattern_source as string) || 'inline'
  Object.assign(form, {
    pattern_source: src === 'url' ? 'url' : 'inline',
    name_pattern: (row.name_pattern as string) || '',
    pattern_url: (row.pattern_url as string) || '',
    pattern_format: (row.pattern_format as string) || 'regex_list',
    target_group_id: row.target_group_id as number,
    mode: row.mode,
    priority: row.priority,
    enabled: row.disabled !== true,
  })
  if (
    form.pattern_source === 'url' &&
    !['regex_list', 'autoproxy', 'autoproxy_base64'].includes(
      form.pattern_format,
    )
  ) {
    form.pattern_format = 'regex_list'
  }
  if (
    form.pattern_source === 'inline' &&
    !patternFormatOptions.some((o) => o.value === form.pattern_format)
  ) {
    form.pattern_format = 'regex_list'
  }
  dlg.value = true
}

async function submitRule() {
  try {
    await ruleFormRef.value?.validate()
  } catch {
    return
  }
  await save()
}

async function save() {
  if (!instanceId.value) return
  saveLoading.value = true
  try {
    const body: Record<string, unknown> = {
      pattern_source: form.pattern_source,
      pattern_format: form.pattern_format,
      name_pattern: form.pattern_source === 'inline' ? form.name_pattern : '',
      pattern_url:
        form.pattern_source === 'url' ? form.pattern_url.trim() || null : null,
      target_group_id: form.target_group_id as number,
      mode: form.mode,
      priority: form.priority,
      disabled: !form.enabled,
    }
    if (editId.value) {
      await api.patchForwardRule(editId.value, body)
    } else {
      await api.createForwardRule(instanceId.value, body)
    }
    dlg.value = false
    await load()
    message.success('已保存')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '保存失败'))
  } finally {
    saveLoading.value = false
  }
}

async function refreshPattern(row: RuleRow) {
  if (row.pattern_source !== 'url') return
  refreshingId.value = row.id
  try {
    await api.postRefreshForwardRulePattern(row.id)
    await load()
    message.success('已重新拉取')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '拉取失败'))
  } finally {
    refreshingId.value = null
  }
}

async function toggleRuleDisabled(row: RuleRow) {
  const next = !row.disabled
  try {
    await api.patchForwardRuleDisabled(row.id, next)
    await load()
    message.success(next ? '已禁用' : '已启用')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '操作失败'))
  }
}

function ruleRowClassName(record: RuleRow) {
  return record.disabled ? 'dns-forward-rule-row--disabled' : ''
}

function remove(row: Record<string, unknown>) {
  Modal.confirm({
    title: '确认',
    content: '确定删除？',
    async onOk() {
      await api.deleteForwardRule(row.id as number)
      await load()
    },
  })
}

onMounted(async () => {
  upstreamGroups.value = (await api.listUpstreamGroups()) as {
    id: number
    name: string
  }[]
  instances.value = (await api.listInstances()) as {
    id: number
    name: string
  }[]
  if (instances.value.length) instanceId.value = instances.value[0].id
  await load()
})
</script>

<style scoped>
:deep(.dns-forward-rule-row--disabled > td) {
  opacity: 0.72;
}
</style>
