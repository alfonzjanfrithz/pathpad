<script>
  import { navigateTo } from '../lib/utils.js';

  let inputValue = $state('');

  function handleSubmit(e) {
    e.preventDefault();
    const name = inputValue.trim().toLowerCase().replace(/[^a-z0-9/_-]/g, '');
    if (!name) return;
    inputValue = '';
    navigateTo(name);
  }
</script>

<div class="flex items-center justify-center h-screen w-screen">
  <div class="text-center max-w-xl px-8">
    <h1 class="text-8xl font-bold tracking-widest text-gray-900 mb-5">PATHPAD</h1>
    <p class="text-2xl text-gray-500 mb-14">The simplest way to share text online</p>

    <form onsubmit={handleSubmit} class="flex items-center justify-center mb-6">
      <span class="px-5 py-3.5 text-lg text-gray-400 bg-gray-50 border border-gray-300 border-r-0 rounded-l-lg select-none font-mono">
        {window.location.host}/
      </span>
      <input
        type="text"
        bind:value={inputValue}
        placeholder="your-secret-page"
        autocomplete="off"
        spellcheck="false"
        class="px-5 py-3.5 text-lg border border-gray-300 outline-none focus:border-indigo-500 w-64 font-mono"
      />
      <button
        type="submit"
        class="px-7 py-3.5 text-lg font-semibold text-white bg-indigo-600 border border-indigo-600 rounded-r-lg hover:bg-indigo-700 transition-colors cursor-pointer"
      >
        Go!
      </button>
    </form>

    <p class="text-lg text-gray-400">No login required</p>

    <p class="mt-10 text-base text-gray-300">
      Press <kbd class="px-2 py-1 bg-gray-100 border border-gray-200 rounded text-gray-500 font-mono text-base">Ctrl+K</kbd> to open command palette
    </p>
  </div>
</div>
