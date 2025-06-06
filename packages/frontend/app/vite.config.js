import {defineConfig, loadEnv} from 'vite';
import tailwindcss from '@tailwindcss/vite'
import { createHtmlPlugin } from 'vite-plugin-html';

export default defineConfig(({mode}) => {
  const env = loadEnv(mode, process.cwd(), '');

  console.log(`Environment: ${env.VITE_RECAPTCHA_SITE_KEY}`);

  return {
    root: 'src', // Assuming your source files are in the 'src' directory
    build: {
      outDir: '../dist', // Output directory for the build
    },
    plugins: [
      tailwindcss(),
      createHtmlPlugin({
        minify: true, // Disable minification during development for readability
        inject: {
          data: {
            recaptchaSiteKey: env.VITE_RECAPTCHA_SITE_KEY,
          },
        },
      }),
    ],
    server: {
      historyApiFallback: true, // This will route all requests to index.html
    },
    assetsInclude: [],
  }
});

