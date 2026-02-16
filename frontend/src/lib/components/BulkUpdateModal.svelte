<script lang="ts">
    import {
        X,
        AlertCircle,
        CheckCircle,
        Loader2,
        Filter,
    } from "lucide-svelte";
    import { containers, hosts, language } from "$lib/stores/docker";
    import { triggerBulkUpdate, type BulkUpdateResult } from "$lib/api/docker";

    interface Props {
        open?: boolean;
        onClose: () => void;
    }

    let { open = $bindable(false), onClose }: Props = $props();

    // Form state
    let selectedHostId = $state("");
    let nameFilter = $state("");
    let isLoading = $state(false);
    let error = $state<string | null>(null);
    let result = $state<BulkUpdateResult | null>(null);

    // Derived: filtered containers preview
    let matchedContainers = $derived(
        $containers.filter((c) => {
            if (selectedHostId && c.hostId !== selectedHostId) return false;
            if (
                nameFilter &&
                !c.name.toLowerCase().includes(nameFilter.toLowerCase())
            )
                return false;
            return true;
        }),
    );

    async function handleSubmit() {
        error = null;
        result = null;

        if (!nameFilter && !selectedHostId) {
            error =
                $language === "es"
                    ? "Debes especificar al menos un filtro (host o nombre)"
                    : "You must specify at least one filter (host or name)";
            return;
        }

        if (matchedContainers.length === 0) {
            error =
                $language === "es"
                    ? "No se encontraron contenedores que coincidan con los filtros"
                    : "No containers match the current filters";
            return;
        }

        isLoading = true;
        try {
            const bulkResult = await triggerBulkUpdate(
                selectedHostId || undefined,
                nameFilter || undefined,
                false,
            );
            result = bulkResult;
        } catch (e: any) {
            error =
                e.message ||
                ($language === "es"
                    ? "Error al actualizar contenedores"
                    : "Failed to update containers");
        } finally {
            isLoading = false;
        }
    }

    function handleClose() {
        if (!isLoading) {
            selectedHostId = "";
            nameFilter = "";
            error = null;
            result = null;
            onClose();
        }
    }

    function handleBackdropClick(e: MouseEvent) {
        if (e.target === e.currentTarget) {
            handleClose();
        }
    }
</script>

