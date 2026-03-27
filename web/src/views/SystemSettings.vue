<template>
  <div class="max-w-3xl">
    <h2 class="text-lg font-semibold m-0 mb-6">系统设置</h2>
    <div class="flex flex-col gap-12">
      <a-card title="账户" class="border-[#303030]! bg-[#141414]!">
        <a-form :model="profileForm" :rules="profileRules" layout="vertical" @finish="saveProfile">
          <a-form-item label="用户名" name="username">
            <a-input v-model:value="profileForm.username" allow-clear placeholder="登录用户名" />
          </a-form-item>
          <a-form-item label="新密码" name="password">
            <a-input-password
              v-model:value="profileForm.password"
              placeholder="修改密码时填写，不改请留空"
            />
          </a-form-item>
          <a-form-item>
            <a-button type="primary" html-type="submit" :loading="profileLoading">保存账户</a-button>
          </a-form-item>
        </a-form>
      </a-card>

      <a-card title="Telegram 机器人通知" class="border-[#303030]! bg-[#141414]!">
        <p class="text-sm text-[rgba(255,255,255,0.45)] m-0 mb-4">
          启用后按下方勾选的级别推送：系统日志（与运行日志级别一致）与审计事件（登录、改密、设置变更等）。
        </p>
        <a-form :model="tgForm" layout="vertical" @finish="saveTelegram">
          <a-form-item label="启用通知">
            <a-switch v-model:checked="tgForm.enabled" />
          </a-form-item>
          <a-form-item label="推送级别">
            <a-checkbox-group v-model:value="tgForm.notify_levels" class="flex flex-col gap-1">
              <a-checkbox value="debug">debug（系统）</a-checkbox>
              <a-checkbox value="info">info（系统）</a-checkbox>
              <a-checkbox value="warn">warn（系统）</a-checkbox>
              <a-checkbox value="error">error（系统）</a-checkbox>
              <a-checkbox value="audit">audit（审计）</a-checkbox>
            </a-checkbox-group>
            <p class="text-xs text-[rgba(255,255,255,0.35)] m-0 mt-2">
              默认仅 warn、error；勾选 audit 可收到登录与账户相关操作。
            </p>
          </a-form-item>
          <a-form-item label="Bot Token" name="bot_token">
            <a-input-password
              v-model:value="tgForm.bot_token"
              allow-clear
              :placeholder="tgTokenPlaceholder"
              autocomplete="off"
            />
          </a-form-item>
          <a-form-item label="Chat ID" name="chat_id">
            <a-input
              v-model:value="tgForm.chat_id"
              allow-clear
              placeholder="频道或用户数字 ID，如 -100xxxxxxxxxx"
            />
          </a-form-item>
          <a-form-item>
            <a-space wrap>
              <a-button type="primary" html-type="submit" :loading="tgLoading">保存通知设置</a-button>
              <a-button :loading="tgTestLoading" :disabled="!tgForm.enabled" @click="sendTelegramTest">
                发送测试消息
              </a-button>
              <a-button
                v-if="tgLoaded && telegramBotTokenSet"
                danger
                :loading="tgLoading"
                @click="clearTelegramToken"
              >
                清除已保存的 Token
              </a-button>
            </a-space>
          </a-form-item>
        </a-form>
      </a-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import * as api from '../api'
import { apiErrorMessage } from '../api/errors'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const profileLoading = ref(false)
const profileForm = reactive({ username: '', password: '' })

const profileRules: Record<string, Rule[]> = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
}

const tgLoading = ref(false)
const tgTestLoading = ref(false)
const tgLoaded = ref(false)
const telegramBotTokenSet = ref(false)
const tgForm = reactive({
  enabled: false,
  bot_token: '',
  chat_id: '',
  notify_levels: [] as string[],
})

const tgTokenPlaceholder = computed(() =>
  telegramBotTokenSet.value ? '已保存 Token，留空不修改；填写则覆盖' : '从 @BotFather 获取',
)

onMounted(async () => {
  try {
    const me = await api.me()
    profileForm.username = me.username
    auth.username = me.username
  } catch {
    /* ignore */
  }
  try {
    const s = await api.getSystemSettings()
    tgForm.enabled = s.telegram_enabled
    tgForm.chat_id = s.telegram_chat_id ?? ''
    tgForm.bot_token = ''
    tgForm.notify_levels = Array.isArray(s.telegram_notify_levels) ? [...s.telegram_notify_levels] : []
    telegramBotTokenSet.value = s.telegram_bot_token_set
    tgLoaded.value = true
  } catch {
    tgLoaded.value = true
  }
})

async function saveProfile() {
  profileLoading.value = true
  try {
    const body: { username?: string; password?: string } = {}
    if (profileForm.username) body.username = profileForm.username
    if (profileForm.password) body.password = profileForm.password
    await api.patchProfile(body)
    profileForm.password = ''
    auth.username = profileForm.username
    message.success('账户已保存')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '保存失败'))
  } finally {
    profileLoading.value = false
  }
}

async function saveTelegram() {
  tgLoading.value = true
  try {
    const body: Record<string, unknown> = {
      telegram_enabled: tgForm.enabled,
      telegram_chat_id: tgForm.chat_id.trim(),
      telegram_notify_levels: [...tgForm.notify_levels],
    }
    const t = tgForm.bot_token.trim()
    if (t) {
      body.telegram_bot_token = t
    }
    await api.patchSystemSettings(body)
    tgForm.bot_token = ''
    const s = await api.getSystemSettings()
    telegramBotTokenSet.value = s.telegram_bot_token_set
    tgForm.notify_levels = Array.isArray(s.telegram_notify_levels) ? [...s.telegram_notify_levels] : []
    message.success('通知设置已保存')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '保存失败'))
  } finally {
    tgLoading.value = false
  }
}

async function sendTelegramTest() {
  tgTestLoading.value = true
  try {
    await api.postTelegramTest()
    message.success('测试消息已发送，请查看 Telegram')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '发送失败'))
  } finally {
    tgTestLoading.value = false
  }
}

function clearTelegramToken() {
  Modal.confirm({
    title: '清除 Bot Token',
    content: '确定清除已保存的 Telegram Bot Token？',
    okText: '清除',
    okType: 'danger',
    async onOk() {
      tgLoading.value = true
      try {
        await api.patchSystemSettings({ telegram_bot_token: '' })
        tgForm.bot_token = ''
        const s = await api.getSystemSettings()
        telegramBotTokenSet.value = s.telegram_bot_token_set
        message.success('已清除')
      } catch (e: unknown) {
        message.error(apiErrorMessage(e, '操作失败'))
      } finally {
        tgLoading.value = false
      }
    },
  })
}
</script>
