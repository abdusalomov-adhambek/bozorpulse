import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api, { getToken, setToken, clearToken } from '../api/client'

const DEMO_ACCOUNTS = {
  admin: { username: 'admin', password: 'admin123' },
  supplier: { username: 'supplier1', password: 'sup123' },
  customer: { username: 'customer1', password: 'cust123' }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(getToken())
  const user = ref(null)

  const isAuthenticated = computed(() => Boolean(token.value))
  const role = computed(() => user.value?.role || null)
  const status = computed(() => user.value?.status || null)
  const isConfirmed = computed(() => status.value === 'confirmed')
  const isAdmin = computed(() => role.value === 'admin')
  const isSupplier = computed(() => role.value === 'supplier')
  const isCustomer = computed(() => role.value === 'customer')
  const displayName = computed(() => user.value?.username || user.value?.company_name || 'User')

  async function login(username, password) {
    const data = await api.post('/auth/login', { username, password }, { auth: false })
    if (data?.token) {
      setToken(data.token)
      token.value = data.token
    }
    user.value = data?.user || null
    return user.value
  }

  async function register(payload) {
    const data = await api.post('/auth/register', payload, { auth: false })
    // New accounts are "waiting"; a token may or may not be issued before approval.
    if (data?.token) {
      setToken(data.token)
      token.value = data.token
    }
    if (data?.user) user.value = data.user
    return data
  }

  async function me() {
    const data = await api.get('/auth/me')
    user.value = data
    return data
  }

  function logout() {
    clearToken()
    token.value = null
    user.value = null
  }

  function forceLogout() {
    logout()
  }

  async function demoLogin(demoRole) {
    const account = DEMO_ACCOUNTS[demoRole]
    if (!account) throw new Error('Unknown demo account')
    return login(account.username, account.password)
  }

  return {
    token,
    user,
    isAuthenticated,
    role,
    status,
    isConfirmed,
    isAdmin,
    isSupplier,
    isCustomer,
    displayName,
    login,
    register,
    me,
    logout,
    forceLogout,
    demoLogin,
    DEMO_ACCOUNTS
  }
})
