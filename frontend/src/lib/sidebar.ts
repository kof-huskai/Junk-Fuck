// Pure state model for the collapsible Sidebar. The transition rules live
// outside React so they are trivially testable and the component never
// juggles contradictory booleans.
//
// Model (single user preference + transient hover):
//   userCollapsed - the persisted user pin. true = user pinned it closed.
//   hovered       - transient pointer state; ONLY meaningful while
//                   userCollapsed is true (temporary expansion).
//
//   if (!userCollapsed)            width = expanded
//   if (userCollapsed && hovered)  width = expanded   (temporary)
//   if (userCollapsed && !hovered) width = collapsed
export const SIDEBAR_COLLAPSED_KEY = "jf.sidebarCollapsed";

/** Effective collapsed state: collapsed only when the user pinned it closed
 *  AND the pointer is not temporarily expanding it. A user-pinned *expanded*
 *  Sidebar never collapses on hover. */
export function effectiveCollapsed(userCollapsed: boolean, hovered: boolean): boolean {
  return userCollapsed && !hovered;
}

/** Invert the persisted user preference (clicking the pin control). */
export function toggleUserCollapsed(userCollapsed: boolean): boolean {
  return !userCollapsed;
}

/** Load the persisted preference. Defaults to expanded (false). */
export function loadSidebarCollapsed(storage: Pick<Storage, "getItem">): boolean {
  return storage.getItem(SIDEBAR_COLLAPSED_KEY) === "1";
}

/** Persist the user preference only — never the transient hover state. */
export function persistSidebarCollapsed(storage: Pick<Storage, "setItem">, collapsed: boolean): void {
  storage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? "1" : "0");
}
