<script lang="ts">
  import {
    Lock,
    Shield,
    Check,
    Key,
    LogOut,
    RefreshCw,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import {
    auth,
    getAutoLogoutMinutes,
    setAutoLogoutMinutes,
  } from '$lib/stores/auth';
  import { API_BASE } from '$lib/api/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  // TOTP/2FA State
  let totpState = $state({
    enabled: false,
    setupMode: false,
    secret: '',
    qrUrl: '',
    verifyCode: '',
    recoveryCodes: [] as string[],
    recoveryCount: 0,
    loading: false,
    error: null as string | null,
    showRecoveryCodes: false,
    confirmDisable: false,
    disablePassword: '',
  });

  // Auto-logout setting
  let autoLogoutMinutes = $state(getAutoLogoutMinutes());

  // Password change form
  let passwordForm = $state({
    current: '',
    new: '',
    confirm: '',
    error: null as string | null,
    success: false,
    loading: false,
  });

  async function handlePasswordChange() {
    passwordForm.error = null;

    if (passwordForm.new !== passwordForm.confirm) {
      passwordForm.error = st.passwordMismatch;
      return;
    }

    if (passwordForm.new.length < 6) {
      passwordForm.error = st.passwordRequirements;
      return;
    }

    passwordForm.loading = true;

    try {
      const success = await auth.changePassword(passwordForm.current, passwordForm.new);

      if (success) {
        passwordForm.success = true;
        passwordForm.current = '';
        passwordForm.new = '';
        passwordForm.confirm = '';
        setTimeout(() => (passwordForm.success = false), 3000);
      } else {
        passwordForm.error = $language === 'es'
          ? 'Error al cambiar la contraseña. Verifica tu contraseña actual.'
          : 'Failed to change password. Check your current password.';
      }
    } catch (err) {
      passwordForm.error = $language === 'es'
        ? 'Error de conexión. Intenta de nuevo.'
        : 'Connection error. Please try again.';
    } finally {
      passwordForm.loading = false;
    }
  }

  // TOTP Functions
  async function loadTOTPStatus() {
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/status`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        totpState.enabled = data.enabled;
        totpState.recoveryCount = data.recoveryCount || 0;
      }
    } catch (err) {
      console.error('Failed to load TOTP status:', err);
    }
  }

  async function setupTOTP() {
    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem('auth_access_token');

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/setup`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        totpState.secret = data.secret;
        totpState.qrUrl = data.url;
        totpState.setupMode = true;
      } else {
        const err = await res.json();
        totpState.error = err.error || 'Failed to setup 2FA';
      }
    } catch (err) {
      totpState.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      totpState.loading = false;
    }
  }

  async function enableTOTP() {
    if (!totpState.verifyCode || totpState.verifyCode.length !== 6) {
      totpState.error = st.invalidCode;
      return;
    }

    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem('auth_access_token');

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/enable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ code: totpState.verifyCode }),
      });

      if (res.ok) {
        const data = await res.json();
        totpState.enabled = true;
        totpState.setupMode = false;
        totpState.recoveryCodes = data.recoveryCodes || [];
        totpState.showRecoveryCodes = true;
        totpState.verifyCode = '';
        auth.refreshUser();
      } else {
        const err = await res.json();
        totpState.error = err.error || st.invalidCode;
      }
    } catch (err) {
      totpState.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      totpState.loading = false;
    }
  }

  async function disableTOTP() {
    if (!totpState.disablePassword) {
      totpState.error = $language === 'es' ? 'Ingresa tu contraseña' : 'Enter your password';
      return;
    }

    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem('auth_access_token');

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/disable`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ password: totpState.disablePassword }),
      });

      if (res.ok) {
        totpState.enabled = false;
        totpState.confirmDisable = false;
        totpState.disablePassword = '';
        totpState.recoveryCodes = [];
        totpState.recoveryCount = 0;
        auth.refreshUser();
      } else {
        const err = await res.json();
        totpState.error = err.error || 'Failed to disable 2FA';
      }
    } catch (err) {
      totpState.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      totpState.loading = false;
    }
  }

  async function regenerateRecoveryCodes() {
    if (!totpState.disablePassword) {
      totpState.error = $language === 'es' ? 'Ingresa tu contraseña' : 'Enter your password';
      return;
    }

    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem('auth_access_token');

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/regenerate-recovery`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ password: totpState.disablePassword }),
      });

      if (res.ok) {
        const data = await res.json();
        totpState.recoveryCodes = data.recoveryCodes || [];
        totpState.recoveryCount = data.recoveryCodes?.length || 0;
        totpState.showRecoveryCodes = true;
        totpState.disablePassword = '';
      } else {
        const err = await res.json();
        totpState.error = err.error || 'Failed to regenerate codes';
      }
    } catch (err) {
      totpState.error = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      totpState.loading = false;
    }
  }

  function generateQRCodeUrl(otpauthUrl: string): string {
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpauthUrl)}`;
  }

  // Load TOTP status on mount
  $effect(() => {
    loadTOTPStatus();
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_label_has_associated_control -->
<div class="p-4 space-y-6">
  <!-- Auto-Logout Section -->
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
      <LogOut class="w-5 h-5 text-primary" />
      {st.autoLogout}
    </h3>
    <p class="text-sm text-foreground-muted">{st.autoLogoutDesc}</p>
    <div class="grid grid-cols-4 gap-2">
      {#each [5, 10, 15, 30, 60, 120, 0] as minutes}
        <button
          onclick={() => {
            autoLogoutMinutes = minutes;
            setAutoLogoutMinutes(minutes);
          }}
          class="py-2 px-3 rounded-lg border text-sm font-medium transition-all
            {autoLogoutMinutes === minutes
            ? 'border-primary bg-primary/10 text-primary'
            : 'border-border text-foreground-muted hover:border-foreground-muted hover:text-foreground'}"
        >
          {#if minutes === 0}
            {st.autoLogoutDisabled}
          {:else if minutes === 60}
            1 {st.autoLogoutHour}
          {:else if minutes === 120}
            2 {st.autoLogoutHours}
          {:else}
            {minutes} {st.autoLogoutMinutes}
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <hr class="border-border" />

  <!-- Password Section -->
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
      <Lock class="w-5 h-5 text-primary" />
      {st.password}
    </h3>

    {#if passwordForm.success}
      <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm">
        <Check class="w-4 h-4" />
        {st.passwordChanged}
      </div>
    {/if}

    {#if passwordForm.error}
      <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
        {passwordForm.error}
      </div>
    {/if}

    <div>
      <label class="block text-sm font-medium text-foreground mb-1">{st.currentPassword}</label>
      <input
        type="password"
        bind:value={passwordForm.current}
        class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
      />
    </div>
    <div>
      <label class="block text-sm font-medium text-foreground mb-1">{st.newPassword}</label>
      <input
        type="password"
        bind:value={passwordForm.new}
        class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
      />
      <p class="text-xs text-foreground-muted mt-1">{st.passwordRequirements}</p>
    </div>
    <div>
      <label class="block text-sm font-medium text-foreground mb-1">{st.confirmPassword}</label>
      <input
        type="password"
        bind:value={passwordForm.confirm}
        class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
      />
    </div>

    <button
      onclick={handlePasswordChange}
      disabled={!passwordForm.current || !passwordForm.new || !passwordForm.confirm || passwordForm.loading}
      class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
    >
      {#if passwordForm.loading}
        <RefreshCw class="w-4 h-4 animate-spin" />
        {$language === 'es' ? 'Guardando...' : 'Saving...'}
      {:else}
        {st.save}
      {/if}
    </button>
  </div>

  <hr class="border-border" />

  <!-- Two-Factor Authentication Section -->
  <div class="space-y-4">
    <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
      <Shield class="w-5 h-5 text-primary" />
      {st.twoFactorAuth}
    </h3>
    <p class="text-sm text-foreground-muted">{st.twoFactorDesc}</p>

    {#if totpState.error}
      <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
        {totpState.error}
      </div>
    {/if}

    <!-- Status Badge -->
    <div class="flex items-center justify-between p-4 bg-background-tertiary rounded-lg">
      <div class="flex items-center gap-3">
        {#if totpState.enabled}
          <div class="w-10 h-10 bg-running/20 rounded-full flex items-center justify-center">
            <Check class="w-5 h-5 text-running" />
          </div>
          <div>
            <p class="font-medium text-foreground">{st.twoFactorEnabled}</p>
            <p class="text-xs text-foreground-muted">{totpState.recoveryCount} {st.codesRemaining}</p>
          </div>
        {:else}
          <div class="w-10 h-10 bg-foreground-muted/20 rounded-full flex items-center justify-center">
            <Shield class="w-5 h-5 text-foreground-muted" />
          </div>
          <div>
            <p class="font-medium text-foreground">{st.twoFactorDisabled}</p>
          </div>
        {/if}
      </div>
    </div>

    <!-- Setup 2FA Flow -->
    {#if !totpState.enabled}
      {#if !totpState.setupMode}
        <button
          onclick={setupTOTP}
          disabled={totpState.loading}
          class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {#if totpState.loading}
            <RefreshCw class="w-4 h-4 animate-spin" />
          {:else}
            <Key class="w-4 h-4" />
          {/if}
          {st.setup2FA}
        </button>
      {:else}
        <!-- QR Code and Setup -->
        <div class="space-y-4 p-4 bg-background rounded-lg border border-border">
          <p class="text-sm text-foreground-muted text-center">{st.scanQRCode}</p>

          <div class="flex justify-center">
            <img
              src={generateQRCodeUrl(totpState.qrUrl)}
              alt="QR Code"
              class="w-48 h-48 rounded-lg bg-white p-2"
            />
          </div>

          <div class="text-center">
            <p class="text-xs text-foreground-muted mb-2">{st.manualEntry}</p>
            <div class="inline-flex items-center gap-2 px-3 py-2 bg-background-tertiary rounded-lg">
              <code class="text-sm font-mono text-primary select-all">{totpState.secret}</code>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-foreground mb-1">{st.enterCode}</label>
            <input
              type="text"
              bind:value={totpState.verifyCode}
              placeholder="000000"
              maxlength="6"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground text-center text-lg font-mono tracking-widest"
            />
          </div>

          <div class="flex gap-2">
            <button
              onclick={() => {
                totpState.setupMode = false;
                totpState.error = null;
              }}
              class="flex-1 py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors"
            >
              {st.cancel}
            </button>
            <button
              onclick={enableTOTP}
              disabled={totpState.loading || totpState.verifyCode.length !== 6}
              class="flex-1 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {#if totpState.loading}
                <RefreshCw class="w-4 h-4 animate-spin" />
              {/if}
              {st.verify2FA}
            </button>
          </div>
        </div>
      {/if}
    {:else}
      <!-- 2FA Enabled - Show disable option and recovery codes -->
      <div class="space-y-3">
        {#if !totpState.confirmDisable}
          <button
            onclick={() => (totpState.confirmDisable = true)}
            class="w-full py-2 border border-stopped/50 text-stopped rounded-lg hover:bg-stopped/10 transition-colors"
          >
            {st.disable2FA}
          </button>

          <button
            onclick={() => (totpState.showRecoveryCodes = !totpState.showRecoveryCodes)}
            class="w-full py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors"
          >
            {st.regenerateCodes}
          </button>
        {:else}
          <!-- Confirm Disable -->
          <div class="p-4 bg-stopped/5 border border-stopped/30 rounded-lg space-y-3">
            <p class="text-sm text-stopped font-medium">{st.confirmDisableTitle}</p>
            <p class="text-sm text-foreground-muted">{st.confirmDisableDesc}</p>

            <div>
              <label class="block text-sm font-medium text-foreground mb-1">{st.enterPassword}</label>
              <input
                type="password"
                bind:value={totpState.disablePassword}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
              />
            </div>

            <div class="flex gap-2">
              <button
                onclick={() => {
                  totpState.confirmDisable = false;
                  totpState.disablePassword = '';
                  totpState.error = null;
                }}
                class="flex-1 py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors"
              >
                {st.cancel}
              </button>
              <button
                onclick={disableTOTP}
                disabled={totpState.loading || !totpState.disablePassword}
                class="flex-1 py-2 bg-stopped text-white rounded-lg hover:bg-stopped/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
              >
                {#if totpState.loading}
                  <RefreshCw class="w-4 h-4 animate-spin" />
                {/if}
                {st.disable2FA}
              </button>
            </div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Recovery Codes Section -->
    {#if totpState.showRecoveryCodes && totpState.recoveryCodes.length > 0}
      <div class="p-4 bg-primary/5 border border-primary/30 rounded-lg space-y-3">
        <h4 class="font-medium text-foreground">{st.recoveryCodes}</h4>
        <p class="text-xs text-foreground-muted">{st.recoveryCodesDesc}</p>

        <div class="grid grid-cols-2 gap-2">
          {#each totpState.recoveryCodes as code}
            <div class="px-3 py-2 bg-background rounded border border-border font-mono text-sm text-center select-all">
              {code}
            </div>
          {/each}
        </div>

        <button
          onclick={() => {
            totpState.showRecoveryCodes = false;
            totpState.recoveryCodes = [];
          }}
          class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        >
          {$language === 'es' ? 'Entendido, los guardé' : 'Got it, I saved them'}
        </button>
      </div>
    {/if}
  </div>
</div>
