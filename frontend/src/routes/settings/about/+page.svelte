<script lang="ts">
  import {
    Server,
    Globe,
    Shield,
    Github,
    Bug,
    Heart,
    Code,
    Cpu,
    Container,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);
</script>

<div class="p-4 space-y-6">
  <!-- Header -->
  <div class="text-center space-y-3">
    <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 border border-primary/20">
      <span class="text-4xl">🐳</span>
    </div>
    <div>
      <h2 class="text-2xl font-bold text-foreground">DockerVerse</h2>
      <p class="text-foreground-muted text-sm">Multi-Host Docker Management</p>
    </div>
    <span class="inline-block px-3 py-1 text-xs font-medium bg-primary/10 text-primary rounded-full border border-primary/20">
      v2.4.0
    </span>
  </div>

  <!-- Info Cards -->
  <div class="grid grid-cols-3 gap-3">
    <div class="card p-3 text-center space-y-1">
      <Server class="w-5 h-5 text-primary mx-auto" />
      <p class="text-xs text-foreground-muted">{st.version}</p>
      <p class="text-sm font-semibold text-foreground">2.4.0</p>
    </div>
    <div class="card p-3 text-center space-y-1">
      <Globe class="w-5 h-5 text-primary mx-auto" />
      <p class="text-xs text-foreground-muted">{st.buildDate}</p>
      <p class="text-sm font-semibold text-foreground">2026-02-09</p>
    </div>
    <div class="card p-3 text-center space-y-1">
      <Shield class="w-5 h-5 text-primary mx-auto" />
      <p class="text-xs text-foreground-muted">{st.license}</p>
      <p class="text-sm font-semibold text-foreground">MIT</p>
    </div>
  </div>

  <!-- Tech Stack -->
  <div class="space-y-2">
    <h3 class="text-sm font-medium text-foreground-muted uppercase tracking-wider">
      {$language === 'es' ? 'Stack Tecnológico' : 'Tech Stack'}
    </h3>
    <div class="grid grid-cols-3 gap-3">
      <div class="flex items-center gap-2 p-2.5 bg-background rounded-lg border border-border">
        <Code class="w-4 h-4 text-blue-400 shrink-0" />
        <span class="text-sm text-foreground truncate">Go 1.23</span>
      </div>
      <div class="flex items-center gap-2 p-2.5 bg-background rounded-lg border border-border">
        <Cpu class="w-4 h-4 text-orange-400 shrink-0" />
        <span class="text-sm text-foreground truncate">Svelte 5</span>
      </div>
      <div class="flex items-center gap-2 p-2.5 bg-background rounded-lg border border-border">
        <Container class="w-4 h-4 text-green-400 shrink-0" />
        <span class="text-sm text-foreground truncate">Nginx</span>
      </div>
    </div>
  </div>

  <!-- Action Links -->
  <div class="flex gap-3">
    <a
      href="https://github.com/vhm-tools/dockerverse"
      target="_blank"
      rel="noopener noreferrer"
      class="flex-1 flex items-center justify-center gap-2 py-2.5 px-4 bg-background border border-border rounded-lg text-foreground hover:bg-background-tertiary transition-colors text-sm"
    >
      <Github class="w-4 h-4" />
      {st.documentation}
    </a>
    <a
      href="https://github.com/vhm-tools/dockerverse/issues/new"
      target="_blank"
      rel="noopener noreferrer"
      class="flex-1 flex items-center justify-center gap-2 py-2.5 px-4 bg-background border border-border rounded-lg text-foreground hover:bg-background-tertiary transition-colors text-sm"
    >
      <Bug class="w-4 h-4" />
      {st.reportBug}
    </a>
  </div>

  <!-- Footer -->
  <div class="text-center pt-2 border-t border-border">
    <p class="text-xs text-foreground-muted flex items-center justify-center gap-1">
      {$language === 'es' ? 'Hecho con' : 'Made with'}
      <Heart class="w-3 h-3 text-stopped inline" />
      {$language === 'es' ? 'por' : 'by'} Victor Heredia
    </p>
  </div>
</div>
