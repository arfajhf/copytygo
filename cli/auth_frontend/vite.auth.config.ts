import { defineConfig, mergeConfig } from "vite";
import { resolve } from "node:path";
import baseConfig from "./vite.config";

// Keep your existing preset and index.html. The auth UI has its own TS entry.
export default defineConfig(async (environment) => {
  const base = typeof baseConfig === "function" ? await baseConfig(environment) : await baseConfig;
  return mergeConfig(base, {
    build: {
      rollupOptions: {
        input: { app: resolve("index.html"), auth: resolve("auth.html") },
      },
    },
    server: {
      proxy: { "/api": process.env.COPYTYGO_BACKEND_ORIGIN || "http://127.0.0.1:8080" },
      hmr: { clientPort: Number(process.env.COPYTYGO_BACKEND_PORT || 8080) },
    },
  });
});
