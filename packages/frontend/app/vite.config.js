import {defineConfig} from 'vite';
import tailwindcss from '@tailwindcss/vite'
import handlebars from 'vite-plugin-handlebars';
import {resolve} from 'path';

export default defineConfig({
  root: 'src', // Assuming your source files are in the 'src' directory
  build: {
    outDir: '../dist', // Output directory for the build
  },
  plugins: [
    tailwindcss(),
    handlebars({
      partialDirectory: resolve(__dirname, 'partials'),
      context: {
        title: 'Hello, world!',
      },
    }),
  ],
  server: {
    historyApiFallback: true, // This will route all requests to index.html
  },
  assetsInclude: [
  ]
});