{#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4"
        onclick={handleBackdropClick}
    >
        <div
            class="bg-background border border-border rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto"
        >
            <!-- Header -->
            <div
                class="flex items-center justify-between p-6 border-b border-border"
            >
                <div class="flex items-center gap-3">
                    <div class="p-2 bg-primary/10 rounded-lg">
                        <Filter class="w-5 h-5 text-primary" />
                    </div>
                    <div>
                        <h2 class="text-xl font-semibold text-foreground">
                            {$language === "es"
                                ? "Actualización Masiva"
                                : "Bulk Update"}
                        </h2>
                        <p class="text-sm text-foreground-muted">
                            {$language === "es"
                                ? "Actualiza múltiples contenedores con Watchtower"
                                : "Update multiple containers with Watchtower"}
                        </p>
                    </div>
                </div>
                <button
                    onclick={handleClose}
                    disabled={isLoading}
                    class="p-2 hover:bg-background-tertiary rounded-lg transition-colors disabled:opacity-50"
                >
                    <X class="w-5 h-5 text-foreground-muted" />
                </button>
            </div>

            <!-- Body -->
            <div class="p-6 space-y-6">
                {#if error}
                    <div
                        class="flex items-start gap-3 p-4 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped"
                    >
                        <AlertCircle class="w-5 h-5 flex-shrink-0 mt-0.5" />
                        <p class="text-sm">{error}</p>
                    </div>
                {/if}

                {#if result}
                    <div class="space-y-4">
                        <!-- Success Summary -->
                        <div
                            class="flex items-start gap-3 p-4 bg-running/10 border border-running/30 rounded-lg text-running"
                        >
                            <CheckCircle class="w-5 h-5 flex-shrink-0 mt-0.5" />
                            <div class="flex-1">
                                <p class="font-medium">
                                    {$language === "es"
                                        ? `Actualización completada: ${result.updated}/${result.matched} exitosos`
                                        : `Update complete: ${result.updated}/${result.matched} successful`}
                                </p>
                                {#if result.failed > 0}
                                    <p class="text-sm mt-1 text-stopped">
                                        {$language === "es"
                                            ? `${result.failed} fallidos`
                                            : `${result.failed} failed`}
                                    </p>
                                {/if}
                            </div>
                        </div>

                        <!-- Detailed Results -->
                        {#if result.results && result.results.length > 0}
                            <div class="space-y-2">
                                <h4 class="text-sm font-medium text-foreground">
                                    {$language === "es"
                                        ? "Resultados detallados:"
                                        : "Detailed results:"}
                                </h4>
                                <div class="max-h-60 overflow-y-auto space-y-2">
                                    {#each result.results as item}
                                        <div
                                            class="flex items-center justify-between p-3 bg-background-tertiary rounded-lg"
                                        >
                                            <div class="flex-1 min-w-0">
                                                <p
                                                    class="text-sm font-medium text-foreground truncate"
                                                >
                                                    {item.containerName}
                                                </p>
                                                <p
                                                    class="text-xs text-foreground-muted"
                                                >
                                                    {item.hostId}
                                                </p>
                                            </div>
                                            <div
                                                class="flex items-center gap-2"
                                            >
                                                {#if item.success}
                                                    <CheckCircle
                                                        class="w-4 h-4 text-running"
                                                    />
                                                {:else}
                                                    <AlertCircle
                                                        class="w-4 h-4 text-stopped"
                                                    />
                                                {/if}
                                                {#if item.error}
                                                    <span
                                                        class="text-xs text-stopped max-w-[200px] truncate"
                                                        >{item.error}</span
                                                    >
                                                {/if}
                                            </div>
                                        </div>
                                    {/each}
                                </div>
                            </div>
                        {/if}

                        <!-- Close Button -->
                        <button
                            onclick={handleClose}
                            class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                        >
                            {$language === "es" ? "Cerrar" : "Close"}
                        </button>
                    </div>
                {:else}
                    <!-- Filters Form -->
                    <div class="space-y-4">
                        <!-- Host Filter -->
                        <div>
                            <label
                                class="block text-sm font-medium text-foreground mb-2"
                                for="bulk-update-host"
                            >
                                {$language === "es"
                                    ? "Filtrar por Host (opcional)"
                                    : "Filter by Host (optional)"}
                            </label>
                            <select
                                id="bulk-update-host"
                                bind:value={selectedHostId}
                                disabled={isLoading}
                                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground disabled:opacity-50"
                            >
                                <option value=""
                                    >{$language === "es"
                                        ? "Todos los hosts"
                                        : "All hosts"}</option
                                >
                                {#each $hosts as host}
                                    <option value={host.id}>{host.name}</option>
                                {/each}
                            </select>
                        </div>

                        <!-- Name Filter -->
                        <div>
                            <label
                                class="block text-sm font-medium text-foreground mb-2"
                                for="bulk-update-name"
                            >
                                {$language === "es"
                                    ? "Filtrar por nombre (opcional)"
                                    : "Filter by name (optional)"}
                            </label>
                            <input
                                id="bulk-update-name"
                                type="text"
                                bind:value={nameFilter}
                                disabled={isLoading}
                                placeholder={$language === "es"
                                    ? "ej: raspi, portainer, etc."
                                    : "e.g.: raspi, portainer, etc."}
                                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground disabled:opacity-50"
                            />
                        </div>

                        <!-- Preview -->
                        <div class="p-4 bg-background-tertiary rounded-lg">
                            <p class="text-sm font-medium text-foreground mb-2">
                                {$language === "es"
                                    ? "Vista previa:"
                                    : "Preview:"}
                            </p>
                            <p class="text-sm text-foreground-muted">
                                {matchedContainers.length === 0
                                    ? $language === "es"
                                        ? "No hay contenedores que coincidan"
                                        : "No matching containers"
                                    : $language === "es"
                                      ? `${matchedContainers.length} contenedor(es) será(n) actualizado(s)`
                                      : `${matchedContainers.length} container(s) will be updated`}
                            </p>
                            {#if matchedContainers.length > 0 && matchedContainers.length <= 10}
                                <ul class="mt-2 space-y-1">
                                    {#each matchedContainers as container}
                                        <li
                                            class="text-xs text-foreground-muted"
                                        >
                                            • {container.name} ({container.hostName})
                                        </li>
                                    {/each}
                                </ul>
                            {/if}
                        </div>
                    </div>

                    <!-- Actions -->
                    <div class="flex gap-3">
                        <button
                            onclick={handleClose}
                            disabled={isLoading}
                            class="flex-1 py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors disabled:opacity-50"
                        >
                            {$language === "es" ? "Cancelar" : "Cancel"}
                        </button>
                        <button
                            onclick={handleSubmit}
                            disabled={isLoading ||
                                matchedContainers.length === 0}
                            class="flex-1 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {#if isLoading}
                                <Loader2 class="w-4 h-4 animate-spin" />
                                {$language === "es"
                                    ? "Actualizando..."
                                    : "Updating..."}
                            {:else}
                                {$language === "es" ? "Actualizar" : "Update"}
                            {/if}
                        </button>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}
