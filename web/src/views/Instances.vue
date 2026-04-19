<template>
  <div class="space-y-4">
    <div class="flex flex-wrap justify-between items-center gap-2">
      <h2 class="text-lg font-semibold m-0">DNS 实例</h2>
      <div class="flex flex-wrap items-center gap-2">
        <a-button :loading="listLoading" @click="load">刷新</a-button>
        <a-button type="primary" @click="openCreate">添加实例</a-button>
      </div>
    </div>
    <a-table
      :columns="instColumns"
      :data-source="rows"
      :pagination="false"
      bordered
      row-key="id"
      size="middle"
      :row-class-name="instanceRowClassName"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <span class="inline-flex items-center gap-1.5 min-w-0">
            <span class="truncate">{{ (record as Inst).name }}</span>
            <a-tooltip
              v-if="(record as Inst).runtime_error"
              :title="(record as Inst).runtime_error"
            >
              <ExclamationCircleOutlined
                class="shrink-0 cursor-help text-[14px] leading-none text-[#ff4d4f]!"
                aria-label="运行时错误"
              />
            </a-tooltip>
          </span>
        </template>
        <template v-else-if="column.key === 'default_upstream'">
          <div
            v-if="(record as Inst).default_upstream"
            class="text-sm max-w-md"
          >
            <div class="font-medium text-[rgba(255,255,255,0.88)]">
              {{ (record as Inst).default_upstream!.name }} (#{{
                (record as Inst).default_upstream!.id
              }})
            </div>
            <div
              class="text-xs mt-0.5 text-[rgba(255,255,255,0.45)] break-words"
            >
              {{ formatServerList((record as Inst).default_upstream!.servers) }}
            </div>
          </div>
          <span v-else class="text-[rgba(255,255,255,0.45)]">—</span>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button
              type="text"
              size="small"
              :class="dnsTableActionBtn"
              :title="(record as Inst).paused ? '运行' : '暂停'"
              :aria-label="(record as Inst).paused ? '运行' : '暂停'"
              @click="togglePause(record as Inst)"
            >
              <CaretRightOutlined v-if="(record as Inst).paused" />
              <PauseOutlined v-else />
              <span>{{ (record as Inst).paused ? '运行' : '暂停' }}</span>
            </a-button>
            <a-button
              type="text"
              size="small"
              :class="dnsTableActionBtn"
              @click="edit(record as Inst)"
            >
              <EditOutlined />
              <span>编辑</span>
            </a-button>
            <a-button
              type="text"
              size="small"
              :class="dnsTableActionBtn"
              @click="goRecords(record as Inst)"
            >
              <DatabaseOutlined />
              <span>记录（{{ (record as Inst).record_count ?? 0 }}）</span>
            </a-button>
            <a-button
              type="text"
              size="small"
              :class="dnsTableActionBtn"
              danger
              @click="remove(record as Inst)"
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
      :title="editId ? '编辑实例' : '新建实例'"
      :width="520"
      destroy-on-close
      wrap-class-name="dns-modal"
    >
      <a-form
        ref="instFormRef"
        :model="form"
        :rules="instFormRules"
        layout="vertical"
      >
        <a-form-item label="名称" name="name">
          <a-input
            v-model:value="form.name"
            placeholder="例如 prod-dns"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="监听地址" name="listen_addr">
          <a-input
            v-model:value="form.listen_addr"
            placeholder="0.0.0.0 或本机 IP"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="端口" name="listen_port">
          <a-input-number
            v-model:value="form.listen_port"
            :min="1"
            :max="65535"
            class="w-full"
            placeholder="1–65535"
          />
        </a-form-item>
        <a-form-item label="默认转发组" name="default_upstream_group_id">
          <a-auto-complete
            v-model:value="groupACDisplay"
            :options="groupACOptions"
            allow-clear
            class="w-full"
            placeholder="搜索组名或 ID 选择"
            :filter-option="filterGroupOption"
            @select="onGroupSelect"
            @clear="onGroupClear"
          />
          <span class="mt-1 block text-xs text-[rgba(255,255,255,0.45)]"
            >留空表示不设置默认转发组</span
          >
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button type="button" @click="dlg = false">取消</a-button>
        <a-button type="primary" :loading="saveLoading" @click="submitInst"
          >保存</a-button
        >
      </template>
    </a-modal>

    <a-modal
      v-model:open="recDrawer"
      :title="'记录 — ' + (recInst?.name ?? '')"
      :width="920"
      :footer="null"
      destroy-on-close
      wrap-class-name="dns-modal"
    >
      <div class="mb-4">
        <a-button type="primary" @click="openRec">添加记录</a-button>
      </div>
      <a-table
        :columns="recColumns"
        :data-source="records"
        :pagination="false"
        bordered
        size="small"
        row-key="id"
      >
        <template #bodyCell="{ column, record: row }">
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button
                type="text"
                size="small"
                :class="dnsTableActionBtn"
                @click="editRec(row as Record<string, unknown>)"
              >
                <EditOutlined />
                <span>编辑</span>
              </a-button>
              <a-button
                type="text"
                size="small"
                :class="dnsTableActionBtn"
                danger
                @click="delRec(row as Record<string, unknown>)"
              >
                <DeleteOutlined />
                <span>删除</span>
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-modal>

    <a-modal
      v-model:open="recDlg"
      :title="recEditId ? '编辑记录' : '新建记录'"
      :width="520"
      destroy-on-close
      wrap-class-name="dns-modal"
    >
      <a-form
        ref="recFormRef"
        :model="recForm"
        :rules="recFormRules"
        layout="vertical"
      >
        <a-form-item label="名称" name="name">
          <a-input
            v-model:value="recForm.name"
            placeholder="主机记录，如 www 或 @"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="类型" name="rtype">
          <a-select
            v-model:value="recForm.rtype"
            show-search
            placeholder="选择记录类型"
            :options="dnsQtypeSelectOptions"
            option-filter-prop="label"
            class="w-full"
          />
        </a-form-item>
        <a-form-item label="TTL" name="ttl">
          <a-input-number
            v-model:value="recForm.ttl"
            :min="1"
            class="w-full"
            placeholder="秒，如 300"
          />
        </a-form-item>
        <a-form-item label="内容" name="content">
          <a-input
            v-model:value="recForm.content"
            placeholder="IP 或目标域名"
            allow-clear
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button type="button" @click="recDlg = false">取消</a-button>
        <a-button type="primary" :loading="recSaveLoading" @click="submitRec"
          >保存</a-button
        >
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import {
  CaretRightOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  PauseOutlined,
} from '@ant-design/icons-vue'
import { ref, reactive, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import type { TableColumnType } from 'ant-design-vue'
import * as api from '../api'
import { apiErrorMessage } from '../api/errors'
import { dnsQtypeSelectOptions } from '../dnsQtypes'

const dnsTableActionBtn = 'inline-flex items-center justify-center'

const instFormRules: Record<string, Rule[]> = {
  name: [
    { required: true, message: '请输入名称', trigger: 'blur' },
    { max: 128, message: '名称过长', trigger: 'blur' },
  ],
  listen_addr: [{ required: true, message: '请输入监听地址', trigger: 'blur' }],
  listen_port: [
    {
      required: true,
      validator: async (_rule, value: unknown) => {
        if (value === null || value === undefined) {
          return Promise.reject(new Error('请输入端口'))
        }
        const n = Number(value)
        if (!Number.isFinite(n) || n < 1 || n > 65535) {
          return Promise.reject(new Error('端口须为 1–65535 的整数'))
        }
        return Promise.resolve()
      },
      trigger: 'change',
    },
  ],
}

const recFormRules: Record<string, Rule[]> = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  rtype: [{ required: true, message: '请选择类型', trigger: 'change' }],
  ttl: [
    {
      validator: async (_rule, value: unknown) => {
        if (value === null || value === undefined) {
          return Promise.reject(new Error('请输入 TTL'))
        }
        const n = Number(value)
        if (!Number.isFinite(n) || n < 1) {
          return Promise.reject(new Error('TTL 须为 ≥1 的整数'))
        }
        return Promise.resolve()
      },
      trigger: 'change',
    },
  ],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
}

