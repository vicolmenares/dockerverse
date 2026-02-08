<script lang="ts">
  import {
    Bell,
    Activity,
    Cpu,
    MemoryStick,
    Mail,
    Send,
    Settings2,
    RefreshCw,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { API_BASE } from '$lib/api/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  // App settings from backend
  let appSettings = $state({
    cpuThreshold: 80,
    memoryThreshold: 80,
    appriseUrl: 'https://apprise.nerdslabs.com',
    appriseKey: 'dockerverse',
    telegramEnabled: false,
    telegramUrl: '',
    emailEnabled: false,
    notifyOnStop: true,
    notifyOnStart: true,
    notifyOnHighCpu: true,
    notifyOnHighMem: true,
    notifyTags: [] as string[],
  });
  let testingNotification = $state(false);
  let testChannel = $state<'telegram' | 'email' | 'both'>('both');

  async function loadSettings() {
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        appSettings = { ...appSettings, ...data };
      }
    } catch (e) {
      console.error(e);
    }
  }

  async function saveSettings() {
    const token = localStorage.getItem('auth_access_token');
    try {
      await fetch(`${API_BASE}/api/settings`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(appSettings),
      });
    } catch (e) {
      console.error(e);
    }
  }

  async function testNotification(channel?: 'telegram' | 'email' | 'both') {
    testingNotification = true;
    const token = localStorage.getItem('auth_access_token');
    const selectedChannel = channel || testChannel;
    try {
      const res = await fetch(`${API_BASE}/api/notify/test`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          title: 'DockerVerse Test',
          body: `Test notification via ${selectedChannel.toUpperCase()} - If you see this, notifications are working!`,
          type: 'info',
          channel: selectedChannel,
        }),
      });

      const result = await res.json();

      if (result.success) {
        const parts = [];
        if (result.telegram === 'sent') parts.push('Telegram OK');
        if (result.email === 'sent') parts.push('Email OK');
        if (result.telegram === 'failed') parts.push('Telegram failed');
        if (result.email === 'failed') parts.push('Email failed');
        if (result.email === 'no_email') parts.push($language === 'es' ? 'Sin email configurado' : 'No email configured');

        alert(
          $language === 'es'
            ? `Resultado: ${parts.join(', ') || 'Enviado'}`
            : `Result: ${parts.join(', ') || 'Sent'}`
        );
      } else {
        const errors = result.errors?.join('\n') || 'Unknown error';
        alert(
          $language === 'es'
            ? `Parcialmente enviado:\n${errors}`
            : `Partially sent:\n${errors}`
        );
      }
    } catch (e) {
      console.error(e);
      alert($language === 'es' ? 'Error de conexión' : 'Connection error');
    }
    testingNotification = false;
  }

  // Load settings on mount
  $effect(() => {
    loadSettings();
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_label_has_associated_control -->
<div class="p-4 space-y-4">
  <!-- Threshold sliders -->
  <div class="p-4 bg-background rounded-lg border border-border space-y-4">
    <h4 class="font-medium text-foreground flex items-center gap-2">
      <Settings2 class="w-4 h-4" />
      {st.alertThresholds}
    </h4>
    <div class="space-y-3">
      <div>
        <div class="flex justify-between text-sm mb-1">
          <span class="text-foreground-muted flex items-center gap-1"><Cpu class="w-4 h-4" /> {st.cpuThreshold}</span>
          <span class="text-primary font-medium">{appSettings.cpuThreshold}%</span>
        </div>
        <input
          type="range"
          min="50"
          max="100"
          bind:value={appSettings.cpuThreshold}
          onchange={saveSettings}
          class="w-full h-2 bg-background-tertiary rounded-lg appearance-none cursor-pointer accent-primary"
        />
      </div>
      <div>
        <div class="flex justify-between text-sm mb-1">
          <span class="text-foreground-muted flex items-center gap-1"><MemoryStick class="w-4 h-4" /> {st.memoryThreshold}</span>
          <span class="text-primary font-medium">{appSettings.memoryThreshold}%</span>
        </div>
        <input
          type="range"
          min="50"
          max="100"
          bind:value={appSettings.memoryThreshold}
          onchange={saveSettings}
          class="w-full h-2 bg-background-tertiary rounded-lg appearance-none cursor-pointer accent-primary"
        />
      </div>
    </div>
  </div>

  <!-- Toggle switches -->
  <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
    <div class="flex items-center gap-3">
      <Activity class="w-5 h-5 text-foreground-muted" />
      <div>
        <p class="font-medium text-foreground">{st.containerStopped}</p>
        <p class="text-sm text-foreground-muted">{st.containerStoppedDesc}</p>
      </div>
    </div>
    <label class="relative inline-flex items-center cursor-pointer">
      <input type="checkbox" bind:checked={appSettings.notifyOnStop} onchange={saveSettings} class="sr-only peer" />
      <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
    </label>
  </div>

  <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
    <div class="flex items-center gap-3">
      <Activity class="w-5 h-5 text-foreground-muted" />
      <div>
        <p class="font-medium text-foreground">{st.containerStarted}</p>
        <p class="text-sm text-foreground-muted">{st.containerStartedDesc}</p>
      </div>
    </div>
    <label class="relative inline-flex items-center cursor-pointer">
      <input type="checkbox" bind:checked={appSettings.notifyOnStart} onchange={saveSettings} class="sr-only peer" />
      <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
    </label>
  </div>

  <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
    <div class="flex items-center gap-3">
      <Cpu class="w-5 h-5 text-foreground-muted" />
      <div>
        <p class="font-medium text-foreground">{st.highCpu}</p>
        <p class="text-sm text-foreground-muted">{st.highCpuDesc}</p>
      </div>
    </div>
    <label class="relative inline-flex items-center cursor-pointer">
      <input type="checkbox" bind:checked={appSettings.notifyOnHighCpu} onchange={saveSettings} class="sr-only peer" />
      <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
    </label>
  </div>

  <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
    <div class="flex items-center gap-3">
      <MemoryStick class="w-5 h-5 text-foreground-muted" />
      <div>
        <p class="font-medium text-foreground">{st.highMemory}</p>
        <p class="text-sm text-foreground-muted">{st.highMemoryDesc}</p>
      </div>
    </div>
    <label class="relative inline-flex items-center cursor-pointer">
      <input type="checkbox" bind:checked={appSettings.notifyOnHighMem} onchange={saveSettings} class="sr-only peer" />
      <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
    </label>
  </div>

  <!-- Notification Channels Section -->
  <div class="border-t border-border mt-4 pt-4">
    <h4 class="font-medium text-foreground flex items-center gap-2 mb-4">
      <Send class="w-4 h-4" />
      {st.notificationChannels}
    </h4>

    <!-- Email Notifications -->
    <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
      <div class="flex items-center gap-3">
        <Mail class="w-5 h-5 text-foreground-muted" />
        <div>
          <p class="font-medium text-foreground">{st.emailNotifications}</p>
          <p class="text-sm text-foreground-muted">{st.emailNotificationsDesc}</p>
        </div>
      </div>
      <label class="relative inline-flex items-center cursor-pointer">
        <input type="checkbox" bind:checked={appSettings.emailEnabled} onchange={saveSettings} class="sr-only peer" />
        <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
      </label>
    </div>

    <!-- Telegram Notifications -->
    <div class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50">
      <div class="flex items-center gap-3">
        <Send class="w-5 h-5 text-foreground-muted" />
        <div>
          <p class="font-medium text-foreground">{st.telegramNotifications}</p>
          <p class="text-sm text-foreground-muted">{st.telegramNotificationsDesc}</p>
        </div>
      </div>
      <label class="relative inline-flex items-center cursor-pointer">
        <input type="checkbox" bind:checked={appSettings.telegramEnabled} onchange={saveSettings} class="sr-only peer" />
        <div class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
      </label>
    </div>
  </div>

  <!-- Apprise Configuration -->
  <div class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4">
    <h4 class="font-medium text-foreground">{st.appriseServer}</h4>
    <div>
      <label class="block text-sm text-foreground-muted mb-1">{st.appriseUrl}</label>
      <input
        type="url"
        bind:value={appSettings.appriseUrl}
        onchange={saveSettings}
        placeholder="https://apprise.example.com"
        class="w-full px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
      />
    </div>
    <div>
      <label class="block text-sm text-foreground-muted mb-1">{st.appriseKey}</label>
      <input
        type="text"
        bind:value={appSettings.appriseKey}
        onchange={saveSettings}
        placeholder="dockerverse"
        class="w-full px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
      />
    </div>
    <p class="text-xs text-foreground-muted">{st.appriseHelp}</p>
  </div>

  <!-- Telegram Configuration -->
  {#if appSettings.telegramEnabled}
    <div class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4">
      <h4 class="font-medium text-foreground flex items-center gap-2">
        <Send class="w-4 h-4" />
        Telegram
      </h4>
      <div>
        <label class="block text-sm text-foreground-muted mb-1">{st.telegramUrl}</label>
        <input
          type="text"
          bind:value={appSettings.telegramUrl}
          onchange={saveSettings}
          placeholder={st.telegramUrlPlaceholder}
          class="w-full px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground font-mono text-sm"
        />
        <p class="text-xs text-foreground-muted mt-1">{st.telegramUrlHelp}</p>
      </div>
    </div>
  {/if}

  <!-- Test Channel Selection -->
  <div class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4">
    <h4 class="font-medium text-foreground">{st.testChannelLabel}</h4>
    <div class="grid grid-cols-3 gap-2">
      {#each [{ id: 'telegram', label: st.testTelegram, icon: Send }, { id: 'email', label: st.testEmail, icon: Mail }, { id: 'both', label: st.testBoth, icon: Bell }] as channel}
        <button
          onclick={() => (testChannel = channel.id as 'telegram' | 'email' | 'both')}
          class="flex items-center justify-center gap-2 py-2 px-3 rounded-lg border transition-all
            {testChannel === channel.id
            ? 'border-primary bg-primary/10 text-primary'
            : 'border-border text-foreground-muted hover:border-foreground-muted'}"
        >
          <channel.icon class="w-4 h-4" />
          <span class="text-sm">{channel.label}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- Test notification button -->
  <button
    onclick={() => testNotification()}
    disabled={testingNotification}
    class="w-full flex items-center justify-center gap-2 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
  >
    {#if testingNotification}
      <RefreshCw class="w-4 h-4 animate-spin" />
    {:else}
      <Send class="w-4 h-4" />
    {/if}
    {testingNotification ? st.sending : st.testNotification}
  </button>
</div>
