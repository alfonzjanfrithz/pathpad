<script>
  import { onMount, onDestroy } from 'svelte';
  import { getPad, savePad, savePadBeacon } from '../lib/api.js';
  import { connectSSE } from '../lib/sse.js';
  import { clientId, connected, saveStatus } from '../lib/state.js';
  import { parentPath, navigateTo } from '../lib/utils.js';

  let { path = '' } = $props();

  let textareaEl;
  let content = $state('');
  let lastSavedContent = '';
  let saveTimeout = null;
  let isSaving = false;
  let sseCleanup = null;

  async function loadPad() {
    try {
      const data = await getPad(path);
      content = data.content || '';
      lastSavedContent = content;
      saveStatus.set('');
    } catch (err) {
      console.error('Failed to load pad:', err);
      saveStatus.set('error');
    }
  }

  async function doSave() {
    if (isSaving) return;
    if (content === lastSavedContent) {
      saveStatus.set('saved');
      return;
    }
    isSaving = true;
    saveStatus.set('saving');
    try {
      const data = await savePad(path, content, clientId);
      lastSavedContent = data.content;
      saveStatus.set('saved');
    } catch (err) {
      console.error('Failed to save:', err);
      saveStatus.set('error');
    } finally {
      isSaving = false;
    }
  }

  function handleInput() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveStatus.set('saving');
    saveTimeout = setTimeout(() => {
      saveTimeout = null;
      doSave();
    }, 500);
  }

  function flushSave() {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      saveTimeout = null;
    }
    if (content !== lastSavedContent && path) {
      savePadBeacon(path, content, clientId);
      lastSavedContent = content;
    }
  }

  function onForceSave() {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      saveTimeout = null;
    }
    doSave();
  }

  function setupSSE() {
    if (sseCleanup) sseCleanup();
    sseCleanup = connectSSE(path, clientId, {
      onUpdate(newContent) {
        content = newContent;
        lastSavedContent = newContent;
        saveStatus.set('saved');
      },
      onDelete() {
        navigateTo(parentPath(path));
      },
      onChildrenChanged() {
        window.dispatchEvent(new CustomEvent('children-changed'));
      },
      onConnect() {
        connected.set(true);
      },
      onDisconnect() {
        connected.set(false);
      },
    });
  }

  onMount(() => {
    loadPad();
    setupSSE();
    window.addEventListener('force-save', onForceSave);
    if (textareaEl) textareaEl.focus();

    return () => {
      flushSave();
      if (sseCleanup) sseCleanup();
      window.removeEventListener('force-save', onForceSave);
    };
  });

  $effect(() => {
    if (path !== undefined) {
      flushSave();
      loadPad();
      setupSSE();
      if (textareaEl) textareaEl.focus();
    }
  });

  onDestroy(() => {
    flushSave();
    if (sseCleanup) sseCleanup();
    if (saveTimeout) clearTimeout(saveTimeout);
  });
</script>

<textarea
  bind:this={textareaEl}
  bind:value={content}
  oninput={handleInput}
  placeholder="Start typing..."
  spellcheck="false"
  class="w-full flex-1 border-none outline-none resize-none px-7 py-6 font-mono text-lg leading-8 text-gray-900 bg-white placeholder:text-gray-300"
></textarea>