interface UpstreamServerRow {
  address: string
  port: number
  protocol?: 'udp' | 'dot' | 'doh' | string
  path?: string | null
}

interface DefaultUpstream {
  id: number
  name: string
  servers: UpstreamServerRow[]
}

interface Inst {
  id: number
  name: string
  listen_addr: string
  listen_port: number
  paused: boolean
  /** 静态 DNS 记录条数 */
  record_count?: number
  runtime_error?: string | null
  default_upstream_group_id?: number | null
  default_upstream?: DefaultUpstream | null
}

const instColumns: TableColumnType<Inst>[] = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '监听地址', dataIndex: 'listen_addr', key: 'listen_addr' },
  { title: '端口', dataIndex: 'listen_port', key: 'listen_port', width: 90 },
  { title: '默认转发组', key: 'default_upstream', width: 280 },
  { title: '操作', key: 'action', width: 340 },
]

function formatServerList(servers: UpstreamServerRow[]) {
  if (!servers?.length) return '（无服务器）'
  return servers
    .map((s) => {
      const hostPort = s.address.includes(':')
        ? `[${s.address}]:${s.port}`
        : `${s.address}:${s.port}`
      switch (s.protocol) {
        case 'doh':
          return `https://${hostPort}${s.path || '/dns-query'}`
        case 'dot':
          return `tls://${hostPort}`
        default:
          return hostPort
      }
    })
    .join('，')
}

