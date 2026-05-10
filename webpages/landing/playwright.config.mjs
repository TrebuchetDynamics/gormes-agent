import { defineConfig, devices } from '@playwright/test';

const e2eHost = '127.0.0.1';
const e2ePort = Number(process.env.LANDING_E2E_PORT ?? 18080);
const e2eBaseURL = `http://${e2eHost}:${e2ePort}`;
const assertPortFree = `node -e "const net=require('node:net');const port=${e2ePort};const host='${e2eHost}';const server=net.createServer();server.once('error',err=>{console.error('LANDING_E2E_PORT '+port+' unavailable: '+err.message);process.exit(1)});server.listen(port,host,()=>server.close(()=>process.exit(0)))"`;

export default defineConfig({
  testDir: 'tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: e2eBaseURL,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: `${assertPortFree} && node ../scripts/with-compatible-node.mjs ./node_modules/.bin/astro dev --host ${e2eHost} --port ${e2ePort}`,
    url: e2eBaseURL,
    reuseExistingServer: false,
  },
});
