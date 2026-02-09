/**
 * SSE connection manager for real-time pad events.
 */

/**
 * Connect to SSE event stream for a pad path.
 * @param {string} path - pad path
 * @param {string} clientId - this client's unique ID
 * @param {object} handlers - { onUpdate, onDelete, onChildrenChanged, onConnect, onDisconnect }
 * @returns {function} cleanup function to close the connection
 */
export function connectSSE(path, clientId, handlers) {
  const url = `/api/pad/events/${path}?client_id=${encodeURIComponent(clientId)}`;
  const es = new EventSource(url);

  es.onopen = () => {
    handlers.onConnect?.();
  };

  es.onmessage = (e) => {
    try {
      const event = JSON.parse(e.data);

      // Skip self-echoed events
      if (event.client_id === clientId) return;

      switch (event.type) {
        case 'update':
          handlers.onUpdate?.(event.content);
          break;
        case 'delete':
          handlers.onDelete?.(event.path);
          break;
        case 'children_changed':
          handlers.onChildrenChanged?.();
          break;
      }
    } catch (err) {
      console.error('Failed to parse SSE event:', err);
    }
  };

  es.onerror = () => {
    handlers.onDisconnect?.();
  };

  return () => {
    es.close();
  };
}
