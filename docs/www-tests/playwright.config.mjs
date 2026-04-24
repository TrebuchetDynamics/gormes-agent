import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  use: {
    baseURL: 'http://127.0.0.1:1313',
  },
  webServer: {
    command: 'hugo server -D --bind 127.0.0.1 --port 1313',
    port: 1313,
    cwd: '..',
    reuseExistingServer: !process.env.CI,
  },
});
