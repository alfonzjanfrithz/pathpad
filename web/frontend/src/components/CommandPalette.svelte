<script>
  import { onMount, onDestroy } from 'svelte';
  import { getChildren } from '../lib/api.js';
  import { currentPath, paletteOpen, sidebarCollapsed } from '../lib/state.js';
  import { navigateTo, fuzzyMatch, parentPath } from '../lib/utils.js';

  let inputEl;
  let query = $state('');
  let selectedIndex = $state(0);
  let allPages = $state([]);

  const actions = [
    { id: 'parent', label: 'Go to parent', icon: '↑', action: () => navigateTo(parentPath($currentPath)) },
    { id: 'root', label: 'Go to root', icon: '⌂', action: () => navigateTo('') },
    { id: 'sidebar', label: 'Toggle sidebar', icon: '☰', action: () => sidebarCollapsed.update(v => !v) },
    { id: 'delete', label: 'Delete current page', icon: '🗑', action: () => {
      paletteOpen.set(false);
      setTimeout(() => window.dispatchEvent(new CustomEvent('palette-delete')), 100);
    }},
  ];

  let results = $derived.by(() => {
    const q = query.trim();

    const matchedPages = allPages
      .filter((p) => fuzzyMatch(q, p.path))
      .slice(0, 10)
      .map((p) => ({ type: 'page', label: '/' + p.path, path: p.path }));

    const matchedActions = q
      ? actions.filter((a) => fuzzyMatch(q, a.label)).map((a) => ({ type: 'action', ...a }))
      : actions.map((a) => ({ type: 'action', ...a }));

    const items = [...matchedPages, ...matchedActions];
    const normalized = q.toLowerCase().replace(/[^a-z0-9/_-]/g, '');
    if (normalized && !allPages.some((p) => p.path === normalized)) {
      items.unshift({
        type: 'create',
        label: `Create /${normalized}`,
        path: normalized,
      });
    }

    return items;
  });

  $effect(() => {
    if (results) selectedIndex = 0;
  });

  function handleKeydown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = Math.min(selectedIndex + 1, results.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = Math.max(selectedIndex - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      selectItem(results[selectedIndex]);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      paletteOpen.set(false);
    }
  }

  function selectItem(item) {
    if (!item) return;
    paletteOpen.set(false);
    if (item.type === 'page' || item.type === 'create') {
      navigateTo(item.path);
    } else if (item.type === 'action') {
      item.action();
    }
  }

  async function loadAllPages() {
    try {
      const gathered = [];
      const queue = [''];
      while (queue.length > 0) {
        const p = queue.shift();
        const data = await getChildren(p);
        for (const child of data.children || []) {
          gathered.push(child);
          queue.push(child.path);
        }
        if (gathered.length > 200) break;
      }
      allPages = gathered;
    } catch (err) {
      console.error('Failed to load pages for palette:', err);
    }
  }

  function onPrefill(e) {
    query = e.detail.text || '';
  }

  onMount(() => {
    loadAllPages();
    window.addEventListener('palette-prefill', onPrefill);
    if (inputEl) inputEl.focus();
    return () => {
      window.removeEventListener('palette-prefill', onPrefill);
    };
  });
</script>

<!-- Backdrop -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 bg-black/30 backdrop-blur-sm z-50 flex items-start justify-center pt-[15vh]"
  onmousedown={(e) => { if (e.target === e.currentTarget) paletteOpen.set(false); }}
  onkeydown={() => {}}
>
  <!-- Palette -->
  <div class="w-full max-w-2xl mx-4 bg-white rounded-xl shadow-2xl border border-gray-200 overflow-hidden">
    <!-- Input -->
    <div class="flex items-center px-6 border-b border-gray-100">
      <span class="text-gray-400 text-lg mr-3">⌘</span>
      <input
        bind:this={inputEl}
        bind:value={query}
        onkeydown={handleKeydown}
        placeholder="Type a page name, path, or action..."
        autocomplete="off"
        spellcheck="false"
        class="flex-1 py-5 text-lg outline-none bg-transparent placeholder:text-gray-400"
      />
    </div>

    <!-- Results -->
    {#if results.length > 0}
      <div class="max-h-96 overflow-y-auto py-2">
        {#each results as item, i}
          <button
            class="w-full flex items-center gap-4 px-6 py-3.5 text-lg text-left cursor-pointer transition-colors border-none {i === selectedIndex ? 'bg-indigo-600 text-white' : 'bg-transparent text-gray-700'}"
            onmouseenter={() => selectedIndex = i}
            onclick={() => selectItem(item)}
          >
            {#if item.type === 'create'}
              <span class="text-base w-7 text-center font-bold {i === selectedIndex ? 'text-indigo-200' : 'text-indigo-500'}">+</span>
              <span class="font-medium">{item.label}</span>
            {:else if item.type === 'page'}
              <span class="text-base w-7 text-center font-mono {i === selectedIndex ? 'text-indigo-200' : 'text-gray-400'}">/</span>
              <span class="font-mono">{item.label}</span>
            {:else}
              <span class="w-7 text-center">{item.icon}</span>
              <span>{item.label}</span>
            {/if}
          </button>
        {/each}
      </div>
    {:else}
      <div class="px-6 py-10 text-center text-lg text-gray-400">No results</div>
    {/if}

    <!-- Footer hints (desktop only) -->
    <div class="hidden md:flex px-6 py-3.5 border-t border-gray-100 text-base text-gray-400 gap-6">
      <span><kbd class="px-2 py-1 bg-gray-100 rounded text-gray-500 font-mono text-base">↑↓</kbd> navigate</span>
      <span><kbd class="px-2 py-1 bg-gray-100 rounded text-gray-500 font-mono text-base">↵</kbd> select</span>
      <span><kbd class="px-2 py-1 bg-gray-100 rounded text-gray-500 font-mono text-base">esc</kbd> close</span>
    </div>
  </div>
</div>
