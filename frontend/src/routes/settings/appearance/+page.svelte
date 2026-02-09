<script lang="ts">
  import {
    Moon,
    Sun,
    Monitor,
    Check,
    Globe,
    Eye,
    EyeOff,
    Clock,
    Calendar,
    Type,
    TerminalSquare,
  } from 'lucide-svelte';
  import { language, type Language } from '$lib/stores/docker';
  import { settingsText, type Theme } from '$lib/settings';
  import { browser } from '$app/environment';

  let st = $derived(settingsText[$language]);

  // Theme
  let theme = $state<Theme>('dark');

  // Display settings (persisted to localStorage)
  let showStopped = $state(true);
  let highlightUpdates = $state(true);
  let timeFormat = $state<'24h' | '12h'>('24h');
  let dateFormat = $state<string>('DD.MM.YYYY');
  let font = $state<string>('System UI');
  let fontSize = $state<string>('Normal');
  let gridFontSize = $state<string>('Normal');
  let terminalFont = $state<string>('System Monospace');

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

  function saveSetting(key: string, value: string | boolean) {
    localStorage.setItem(`dockerverse_appearance_${key}`, String(value));
  }

  function loadSettings() {
    showStopped = localStorage.getItem('dockerverse_appearance_showStopped') !== 'false';
    highlightUpdates = localStorage.getItem('dockerverse_appearance_highlightUpdates') !== 'false';
    timeFormat = (localStorage.getItem('dockerverse_appearance_timeFormat') as '24h' | '12h') || '24h';
    dateFormat = localStorage.getItem('dockerverse_appearance_dateFormat') || 'DD.MM.YYYY';
    font = localStorage.getItem('dockerverse_appearance_font') || 'System UI';
    fontSize = localStorage.getItem('dockerverse_appearance_fontSize') || 'Normal';
    gridFontSize = localStorage.getItem('dockerverse_appearance_gridFontSize') || 'Normal';
    terminalFont = localStorage.getItem('dockerverse_appearance_terminalFont') || 'System Monospace';
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

  // Load saved theme and settings on mount
  $effect(() => {
    const saved = (localStorage.getItem('dockerverse-theme') as Theme) || 'dark';
    theme = saved;
    applyTheme(saved);
    loadSettings();
  });

  const dateFormats = ['DD.MM.YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD'];
  const fonts = ['System UI', 'Inter', 'JetBrains Mono'];
  const sizes = [$language === 'es' ? 'Pequeno' : 'Small', 'Normal', $language === 'es' ? 'Grande' : 'Large'];
  const terminalFonts = ['System Monospace', 'JetBrains Mono'];
</script>

<div class="p-4 space-y-8">
  <!-- Two column layout -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">

    <!-- Left Column: Display Settings -->
    <div class="space-y-6">
      <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
        <Eye class="w-5 h-5 text-primary" />
        {st.displaySettings}
      </h3>

      <!-- Show stopped containers -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div>
          <p class="text-sm font-medium text-foreground">{st.showStopped}</p>
          <p class="text-xs text-foreground-muted">{st.showStoppedDesc}</p>
        </div>
        <button
          class="relative w-11 h-6 rounded-full transition-colors {showStopped ? 'bg-primary' : 'bg-background-tertiary'}"
          onclick={() => { showStopped = !showStopped; saveSetting('showStopped', showStopped); }}
        >
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {showStopped ? 'translate-x-5' : ''}"></span>
        </button>
      </div>

      <!-- Highlight updates -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div>
          <p class="text-sm font-medium text-foreground">{st.highlightUpdates}</p>
          <p class="text-xs text-foreground-muted">{st.highlightUpdatesDesc}</p>
        </div>
        <button
          class="relative w-11 h-6 rounded-full transition-colors {highlightUpdates ? 'bg-primary' : 'bg-background-tertiary'}"
          onclick={() => { highlightUpdates = !highlightUpdates; saveSetting('highlightUpdates', highlightUpdates); }}
        >
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {highlightUpdates ? 'translate-x-5' : ''}"></span>
        </button>
      </div>

      <!-- Time format -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div class="flex items-center gap-2 mb-2">
          <Clock class="w-4 h-4 text-foreground-muted" />
          <p class="text-sm font-medium text-foreground">{st.timeFormat}</p>
        </div>
        <div class="flex gap-2">
          {#each ['24h', '12h'] as fmt}
            <button
              class="px-4 py-1.5 text-sm rounded-lg transition-colors {timeFormat === fmt ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
              onclick={() => { timeFormat = fmt as '24h' | '12h'; saveSetting('timeFormat', fmt); }}
            >
              {fmt}
            </button>
          {/each}
        </div>
      </div>

      <!-- Date format -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div class="flex items-center gap-2 mb-2">
          <Calendar class="w-4 h-4 text-foreground-muted" />
          <p class="text-sm font-medium text-foreground">{st.dateFormat}</p>
        </div>
        <select
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          bind:value={dateFormat}
          onchange={() => saveSetting('dateFormat', dateFormat)}
        >
          {#each dateFormats as fmt}
            <option value={fmt}>{fmt}</option>
          {/each}
        </select>
      </div>
    </div>

    <!-- Right Column: Theme & Font Settings -->
    <div class="space-y-6">
      <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
        <Moon class="w-5 h-5 text-primary" />
        {st.themeSettings}
      </h3>

      <!-- Theme Selection -->
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

      <!-- Font -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div class="flex items-center gap-2 mb-2">
          <Type class="w-4 h-4 text-foreground-muted" />
          <p class="text-sm font-medium text-foreground">{st.font}</p>
        </div>
        <select
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          bind:value={font}
          onchange={() => saveSetting('font', font)}
        >
          {#each fonts as f}
            <option value={f}>{f}</option>
          {/each}
        </select>
      </div>

      <!-- Font Size -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <p class="text-sm font-medium text-foreground mb-2">{st.fontSize}</p>
        <div class="flex gap-2">
          {#each sizes as size}
            <button
              class="flex-1 px-3 py-1.5 text-sm rounded-lg transition-colors {fontSize === size ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
              onclick={() => { fontSize = size; saveSetting('fontSize', size); }}
            >
              {size}
            </button>
          {/each}
        </div>
      </div>

      <!-- Grid Font Size -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <p class="text-sm font-medium text-foreground mb-2">{st.gridFontSize}</p>
        <div class="flex gap-2">
          {#each sizes as size}
            <button
              class="flex-1 px-3 py-1.5 text-sm rounded-lg transition-colors {gridFontSize === size ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
              onclick={() => { gridFontSize = size; saveSetting('gridFontSize', size); }}
            >
              {size}
            </button>
          {/each}
        </div>
      </div>

      <!-- Terminal Font -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div class="flex items-center gap-2 mb-2">
          <TerminalSquare class="w-4 h-4 text-foreground-muted" />
          <p class="text-sm font-medium text-foreground">{st.terminalFont}</p>
        </div>
        <select
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          bind:value={terminalFont}
          onchange={() => saveSetting('terminalFont', terminalFont)}
        >
          {#each terminalFonts as tf}
            <option value={tf}>{tf}</option>
          {/each}
        </select>
      </div>
    </div>
  </div>

  <!-- Language Section (full width) -->
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
      <Globe class="w-5 h-5 text-primary" />
      {st.language}
    </h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
      {#each [{ id: 'es', flag: '\u{1F1EA}\u{1F1F8}', label: 'Español' }, { id: 'en', flag: '\u{1F1EC}\u{1F1E7}', label: 'English' }] as item}
        <button
          onclick={() => setLanguage(item.id as Language)}
          class="flex items-center gap-4 p-3 rounded-lg border-2 transition-all text-left
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
