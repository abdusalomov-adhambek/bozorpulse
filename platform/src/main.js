import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/main.css'
import { t, setLocale, locale, LOCALES } from './i18n'

const app = createApp(App)

// Available in every template as t(...) / $t(...) without imports.
app.config.globalProperties.t = t
app.config.globalProperties.$t = t
app.config.globalProperties.setLocale = setLocale
app.config.globalProperties.locale = locale
app.config.globalProperties.LOCALES = LOCALES
setLocale(locale.value)

app.use(createPinia())
app.use(router)

app.mount('#app')
