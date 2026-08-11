import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    // Pure-logic unit tests only (no DOM components) — node is enough and
    // keeps the test run fast.
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
});
