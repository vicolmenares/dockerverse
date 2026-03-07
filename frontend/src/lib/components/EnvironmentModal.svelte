<script module lang="ts">
  export interface EnvironmentData {
    id: string;
    name: string;
    connectionType: string;   // "socket" | "tcp" | "tcp+tls"
    socketPath: string;
    host: string;
    port: number;
    protocol: string;
    tlsCa: string;
    tlsCert: string;
    tlsKey: string;
    tlsSkipVerify: boolean;
    labels: string[];
    publicIp: string;
    timezone: string;
    // Updates
    autoUpdate: boolean;
    updateSchedule: string;
    imagePrune: boolean;
    imagePruneMode: string;   // "dangling" | "all"
    imagePruneCron: string;
    // Monitoring
    eventTracking: boolean;
    vulnScanning: boolean;
    collectMetrics: boolean;
    highlightChanges: boolean;
    // Advanced
    diskWarningEnabled: boolean;
    diskWarningMode: string;   // "percentage" | "absolute"
    diskWarningThreshold: number;
  }
</script>

<script lang="ts">
  import {
    X, Server, RefreshCw, Activity, Shield, Settings,
    Globe, Unplug, Loader2, Check, AlertCircle, Plus, ChevronDown
  } from "lucide-svelte";
  import { API_BASE, getAuthHeaders } from "$lib/api/docker";

  const TIMEZONES = [
    "UTC", "America/New_York", "America/Chicago", "America/Denver",
    "America/Los_Angeles", "America/Sao_Paulo", "Europe/London",
    "Europe/Paris", "Europe/Berlin", "Europe/Madrid", "Europe/Rome",
    "Europe/Amsterdam", "Europe/Moscow", "Asia/Tokyo", "Asia/Shanghai",
    "Asia/Singapore", "Asia/Dubai", "Asia/Kolkata", "Australia/Sydney",
    "Pacific/Auckland"
  ];

  let {
    environment = null,
    onclose,
    onsave,
  }: {
    environment?: EnvironmentData | null;
    onclose: () => void;
    onsave: (env: EnvironmentData) => void;
  } = $props();

  let activeTab = $state<"general" | "updates" | "monitoring" | "advanced">("general");
  let testing = $state(false);
  let testResult = $state<{ success: boolean; error?: string; info?: { serverVersion: string; containers: number } } | null>(null);
  let detectingSocket = $state(false);
  let detectedSockets = $state<{ path: string; name: string }[]>([]);
  let showSocketList = $state(false);
  let labelInput = $state("");
  let saving = $state(false);

  const emptyForm: EnvironmentData = {
    id: "",
    name: "",
    connectionType: "socket",
    socketPath: "/var/run/docker.sock",
    host: "",
    port: 2375,
    protocol: "http",
    tlsCa: "",
    tlsCert: "",
    tlsKey: "",
    tlsSkipVerify: false,
    labels: [],
    publicIp: "",
    timezone: "UTC",
    autoUpdate: false,
    updateSchedule: "0 3 * * *",
    imagePrune: false,
    imagePruneMode: "dangling",
    imagePruneCron: "0 4 * * *",
    eventTracking: true,
    vulnScanning: false,
    collectMetrics: false,
    highlightChanges: false,
    diskWarningEnabled: false,
    diskWarningMode: "percentage",
    diskWarningThreshold: 85,
  };

  let form = $state<EnvironmentData>({ ...emptyForm });
  let isEditing = $derived(environment !== null);

  $effect(() => {
    if (environment) {
      // Merge with defaults to ensure all new fields exist
      form = {
        ...emptyForm,
        ...environment,
        // Normalize labels
        labels: Array.isArray(environment.labels)
          ? environment.labels
          : environment.labels
            ? (environment.labels as unknown as string).split(",").map((l: string) => l.trim()).filter(Boolean)
            : [],
      };
    } else {
      form = { ...emptyForm };
    }
    testResult = null;
    activeTab = "general";
  });

  function handleConnectionTypeChange(type: string) {
    form.connectionType = type;
    if (type === "socket") {
      form.socketPath = form.socketPath || "/var/run/docker.sock";
    } else if (type === "tcp") {
      form.port = form.port || 2375;
    } else if (type === "tcp+tls") {
      form.port = form.port || 2376;
    }
    testResult = null;
  }

  function addLabel() {
    const label = labelInput.trim();
    if (label && !form.labels.includes(label)) {
      form.labels = [...form.labels, label];
    }
    labelInput = "";
  }

  function removeLabel(label: string) {
    form.labels = form.labels.filter(l => l !== label);
  }

  function handleLabelKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      addLabel();
    }
  }

  async function testConnection() {
    testing = true;
    testResult = null;
    try {
      const res = await fetch(`${API_BASE}/api/environments/test`, {
        method: "POST",
        headers: getAuthHeaders(),
        body: JSON.stringify({
          connectionType: form.connectionType,
          socketPath: form.socketPath,
          host: form.host,
          port: form.port,
          protocol: form.protocol,
          tlsCa: form.tlsCa,
          tlsCert: form.tlsCert,
          tlsKey: form.tlsKey,
          tlsSkipVerify: form.tlsSkipVerify,
        }),
      });
      testResult = await res.json();
    } catch {
      testResult = { success: false, error: "Request failed" };
    }
    testing = false;
  }

  async function detectSockets() {
    detectingSocket = true;
    try {
      const res = await fetch(`${API_BASE}/api/environments/detect-sockets`, {
        headers: getAuthHeaders(),
      });
      const data = await res.json();
      detectedSockets = data.sockets || [];
      showSocketList = detectedSockets.length > 0;
    } catch {
      detectedSockets = [];
    }
    detectingSocket = false;
  }

  function selectSocket(path: string) {
    form.socketPath = path;
    showSocketList = false;
  }

  async function handleSave() {
    if (!form.id || !form.name) return;
    saving = true;
    onsave({ ...form });
    saving = false;
  }

  const tabs = [
    { id: "general" as const, label: "General", icon: Server },
    { id: "updates" as const, label: "Updates", icon: RefreshCw },
    { id: "monitoring" as const, label: "Monitoring", icon: Activity },
    { id: "advanced" as const, label: "Advanced", icon: Settings },
  ];
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 bg-black/60 z-[100] flex items-center justify-center p-4"
  onclick={onclose}
