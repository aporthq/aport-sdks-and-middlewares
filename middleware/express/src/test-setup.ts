/**
 * Jest test setup - use local dev server unless in CI/prod
 */
if (!process.env.CI && process.env.NODE_ENV !== "production") {
  process.env.AGENT_PASSPORT_BASE_URL =
    process.env.AGENT_PASSPORT_BASE_URL || "http://localhost:8787";
}
