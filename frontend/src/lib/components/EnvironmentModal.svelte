<script module lang="ts">
  export interface EnvironmentData {
    id: string;
    name: string;
    connectionType: string;
    address: string;
    protocol: string;
    isLocal: boolean;
    labels: string;
    autoUpdate: boolean;
    updateSchedule: string;
    imagePrune: boolean;
    eventTracking: boolean;
    vulnScanning: boolean;
  }
</script>

<script lang="ts">
  import { X, Server, Wifi, RefreshCw, Shield, Bell } from "lucide-svelte";
  import { language } from "$lib/stores/docker";
  import { settingsText } from "$lib/settings";

  let {
    environment = null,
    onclose,
    onsave,
  }: {
    environment?: EnvironmentData | null;
    onclose: () => void;
    onsave: (env: EnvironmentData) => void;
  } = $props();

  let st = $derived(settingsText[$language]);
  let activeTab = $state("general");

  // Form state
  let form = $state<EnvironmentData>(
    environment
      ? { ...environment }
      : {
          id: "",
          name: "",
          connectionType: "socket",
          address: "/var/run/docker.sock",
          protocol: "http",
          isLocal: true,
          labels: "",
          autoUpdate: false,
          updateSchedule: "0 3 * * *",
          imagePrune: false,
          eventTracking: true,
          vulnScanning: false,
        },
  );

  function handleConnectionTypeChange(type: string) {
    form.connectionType = type;
    if (type === "socket") {
      form.address = "/var/run/docker.sock";
      form.isLocal = true;
    } else {
      form.address = "http://";
      form.isLocal = false;
    }
  }

  function handleSave() {
    if (!form.id || !form.name) return;
    onsave(form);
  }

  let isEditing = $derived(environment !== null);

  const tabs = $derived([
    { id: "general", label: st.general, icon: Server },
    { id: "updates", label: st.updates, icon: RefreshCw },
    { id: "activity", label: st.activity, icon: Wifi },
    { id: "security", label: st.security || "Security", icon: Shield },
  ]);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 bg-black/60 z-[100] flex items-center justify-center p-4"
  onclick={onclose}
>
  <div
    class="bg-background-secondary border border-border rounded-xl shadow-2xl w-full max-w-2xl overflow-hidden"
    onclick={(e) => e.stopPropagation()}
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between px-5 py-4 border-b border-border"
    >
      <h3 class="font-semibold text-foreground">
        {isEditing ? st.editEnvironment : st.addEnvironment}
      </h3>
      <button
        class="btn-icon hover:bg-background-tertiary"
        onclick={onclose}
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-border px-5">
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
    <div class="px-5 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
      {#if activeTab === "general"}
        <!-- ID -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">ID</label>
          <input
            type="text"
            bind:value={form.id}
            disabled={isEditing}
            placeholder="raspi1"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none disabled:opacity-50"
          />
        </div>

        <!-- Name -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">{st.firstName || "Name"}</label>
          <input
            type="text"
            bind:value={form.name}
            placeholder="Raspberry Pi Main"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Labels -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">Labels</label>
          <input
            type="text"
            bind:value={form.labels}
            placeholder="production, arm64"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Connection Type -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-2">{st.connectionType}</label>
          <div class="flex gap-3">
            <button
              class="flex-1 flex items-center gap-2 p-3 rounded-lg border-2 transition-all {form.connectionType === 'socket' ? 'border-primary bg-primary/10' : 'border-border hover:border-foreground-muted'}"
              onclick={() => handleConnectionTypeChange("socket")}
            >
              <Server class="w-5 h-5 {form.connectionType === 'socket' ? 'text-primary' : 'text-foreground-muted'}" />
              <div class="text-left">
                <p class="text-sm font-medium">{st.unixSocket}</p>
                <p class="text-xs text-foreground-muted">/var/run/docker.sock</p>
              </div>
            </button>
            <button
              class="flex-1 flex items-center gap-2 p-3 rounded-lg border-2 transition-all {form.connectionType === 'tcp' ? 'border-primary bg-primary/10' : 'border-border hover:border-foreground-muted'}"
              onclick={() => handleConnectionTypeChange("tcp")}
            >
              <Wifi class="w-5 h-5 {form.connectionType === 'tcp' ? 'text-primary' : 'text-foreground-muted'}" />
              <div class="text-left">
                <p class="text-sm font-medium">{st.directTcp}</p>
                <p class="text-xs text-foreground-muted">http://host:2375</p>
              </div>
            </button>
          </div>
        </div>

        <!-- Address -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {form.connectionType === "socket" ? st.socketPath : st.hostPort}
          </label>
          <input
            type="text"
            bind:value={form.address}
            placeholder={form.connectionType === "socket" ? "/var/run/docker.sock" : "http://192.168.1.100:2375"}
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
          />
        </div>

        {#if form.connectionType === "tcp"}
          <!-- Protocol -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Protocol</label>
            <div class="flex gap-2">
              {#each ["http", "https"] as proto}
                <button
                  class="px-4 py-1.5 text-sm rounded-lg transition-colors {form.protocol === proto ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
                  onclick={() => (form.protocol = proto)}
                >
                  {proto.toUpperCase()}
                </button>
              {/each}
            </div>
          </div>
        {/if}

      {:else if activeTab === "updates"}
        <!-- Auto Update -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">{st.autoUpdate}</p>
            <p class="text-xs text-foreground-muted">{$language === "es" ? "Actualizar imágenes automáticamente" : "Automatically update container images"}</p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {form.autoUpdate ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.autoUpdate = !form.autoUpdate)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.autoUpdate ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        {#if form.autoUpdate}
          <!-- Schedule -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">{st.updateSchedule}</label>
            <input
              type="text"
              bind:value={form.updateSchedule}
              placeholder="0 3 * * *"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground font-mono placeholder:text-foreground-muted focus:border-primary focus:outline-none"
            />
            <p class="text-xs text-foreground-muted mt-1">Cron format (e.g. "0 3 * * *" = daily at 3am)</p>
          </div>
        {/if}

        <!-- Image Prune -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">{st.imagePrune}</p>
            <p class="text-xs text-foreground-muted">{$language === "es" ? "Eliminar imágenes sin uso después de actualizar" : "Remove unused images after updates"}</p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {form.imagePrune ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.imagePrune = !form.imagePrune)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.imagePrune ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

      {:else if activeTab === "activity"}
        <!-- Event Tracking -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">{st.eventTracking}</p>
            <p class="text-xs text-foreground-muted">{$language === "es" ? "Registrar eventos de contenedores" : "Log container events"}</p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {form.eventTracking ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.eventTracking = !form.eventTracking)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.eventTracking ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

      {:else if activeTab === "security"}
        <!-- Vulnerability Scanning -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">{st.vulnScanning}</p>
            <p class="text-xs text-foreground-muted">{$language === "es" ? "Escanear imágenes en busca de vulnerabilidades" : "Scan images for vulnerabilities"}</p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {form.vulnScanning ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (form.vulnScanning = !form.vulnScanning)}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {form.vulnScanning ? 'translate-x-5' : ''}"></span>
          </button>
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div
      class="flex items-center justify-end gap-3 px-5 py-3 border-t border-border bg-background-tertiary/30"
    >
      <button
        class="px-4 py-2 text-sm text-foreground-muted hover:text-foreground transition-colors"
        onclick={onclose}
      >
        {st.cancel}
      </button>
      <button
        class="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        onclick={handleSave}
        disabled={!form.id || !form.name}
      >
        {st.save}
      </button>
    </div>
  </div>
</div>