>
  <div
    class="bg-background-secondary border border-border rounded-xl shadow-2xl w-full max-w-2xl overflow-hidden flex flex-col max-h-[90vh]"
    onclick={(e) => e.stopPropagation()}
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
      <h3 class="font-semibold text-foreground">
        {isEditing ? "Edit Environment" : "Add Environment"}
      </h3>
      <button class="btn-icon hover:bg-background-tertiary" onclick={onclose}>
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-border px-5 shrink-0">
      {#each tabs as tab}
        {@const Icon = tab.icon}
        <button
          class="flex items-center gap-1.5 px-4 py-2.5 text-sm border-b-2 transition-colors -mb-px {activeTab === tab.id ? 'border-primary text-primary' : 'border-transparent text-foreground-muted hover:text-foreground'}"
          onclick={() => (activeTab = tab.id)}
        >
          <Icon class="w-4 h-4" />
          {tab.label}
        </button>
      {/each}
    </div>

    <!-- Content -->
    <div class="px-5 py-4 space-y-4 overflow-y-auto flex-1">

      {#if activeTab === "general"}
        <!-- ID -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1" for="env-id">ID</label>
          <input
            id="env-id"
            type="text"
            bind:value={form.id}
            disabled={isEditing}
            placeholder="raspi1"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none disabled:opacity-50"
          />
          <p class="text-xs text-foreground-muted mt-1">Unique machine-readable identifier</p>
        </div>

        <!-- Name -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1" for="env-name">Name</label>
          <input
            id="env-name"
            type="text"
            bind:value={form.name}
            placeholder="Raspberry Pi Main"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Labels tag input -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">Labels</label>
          <div class="flex flex-wrap gap-1.5 mb-2">
            {#each form.labels as label}
              <button
                type="button"
                class="flex items-center gap-1 text-xs px-2 py-0.5 bg-primary/10 text-primary rounded-full hover:bg-primary/20 transition-colors"
                onclick={() => removeLabel(label)}
              >
                {label}
                <X class="w-3 h-3" />
              </button>
            {/each}
          </div>
          <div class="flex gap-2">
            <input
              type="text"
              bind:value={labelInput}
              onkeydown={handleLabelKeydown}
              placeholder="Add label... (Enter to add)"
              class="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
            />
            <button
              type="button"
              onclick={addLabel}
              class="px-3 py-2 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors"
            >
              <Plus class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- Connection Type -->
        <div>
          <p class="block text-sm font-medium text-foreground mb-2">Connection type</p>
          <div class="flex gap-2">
            {#each [{ id: "socket", label: "Socket", icon: Unplug }, { id: "tcp", label: "TCP", icon: Globe }, { id: "tcp+tls", label: "TCP+TLS", icon: Shield }] as ct}
              {@const Icon = ct.icon}
              <button
                type="button"
                class="flex-1 flex items-center justify-center gap-1.5 p-2.5 rounded-lg border-2 text-sm transition-all {form.connectionType === ct.id ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground-muted hover:border-foreground-muted'}"
                onclick={() => handleConnectionTypeChange(ct.id)}
              >
                <Icon class="w-4 h-4" />
                {ct.label}
              </button>
            {/each}
          </div>
        </div>

        <!-- Socket Path (shown when socket) -->
        {#if form.connectionType === "socket" || !form.connectionType}
          <div>
            <label class="block text-sm font-medium text-foreground mb-1" for="env-socket">Socket path</label>
            <div class="flex gap-2">
              <input
                id="env-socket"
                type="text"
                bind:value={form.socketPath}
                placeholder="/var/run/docker.sock"
                class="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
              />
              <button
                type="button"
                class="flex items-center gap-1.5 px-3 py-2 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors disabled:opacity-50"
                onclick={detectSockets}
                disabled={detectingSocket}
              >
                {#if detectingSocket}
                  <Loader2 class="w-3.5 h-3.5 animate-spin" />
                {:else}
                  <ChevronDown class="w-3.5 h-3.5" />
                {/if}
                Detect
              </button>
            </div>
            {#if showSocketList && detectedSockets.length > 0}
              <div class="mt-1 border border-border rounded-lg bg-background overflow-hidden">
                {#each detectedSockets as sock}
                  <button
                    type="button"
                    class="w-full flex items-center justify-between px-3 py-2 text-sm hover:bg-background-tertiary transition-colors text-left"
                    onclick={() => selectSocket(sock.path)}
                  >
                    <span class="font-mono text-foreground text-xs">{sock.path}</span>
                    <span class="text-foreground-muted text-xs">{sock.name}</span>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}

        <!-- Host + Port (shown when tcp/tcp+tls) -->
        {#if form.connectionType === "tcp" || form.connectionType === "tcp+tls"}
          <div class="grid grid-cols-3 gap-3">
            <div class="col-span-2">
              <label class="block text-sm font-medium text-foreground mb-1" for="env-host">Host</label>
              <input
                id="env-host"
                type="text"
                bind:value={form.host}
                placeholder="192.168.1.100"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1" for="env-port">Port</label>
              <input
                id="env-port"
                type="number"
                bind:value={form.port}
                placeholder="2375"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
              />
            </div>
          </div>
        {/if}

        <!-- TLS fields (shown when tcp+tls) -->
        {#if form.connectionType === "tcp+tls"}
          <div class="space-y-3 p-3 rounded-lg bg-background-tertiary/30 border border-border">
            <p class="text-xs font-semibold text-foreground-muted uppercase tracking-wider">TLS Configuration</p>
            <div>
              <label class="block text-xs font-medium text-foreground mb-1">CA Certificate</label>
              <textarea
                bind:value={form.tlsCa}
                placeholder="-----BEGIN CERTIFICATE-----"
                rows="3"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-xs text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none resize-none"
              ></textarea>
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground mb-1">Client Certificate</label>
              <textarea
                bind:value={form.tlsCert}
                placeholder="-----BEGIN CERTIFICATE-----"
                rows="3"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-xs text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none resize-none"
              ></textarea>
            </div>
            <div>
              <label class="block text-xs font-medium text-foreground mb-1">Client Key</label>
              <textarea
                bind:value={form.tlsKey}
                placeholder="-----BEGIN PRIVATE KEY-----"
                rows="3"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-xs text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none resize-none"
              ></textarea>
            </div>
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium text-foreground">Skip TLS verify</p>
                <p class="text-xs text-foreground-muted">Accept any certificate (insecure)</p>
              </div>
              <button
                type="button"
                class="relative w-11 h-6 rounded-full transition-colors {form.tlsSkipVerify ? 'bg-primary' : 'bg-background-tertiary'}"
                onclick={() => (form.tlsSkipVerify = !form.tlsSkipVerify)}
              >
                <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.tlsSkipVerify ? 'translate-x-5' : ''}"></span>
              </button>
            </div>
          </div>
        {/if}

        <!-- Test Connection -->
        <div>
          <button
            type="button"
            class="flex items-center gap-2 px-4 py-2 text-sm bg-background-tertiary text-foreground rounded-lg hover:bg-background-tertiary/80 transition-colors disabled:opacity-50"
            onclick={testConnection}
            disabled={testing}
          >
            {#if testing}
              <Loader2 class="w-4 h-4 animate-spin" />
              Testing...
            {:else}
              <Globe class="w-4 h-4" />
              Test connection
            {/if}
          </button>
          {#if testResult !== null}
            <div class="mt-2 flex items-start gap-2 text-sm {testResult.success ? 'text-green-400' : 'text-red-400'}">
              {#if testResult.success}
                <Check class="w-4 h-4 shrink-0 mt-0.5" />
                <div>
                  <span>Connected</span>
                  {#if testResult.info}
                    <p class="text-xs text-foreground-muted">Docker {testResult.info.serverVersion} · {testResult.info.containers} containers</p>
                  {/if}
                </div>
              {:else}
                <AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
                <span>{testResult.error || "Connection failed"}</span>
              {/if}
            </div>
          {/if}
        </div>

        <!-- Public IP -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1" for="env-publicip">Public IP (optional)</label>
          <input
            id="env-publicip"
            type="text"
            bind:value={form.publicIp}
            placeholder="203.0.113.1"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>

      {:else if activeTab === "updates"}
        <!-- Update Check -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Update check</p>
            <p class="text-xs text-foreground-muted">Check for newer image versions</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.autoUpdate ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.autoUpdate = !form.autoUpdate)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.autoUpdate ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        {#if form.autoUpdate}
          <div>
            <label class="block text-sm font-medium text-foreground mb-1" for="env-schedule">Update schedule (cron)</label>
            <input
              id="env-schedule"
              type="text"
              bind:value={form.updateSchedule}
              placeholder="0 3 * * *"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
            />
            <p class="text-xs text-foreground-muted mt-1">e.g. "0 3 * * *" = daily at 3am</p>
          </div>
        {/if}

        <!-- Image Prune -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Image prune</p>
            <p class="text-xs text-foreground-muted">Remove unused images on schedule</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.imagePrune ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.imagePrune = !form.imagePrune)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.imagePrune ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        {#if form.imagePrune}
          <div class="space-y-3">
            <div>
              <p class="text-sm font-medium text-foreground mb-2">Prune mode</p>
              <div class="flex gap-2">
                {#each [{ id: "dangling", label: "Dangling only" }, { id: "all", label: "All unused" }] as mode}
                  <button
                    type="button"
                    class="flex-1 px-3 py-1.5 text-sm rounded-lg border-2 transition-all {form.imagePruneMode === mode.id ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground-muted hover:border-foreground-muted'}"
                    onclick={() => (form.imagePruneMode = mode.id)}
                  >
                    {mode.label}
                  </button>
                {/each}
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1" for="env-prune-cron">Prune schedule (cron)</label>
              <input
                id="env-prune-cron"
                type="text"
                bind:value={form.imagePruneCron}
                placeholder="0 4 * * *"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
              />
            </div>
          </div>
        {/if}

      {:else if activeTab === "monitoring"}
        <!-- Event Tracking -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Event tracking</p>
            <p class="text-xs text-foreground-muted">Log container start/stop/die events</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.eventTracking ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.eventTracking = !form.eventTracking)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.eventTracking ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Collect Metrics -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Collect metrics</p>
            <p class="text-xs text-foreground-muted">CPU/memory polling</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.collectMetrics ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.collectMetrics = !form.collectMetrics)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.collectMetrics ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Vulnerability Scanning -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Vulnerability scanning</p>
            <p class="text-xs text-foreground-muted">Trivy scan on image pull/update</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.vulnScanning ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.vulnScanning = !form.vulnScanning)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.vulnScanning ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Highlight Changes -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Highlight changes</p>
            <p class="text-xs text-foreground-muted">Visual indicator on changed containers</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.highlightChanges ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.highlightChanges = !form.highlightChanges)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.highlightChanges ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

      {:else if activeTab === "advanced"}
        <!-- Timezone -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1" for="env-tz">Timezone</label>
          <select
            id="env-tz"
            bind:value={form.timezone}
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          >
            {#each TIMEZONES as tz}
              <option value={tz}>{tz}</option>
            {/each}
          </select>
        </div>

        <!-- Disk Warning -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">Disk warning</p>
            <p class="text-xs text-foreground-muted">Alert when disk usage exceeds threshold</p>
          </div>
          <button
            type="button"
            class="relative w-11 h-6 rounded-full transition-colors {form.diskWarningEnabled ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.diskWarningEnabled = !form.diskWarningEnabled)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.diskWarningEnabled ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        {#if form.diskWarningEnabled}
          <div class="space-y-3">
            <div>
              <p class="text-sm font-medium text-foreground mb-2">Warning mode</p>
              <div class="flex gap-2">
                {#each [{ id: "percentage", label: "Percentage" }, { id: "absolute", label: "Absolute (GB)" }] as mode}
                  <button
                    type="button"
                    class="flex-1 px-3 py-1.5 text-sm rounded-lg border-2 transition-all {form.diskWarningMode === mode.id ? 'border-primary bg-primary/10 text-primary' : 'border-border text-foreground-muted hover:border-foreground-muted'}"
                    onclick={() => (form.diskWarningMode = mode.id)}
                  >
                    {mode.label}
                  </button>
                {/each}
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1" for="env-disk-threshold">
                Threshold {form.diskWarningMode === "percentage" ? "(%)" : "(GB)"}
              </label>
              <input
                id="env-disk-threshold"
                type="number"
                bind:value={form.diskWarningThreshold}
                min="1"
                max={form.diskWarningMode === "percentage" ? 100 : 10000}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
              />
            </div>
          </div>
        {/if}
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end gap-3 px-5 py-3 border-t border-border bg-background-tertiary/30 shrink-0">
      <button
        class="px-4 py-2 text-sm text-foreground-muted hover:text-foreground transition-colors"
        onclick={onclose}
      >
        Cancel
      </button>
      <button
        class="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        onclick={handleSave}
        disabled={!form.id || !form.name || saving}
      >
        {saving ? "Saving..." : "Save"}
      </button>
    </div>
  </div>
</div>
