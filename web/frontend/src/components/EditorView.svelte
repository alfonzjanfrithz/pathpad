<script>
  import Sidebar from './Sidebar.svelte';
  import Breadcrumbs from './Breadcrumbs.svelte';
  import Editor from './Editor.svelte';

  let { path = '' } = $props();

  // Update document title
  $effect(() => {
    document.title = path ? `${path} - Pathpad` : 'Pathpad';
  });
</script>

<div class="flex h-screen w-screen">
  <Sidebar {path} />

  <main class="flex-1 flex flex-col min-w-0 h-screen">
    <Breadcrumbs {path} />
    <Editor {path} />
  </main>
</div>
