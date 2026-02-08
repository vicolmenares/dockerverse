<script lang="ts">
  import { language } from '$lib/stores/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);
</script>

<div class="p-4 space-y-4">
  <div class="text-center mb-6">
    <span class="text-5xl">🐳</span>
    <h3 class="text-xl font-bold text-foreground mt-2">DockerVerse</h3>
    <p class="text-foreground-muted">Multi-Host Docker Management</p>
  </div>

  <div class="space-y-3">
    <div class="flex justify-between p-3 bg-background rounded-lg">
      <span class="text-foreground-muted">{st.version}</span>
      <span class="text-foreground font-medium">2.1.0</span>
    </div>
    <div class="flex justify-between p-3 bg-background rounded-lg">
      <span class="text-foreground-muted">{st.buildDate}</span>
      <span class="text-foreground font-medium">2026-02-08</span>
    </div>
    <div class="flex justify-between p-3 bg-background rounded-lg">
      <span class="text-foreground-muted">{st.license}</span>
      <span class="text-foreground font-medium">MIT</span>
    </div>
  </div>

  <div class="flex gap-2 pt-4">
    <button
      class="flex-1 py-2 px-4 bg-background text-foreground rounded-lg hover:bg-background-tertiary transition-colors text-sm"
    >
      {st.documentation}
    </button>
    <button
      class="flex-1 py-2 px-4 bg-background text-foreground rounded-lg hover:bg-background-tertiary transition-colors text-sm"
    >
      {st.reportBug}
    </button>
  </div>
</div>