function instanceRowClassName(record: Inst) {
  if (record.runtime_error) return 'dns-instance-row--error'
  if (record.paused) return 'dns-instance-row--paused'
  return ''
}

const recColumns: TableColumnType<Record<string, unknown>>[] = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'rtype', key: 'rtype', width: 80 },
  { title: 'TTL', dataIndex: 'ttl', key: 'ttl', width: 70 },
  { title: '内容', dataIndex: 'content', key: 'content' },
  { title: '操作', key: 'action', width: 160 },
]

const rows = ref<Inst[]>([])
const listLoading = ref(false)
const dlg = ref(false)
const instFormRef = ref<FormInstance>()
const saveLoading = ref(false)
const editId = ref<number | null>(null)
const form = reactive({
  name: '',
  listen_addr: '0.0.0.0',
  listen_port: 5353,
  default_upstream_group_id: undefined as number | undefined,
})

const upstreamGroups = ref<{ id: number; name: string }[]>([])
const groupACDisplay = ref('')
const selectedGroupId = ref<number | undefined>(undefined)

const groupACOptions = computed(() =>
  upstreamGroups.value.map((g) => ({
    value: `${g.name} (#${g.id})`,
    label: `${g.name} (#${g.id})`,
    id: g.id,
  })),
)

function filterGroupOption(input: string, opt: { value?: string }) {
  return (opt.value || '').toLowerCase().includes(input.toLowerCase())
}

function onGroupSelect(value: string) {
  const o = groupACOptions.value.find((x) => x.value === value)
  if (o) selectedGroupId.value = o.id
  else selectedGroupId.value = undefined
}

function onGroupClear() {
  selectedGroupId.value = undefined
  groupACDisplay.value = ''
}

function syncDisplayFromGroupId(id: number | undefined) {
  selectedGroupId.value = id
  if (id == null) {
    groupACDisplay.value = ''
    return
  }
  const g = upstreamGroups.value.find((x) => x.id === id)
  groupACDisplay.value = g ? `${g.name} (#${g.id})` : `#${id}`
}

const recDrawer = ref(false)
const recInst = ref<Inst | null>(null)
const records = ref<Record<string, unknown>[]>([])
const recDlg = ref(false)
const recFormRef = ref<FormInstance>()
const recSaveLoading = ref(false)
const recEditId = ref<number | null>(null)
const recForm = reactive({ name: '', rtype: 'A', ttl: 300, content: '' })

async function loadGroups() {
  upstreamGroups.value = (await api.listUpstreamGroups()) as {
    id: number
    name: string
  }[]
}

