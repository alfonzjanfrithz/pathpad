/**
 * API client for Pathpad backend.
 */

const BASE = '/api/pad';

/**
 * Get pad content by path. Always returns 200 (empty content for implicit pads).
 * @param {string} path
 * @returns {Promise<{path: string, content: string, updated_at: number, created_at: number}>}
 */
export async function getPad(path) {
  const res = await fetch(`${BASE}/content/${path}`);
  if (!res.ok) throw new Error(`Failed to get pad: ${res.status}`);
  return res.json();
}

/**
 * Save pad content (upsert).
 * @param {string} path
 * @param {string} content
 * @param {string} clientId
 * @returns {Promise<{path: string, content: string, updated_at: number, created_at: number}>}
 */
export async function savePad(path, content, clientId) {
  const url = `${BASE}/content/${path}?client_id=${encodeURIComponent(clientId)}`;
  const res = await fetch(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  });
  if (!res.ok) throw new Error(`Failed to save pad: ${res.status}`);
  return res.json();
}

/**
 * Delete pad and all descendants.
 * @param {string} path
 * @param {string} clientId
 * @returns {Promise<{deleted: number}>}
 */
export async function deletePad(path, clientId) {
  const url = `${BASE}/content/${path}?client_id=${encodeURIComponent(clientId)}`;
  const res = await fetch(url, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete pad: ${res.status}`);
  return res.json();
}

/**
 * Get direct children of a pad path.
 * @param {string} path
 * @returns {Promise<{children: Array<{path: string, updated_at: number}>}>}
 */
export async function getChildren(path) {
  const res = await fetch(`${BASE}/children/${path}`);
  if (!res.ok) throw new Error(`Failed to get children: ${res.status}`);
  return res.json();
}

/**
 * Fire-and-forget save using fetch with keepalive (for navigating away).
 * @param {string} path
 * @param {string} content
 * @param {string} clientId
 */
export function savePadBeacon(path, content, clientId) {
  const url = `${BASE}/content/${path}?client_id=${encodeURIComponent(clientId)}`;
  fetch(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
    keepalive: true,
  }).catch(() => {});
}
