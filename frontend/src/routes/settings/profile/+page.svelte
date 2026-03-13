<script lang="ts">
  import { onMount } from 'svelte';
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

  // Change password state
  let pwForm = $state({
    current: '',
    next: '',
    confirm: '',
    loading: false,
    success: false,
    error: null as string | null,
  });

  async function changePassword() {
    pwForm.error = null;
    pwForm.success = false;

    if (pwForm.next !== pwForm.confirm) {
      pwForm.error = $language === 'es' ? 'Las contraseñas no coinciden' : 'Passwords do not match';
      return;
    }
    if (pwForm.next.length < 8) {
      pwForm.error = $language === 'es' ? 'Mínimo 8 caracteres' : 'Minimum 8 characters';
      return;
    }

    pwForm.loading = true;
    try {
      const token = localStorage.getItem('auth_access_token');
      const res = await fetch(`${API_BASE}/api/auth/password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ currentPassword: pwForm.current, newPassword: pwForm.next }),
      });
      if (res.ok) {
        pwForm.success = true;
        pwForm.current = '';
        pwForm.next = '';
        pwForm.confirm = '';
        setTimeout(() => (pwForm.success = false), 3000);
      } else {
        const err = await res.json();
        pwForm.error = err.error || ($language === 'es' ? 'Error al cambiar contraseña' : 'Failed to change password');
      }
    } catch {
      pwForm.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      pwForm.loading = false;
    }
  }

  // 2FA state
  let totpStatus = $state({ enabled: false, loading: true });
  let totpSetup = $state({
    active: false,
    secret: '',
    qrUrl: '',
    code: '',
    loading: false,
    error: null as string | null,
    recoveryCodes: [] as string[],
    showCodes: false,
  });
  let totpDisable = $state({
    active: false,
    password: '',
    loading: false,
    error: null as string | null,
  });

  async function loadTotpStatus() {
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/status`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        totpStatus.enabled = data.enabled ?? false;
      }
    } catch { /* ignore */ } finally {
      totpStatus.loading = false;
    }
  }

  async function startTotpSetup() {
    totpSetup.loading = true;
    totpSetup.error = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/setup`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        totpSetup.secret = data.secret;
        totpSetup.qrUrl = data.url;
        totpSetup.active = true;
      } else {
        const err = await res.json().catch(() => ({}));
        totpSetup.error = err.error || 'Failed to start 2FA setup';
      }
    } catch {
      totpSetup.error = 'Failed to start 2FA setup';
    } finally {
      totpSetup.loading = false;
    }
  }

  async function confirmTotpEnable() {
    totpSetup.loading = true;
    totpSetup.error = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/enable`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ code: totpSetup.code }),
      });
      if (res.ok) {
        const data = await res.json();
        totpStatus.enabled = true;
        totpSetup.active = false;
        totpSetup.code = '';
        totpSetup.recoveryCodes = data.recoveryCodes ?? [];
        totpSetup.showCodes = true;
      } else {
        const err = await res.json();
        totpSetup.error = err.error || 'Invalid code';
      }
    } finally {
      totpSetup.loading = false;
    }
  }

  async function confirmTotpDisable() {
    totpDisable.loading = true;
    totpDisable.error = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/disable`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ password: totpDisable.password }),
      });
      if (res.ok) {
        totpStatus.enabled = false;
        totpDisable.active = false;
        totpDisable.password = '';
      } else {
        const err = await res.json();
        totpDisable.error = err.error || 'Failed to disable 2FA';
      }
    } finally {
      totpDisable.loading = false;
    }
  }

  onMount(() => { loadTotpStatus(); });
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

  <!-- Change Password -->
  <div class="border-t border-border pt-4">
    <h4 class="text-sm font-semibold text-foreground mb-3">
      {$language === 'es' ? 'Cambiar contraseña' : 'Change Password'}
    </h4>

    {#if pwForm.success}
      <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm mb-3">
        <Check class="w-4 h-4" />
        <span>{$language === 'es' ? 'Contraseña actualizada' : 'Password updated'}</span>
      </div>
    {/if}
    {#if pwForm.error}
      <div class="p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm mb-3">{pwForm.error}</div>
    {/if}

    <div class="space-y-3">
      <!-- svelte-ignore a11y_label_has_associated_control -->
      <div>
        <label class="block text-sm font-medium text-foreground mb-1">
          {$language === 'es' ? 'Contraseña actual' : 'Current Password'}
        </label>
        <input
          type="password"
          bind:value={pwForm.current}
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
        />
      </div>
      <!-- svelte-ignore a11y_label_has_associated_control -->
      <div>
        <label class="block text-sm font-medium text-foreground mb-1">
          {$language === 'es' ? 'Nueva contraseña' : 'New Password'}
        </label>
        <input
          type="password"
          bind:value={pwForm.next}
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
        />
      </div>
      <!-- svelte-ignore a11y_label_has_associated_control -->
      <div>
        <label class="block text-sm font-medium text-foreground mb-1">
          {$language === 'es' ? 'Confirmar contraseña' : 'Confirm Password'}
        </label>
        <input
          type="password"
          bind:value={pwForm.confirm}
          class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
        />
      </div>
      <button
        onclick={changePassword}
        disabled={pwForm.loading || !pwForm.current || !pwForm.next || !pwForm.confirm}
        class="w-full py-2 bg-background-secondary border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors disabled:opacity-50 flex items-center justify-center gap-2 text-sm"
      >
        {#if pwForm.loading}<RefreshCw class="w-4 h-4 animate-spin" />{/if}
        {$language === 'es' ? 'Cambiar contraseña' : 'Change Password'}
      </button>
    </div>
  </div>

  <!-- 2FA Management -->
  <div class="border-t border-border pt-4">
    <div class="flex items-center justify-between mb-3">
      <div>
        <h4 class="text-sm font-semibold text-foreground">
          {$language === 'es' ? 'Autenticación de dos factores' : 'Two-Factor Authentication'}
        </h4>
        <p class="text-xs text-foreground-muted mt-0.5">
          {$language === 'es' ? 'Protege tu cuenta con TOTP (Google Authenticator, Authy)' : 'Protect your account with TOTP (Google Authenticator, Authy)'}
        </p>
      </div>
      <span class="px-2 py-0.5 rounded text-xs font-medium {totpStatus.enabled ? 'bg-running/10 text-running' : 'bg-background-tertiary text-foreground-muted'}">
        {totpStatus.enabled ? ($language === 'es' ? 'Activo' : 'Enabled') : ($language === 'es' ? 'Inactivo' : 'Disabled')}
      </span>
    </div>

    {#if totpSetup.showCodes}
      <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg mb-3">
        <p class="text-xs font-semibold text-amber-400 mb-2">
          {$language === 'es' ? 'Guarda estos códigos de recuperación' : 'Save these recovery codes'}
        </p>
        <div class="grid grid-cols-2 gap-1">
          {#each totpSetup.recoveryCodes as code}
            <code class="text-xs font-mono text-foreground bg-background px-2 py-1 rounded">{code}</code>
          {/each}
        </div>
        <button
          onclick={() => (totpSetup.showCodes = false)}
          class="mt-2 text-xs text-foreground-muted hover:text-foreground"
        >
          {$language === 'es' ? 'He guardado mis códigos ✓' : "I've saved my codes ✓"}
        </button>
      </div>
    {/if}

    {#if !totpStatus.loading && !totpStatus.enabled && !totpSetup.active}
      <button
        onclick={startTotpSetup}
        disabled={totpSetup.loading}
        class="flex items-center gap-2 text-sm text-primary hover:text-primary/80 disabled:opacity-50"
      >
        {#if totpSetup.loading}<RefreshCw class="w-4 h-4 animate-spin" />{/if}
        {$language === 'es' ? 'Activar 2FA' : 'Enable 2FA'}
      </button>
    {/if}

    {#if totpSetup.active}
      <div class="space-y-3">
        <p class="text-xs text-foreground-muted">
          {$language === 'es' ? 'Escanea este código QR con tu app de autenticación:' : 'Scan this QR code with your authenticator app:'}
        </p>
        <img
          src={`https://api.qrserver.com/v1/create-qr-code/?size=160x160&data=${encodeURIComponent(totpSetup.qrUrl)}`}
          alt="QR Code"
          class="w-40 h-40 rounded-lg border border-border"
        />
        <p class="text-xs text-foreground-muted">
          {$language === 'es' ? 'O ingresa el secreto manualmente:' : 'Or enter the secret manually:'}
          <code class="font-mono text-foreground bg-background-tertiary px-1.5 py-0.5 rounded ml-1">{totpSetup.secret}</code>
        </p>
        {#if totpSetup.error}
          <div class="p-2 bg-stopped/10 border border-stopped/30 rounded text-stopped text-xs">{totpSetup.error}</div>
        {/if}
        <div class="flex gap-2">
          <input
            type="text"
            placeholder={$language === 'es' ? 'Código de 6 dígitos' : '6-digit code'}
            bind:value={totpSetup.code}
            maxlength="6"
            class="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-foreground text-sm font-mono tracking-widest"
          />
          <button
            onclick={confirmTotpEnable}
            disabled={totpSetup.loading || totpSetup.code.length !== 6}
            class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90 disabled:opacity-50"
          >
            {$language === 'es' ? 'Verificar' : 'Verify'}
          </button>
        </div>
        <button onclick={() => (totpSetup.active = false)} class="text-xs text-foreground-muted hover:text-foreground">
          {$language === 'es' ? 'Cancelar' : 'Cancel'}
        </button>
      </div>
    {/if}

    {#if !totpStatus.loading && totpStatus.enabled && !totpDisable.active && !totpSetup.showCodes}
      <button
        onclick={() => (totpDisable.active = true)}
        class="text-sm text-stopped hover:text-stopped/80"
      >
        {$language === 'es' ? 'Desactivar 2FA' : 'Disable 2FA'}
      </button>
    {/if}

    {#if totpDisable.active}
      <div class="space-y-3">
        <p class="text-xs text-foreground-muted">
          {$language === 'es' ? 'Ingresa tu contraseña para desactivar 2FA:' : 'Enter your password to disable 2FA:'}
        </p>
        {#if totpDisable.error}
          <div class="p-2 bg-stopped/10 border border-stopped/30 rounded text-stopped text-xs">{totpDisable.error}</div>
        {/if}
        <div class="flex gap-2">
          <input
            type="password"
            placeholder={$language === 'es' ? 'Tu contraseña actual' : 'Your current password'}
            bind:value={totpDisable.password}
            class="flex-1 px-3 py-2 bg-background border border-border rounded-lg text-foreground text-sm"
          />
          <button
            onclick={confirmTotpDisable}
            disabled={totpDisable.loading || !totpDisable.password}
            class="px-4 py-2 bg-stopped text-white rounded-lg text-sm hover:bg-stopped/90 disabled:opacity-50"
          >
            {$language === 'es' ? 'Desactivar' : 'Disable'}
          </button>
        </div>
        <button onclick={() => (totpDisable.active = false)} class="text-xs text-foreground-muted hover:text-foreground">
          {$language === 'es' ? 'Cancelar' : 'Cancel'}
        </button>
      </div>
    {/if}
  </div>
</div>
