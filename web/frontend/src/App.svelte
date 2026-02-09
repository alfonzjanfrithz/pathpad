<script>
  import { onMount } from 'svelte';
  import { currentPath, paletteOpen, sidebarCollapsed } from './lib/state.js';
  import { getPathFromURL } from './lib/utils.js';
  import LandingPage from './components/LandingPage.svelte';
  import EditorView from './components/EditorView.svelte';
  import CommandPalette from './components/CommandPalette.svelte';

  let path = $state(getPathFromURL());

  function updatePath() {
    path = getPathFromURL();
    currentPath.set(path);
  }

  onMount(() => {
    updatePath();

    // Listen for custom navigation events
    const onNav = (e) => {
      path = e.detail.path;
      currentPath.set(path);
    };
    window.addEventListener('navigate', onNav);
    window.addEventListener('popstate', updatePath);

    // Global keyboard shortcuts
    const onKeydown = (e) => {
      // Ctrl+K — command palette
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        paletteOpen.update((v) => !v);
        return;
      }
      // Ctrl+\ — toggle sidebar
      if ((e.ctrlKey || e.metaKey) && e.key === '\\') {
        e.preventDefault();
        sidebarCollapsed.update((v) => !v);
        return;
      }
      // Ctrl+N — new child page (open palette with prefix)
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        paletteOpen.set(true);
        // Dispatch event with prefill
        setTimeout(() => {
          window.dispatchEvent(new CustomEvent('palette-prefill', {
            detail: { text: path ? path + '/' : '' }
          }));
        }, 50);
        return;
      }
      // Escape — close palette
      if (e.key === 'Escape') {
        if ($paletteOpen) {
          e.preventDefault();
          paletteOpen.set(false);
        }
        return;
      }
      // Ctrl+S — force save
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        window.dispatchEvent(new CustomEvent('force-save'));
        return;
      }
    };
    window.addEventListener('keydown', onKeydown);

    return () => {
      window.removeEventListener('navigate', onNav);
      window.removeEventListener('popstate', updatePath);
      window.removeEventListener('keydown', onKeydown);
    };
  });
</script>

{#if path === '' && getPathFromURL() === ''}
  <LandingPage />
{:else}
  <EditorView {path} />
{/if}

{#if $paletteOpen}
  <CommandPalette />
{/if}
