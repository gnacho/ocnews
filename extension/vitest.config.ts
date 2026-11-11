import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    globals: true,
    environment: 'happy-dom',
    include: ['src/**/*.spec.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary'],
      include: ['src/api.ts', 'src/index.ts', 'src/views/NewsApp.vue'],
      exclude: ['src/**/*.spec.ts', 'src/shims.d.ts']
    }
  }
})
