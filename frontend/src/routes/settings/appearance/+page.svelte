<script lang="ts">
  import {
    Moon,
    Sun,
    Monitor,
    Check,
    Globe,
  } from 'lucide-svelte';
  import { language, type Language } from '$lib/stores/docker';
  import { settingsText, type Theme } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  // Theme
  let theme = $state<Theme>('dark');

  function setTheme(newTheme: Theme) {
    theme = newTheme;
    localStorage.setItem('dockerverse-theme', newTheme);
    applyTheme(newTheme);
  }

  function applyTheme(t: Theme) {
    const root = document.documentElement;
    let effectiveTheme = t;

    if (t === 'system') {
      effectiveTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    if (effectiveTheme === 'light') {
      root.classList.add('light');
      root.classList.remove('dark');
    } else {
      root.classList.remove('light');
      root.classList.add('dark');
    }
  }

  function setLanguage(lang: Language) {
    language.set(lang);
    localStorage.setItem('dockerverse-language', lang);
  }

  // Listen for system theme changes
  $effect(() => {
    if (theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      const handleChange = () => applyTheme('system');
      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }
  });

  // Load saved theme on mount
  $effect(() => {
    const saved = (localStorage.getItem('dockerverse-theme') as Theme) || 'dark';
    theme = saved;
    applyTheme(saved);
  });
</script>

<div class="p-4 space-y-8">
  <!-- Theme Section -->
  <div class="space-y-4">
    <p class="text-sm text-foreground-muted">{st.themeSelect}</p>
    <div class="grid grid-cols-3 gap-3">
      {#each [{ id: 'dark', icon: Moon, label: st.dark }, { id: 'light', icon: Sun, label: st.light }, { id: 'system', icon: Monitor, label: st.system }] as item}
        <button
          onclick={() => setTheme(item.id as Theme)}
          class="flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all
            {theme === item.id
            ? 'border-primary bg-primary/10'
            : 'border-border hover:border-foreground-muted'}"
        >
          <item.icon class="w-6 h-6 {theme === item.id ? 'text-primary' : 'text-foreground-muted'}" />
          <span class="text-sm {theme === item.id ? 'text-primary' : 'text-foreground'}">{item.label}</span>
          {#if theme === item.id}
            <Check class="w-4 h-4 text-primary" />
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <!-- Language Section -->
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
      <Globe class="w-5 h-5 text-primary" />
      {st.language}
    </h3>
    <div class="space-y-2">
      {#each [{ id: 'es', flag: '\u{1F1EA}\u{1F1F8}', label: 'Español' }, { id: 'en', flag: '\u{1F1EC}\u{1F1E7}', label: 'English' }] as item}
        <button
          onclick={() => setLanguage(item.id as Language)}
          class="w-full flex items-center gap-4 p-3 rounded-lg border-2 transition-all text-left
            {$language === item.id
            ? 'border-primary bg-primary/10'
            : 'border-transparent hover:bg-background-tertiary/50'}"
        >
          <span class="text-2xl">{item.flag}</span>
          <span class="flex-1 font-medium text-foreground">{item.label}</span>
          {#if $language === item.id}
            <Check class="w-5 h-5 text-primary" />
          {/if}
        </button>
      {/each}
    </div>
  </div>
</div>
