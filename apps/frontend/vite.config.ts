import path from 'node:path'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    watch: {
      usePolling: true,
      interval: 200,
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      reportsDirectory: './coverage',
      include: ['src/features/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}'],
      thresholds: {
        perFile: true,
        branches: 75,
        functions: 80,
        lines: 80,
        statements: 80,
      },
    },
  },
})
