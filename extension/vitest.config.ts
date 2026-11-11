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
      include: ['src/**/*.ts', 'src/**/*.vue'],
      exclude: ['src/**/*.spec.ts', 'src/shims.d.ts'],
      // The view is still a single 2k line component with no tests (#82), so
      // the gate covers what is extracted and tested: the pure helpers and
      // the components that have their own suite.
      thresholds: {
        'src/lib/**': { statements: 95, branches: 90, functions: 95, lines: 95 },
        'src/components/**': { statements: 88, branches: 85, functions: 70, lines: 88 }
      }
    }
  }
})
