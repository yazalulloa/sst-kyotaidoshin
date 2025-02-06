import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  root: 'src', // Assuming your source files are in the 'src' directory
  build: {
    outDir: '../dist', // Output directory for the build
  },
  plugins: [
    tailwindcss(),
  ],
  server: {
    historyApiFallback: true, // This will route all requests to index.html
  },
});
