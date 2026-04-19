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
        <a-button type="primary" class="dns-upstream-add-group" @click="addGroup">添加</a-button>
      </div>
    </div>

    <div
      v-if="groups?.length > 0"
      class="grid min-h-0 grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3"
    >
      <a-card
        v-for="g in groups"
        :key="g.id"
        class="dns-upstream-card flex h-[350px] flex-col overflow-hidden border-[#303030]! bg-[#141414]!"
        :bordered="true"
      >
        <template #title>
          <div class="flex min-w-0 items-center justify-between gap-2 pr-0">
            <div class="flex min-w-0 flex-1 items-center gap-2">
              <span class="shrink-0 text-sm text-[rgba(255,255,255,0.45)]">#{{ g.id }}</span>
              <a-input
                v-if="editingGroupId === g.id"
                v-model:value="groupNameDraft[g.id]"
                placeholder="转发组名称"
                allow-clear
                class="dns-upstream-group-name min-w-0 flex-1"
                @pressEnter="saveGroupName(g.id)"
                @blur="stopEditGroupName(g.id)"
              />
              <span
                v-else
                class="dns-upstream-group-title-text min-w-0 flex-1 truncate"
                :title="groupNameDraft[g.id] || g.name"
                @click="startEditGroupName(g.id)"
              >
                {{ groupNameDraft[g.id] || g.name }}
              </span>
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
        <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div class="mb-[8px] flex shrink-0 flex-col gap-2">
            <div class="flex items-stretch gap-2">
              <a-select
                v-model:value="srvProto[g.id]"
                class="w-[88px] shrink-0"
                :options="protoOptions"
                @change="onProtoChange(g.id)"
              />
              <a-input
                v-model:value="srvAddr[g.id]"
                allow-clear
                placeholder="IPv4 或 IPv6，如 8.8.8.8 或 2001:db8::1"
                class="min-w-0 flex-1"
              />
              <a-input-number
                v-model:value="srvPort[g.id]"
                :min="1"
                :max="65535"
                :controls="false"
                placeholder="端口"
                class="w-17"
              />
              <a-button type="primary" class="shrink-0" @click="addSrv(g.id)">
                <span class="inline-flex items-center gap-1">
                  <PlusOutlined />
                  <span>添加</span>
                </span>
              </a-button>
            </div>
            <div
              v-if="srvProto[g.id] && srvProto[g.id] !== 'udp'"
              class="flex items-stretch gap-2"
            >
              <a-input
                v-if="srvProto[g.id] === 'doh'"
                v-model:value="srvPath[g.id]"
                allow-clear
                placeholder="DoH 路径，缺省 /dns-query"
                class="min-w-0 flex-1"
              />
              <a-input
                v-model:value="srvSNI[g.id]"
                allow-clear
                placeholder="可选 TLS SNI，留空则用地址"
                class="min-w-0 flex-1"
              />
            </div>
          </div>
          <div class="min-h-0 flex-1 overflow-hidden">
            <a-table
              :columns="srvColumns"
              :data-source="servers[g.id] || []"
              :pagination="false"
              bordered
              size="small"
              row-key="id"
              :scroll="{ x: 'max-content', y: 205 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'protocol'">
                  <a-tag :color="protocolTagColor((record as ServerRow).protocol)">
                    {{ protocolLabel((record as ServerRow).protocol || 'udp') }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'address'">
                  <span class="font-mono text-xs">
                    {{ formatServerEndpoint(record as ServerRow) }}
                  </span>
                </template>
                <template v-else-if="column.key === 'action'">
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
import { DeleteOutlined, PlusOutlined, SaveOutlined } from "@ant-design/icons-vue";
import { ref, reactive, onMounted } from "vue";
import { message, Modal } from "ant-design-vue";
import type { TableColumnType } from "ant-design-vue";
import * as api from "../api";
import { apiErrorMessage } from "../api/errors";
import {
  defaultPortForProtocol,
  isUpstreamIpV4OrV6,
  normalizeDoHPath,
  normalizeTLSServerName,
  normalizeUpstreamAddr,
  protocolLabel,
  type UpstreamProtocol,
} from "../upstreamAddr";

interface G {
  id: number;
  name: string;
}

interface ServerRow {
  id: number;
  address: string;
  port: number;
  sort_order: number;
  protocol?: UpstreamProtocol | string;
  path?: string | null;
  tls_server_name?: string | null;
}

const protoOptions = [
  { label: "UDP", value: "udp" },
  { label: "DoT", value: "dot" },
  { label: "DoH", value: "doh" },
];

function protocolTagColor(p?: string): string {
  switch (p) {
    case "dot":
      return "geekblue";
    case "doh":
      return "purple";
    default:
      return "default";
  }
}

/**
 * 计算新增上游时使用的 sort_order：
 *   - 取当前组内已有 sort_order 的最大值 + 1；
 *   - 空组返回 0。
 * 不能用 servers.length：删除中间项后会与剩余项重复。
 */
function nextSortOrder(rows: ServerRow[] | undefined): number {
  if (!rows || rows.length === 0) return 0;
  let max = -1;
  for (const r of rows) {
    const v = typeof r.sort_order === "number" ? r.sort_order : 0;
    if (v > max) max = v;
  }
  return max + 1;
}

function formatServerEndpoint(row: ServerRow): string {
  const proto = (row.protocol || "udp") as UpstreamProtocol;
  const hostPort = row.address.includes(":")
    ? `[${row.address}]:${row.port}`
    : `${row.address}:${row.port}`;
  if (proto === "doh") {
    const path = row.path || "/dns-query";
    return `https://${hostPort}${path}`;
  }
  if (proto === "dot") {
    return `tls://${hostPort}`;
  }
  return `udp://${hostPort}`;
}

const srvColumns: TableColumnType<Record<string, unknown>>[] = [
  { title: "协议", key: "protocol", width: 72 },
  { title: "端点", dataIndex: "address", key: "address", ellipsis: true },
  { title: "顺序", dataIndex: "sort_order", key: "sort_order", width: 64 },
  { title: "操作", key: "action", width: 88 },
];

const groups = ref<G[]>([]);
const servers = reactive<Record<number, ServerRow[]>>({});
const srvAddr = reactive<Record<number, string>>({});
const srvPort = reactive<Record<number, number>>({});
const srvProto = reactive<Record<number, UpstreamProtocol>>({});
const srvPath = reactive<Record<number, string>>({});
const srvSNI = reactive<Record<number, string>>({});
const newName = ref("");
/** 卡片标题内编辑中的组名，与列表同步 */
const groupNameDraft = reactive<Record<number, string>>({});
/** 当前处于编辑态的组 ID（null 表示全部文本态） */
const editingGroupId = ref<number | null>(null);

async function load() {
  const raw = (await api.listUpstreamGroups()) as G[] | null | undefined;
  groups.value = Array.isArray(raw) ? raw : [];
  for (const g of groups.value) {
    groupNameDraft[g.id] = g.name;
    srvAddr[g.id] = "";
    if (!srvProto[g.id]) srvProto[g.id] = "udp";
    srvPort[g.id] = defaultPortForProtocol(srvProto[g.id]);
    if (srvPath[g.id] === undefined) srvPath[g.id] = "";
    if (srvSNI[g.id] === undefined) srvSNI[g.id] = "";
    const srvRaw = (await api.listGroupServers(g.id)) as ServerRow[] | null | undefined;
    servers[g.id] = Array.isArray(srvRaw) ? srvRaw : [];
  }
}

/** 切换协议时重置端口为协议默认值（仅当当前是另一协议默认端口时才覆盖，避免吞用户手填值）。 */
function onProtoChange(gid: number) {
  const proto = srvProto[gid];
  const cur = srvPort[gid];
  const isLikelyDefault = cur === 53 || cur === 853 || cur === 443;
  if (isLikelyDefault) {
    srvPort[gid] = defaultPortForProtocol(proto);
  }
}

/** 组名输入与已保存值不一致时显示「保存」 */
function groupNameDirty(gid: number): boolean {
  const g = groups.value.find((x) => x.id === gid);
  if (!g) return false;
  const draft = (groupNameDraft[gid] ?? "").trim();
  return draft !== g.name;
}

async function addGroup() {
  const name = newName.value?.trim();
  if (!name) {
    message.warning("请输入转发组名称");
    return;
  }
  await api.createGroup(name);
  newName.value = "";
  await load();
  message.success("已创建");
}

async function saveGroupName(gid: number) {
  const name = groupNameDraft[gid]?.trim();
  if (!name) {
    message.warning("请输入转发组名称");
    const g = groups.value.find((x) => x.id === gid);
    if (g) groupNameDraft[gid] = g.name;
    return;
  }
  const prev = groups.value.find((x) => x.id === gid);
  if (prev && prev.name === name) return;
  try {
    await api.patchUpstreamGroup(gid, { name });
    await load();
    if (editingGroupId.value === gid) editingGroupId.value = null;
    message.success("已保存");
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, "保存失败"));
    if (prev) groupNameDraft[gid] = prev.name;
  }
}

