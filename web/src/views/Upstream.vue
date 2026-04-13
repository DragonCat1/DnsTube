<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-2 shrink-0">
      <h2 class="m-0 text-lg font-semibold">转发服务器</h2>
      <div class="flex flex-wrap items-center gap-2">
        <a-input
          v-model:value="newName"
          allow-clear
          placeholder="添加新转发组"
          class="dns-upstream-new-name"
        />
        <a-button
          type="primary"
          class="dns-upstream-add-group"
          @click="addGroup"
          >添加</a-button
        >
      </div>
    </div>

    <div
      v-if="groups?.length > 0"
      class="grid min-h-0 grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3"
    >
      <a-card
        v-for="g in groups"
        :key="g.id"
        class="dns-upstream-card flex h-[22rem] flex-col overflow-hidden border-[#303030]! bg-[#141414]!"
        :bordered="true"
      >
        <template #title>
          <div class="flex min-w-0 items-center justify-between gap-2 pr-0">
            <div class="flex min-w-0 flex-1 items-center gap-2">
              <a-input
                v-model:value="groupNameDraft[g.id]"
                placeholder="转发组名称"
                allow-clear
                class="dns-upstream-group-name min-w-0 flex-1"
                @pressEnter="saveGroupName(g.id)"
              />
              <span class="shrink-0 text-xs text-[rgba(255,255,255,0.45)]"
                >#{{ g.id }}</span
              >
              <a-button
                v-if="groupNameDirty(g.id)"
                type="text"
                size="small"
                class="inline-flex shrink-0 items-center"
                @click="saveGroupName(g.id)"
              >
                <SaveOutlined />
                <span>保存</span>
              </a-button>
            </div>
            <a-button
              type="text"
              size="small"
              danger
              class="inline-flex shrink-0 items-center"
              @click="delGroup(g.id)"
            >
              <DeleteOutlined />
              <span>删除</span>
            </a-button>
          </div>
        </template>
        <div
          class="dns-upstream-card__body flex min-h-0 flex-1 flex-col overflow-hidden"
        >
          <div class="mb-3 flex shrink-0 flex-col gap-2 dns-upstream-add-row">
            <a-input
              v-model:value="srvAddr[g.id]"
              allow-clear
              placeholder="IPv4 或 IPv6，如 8.8.8.8 或 2001:db8::1"
              class="dns-upstream-addr w-full"
            />
            <div class="flex flex-wrap items-center gap-2">
              <a-input-number
                v-model:value="srvPort[g.id]"
                :min="1"
                :max="65535"
                placeholder="端口"
                class="dns-upstream-port"
              />
              <a-button
                type="primary"
                class="dns-upstream-add-srv shrink-0"
                @click="addSrv(g.id)"
              >
                <span class="inline-flex items-center gap-1">
                  <PlusOutlined />
                  <span>添加</span>
                </span>
              </a-button>
            </div>
          </div>
          <div class="dns-upstream-table-host min-h-0 flex-1 overflow-hidden">
            <a-table
              :columns="srvColumns"
              :data-source="servers[g.id] || []"
              :pagination="false"
              bordered
              size="small"
              row-key="id"
              :scroll="{ x: 'max-content', y: UPSTREAM_TABLE_BODY_SCROLL_Y }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'action'">
                  <a-button
                    type="text"
                    size="small"
                    danger
                    class="inline-flex items-center"
                    @click="delSrv((record as { id: number }).id)"
                  >
                    <DeleteOutlined />
                    <span>删除</span>
                  </a-button>
                </template>
              </template>
            </a-table>
          </div>
        </div>
      </a-card>
    </div>
    <div v-else class="shrink-0">
      <a-empty description="暂无转发服务器" />
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  DeleteOutlined,
  PlusOutlined,
  SaveOutlined,
} from '@ant-design/icons-vue'
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { TableColumnType } from 'ant-design-vue'
import * as api from '../api'
import { apiErrorMessage } from '../api/errors'
import { isUpstreamIpV4OrV6, normalizeUpstreamAddr } from '../upstreamAddr'

interface G {
  id: number
  name: string
}

/** 与卡片固定高度 `h-[22rem]` 匹配的表体滚动区高度（px），表头固定 */
const UPSTREAM_TABLE_BODY_SCROLL_Y = 140

