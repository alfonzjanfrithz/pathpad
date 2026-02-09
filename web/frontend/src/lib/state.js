/**
 * Shared application state using Svelte 5 runes-compatible stores.
 */

import { writable } from 'svelte/store';

/** Unique client ID for SSE event filtering. */
export const clientId = crypto.randomUUID?.() ||
  'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });

/** Current pad path (derived from URL). */
export const currentPath = writable('');

/** Whether the sidebar is collapsed. */
export const sidebarCollapsed = writable(
  localStorage.getItem('sidebarCollapsed') === 'true'
);

// Persist sidebar state
sidebarCollapsed.subscribe((v) => {
  localStorage.setItem('sidebarCollapsed', String(v));
});

/** Whether the command palette is open. */
export const paletteOpen = writable(false);

/** SSE connection status. */
export const connected = writable(false);

/** Save status: '', 'saving', 'saved', 'error' */
export const saveStatus = writable('');
