<script>
  import { onMount, onDestroy } from 'svelte';
  import { getChildren, savePad, deletePad } from '../lib/api.js';
  import { clientId, sidebarCollapsed, mobileMenuOpen, connected, saveStatus } from '../lib/state.js';
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
      mobileMenuOpen.set(false);
      navigateTo(childPath);
    } catch {
      mobileMenuOpen.set(false);
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
      mobileMenuOpen.set(false);
      navigateTo(parent);
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  }

  function handleChildClick(e, childPath) {
    e.preventDefault();
    mobileMenuOpen.set(false);
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

<!-- Desktop sidebar (hidden on mobile) -->
<aside
  class="hidden md:flex h-screen bg-gray-50 border-r border-gray-200 flex-col transition-all duration-200 shrink-0 overflow-hidden"
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

<!-- Mobile drawer overlay -->
{#if $mobileMenuOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="md:hidden fixed inset-0 z-40 flex"
    onkeydown={() => {}}
  >
    <!-- Backdrop -->
    <div
      class="fixed inset-0 bg-black/30"
      onclick={() => mobileMenuOpen.set(false)}
      role="button"
      tabindex="-1"
      onkeydown={() => {}}
    ></div>

    <!-- Drawer panel -->
    <aside class="relative z-50 w-72 max-w-[80vw] h-full bg-gray-50 border-r border-gray-200 flex flex-col shadow-xl">
      <!-- Header -->
      <div class="p-4 flex items-center justify-between border-b border-gray-200 shrink-0">
        <span class="text-lg font-semibold text-gray-700">Pages</span>
        <button
          onclick={() => mobileMenuOpen.set(false)}
          class="p-2 text-gray-400 hover:text-gray-700 rounded-md text-2xl leading-none cursor-pointer bg-transparent border-none"
        >&times;</button>
      </div>

      <!-- Children list -->
      <div class="flex-1 overflow-y-auto px-3 py-2">
        {#if children.length === 0}
          <p class="text-base text-gray-400 italic px-2 py-3">No child pages</p>
        {:else}
          {#each children as child}
            <a
              href={'/' + child.path}
              onclick={(e) => handleChildClick(e, child.path)}
              class="block px-3 py-3 text-lg text-gray-700 active:bg-indigo-50 rounded-md transition-colors truncate"
            >{lastSegment(child.path)}</a>
          {/each}
        {/if}
      </div>

      <!-- Footer -->
      <div class="p-4 border-t border-gray-200 shrink-0 space-y-3">
        <div class="flex gap-2">
          <input
            type="text"
            bind:value={newChildName}
            onkeydown={handleChildKeydown}
            placeholder="new page..."
            autocomplete="off"
            spellcheck="false"
            class="flex-1 min-w-0 px-3 py-3 text-base border border-gray-200 rounded-md bg-white outline-none focus:border-indigo-400"
          />
          <button
            onclick={createChild}
            class="px-4 py-3 text-base font-bold text-white bg-indigo-600 rounded-md hover:bg-indigo-700 transition-colors cursor-pointer"
            title="Create child page"
          >+</button>
        </div>

        <div class="flex items-center gap-2 text-base">
          <span
            class="text-sm leading-none transition-colors"
            class:text-green-500={$connected}
            class:text-red-400={!$connected}
          >&#9679;</span>
          <span class={statusClass + ' flex-1'}>{statusText}</span>
          <button
            onclick={handleDelete}
            class="text-gray-300 hover:text-red-500 transition-colors cursor-pointer p-2 bg-transparent border-none text-lg"
            title="Delete this pad"
          >🗑</button>
        </div>
      </div>
    </aside>
  </div>
{/if}
