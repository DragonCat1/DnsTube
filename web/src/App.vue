<template>
  <a-config-provider :locale="antdZhCN" :theme="{ algorithm: theme.darkAlgorithm }">
    <div class="flex min-h-screen flex-1 flex-col">
      <router-view />
    </div>
  </a-config-provider>
</template>

<script setup lang="ts">
import { theme } from 'ant-design-vue'
import antdZhCN from './antd-locale'
</script>
