// Central API client for BozorPulse.
// All requests go to /api/v1 and are proxied by Vite to the backend.

const BASE_URL = '/api/v1'
const TOKEN_KEY = 'bp_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

function buildUrl(path, params) {
  let url = `${BASE_URL}${path}`
  if (params && typeof params === 'object') {
    const query = new URLSearchParams()
    Object.keys(params).forEach((key) => {
      const value = params[key]
      if (value !== undefined && value !== null && value !== '') {
        query.append(key, value)
      }
    })
    const qs = query.toString()
    if (qs) url += `?${qs}`
  }
  return url
}

async function request(path, options = {}) {
  const { method = 'GET', body, params, auth = true } = options
  const url = buildUrl(path, params)

  const headers = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const token = getToken()
  if (auth && token) headers.Authorization = `Bearer ${token}`

  const config = { method, headers }
  if (body !== undefined) config.body = JSON.stringify(body)

  let response
  try {
    response = await fetch(url, config)
  } catch (err) {
    const error = new Error('Cannot reach the server. Please try again.')
    error.code = 'NETWORK_ERROR'
    throw error
  }

  if (response.status === 401) {
    const hadToken = Boolean(token)
    clearToken()
    if (hadToken && typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('auth:unauthorized'))
    }
  }

  let payload = null
  const text = await response.text()
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = text
    }
  }

  if (!response.ok) {
    let message = `Request failed (${response.status})`
    if (payload && typeof payload === 'object') {
      message = payload.message || payload.error || payload.detail || message
    }
    const error = new Error(message)
    error.status = response.status
    error.payload = payload
    throw error
  }

  return payload
}

export function get(path, params, options = {}) {
  return request(path, { ...options, method: 'GET', params })
}

export function post(path, body, options = {}) {
  return request(path, { ...options, method: 'POST', body })
}

export function patch(path, body, options = {}) {
  return request(path, { ...options, method: 'PATCH', body })
}

export function del(path, options = {}) {
  return request(path, { ...options, method: 'DELETE' })
}

// AI assistant
export function aiChat(body) {
  return post('/ai/chat', body)
}
export function listConversations() {
  return get('/ai/conversations')
}
export function getConversation(id) {
  return get(`/ai/conversations/${encodeURIComponent(id)}`)
}
export function deleteConversation(id) {
  return del(`/ai/conversations/${encodeURIComponent(id)}`)
}

// Purchase-timing plans
export function listPlans() {
  return get('/plans')
}
export function getPlan(id) {
  return get(`/plans/${encodeURIComponent(id)}`)
}
export function createPlan(body) {
  return post('/plans', body)
}
export function updatePlan(id, body) {
  return patch(`/plans/${encodeURIComponent(id)}`, body)
}
export function deletePlan(id) {
  return del(`/plans/${encodeURIComponent(id)}`)
}
export function refreshPlan(id) {
  return post(`/plans/${encodeURIComponent(id)}/refresh`)
}
export function previewPlan(body) {
  return post('/plans/preview', body)
}

export default {
  get, post, patch, del, getToken, setToken, clearToken,
  aiChat, listConversations, getConversation, deleteConversation,
  listPlans, getPlan, createPlan, updatePlan, deletePlan, refreshPlan, previewPlan,
}
