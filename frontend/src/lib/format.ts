export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let v = bytes;
  let u = -1;
  do {
    v /= 1024;
    u++;
  } while (v >= 1024 && u < units.length - 1);
  return `${v.toFixed(2)} ${units[u]}`;
}

// Maps backend category ids to i18n keys.
export const CATEGORY_KEYS: Record<string, string> = {
  "temporary-files": "cat.temporary-files",
  cache: "cat.cache",
  logs: "cat.logs",
  "crash-dumps": "cat.crash-dumps",
  backups: "cat.backups",
  "partial-downloads": "cat.partial-downloads",
  "build-artifacts": "cat.build-artifacts",
  "editor-temp": "cat.editor-temp",
  "other-junk": "cat.other-junk",
};
