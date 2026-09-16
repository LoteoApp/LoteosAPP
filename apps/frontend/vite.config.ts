import path from 'node:path'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { loadEnv } from 'vite'
import { defineConfig } from 'vitest/config'
import { contentSecurityPolicyPlugin } from './vite-plugins/content-security-policy'
import { DEFAULT_API_URL, DEFAULT_SUPABASE_URL, resolveUrl } from './src/shared/config/env-defaults'

export default defineConfig(({ mode, command }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')

  return {
    plugins: [
      react(),
      tailwindcss(),
      contentSecurityPolicyPlugin({
        apiUrl: resolveUrl(env.VITE_API_URL, DEFAULT_API_URL),
        supabaseUrl: resolveUrl(env.VITE_SUPABASE_URL, DEFAULT_SUPABASE_URL),
        isDev: command === 'serve',
      }),
    ],
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
  }
})
