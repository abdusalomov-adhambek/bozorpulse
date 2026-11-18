export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',

  devtools: { enabled: false },

  app: {
    head: {
      title: 'BozorPulse — Real market prices. More buyers.',
      htmlAttrs: { lang: 'en' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        {
          name: 'description',
          content:
            'BozorPulse is an AI B2B marketplace where suppliers publish offers and customers compare real market prices, find suppliers and create orders.'
        },
        { name: 'theme-color', content: '#4f46e5' }
      ]
    }
  },

  css: ['~/assets/main.css'],

  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080/api/v1'
    }
  },

  devServer: {
    port: 3001
  }
})
