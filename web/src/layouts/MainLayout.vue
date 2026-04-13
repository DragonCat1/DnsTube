<template>
  <a-layout class="flex min-h-screen w-full flex-1 flex-row">
    <a-layout-sider
      v-if="!isMobile"
      :width="220"
      theme="dark"
      class="dns-sidebar shrink-0 self-stretch border-r border-[#262626] bg-black! min-h-screen"
    >
      <div class="flex min-h-screen flex-col bg-black">
        <div
          class="dns-sidebar-brand flex h-14! min-h-14! max-h-14! box-border items-center gap-2 border-b border-[#303030] bg-black px-4! leading-normal! text-lg font-bold text-white"
        >
          <img :src="logo" class="shrink-0 w-6 h-6" alt="DnsTube" />
          <span>DnsTube</span>
        </div>
        <a-menu
          mode="inline"
          theme="dark"
          :selected-keys="[route.path]"
          class="flex-1 border-0! bg-transparent! [&_.ant-menu-sub]:bg-transparent!"
          @click="onMenuClick"
        >
        <a-menu-item key="/dashboard">
          <template #icon><DashboardOutlined /></template>
          Dashboard
        </a-menu-item>
        <a-menu-item key="/instances">
          <template #icon><CloudServerOutlined /></template>
          DNS 实例
        </a-menu-item>
        <a-menu-item key="/upstream">
          <template #icon><LinkOutlined /></template>
          转发服务器
        </a-menu-item>
        <a-menu-item key="/forward">
          <template #icon><ShareAltOutlined /></template>
          转发规则
        </a-menu-item>
        <a-menu-item key="/logs">
          <template #icon><FileTextOutlined /></template>
          查询日志
        </a-menu-item>
        <a-menu-item key="/cache">
          <template #icon><HddOutlined /></template>
          缓存列表
        </a-menu-item>
        <a-menu-item key="/system-logs">
          <template #icon><HistoryOutlined /></template>
          系统日志
        </a-menu-item>
        <a-menu-item key="/settings">
          <template #icon><SettingOutlined /></template>
          系统设置
        </a-menu-item>
        </a-menu>
      </div>
    </a-layout-sider>
    <a-layout class="dns-main-column flex min-h-screen min-w-0 flex-1 flex-col">
      <a-layout-header
        class="dns-topbar flex h-14! min-h-14! max-h-14! shrink-0 box-border items-center justify-end gap-3 border-b border-[#303030] bg-[#141414]! px-4! leading-normal!"
      >
        <a-button
          v-if="isMobile"
          type="text"
          class="mr-auto inline-flex items-center text-[rgba(255,255,255,0.88)]!"
          aria-label="打开导航菜单"
          @click="mobileMenuOpen = true"
        >
          <MenuOutlined />
        </a-button>
        <span
          class="dns-topbar-user inline-flex max-w-[min(40vw,240px)] items-center gap-1.5 truncate text-sm leading-none text-[rgba(255,255,255,0.85)]"
          :title="username || undefined"
        >
          <UserOutlined class="dns-topbar-user__icon shrink-0 text-[14px] leading-none text-[rgba(255,255,255,0.65)]" />
          <span class="truncate">{{ username || '—' }}</span>
        </span>
        <a-button
          type="link"
          danger
          class="dns-topbar-logout inline-flex! h-auto! min-h-0! items-center! p-0! text-sm! leading-none! [&>span]:inline-flex! [&>span]:min-h-0! [&>span]:items-center! [&>span]:leading-none!"
          @click="logout"
        >
          <span class="inline-flex items-center gap-1.5">
            <LogoutOutlined class="dns-topbar-user__icon shrink-0 text-[14px] leading-none" />
            <span class="leading-none">退出</span>
          </span>
        </a-button>
      </a-layout-header>
      <a-layout-content class="dns-main-content flex-1 overflow-auto bg-[#0a0a0a]! min-h-0 p-6!">
        <router-view />
      </a-layout-content>
    </a-layout>
    <a-drawer
      v-model:open="mobileMenuOpen"
      placement="left"
      :closable="false"
      :width="220"
      class="dns-mobile-nav-drawer"
      :body-style="{ padding: '0' }"
    >
      <div class="flex min-h-full flex-col bg-black">
        <div
          class="dns-sidebar-brand flex h-14! min-h-14! max-h-14! box-border items-center gap-2 border-b border-[#303030] bg-black px-4! leading-normal! text-lg font-bold text-white"
        >
          <img :src="logo" class="shrink-0 w-6 h-6" alt="DnsTube" />
          <span>DnsTube</span>
        </div>
        <a-menu
          mode="inline"
          theme="dark"
          :selected-keys="[route.path]"
          class="flex-1 border-0! bg-transparent! [&_.ant-menu-sub]:bg-transparent!"
          @click="onMobileMenuClick"
        >
          <a-menu-item key="/dashboard">
            <template #icon><DashboardOutlined /></template>
            Dashboard
          </a-menu-item>
          <a-menu-item key="/instances">
            <template #icon><CloudServerOutlined /></template>
            DNS 实例
          </a-menu-item>
          <a-menu-item key="/upstream">
            <template #icon><LinkOutlined /></template>
            转发服务器
          </a-menu-item>
          <a-menu-item key="/forward">
            <template #icon><ShareAltOutlined /></template>
            转发规则
          </a-menu-item>
          <a-menu-item key="/logs">
            <template #icon><FileTextOutlined /></template>
            查询日志
          </a-menu-item>
          <a-menu-item key="/cache">
            <template #icon><HddOutlined /></template>
            缓存列表
          </a-menu-item>
          <a-menu-item key="/system-logs">
            <template #icon><HistoryOutlined /></template>
            系统日志
          </a-menu-item>
          <a-menu-item key="/settings">
            <template #icon><SettingOutlined /></template>
            系统设置
          </a-menu-item>
        </a-menu>
      </div>
    </a-drawer>
  </a-layout>
</template>

<script setup lang="ts">
import {
  DashboardOutlined,
  CloudServerOutlined,
  LinkOutlined,
  ShareAltOutlined,
  FileTextOutlined,
  HistoryOutlined,
  HddOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuOutlined,
} from '@ant-design/icons-vue'
import { Grid } from 'ant-design-vue'
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import * as api from '../api'
import { useAuthStore } from '../stores/auth'
import logo from '../assets/logo.svg'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { username } = storeToRefs(auth)
const mobileMenuOpen = ref(false)
const screens = Grid.useBreakpoint()
const isMobile = computed(() => !screens.value.lg)

onMounted(async () => {
  if (!auth.token || username.value) return
  try {
    const me = await api.me()
    auth.username = me.username
  } catch {
    /* 未授权等由路由/拦截器处理 */
  }
})

function onMenuClick({ key }: { key: string | number }) {
  router.push(String(key))
}

function onMobileMenuClick({ key }: { key: string | number }) {
  mobileMenuOpen.value = false
  router.push(String(key))
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
