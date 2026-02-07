<script lang="ts">
  import { onMount } from "svelte";
  import { Search, X } from "lucide-svelte";
  import { searchContainers, type Container } from "$lib/api/docker";
  import { selectedContainer, showTerminal } from "$lib/stores/docker";

  let { onclose }: { onclose: () => void } = $props();

  let query = $state("");
  let results = $state<Container[]>([]);
  let loading = $state(false);
  let selectedIndex = $state(0);
  let inputRef: HTMLInputElement;

  onMount(() => {
    inputRef?.focus();
  });

  async function search() {
    if (!query.trim()) {
      results = [];
      return;
    }

    loading = true;
    try {
      results = await searchContainers(query);
      selectedIndex = 0;
    } catch (e) {
      console.error("Search error:", e);
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        selectedIndex = Math.min(selectedIndex + 1, results.length - 1);
        break;
      case "ArrowUp":
        e.preventDefault();
        selectedIndex = Math.max(selectedIndex - 1, 0);
        break;
      case "Enter":
        if (results[selectedIndex]) {
          selectContainer(results[selectedIndex]);
        }
        break;
      case "Escape":
        onclose();
        break;
    }
  }

  function selectContainer(container: Container) {
    selectedContainer.set(container);
    showTerminal.set(true);
    onclose();
  }

  // Track query changes and trigger search with debounce
  $effect(() => {
    const q = query; // Force tracking
    const timer = setTimeout(() => {
      if (q !== undefined) search();
    }, 200);
    return () => clearTimeout(timer);
  });
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/60 backdrop-blur-sm"
  role="dialog"
  aria-modal="true"
  onclick={(e) => e.target === e.currentTarget && onclose()}
  onkeydown={handleKeydown}
>
  <div
    class="w-full max-w-xl bg-background-secondary rounded-xl shadow-2xl border border-background-tertiary overflow-hidden animate-fade-in"
  >
    <!-- Search Input -->
    <div
      class="flex items-center gap-3 p-4 border-b border-background-tertiary"
    >
      <Search class="w-5 h-5 text-foreground-muted" />
      <input
        bind:this={inputRef}
        bind:value={query}
        type="text"
        placeholder="Buscar contenedores, imágenes, hosts..."
        class="flex-1 bg-transparent text-foreground placeholder:text-foreground-muted focus:outline-none"
      />
      {#if query}
        <button
          onclick={() => (query = "")}
          class="text-foreground-muted hover:text-foreground"
        >
          <X class="w-4 h-4" />
        </button>
      {/if}
      <kbd
        class="px-2 py-1 text-xs bg-background rounded border border-background-tertiary text-foreground-muted"
      >
        ESC
      </kbd>
    </div>

    <!-- Results -->
    <div class="max-h-80 overflow-y-auto">
      {#if loading}
        <div class="p-8 text-center text-foreground-muted">
          <div
            class="animate-spin w-6 h-6 border-2 border-primary border-t-transparent rounded-full mx-auto"
          ></div>
          <p class="mt-2 text-sm">Buscando...</p>
        </div>
      {:else if results.length === 0 && query}
        <div class="p-8 text-center text-foreground-muted">
          <p>No se encontraron contenedores</p>
        </div>
      {:else if results.length > 0}
        <ul class="py-2">
          {#each results as container, i}
            <li>
              <button
                class="w-full px-4 py-3 flex items-center gap-3 text-left transition-colors
								       {i === selectedIndex
                  ? 'bg-primary/10 text-foreground'
                  : 'hover:bg-background-tertiary/50'}"
                onclick={() => selectContainer(container)}
              >
                <span
                  class="status-dot {container.state === 'running'
                    ? 'status-dot-running'
                    : 'status-dot-stopped'}"
                ></span>
                <div class="flex-1 min-w-0">
                  <p class="font-medium truncate">{container.name}</p>
                  <p class="text-sm text-foreground-muted truncate">
                    {container.image}
                  </p>
                </div>
                <span
                  class="text-xs text-foreground-muted px-2 py-1 bg-background rounded"
                >
                  {container.hostName}
                </span>
              </button>
            </li>
          {/each}
        </ul>
      {:else}
        <div class="p-4 text-sm text-foreground-muted">
          <p class="font-medium mb-2">Sugerencias:</p>
          <ul class="space-y-1">
            <li>• Escribe el nombre de un contenedor</li>
            <li>• Busca por imagen (ej: "nginx")</li>
            <li>• Filtra por host (ej: "raspi1")</li>
          </ul>
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div
      class="px-4 py-3 border-t border-background-tertiary flex items-center gap-4 text-xs text-foreground-muted"
    >
      <span>↑↓ Navegar</span>
      <span>↵ Seleccionar</span>
      <span>ESC Cerrar</span>
    </div>
  </div>
</div>