async function load() {
  listLoading.value = true
  try {
    rows.value = (await api.listInstances()) as Inst[]
  } finally {
    listLoading.value = false
  }
}

function openCreate() {
  editId.value = null
  Object.assign(form, {
    name: '',
    listen_addr: '0.0.0.0',
    listen_port: 5353,
    default_upstream_group_id: undefined,
  })
  syncDisplayFromGroupId(undefined)
  dlg.value = true
}

function edit(row: Inst) {
  editId.value = row.id
  Object.assign(form, {
    name: row.name,
    listen_addr: row.listen_addr,
    listen_port: row.listen_port,
    default_upstream_group_id: row.default_upstream_group_id ?? undefined,
  })
  syncDisplayFromGroupId(row.default_upstream_group_id ?? undefined)
  dlg.value = true
}

async function togglePause(row: Inst) {
  try {
    await api.patchInstance(row.id, { paused: !row.paused })
    await load()
    message.success(row.paused ? '已恢复运行' : '已暂停')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '操作失败'))
  }
}

async function submitInst() {
  try {
    await instFormRef.value?.validate()
  } catch {
    return
  }
  await save()
}

async function save() {
  saveLoading.value = true
  try {
    const name = form.name.trim()
    const listenAddr = (form.listen_addr || '').trim() || '0.0.0.0'
    const listenPort = Number(form.listen_port)
    const dg = selectedGroupId.value

    if (editId.value) {
      const body: Record<string, unknown> = {
        name,
        listen_addr: listenAddr,
        listen_port: listenPort,
        default_upstream_group_id: dg ?? null,
      }
      await api.patchInstance(editId.value, body)
    } else {
      const body: Record<string, unknown> = {
        name,
        listen_addr: listenAddr,
        listen_port: listenPort,
      }
      if (dg != null) body.default_upstream_group_id = dg
      await api.createInstance(body)
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

function remove(row: Inst) {
  Modal.confirm({
    title: '删除实例',
    content: `确定删除实例「${row.name} (#${row.id})」吗？关联的静态记录也会一并删除，且不可恢复。`,
    okText: '删除',
    okType: 'danger',
    async onOk() {
      await api.deleteInstance(row.id)
      await load()
    },
  })
}

async function goRecords(row: Inst) {
  recInst.value = row
  records.value = (await api.listRecords(row.id)) as Record<string, unknown>[]
  recDrawer.value = true
}

function openRec() {
  recEditId.value = null
  Object.assign(recForm, { name: '', rtype: 'A', ttl: 300, content: '' })
  recDlg.value = true
}

function editRec(row: Record<string, unknown>) {
  recEditId.value = row.id as number
  Object.assign(recForm, {
    name: row.name,
    rtype: String(row.rtype ?? 'A').toUpperCase(),
    ttl: row.ttl,
    content: row.content,
  })
  recDlg.value = true
}

async function submitRec() {
  try {
    await recFormRef.value?.validate()
  } catch {
    return
  }
  await saveRec()
}

async function saveRec() {
  if (!recInst.value) return
  recSaveLoading.value = true
  try {
    if (recEditId.value) {
      await api.patchRecord(recEditId.value, recForm)
    } else {
      await api.createRecord(recInst.value.id, recForm)
    }
    recDlg.value = false
    records.value = (await api.listRecords(recInst.value.id)) as Record<
      string,
      unknown
    >[]
    message.success('已保存')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '保存失败'))
  } finally {
    recSaveLoading.value = false
  }
}

async function delRec(row: Record<string, unknown>) {
  await api.deleteRecord(row.id as number)
  if (recInst.value)
    records.value = (await api.listRecords(recInst.value.id)) as Record<
      string,
      unknown
    >[]
}

onMounted(async () => {
  await loadGroups()
  await load()
})
</script>

<style scoped>
:deep(.dns-instance-row--paused > td) {
  background-color: rgba(234, 179, 8, 0.16) !important;
}
:deep(.dns-instance-row--error > td) {
  background-color: rgba(220, 38, 38, 0.2) !important;
}
</style>
