<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { ChevronRight } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { isAuthenticated } from '$lib/stores/auth';
  import { settingsText } from '$lib/settings';

  let { children } = $props();

  let st = $derived(settingsText[$language]);

  // Derive current page name from pathname
  let currentPage = $derived(() => {
    const path = $page.url.pathname;
    const segments = path.split('/').filter(Boolean);
    if (segments.length <= 1) return null; // /settings
    return segments[segments.length - 1]; // e.g., "profile", "security"
  });

  let pageTitle = $derived(() => {
    const pg = currentPage();
    if (!pg) return st.settings;
    const key = pg as keyof typeof st;
    return (st[key] as string) || st.settings;
  });

  // Auth guard - redirect to / if not authenticated
  $effect(() => {
    if (browser && !$isAuthenticated) {
      goto('/');
    }
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="max-w-2xl mx-auto">
  <!-- Header with breadcrumb -->
  <div class="flex items-center gap-2 mb-6">
    {#if currentPage()}
      <a
        href="/settings"
        class="flex items-center gap-2 text-foreground-muted hover:text-foreground transition-colors"
      >
        <ChevronRight class="w-5 h-5 rotate-180" />
        <span class="text-sm">{st.back}</span>
      </a>
      <span class="text-foreground-muted">/</span>
      <h2 class="text-lg font-semibold text-foreground">{pageTitle()}</h2>
    {:else}
      <h2 class="text-lg font-semibold text-foreground">{st.settings}</h2>
    {/if}
  </div>

  <!-- Page content -->
  {@render children()}
</div>