const srvColumns: TableColumnType<Record<string, unknown>>[] = [
  { title: '地址', dataIndex: 'address', key: 'address', ellipsis: true },
  { title: '端口', dataIndex: 'port', key: 'port', width: 72 },
  { title: '顺序', dataIndex: 'sort_order', key: 'sort_order', width: 64 },
  { title: '操作', key: 'action', width: 88 },
]

const groups = ref<G[]>([])
const servers = reactive<Record<number, Record<string, unknown>[]>>({})
const srvAddr = reactive<Record<number, string>>({})
const srvPort = reactive<Record<number, number>>({})
const newName = ref('')
/** 卡片标题内编辑中的组名，与列表同步 */
const groupNameDraft = reactive<Record<number, string>>({})

async function load() {
  const raw = (await api.listUpstreamGroups()) as G[] | null | undefined
  groups.value = Array.isArray(raw) ? raw : []
  for (const g of groups.value) {
    groupNameDraft[g.id] = g.name
    srvAddr[g.id] = ''
    srvPort[g.id] = 53
    const srvRaw = (await api.listGroupServers(g.id)) as
      | Record<string, unknown>[]
      | null
      | undefined
    servers[g.id] = Array.isArray(srvRaw) ? srvRaw : []
  }
}

/** 组名输入与已保存值不一致时显示「保存」 */
function groupNameDirty(gid: number): boolean {
  const g = groups.value.find((x) => x.id === gid)
  if (!g) return false
  const draft = (groupNameDraft[gid] ?? '').trim()
  return draft !== g.name
}

async function addGroup() {
  const name = newName.value?.trim()
  if (!name) {
    message.warning('请输入转发组名称')
    return
  }
  await api.createGroup(name)
  newName.value = ''
  await load()
  message.success('已创建')
}

async function saveGroupName(gid: number) {
  const name = groupNameDraft[gid]?.trim()
  if (!name) {
    message.warning('请输入转发组名称')
    const g = groups.value.find((x) => x.id === gid)
    if (g) groupNameDraft[gid] = g.name
    return
  }
  const prev = groups.value.find((x) => x.id === gid)
  if (prev && prev.name === name) return
  try {
    await api.patchUpstreamGroup(gid, { name })
    await load()
    message.success('已保存')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '保存失败'))
    if (prev) groupNameDraft[gid] = prev.name
  }
}

async function addSrv(gid: number) {
  const raw = srvAddr[gid] ?? ''
  if (!raw.trim()) {
    message.warning('请输入服务器地址')
    return
  }
  if (!isUpstreamIpV4OrV6(raw)) {
    message.warning('地址须为合法 IPv4 或 IPv6')
    return
  }
  const addr = normalizeUpstreamAddr(raw)!
  try {
    await api.addServer(gid, {
      address: addr,
      port: srvPort[gid] || 53,
      sort_order: servers[gid]?.length || 0,
    })
    srvAddr[gid] = ''
    servers[gid] = (await api.listGroupServers(gid)) as Record<string, unknown>[]
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '添加失败'))
  }
}

async function delSrv(id: number) {
  await api.deleteServer(id)
  await load()
}

function delGroup(id: number) {
  const g = groups.value.find((x) => x.id === id)
  const groupLabel = g ? `${g.name} (#${g.id})` : `#${id}`
  Modal.confirm({
    title: '删除转发组',
    content: `确定删除转发组「${groupLabel}」吗？组内服务器配置将一并删除，且不可恢复。`,
    okText: '删除',
    okType: 'danger',
    async onOk() {
      await api.deleteGroup(id)
      await load()
    },
  })
}

onMounted(load)
</script>

<style scoped>
.dns-upstream-new-name {
  width: 200px;
  min-height: 32px;
}
.dns-upstream-group-name {
  min-height: 28px;
}
.dns-upstream-add-group {
  min-height: 32px;
}
.dns-upstream-card :deep(.ant-card-head) {
  flex-shrink: 0;
  min-height: 48px;
  padding: 0 12px;
  border-bottom-color: #303030;
}
.dns-upstream-card :deep(.ant-card-body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 12px;
  overflow: hidden;
}
.dns-upstream-add-row :deep(.ant-input),
.dns-upstream-add-row :deep(.ant-input-number),
.dns-upstream-add-srv {
  min-height: 32px;
}
.dns-upstream-port {
  width: 120px;
}
.dns-upstream-card :deep(.ant-table-wrapper) {
  font-size: 12px;
}
</style>
