import { describe, expect, it } from "vitest";
import {
  SIDEBAR_COLLAPSED_KEY,
  effectiveCollapsed,
  loadSidebarCollapsed,
  persistSidebarCollapsed,
  toggleUserCollapsed,
} from "./sidebar";

function fakeStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map<string, string>(Object.entries(initial));
  return {
    getItem: (k) => map.get(k) ?? null,
    setItem: (k, v) => void map.set(k, v),
    removeItem: (k) => void map.delete(k),
    clear: () => map.clear(),
    key: (i) => [...map.keys()][i] ?? null,
    get length() {
      return map.size;
    },
  } as Storage;
}

describe("effectiveCollapsed", () => {
  it("expanded user state ignores hover entirely", () => {
    expect(effectiveCollapsed(false, false)).toBe(false);
    expect(effectiveCollapsed(false, true)).toBe(false);
  });

  it("collapsed user state collapses when not hovered", () => {
    expect(effectiveCollapsed(true, false)).toBe(true);
  });

  it("collapsed user state temporarily expands on hover", () => {
    expect(effectiveCollapsed(true, true)).toBe(false);
  });
});

describe("toggleUserCollapsed", () => {
  it("flips the persisted preference", () => {
    expect(toggleUserCollapsed(false)).toBe(true);
    expect(toggleUserCollapsed(true)).toBe(false);
  });
});

describe("persistence", () => {
  it("loads '1' as collapsed and '0'/missing as expanded", () => {
    expect(loadSidebarCollapsed(fakeStorage({ [SIDEBAR_COLLAPSED_KEY]: "1" }))).toBe(true);
    expect(loadSidebarCollapsed(fakeStorage({ [SIDEBAR_COLLAPSED_KEY]: "0" }))).toBe(false);
    expect(loadSidebarCollapsed(fakeStorage({}))).toBe(false);
  });

  it("round-trips the persisted preference", () => {
    const storage = fakeStorage();
    persistSidebarCollapsed(storage, true);
    expect(loadSidebarCollapsed(storage)).toBe(true);
    persistSidebarCollapsed(storage, false);
    expect(loadSidebarCollapsed(storage)).toBe(false);
  });
});
