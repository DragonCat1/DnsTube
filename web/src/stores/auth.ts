import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('dnstube_token') || '')
  const username = ref('')

  const isAuthed = computed(() => !!token.value)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('dnstube_token', t)
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('dnstube_token')
  }

  return { token, username, isAuthed, setToken, logout }
})
