import { defineConfig, devices } from '@playwright/test'

const PORT = 8099

// E2E tests run against the real Go binary serving the built UI (npm run build
// first — `npm run e2e` does both). The server state is in-memory, so it is
// fresh for every run; tests share it and run serially.
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      // Locally reuse the installed Chrome; CI uses Playwright's bundled Chromium.
      use: { ...devices['Desktop Chrome'], channel: process.env.CI ? undefined : 'chrome', viewport: { width: 1440, height: 900 } },
    },
  ],
  webServer: {
    command: 'go run ./cmd/server',
    cwd: '..',
    url: `http://localhost:${PORT}/healthz`,
    env: { ADDR: `:${PORT}`, UPLOAD_DIR: 'data/e2e-uploads', LOG_LEVEL: 'warn' },
    reuseExistingServer: false,
    timeout: 120_000,
  },
})