function startEditGroupName(gid: number) {
  editingGroupId.value = gid;
}

function stopEditGroupName(gid: number) {
  if (editingGroupId.value === gid) editingGroupId.value = null;
}

async function addSrv(gid: number) {
  const raw = srvAddr[gid] ?? "";
  if (!raw.trim()) {
    message.warning("请输入服务器地址");
    return;
  }
  if (!isUpstreamIpV4OrV6(raw)) {
    message.warning("地址须为合法 IPv4 或 IPv6");
    return;
  }
  const proto = (srvProto[gid] || "udp") as UpstreamProtocol;
  const addr = normalizeUpstreamAddr(raw)!;

  const body: Parameters<typeof api.addServer>[1] = {
    address: addr,
    port: srvPort[gid] || defaultPortForProtocol(proto),
    sort_order: nextSortOrder(servers[gid]),
    protocol: proto,
  };

  if (proto === "doh") {
    const path = normalizeDoHPath(srvPath[gid] || "");
    if (path === null) {
      message.warning("DoH 路径须以 / 开头且不含查询/片段");
      return;
    }
    body.path = path;
  }
  if (proto !== "udp") {
    const sni = normalizeTLSServerName(srvSNI[gid] || "");
    if (sni === null) {
      message.warning("TLS SNI 不合法");
      return;
    }
    if (sni) body.tls_server_name = sni;
  }

  try {
    await api.addServer(gid, body);
    srvAddr[gid] = "";
    srvPath[gid] = "";
    srvSNI[gid] = "";
    servers[gid] = (await api.listGroupServers(gid)) as ServerRow[];
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, "添加失败"));
  }
}

async function delSrv(id: number) {
  await api.deleteServer(id);
  await load();
}

function delGroup(id: number) {
  const g = groups.value.find((x) => x.id === id);
  const groupLabel = g ? `${g.name} (#${g.id})` : `#${id}`;
  Modal.confirm({
    title: "删除转发组",
    content: `确定删除转发组「${groupLabel}」吗？组内服务器配置将一并删除，且不可恢复。`,
    okText: "删除",
    okType: "danger",
    async onOk() {
      await api.deleteGroup(id);
      await load();
    },
  });
}

onMounted(load);
</script>

<style scoped>
.dns-upstream-new-name {
  width: 200px;
  min-height: 32px;
}
.dns-upstream-group-name {
  min-height: 28px;
}
.dns-upstream-group-title-text {
  color: rgba(255, 255, 255, 0.88);
  cursor: pointer;
  transition: color 0.2s ease;
}
.dns-upstream-group-title-text:hover {
  color: #69b1ff;
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
  padding: 8px;
  overflow: hidden;
}
</style>
