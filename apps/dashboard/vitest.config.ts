import { defineConfig } from "vitest/config";
import { resolve } from "path";

export default defineConfig({
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: ["src/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}"],
    css: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html", "lcov"],
      exclude: [
        "node_modules/",
        "src/test/",
        "**/*.spec.ts",
        "**/*.test.ts",
        "src/main.ts",
        "src/environments/",
      ],
      thresholds: {
        lines: 20,
        functions: 60,
        branches: 70,
        statements: 20,
      },
    },
  },
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src"),
      "@app": resolve(__dirname, "./src/app"),
      "@core": resolve(__dirname, "./src/app/core"),
      "@shared": resolve(__dirname, "./src/app/shared"),
      "@features": resolve(__dirname, "./src/app/features"),
    },
  },
});
