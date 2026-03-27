<template>
  <div class="min-h-screen flex items-center justify-center bg-[#0a0a0a] p-4">
    <a-card class="w-full max-w-md shadow-2xl" title="DnsTube 管理登录">
      <a-form :model="form" :rules="loginRules" layout="vertical" @finish="onSubmit">
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="form.username"
            autocomplete="username"
            placeholder="请输入用户名"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="密码" name="password">
          <a-input-password
            v-model:value="form.password"
            autocomplete="current-password"
            placeholder="请输入密码"
          />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" block :loading="loading">登录</a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import * as api from '../api'
import { apiErrorMessage } from '../api/errors'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: 'admin', password: '' })

const loginRules: Record<string, Rule[]> = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  loading.value = true
  try {
    const r = await api.login(form.username, form.password)
    auth.setToken(r.token)
    auth.username = r.username
    await router.push('/dashboard')
  } catch (e: unknown) {
    message.error(apiErrorMessage(e, '登录失败'))
  } finally {
    loading.value = false
  }
}
</script>
