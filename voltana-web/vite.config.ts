import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";
import { componentTagger } from "lovable-tagger";

// Proxy the API routes to the Go stack (via nginx :80) so dev/preview run
// same-origin — the httpOnly refresh cookie + credentials:"include" work, and
// no CORS is needed. In production nginx serves the built app + the API together.
const apiProxy = {
  "/auth": { target: "http://localhost:80", changeOrigin: true, cookieDomainRewrite: "" },
  "/v1": { target: "http://localhost:80", changeOrigin: true, cookieDomainRewrite: "" },
  "/health": { target: "http://localhost:80", changeOrigin: true },
};

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: "::",
    port: 8080,
    proxy: apiProxy,
  },
  preview: {
    host: "::",
    port: 4173,
    proxy: apiProxy,
  },
  plugins: [react(), mode === "development" && componentTagger()].filter(Boolean),
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
}));
