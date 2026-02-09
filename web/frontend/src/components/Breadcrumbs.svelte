<script>
  import { navigateTo } from '../lib/utils.js';
  import { mobileMenuOpen } from '../lib/state.js';

  let { path = '' } = $props();

  function segments() {
    if (!path) return [];
    return path.split('/');
  }

  function accumulatedPath(index) {
    return segments().slice(0, index + 1).join('/');
  }

  function handleClick(e, targetPath) {
    e.preventDefault();
    navigateTo(targetPath);
  }
</script>

<nav class="flex items-center px-4 md:px-7 py-3 md:py-3.5 text-base md:text-lg border-b border-gray-200 bg-gray-50/80 shrink-0 overflow-x-auto whitespace-nowrap gap-1.5">
  <!-- Mobile hamburger -->
  <button
    onclick={() => mobileMenuOpen.set(true)}
    class="md:hidden p-1.5 mr-1.5 text-gray-500 hover:text-gray-700 rounded-md text-xl leading-none cursor-pointer bg-transparent border-none"
    title="Open menu"
  >&#9776;</button>

  <button
    onclick={(e) => handleClick(e, '')}
    class="text-indigo-600 hover:text-indigo-800 hover:underline font-medium cursor-pointer bg-transparent border-none p-0 text-base md:text-lg"
  >root</button>

  {#each segments() as seg, i}
    <span class="text-gray-300 select-none mx-0.5 md:mx-1">/</span>
    {#if i < segments().length - 1}
      <button
        onclick={(e) => handleClick(e, accumulatedPath(i))}
        class="text-indigo-600 hover:text-indigo-800 hover:underline cursor-pointer bg-transparent border-none p-0 text-base md:text-lg"
      >{seg}</button>
    {:else}
      <span class="text-gray-900 font-semibold">{seg}</span>
    {/if}
  {/each}
</nav>
