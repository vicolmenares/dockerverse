<script lang="ts">
  import {
    User,
    Check,
    Camera,
    Trash2,
    RefreshCw,
    Upload,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import {
    auth,
    currentUser,
    uploadAvatar,
    deleteAvatar,
  } from '$lib/stores/auth';
  import { API_BASE } from '$lib/api/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  // Avatar state
  let avatarState = $state({
    loading: false,
    error: null as string | null,
  });
  let avatarInput: HTMLInputElement | null = $state(null);

  // Profile form state
  let profileForm = $state({
    firstName: '',
    lastName: '',
    email: '',
    loading: false,
    success: false,
    error: null as string | null,
  });

  // Load profile data
  $effect(() => {
    if ($currentUser) {
      profileForm.firstName = $currentUser.firstName || '';
      profileForm.lastName = $currentUser.lastName || '';
      profileForm.email = $currentUser.email || '';
    }
  });

  function triggerAvatarUpload() {
    avatarInput?.click();
  }

  async function handleAvatarChange(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      avatarState.error = $language === 'es' ? 'Solo se permiten imágenes' : 'Only images are allowed';
      return;
    }

    if (file.size > 500 * 1024) {
      avatarState.error = $language === 'es' ? 'La imagen no puede superar 500KB' : 'Image must be under 500KB';
      return;
    }

    avatarState.loading = true;
    avatarState.error = null;

    try {
      await uploadAvatar(file);
    } catch (err) {
      avatarState.error = err instanceof Error ? err.message : ($language === 'es' ? 'Error al subir avatar' : 'Failed to upload avatar');
    } finally {
      avatarState.loading = false;
      input.value = '';
    }
  }

  async function handleDeleteAvatar() {
    avatarState.loading = true;
    avatarState.error = null;

    try {
      await deleteAvatar();
    } catch (err) {
      avatarState.error = err instanceof Error ? err.message : ($language === 'es' ? 'Error al eliminar avatar' : 'Failed to delete avatar');
    } finally {
      avatarState.loading = false;
    }
  }

  async function saveProfile() {
    profileForm.error = null;
    profileForm.success = false;
    profileForm.loading = true;

    const token = localStorage.getItem('auth_access_token');

    try {
      const res = await fetch(`${API_BASE}/api/auth/profile`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          firstName: profileForm.firstName,
          lastName: profileForm.lastName,
          email: profileForm.email,
        }),
      });

      if (res.ok) {
        profileForm.success = true;
        auth.refreshUser();
        setTimeout(() => (profileForm.success = false), 3000);
      } else {
        const err = await res.json();
        profileForm.error = err.error || ($language === 'es' ? 'Error al guardar' : 'Failed to save');
      }
    } catch (err) {
      profileForm.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      profileForm.loading = false;
    }
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_label_has_associated_control -->
<div class="p-4 space-y-6">
  <!-- Success/Error Messages -->
  {#if profileForm.success}
    <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm">
      <Check class="w-4 h-4" />
      <span>{$language === 'es' ? 'Perfil actualizado' : 'Profile updated'}</span>
    </div>
  {/if}
  {#if profileForm.error}
    <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
      <span>{profileForm.error}</span>
    </div>
  {/if}

  <!-- Avatar -->
  <div class="flex flex-col items-center">
    <input
      type="file"
      accept="image/*"
      class="hidden"
      bind:this={avatarInput}
      onchange={handleAvatarChange}
    />

    <button
      onclick={triggerAvatarUpload}
      disabled={avatarState.loading}
      class="relative w-24 h-24 rounded-full mb-3 group overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-background disabled:opacity-50"
    >
      {#if $currentUser?.avatar}
        <img src={$currentUser.avatar} alt="Avatar" class="w-full h-full object-cover" />
      {:else}
        <div class="w-full h-full bg-primary/20 flex items-center justify-center">
          <User class="w-12 h-12 text-primary" />
        </div>
      {/if}

      <div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
        {#if avatarState.loading}
          <RefreshCw class="w-6 h-6 text-white animate-spin" />
        {:else}
          <Camera class="w-6 h-6 text-white" />
        {/if}
      </div>
    </button>

    <div class="flex items-center gap-2">
      <button
        onclick={triggerAvatarUpload}
        disabled={avatarState.loading}
        class="flex items-center gap-2 text-sm text-primary hover:text-primary/80 disabled:opacity-50"
      >
        <Upload class="w-4 h-4" />
        {st.changeAvatar}
      </button>

      {#if $currentUser?.avatar}
        <span class="text-foreground-muted">|</span>
        <button
          onclick={handleDeleteAvatar}
          disabled={avatarState.loading}
          class="flex items-center gap-2 text-sm text-stopped hover:text-stopped/80 disabled:opacity-50"
        >
          <Trash2 class="w-4 h-4" />
          {$language === 'es' ? 'Eliminar' : 'Remove'}
        </button>
      {/if}
    </div>

    {#if avatarState.error}
      <p class="mt-2 text-sm text-stopped">{avatarState.error}</p>
    {/if}

    <p class="mt-1 text-xs text-foreground-muted">
      {$language === 'es' ? 'Máximo 500KB' : 'Max 500KB'}
    </p>
  </div>

  <!-- Form -->
  <div class="space-y-4">
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-foreground mb-1">{st.firstName}</label>
        <input
          type="text"
          bind:value={profileForm.firstName}
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-foreground mb-1">{st.lastName}</label>
        <input
          type="text"
          bind:value={profileForm.lastName}
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
        />
      </div>
    </div>
    <div>
      <label class="block text-sm font-medium text-foreground mb-1">{st.email}</label>
      <input
        type="email"
        bind:value={profileForm.email}
        class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
      />
    </div>
    <div>
      <label class="block text-sm font-medium text-foreground mb-1">{st.username}</label>
      <input
        type="text"
        value={$currentUser?.username ?? 'admin'}
        disabled
        class="w-full px-3 py-2 bg-background-tertiary border border-border rounded-lg text-foreground-muted cursor-not-allowed"
      />
    </div>
  </div>

  <!-- Save button -->
  <button
    onclick={saveProfile}
    disabled={profileForm.loading}
    class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
  >
    {#if profileForm.loading}
      <RefreshCw class="w-4 h-4 animate-spin" />
    {/if}
    {st.save}
  </button>
</div>
