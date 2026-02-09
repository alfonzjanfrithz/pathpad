/**
 * Utility functions for path manipulation and helpers.
 */

/**
 * Get the parent path of a given path.
 * '' -> '', 'a' -> '', 'a/b' -> 'a', 'a/b/c' -> 'a/b'
 */
export function parentPath(path) {
  if (!path) return '';
  const idx = path.lastIndexOf('/');
  return idx === -1 ? '' : path.substring(0, idx);
}

/**
 * Get the last segment of a path.
 * 'a/b/c' -> 'c', 'a' -> 'a'
 */
export function lastSegment(path) {
  if (!path) return '';
  const parts = path.split('/');
  return parts[parts.length - 1];
}

/**
 * Get the current path from the URL.
 */
export function getPathFromURL() {
  let path = window.location.pathname;
  if (path.startsWith('/')) path = path.substring(1);
  if (path.endsWith('/')) path = path.slice(0, -1);
  return path;
}

/**
 * Navigate to a path using History API.
 */
export function navigateTo(path) {
  const url = '/' + path;
  if (window.location.pathname !== url) {
    window.history.pushState(null, '', url);
  }
  // Dispatch a custom event so App.svelte can react
  window.dispatchEvent(new CustomEvent('navigate', { detail: { path } }));
}

/**
 * Simple fuzzy match — checks if query chars appear in order in the target.
 */
export function fuzzyMatch(query, target) {
  if (!query) return true;
  const q = query.toLowerCase();
  const t = target.toLowerCase();
  let qi = 0;
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}
