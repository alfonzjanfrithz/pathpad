<script>
  import { onMount, onDestroy } from 'svelte';
  import { getChildren, savePad, deletePad } from '../lib/api.js';
  import { clientId, sidebarCollapsed, connected, saveStatus } from '../lib/state.js';
  import { lastSegment, navigateTo } from '../lib/utils.js';

  let { path = '' } = $props();

  let children = $state([]);
  let newChildName = $state('');

  async function loadChildren() {
    try {
      const data = await getChildren(path);
      children = data.children || [];
    } catch (err) {
      console.error('Failed to load children:', err);
    }
  }

  async function createChild() {
    const name = newChildName.trim().toLowerCase().replace(/[^a-z0-9_-]/g, '');
    if (!name) return;
    const childPath = path ? path + '/' + name : name;
    newChildName = '';
    try {
      await savePad(childPath, '', clientId);
      await loadChildren();
      navigateTo(childPath);
    } catch {
      navigateTo(childPath);
    }
  }

  function handleChildKeydown(e) {
    if (e.key === 'Enter') createChild();
  }

  async function handleDelete() {
    const name = path || 'root';
    if (!confirm(`Delete "${name}" and all its child pages?`)) return;
    try {
      await deletePad(path, clientId);
      const parent = path.includes('/') ? path.substring(0, path.lastIndexOf('/')) : '';
      navigateTo(parent);
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  }

  function handleChildClick(e, childPath) {
    e.preventDefault();
    navigateTo(childPath);
  }

  function onChildrenChanged() {
    loadChildren();
  }

  function onPaletteDelete() {
    handleDelete();
  }

  onMount(() => {
    loadChildren();
    window.addEventListener('children-changed', onChildrenChanged);
    window.addEventListener('palette-delete', onPaletteDelete);
    return () => {
      window.removeEventListener('children-changed', onChildrenChanged);
      window.removeEventListener('palette-delete', onPaletteDelete);
    };
  });

  $effect(() => {
    if (path !== undefined) {
      loadChildren();
    }
  });

  let statusText = $derived(
    $saveStatus === 'saving' ? 'Saving...'
    : $saveStatus === 'saved' ? 'Saved'
    : $saveStatus === 'error' ? 'Error'
    : ''
  );

  let statusClass = $derived(
    $saveStatus === 'saving' ? 'text-amber-500'
    : $saveStatus === 'saved' ? 'text-green-600'
    : $saveStatus === 'error' ? 'text-red-500'
    : 'text-gray-400'
  );
</script>

<aside
  class="h-screen bg-gray-50 border-r border-gray-200 flex flex-col transition-all duration-200 shrink-0 overflow-hidden"
  class:w-64={!$sidebarCollapsed}
  class:min-w-64={!$sidebarCollapsed}
  class:w-12={$sidebarCollapsed}
  class:min-w-12={$sidebarCollapsed}
>
  <!-- Toggle button -->
  <div class="p-2.5 shrink-0">
    <button
      onclick={() => sidebarCollapsed.update(v => !v)}
      class="p-2.5 text-gray-400 hover:text-gray-700 hover:bg-gray-200/60 rounded-md transition-colors cursor-pointer text-2xl leading-none"
      title="Toggle sidebar (Ctrl+\)"
    >&#9776;</button>
  </div>

  {#if !$sidebarCollapsed}
    <!-- Children list -->
    <div class="flex-1 overflow-y-auto px-3">
      {#if children.length === 0}
        <p class="text-base text-gray-400 italic px-2 py-3">No child pages</p>
      {:else}
        {#each children as child}
          <a
            href={'/' + child.path}
            onclick={(e) => handleChildClick(e, child.path)}
            class="block px-3 py-2.5 text-lg text-gray-700 hover:text-indigo-700 hover:bg-indigo-50 rounded-md transition-colors truncate"
          >{lastSegment(child.path)}</a>
        {/each}
      {/if}
    </div>

    <!-- Footer -->
    <div class="p-3.5 border-t border-gray-200 shrink-0 space-y-3">
      <!-- New child input -->
      <div class="flex gap-2">
        <input
          type="text"
          bind:value={newChildName}
          onkeydown={handleChildKeydown}
          placeholder="new page..."
          autocomplete="off"
          spellcheck="false"
          class="flex-1 min-w-0 px-3 py-2.5 text-base border border-gray-200 rounded-md bg-white outline-none focus:border-indigo-400"
        />
        <button
          onclick={createChild}
          class="px-3.5 py-2.5 text-base font-bold text-white bg-indigo-600 rounded-md hover:bg-indigo-700 transition-colors cursor-pointer"
          title="Create child page"
        >+</button>
      </div>

      <!-- Status row -->
      <div class="flex items-center gap-2 text-base">
        <span
          class="text-sm leading-none transition-colors"
          class:text-green-500={$connected}
          class:text-red-400={!$connected}
          title={$connected ? 'Connected' : 'Disconnected'}
        >&#9679;</span>
        <span class={statusClass + ' flex-1'}>{statusText}</span>
        <button
          onclick={handleDelete}
          class="text-gray-300 hover:text-red-500 transition-colors cursor-pointer p-0 bg-transparent border-none text-lg"
          title="Delete this pad"
        >🗑</button>
      </div>
    </div>
  {/if}
</aside>
