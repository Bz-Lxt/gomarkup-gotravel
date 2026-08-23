import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: '.',
  testMatch: 'e2e_flow.spec.ts',
  use: {
    baseURL: process.env.E2E_BASE || 'http://frontend',
    locale: 'zh-CN',
  },
  retries: 0,
  timeout: 60_000,
})
