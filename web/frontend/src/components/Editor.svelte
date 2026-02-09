<script>
  import { onMount, onDestroy, untrack } from 'svelte';
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
    const targetPath = path; // capture at call time
    try {
      const data = await getPad(targetPath);
      // Ignore stale responses if path changed during the fetch
      if (path !== targetPath) return;
      content = data.content || '';
      lastSavedContent = content;
      saveStatus.set('');
    } catch (err) {
      if (path !== targetPath) return;
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

  function flushSave(savePath) {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      saveTimeout = null;
    }
    const p = savePath !== undefined ? savePath : path;
    if (content !== lastSavedContent && p) {
      savePadBeacon(p, content, clientId);
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
    // loadPad() and setupSSE() are handled by the $effect below on initial run.
    window.addEventListener('force-save', onForceSave);

    return () => {
      flushSave();
      if (sseCleanup) sseCleanup();
      window.removeEventListener('force-save', onForceSave);
    };
  });

  // React to path changes only. flushSave() reads `content` ($state),
  // so we wrap it in untrack() to prevent content changes from
  // re-triggering this effect (which would cause loadPad on every keystroke).
  let prevPath = undefined;
  $effect(() => {
    const p = path; // sole tracked dependency
    if (p !== prevPath) {
      untrack(() => {
        if (prevPath !== undefined) {
          flushSave(prevPath);
        }
      });
      prevPath = p;
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
  class="w-full flex-1 border-none outline-none resize-none px-4 py-4 md:px-7 md:py-6 font-mono text-base md:text-lg leading-7 md:leading-8 text-gray-900 bg-white placeholder:text-gray-300"
></textarea>
