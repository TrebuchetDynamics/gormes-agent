import { defineConfig } from 'astro/config';

import { site } from './src/lib/site.js';

export default defineConfig({
  site,
  output: 'static',
});
