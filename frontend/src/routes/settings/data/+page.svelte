<script lang="ts">
  import {
    Trash2,
    Download,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  function clearCache() {
    localStorage.clear();
    sessionStorage.clear();
    alert(st.cacheCleared);
  }
</script>

<div class="p-4 space-y-2">
  <button
    onclick={clearCache}
    class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 text-left"
  >
    <Trash2 class="w-5 h-5 text-foreground-muted" />
    <div class="flex-1">
      <p class="font-medium text-foreground">{st.clearCache}</p>
      <p class="text-sm text-foreground-muted">{st.clearCacheDesc}</p>
    </div>
  </button>
  <button
    class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 text-left"
  >
    <Download class="w-5 h-5 text-foreground-muted" />
    <div class="flex-1">
      <p class="font-medium text-foreground">{st.exportData}</p>
      <p class="text-sm text-foreground-muted">{st.exportDataDesc}</p>
    </div>
  </button>

  <div class="border-t border-background-tertiary my-4"></div>

  <button
    class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-stopped/10 text-left"
  >
    <Trash2 class="w-5 h-5 text-stopped" />
    <div class="flex-1">
      <p class="font-medium text-stopped">{st.deleteAccount}</p>
      <p class="text-sm text-foreground-muted">{st.deleteAccountDesc}</p>
    </div>
  </button>
</div>
