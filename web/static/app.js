(function () {
  'use strict';

  // === State ===
  var state = {
    currentPath: '',
    clientId: generateUUID(),
    sseConnection: null,
    saveTimeout: null,
    lastSavedContent: '',
    isSaving: false,
  };

  // === DOM Elements ===
  var landing = document.getElementById('landing');
  var padView = document.getElementById('pad-view');
  var editor = document.getElementById('editor');
  var breadcrumbs = document.getElementById('breadcrumbs');
  var childrenList = document.getElementById('children-list');
  var saveStatus = document.getElementById('save-status');
  var connectionStatus = document.getElementById('connection-status');
  var deleteBtn = document.getElementById('delete-btn');
  var newChildInput = document.getElementById('new-child-input');
  var newChildBtn = document.getElementById('new-child-btn');
  var sidebarToggle = document.getElementById('sidebar-toggle');
  var landingForm = document.getElementById('landing-form');
  var landingInput = document.getElementById('landing-input');
  var landingPrefix = document.getElementById('landing-prefix');

  // === Initialization ===
  function init() {
    // Set landing prefix to current host
    landingPrefix.textContent = window.location.host + '/';

    // Restore sidebar state from localStorage
    if (localStorage.getItem('sidebarCollapsed') === 'true') {
      padView.classList.add('collapsed');
    }

    // Event listeners
    editor.addEventListener('input', onEditorInput);
    deleteBtn.addEventListener('click', onDelete);
    newChildBtn.addEventListener('click', onNewChild);
    newChildInput.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') onNewChild();
    });
    sidebarToggle.addEventListener('click', toggleSidebar);
    landingForm.addEventListener('submit', onLandingGo);

    window.addEventListener('popstate', function () {
      loadPage(getPathFromURL());
    });

    // Load initial page
    loadPage(getPathFromURL());
  }

  // === View Switching ===
  function showLanding() {
    landing.classList.remove('hidden');
    padView.classList.add('hidden');
    document.title = 'Dontpad';
    // Disconnect SSE when on landing
    if (state.sseConnection) {
      state.sseConnection.close();
      state.sseConnection = null;
    }
  }

  function showEditor() {
    landing.classList.add('hidden');
    padView.classList.remove('hidden');
  }

  // === Navigation ===
  function getPathFromURL() {
    var path = window.location.pathname;
    if (path.startsWith('/')) path = path.substring(1);
    if (path.endsWith('/')) path = path.slice(0, -1);
    return path;
  }

  function navigateTo(path) {
    var url = '/' + path;
    if (window.location.pathname !== url) {
      window.history.pushState(null, '', url);
    }
    loadPage(path);
  }

  function loadPage(path) {
    // Flush any pending save before navigating away
    flushPendingSave();

    state.currentPath = path;

    if (!path) {
      showLanding();
      return;
    }

    showEditor();
    document.title = path + ' - Dontpad';
    renderBreadcrumbs(path);
    loadPad(path);
    loadChildren(path);
    connectSSE(path);
  }

  // === Landing ===
  function onLandingGo(e) {
    e.preventDefault();
    var name = landingInput.value.trim().toLowerCase().replace(/[^a-z0-9/_-]/g, '');
    if (!name) return;
    landingInput.value = '';
    navigateTo(name);
  }

  // === Sidebar Toggle ===
  function toggleSidebar() {
    padView.classList.toggle('collapsed');
    var isCollapsed = padView.classList.contains('collapsed');
    localStorage.setItem('sidebarCollapsed', isCollapsed);
  }

  // === Breadcrumbs ===
  function renderBreadcrumbs(path) {
    breadcrumbs.innerHTML = '';

    // Root link
    var rootLink = document.createElement('a');
    rootLink.href = '/';
    rootLink.textContent = 'root';
    rootLink.addEventListener('click', function (e) {
      e.preventDefault();
      navigateTo('');
    });
    breadcrumbs.appendChild(rootLink);

    if (!path) return;

    var segments = path.split('/');
    var accumulated = '';

    for (var i = 0; i < segments.length; i++) {
      // Separator
      var sep = document.createElement('span');
      sep.className = 'sep';
      sep.textContent = '/';
      breadcrumbs.appendChild(sep);

      accumulated += (i > 0 ? '/' : '') + segments[i];

      if (i < segments.length - 1) {
        // Clickable ancestor
        var link = document.createElement('a');
        link.href = '/' + accumulated;
        link.textContent = segments[i];
        (function (target) {
          link.addEventListener('click', function (e) {
            e.preventDefault();
            navigateTo(target);
          });
        })(accumulated);
        breadcrumbs.appendChild(link);
      } else {
        // Current segment — not clickable
        var cur = document.createElement('span');
        cur.className = 'current-seg';
        cur.textContent = segments[i];
        breadcrumbs.appendChild(cur);
      }
    }
  }

  // === Pad CRUD ===
  function loadPad(path) {
    fetch('/api/pad/content/' + path)
      .then(function (res) { return res.json(); })
      .then(function (data) {
        editor.value = data.content || '';
        state.lastSavedContent = editor.value;
        updateSaveStatus('', '');
      })
      .catch(function (err) {
        console.error('Failed to load pad:', err);
        updateSaveStatus('error', 'Load error');
      });
  }

  function savePad(path, content) {
    if (state.isSaving) return;
    state.isSaving = true;
    updateSaveStatus('saving', 'Saving...');

    var url = '/api/pad/content/' + path + '?client_id=' + encodeURIComponent(state.clientId);
    fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: content }),
    })
      .then(function (res) {
        if (!res.ok) throw new Error('Save failed: ' + res.status);
        return res.json();
      })
      .then(function (data) {
        state.lastSavedContent = data.content;
        state.isSaving = false;
        updateSaveStatus('saved', 'Saved');
        loadChildren(state.currentPath);
      })
      .catch(function (err) {
        console.error('Failed to save pad:', err);
        state.isSaving = false;
        updateSaveStatus('error', 'Error');
      });
  }

  function deletePad(path) {
    var url = '/api/pad/content/' + path + '?client_id=' + encodeURIComponent(state.clientId);
    fetch(url, { method: 'DELETE' })
      .then(function (res) { return res.json(); })
      .then(function () {
        navigateTo(parentPath(path));
      })
      .catch(function (err) {
        console.error('Failed to delete pad:', err);
      });
  }

  // === Auto-save ===
  function onEditorInput() {
    if (state.saveTimeout) {
      clearTimeout(state.saveTimeout);
    }
    updateSaveStatus('saving', 'Unsaved');

    state.saveTimeout = setTimeout(function () {
      state.saveTimeout = null;
      var content = editor.value;
      if (content !== state.lastSavedContent) {
        savePad(state.currentPath, content);
      } else {
        updateSaveStatus('saved', 'Saved');
      }
    }, 500);
  }

  // Flush pending save immediately (called before navigating away).
  function flushPendingSave() {
    if (state.saveTimeout) {
      clearTimeout(state.saveTimeout);
      state.saveTimeout = null;
      // Save synchronously before leaving the page
      var content = editor.value;
      var path = state.currentPath;
      if (path && content !== state.lastSavedContent) {
        // Fire-and-forget save via sendBeacon or fetch
        var url = '/api/pad/content/' + path + '?client_id=' + encodeURIComponent(state.clientId);
        var body = JSON.stringify({ content: content });
        if (navigator.sendBeacon) {
          navigator.sendBeacon(url + '&_method=PUT', new Blob([body], { type: 'application/json' }));
        }
        // Also do a regular fetch (sendBeacon doesn't support PUT, so we use fetch too)
        fetch(url, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: body,
          keepalive: true,
        }).catch(function () {});
        state.lastSavedContent = content;
      }
    }
  }

  // === SSE ===
  function connectSSE(path) {
    if (state.sseConnection) {
      state.sseConnection.close();
      state.sseConnection = null;
    }

    var url = '/api/pad/events/' + path + '?client_id=' + encodeURIComponent(state.clientId);
    var es = new EventSource(url);

    es.onopen = function () {
      connectionStatus.classList.add('connected');
      connectionStatus.title = 'Connected';
    };

    es.onmessage = function (e) {
      try {
        var event = JSON.parse(e.data);
        if (event.client_id === state.clientId) return;

        if (event.type === 'update') {
          editor.value = event.content;
          state.lastSavedContent = event.content;
          updateSaveStatus('saved', 'Updated');
        } else if (event.type === 'delete') {
          navigateTo(parentPath(state.currentPath));
        } else if (event.type === 'children_changed') {
          loadChildren(state.currentPath);
        }
      } catch (err) {
        console.error('Failed to parse SSE event:', err);
      }
    };

    es.onerror = function () {
      connectionStatus.classList.remove('connected');
      connectionStatus.title = 'Disconnected';
    };

    state.sseConnection = es;
  }

  // === Children ===
  function loadChildren(path) {
    fetch('/api/pad/children/' + path)
      .then(function (res) { return res.json(); })
      .then(function (data) {
        renderChildren(data.children || []);
      })
      .catch(function (err) {
        console.error('Failed to load children:', err);
      });
  }

  function renderChildren(children) {
    childrenList.innerHTML = '';
    children.forEach(function (child) {
      var a = document.createElement('a');
      a.href = '/' + child.path;
      var segments = child.path.split('/');
      a.textContent = segments[segments.length - 1];
      a.addEventListener('click', function (e) {
        e.preventDefault();
        navigateTo(child.path);
      });
      childrenList.appendChild(a);
    });
  }

  // === Delete ===
  function onDelete() {
    var name = state.currentPath || 'root';
    if (!confirm('Delete "' + name + '" and all its child pages?')) return;
    deletePad(state.currentPath);
  }

  // === New Child ===
  function onNewChild() {
    var name = newChildInput.value.trim().toLowerCase().replace(/[^a-z0-9_-]/g, '');
    if (!name) return;
    var childPath = state.currentPath ? state.currentPath + '/' + name : name;
    newChildInput.value = '';

    // Create the child pad in the DB immediately (with empty content)
    // so it appears in the parent's children list right away.
    var url = '/api/pad/content/' + childPath + '?client_id=' + encodeURIComponent(state.clientId);
    fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: '' }),
    })
      .then(function () {
        // Refresh children list on the current page before navigating
        loadChildren(state.currentPath);
        navigateTo(childPath);
      })
      .catch(function () {
        // Navigate anyway even if save failed
        navigateTo(childPath);
      });
  }

  // === UI Helpers ===
  function updateSaveStatus(className, text) {
    saveStatus.className = 'save-text' + (className ? ' ' + className : '');
    saveStatus.textContent = text || '';
  }

  function parentPath(path) {
    if (!path) return '';
    var idx = path.lastIndexOf('/');
    if (idx === -1) return '';
    return path.substring(0, idx);
  }

  function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
      var r = (Math.random() * 16) | 0;
      var v = c === 'x' ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }

  // === Start ===
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
