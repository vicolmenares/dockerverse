<script lang="ts">
  import {
    Lock,
    Shield,
    Check,
    Key,
    LogOut,
    RefreshCw,
    Settings,
    AlertTriangle,
    Clock,
    Server,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import {
    auth,
    currentUser,
    getAutoLogoutMinutes,
    setAutoLogoutMinutes,
  } from '$lib/stores/auth';
  import { API_BASE } from '$lib/api/docker';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);
  let isAdmin = $derived($currentUser?.roles?.includes('admin') ?? false);

  // ── Admin: Auth Config ──────────────────────────────────────────────────────
  type AuthConfig = {
    authEnabled: boolean;
    sessionTimeoutSecs: number;
    maxLoginAttempts: number;
    lockoutDurationSecs: number;
    defaultProvider: string;
  };

  let authConfig = $state<AuthConfig>({
    authEnabled: true,
    sessionTimeoutSecs: 86400,
    maxLoginAttempts: 5,
    lockoutDurationSecs: 900,
    defaultProvider: 'local',
  });
  let authConfigLoading = $state(false);
  let authConfigSaved = $state(false);
  let authConfigError = $state<string | null>(null);

  async function loadAuthConfig() {
    if (!isAdmin) return;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/auth`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) authConfig = await res.json();
    } catch {
      // non-critical
    }
  }

  async function saveAuthConfig() {
    authConfigLoading = true;
    authConfigError = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/auth`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(authConfig),
      });
      if (res.ok) {
        authConfigSaved = true;
        setTimeout(() => (authConfigSaved = false), 3000);
      } else {
        const err = await res.json();
        authConfigError = err.error || 'Failed to save';
      }
    } catch {
      authConfigError = 'Connection error';
    } finally {
      authConfigLoading = false;
    }
  }

  // ── Admin: LDAP Config ───────────────────────────────────────────────────────
  type LdapConfig = {
    enabled: boolean;
    serverURL: string;
    bindDN: string;
    bindPassword: string;
    baseDN: string;
    userFilter: string;
    usernameAttr: string;
    emailAttr: string;
    displayNameAttr: string;
    groupBaseDN: string;
    groupFilter: string;
    adminGroup: string;
    autoCreateUsers: boolean;
    tlsEnabled: boolean;
    startTLS: boolean;
  };

  let ldapConfig = $state<LdapConfig>({
    enabled: false,
    serverURL: '',
    bindDN: '',
    bindPassword: '',
    baseDN: '',
    userFilter: '(uid=%s)',
    usernameAttr: 'uid',
    emailAttr: 'mail',
    displayNameAttr: 'cn',
    groupBaseDN: '',
    groupFilter: '',
    adminGroup: '',
    autoCreateUsers: false,
    tlsEnabled: false,
    startTLS: false,
  });
  let ldapLoading = $state(false);
  let ldapSaved = $state(false);
  let ldapError = $state<string | null>(null);
  let ldapTestResult = $state<{ success: boolean; message: string } | null>(null);
  let ldapTesting = $state(false);
  let ldapShowAdvanced = $state(false);

  async function loadLdapConfig() {
    if (!isAdmin) return;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/ldap`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        ldapConfig = { ...ldapConfig, ...data, bindPassword: '' };
      }
    } catch {
      // non-critical
    }
  }

  async function saveLdapConfig() {
    ldapLoading = true;
    ldapError = null;
    ldapSaved = false;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/ldap`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(ldapConfig),
      });
      if (res.ok) {
        ldapSaved = true;
        ldapConfig.bindPassword = '';
        setTimeout(() => (ldapSaved = false), 3000);
      } else {
        const err = await res.json();
        ldapError = err.error || ($language === 'es' ? 'Error al guardar' : 'Failed to save');
      }
    } catch {
      ldapError = $language === 'es' ? 'Error de conexión' : 'Connection error';
    } finally {
      ldapLoading = false;
    }
  }

  async function testLdapConnection() {
    ldapTesting = true;
    ldapTestResult = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/ldap/test`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok && data.success) {
        ldapTestResult = { success: true, message: data.message || ($language === 'es' ? 'Conexión exitosa' : 'Connection successful') };
      } else {
        ldapTestResult = { success: false, message: data.error || ($language === 'es' ? 'Falló la conexión' : 'Connection failed') };
      }
    } catch {
      ldapTestResult = { success: false, message: $language === 'es' ? 'Error de conexión' : 'Connection error' };
    } finally {
      ldapTesting = false;
    }
  }

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

  // ── Admin: OIDC Config ───────────────────────────────────────────────────────
  type OidcConfig = {
    enabled: boolean;
    providerURL: string;
    clientId: string;
    clientSecret: string;
    scopes: string[];
    redirectURL: string;
    autoCreateUsers: boolean;
    adminGroupClaim: string;
    adminGroupValue: string;
  };

  let oidcConfig = $state<OidcConfig>({
    enabled: false,
    providerURL: '',
    clientId: '',
    clientSecret: '',
    scopes: ['openid', 'email', 'profile'],
    redirectURL: '',
    autoCreateUsers: false,
    adminGroupClaim: '',
    adminGroupValue: '',
  });
  let oidcLoading = $state(false);
  let oidcSaved = $state(false);
  let oidcError = $state<string | null>(null);

  async function loadOidcConfig() {
    if (!isAdmin) return;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/oidc`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        oidcConfig = { ...oidcConfig, ...data, clientSecret: '' };
      }
    } catch { /* non-critical */ }
  }

  async function saveOidcConfig() {
    oidcLoading = true;
    oidcError = null;
    const token = localStorage.getItem('auth_access_token');
    try {
      const res = await fetch(`${API_BASE}/api/settings/oidc`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(oidcConfig),
      });
      if (res.ok) {
        oidcSaved = true;
        oidcConfig.clientSecret = '';
        setTimeout(() => (oidcSaved = false), 3000);
      } else {
        const err = await res.json();
        oidcError = err.error || 'Failed to save';
      }
    } catch {
      oidcError = 'Connection error';
    } finally {
      oidcLoading = false;
    }
  }

  // Load on mount
  $effect(() => {
    loadTOTPStatus();
    loadAuthConfig();
    loadLdapConfig();
    loadOidcConfig();
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_label_has_associated_control -->
<div class="p-4 space-y-6">

  <!-- Admin: Auth Configuration (admin only) -->
  {#if isAdmin}
    <div class="space-y-4">
      <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
        <Settings class="w-5 h-5 text-primary" />
        {$language === 'es' ? 'Configuración de Autenticación' : 'Authentication Configuration'}
      </h3>

      {#if authConfigSaved}
        <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm">
          <Check class="w-4 h-4" />
          {$language === 'es' ? 'Guardado correctamente' : 'Saved successfully'}
        </div>
      {/if}

      {#if authConfigError}
        <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
          <AlertTriangle class="w-4 h-4" />
          {authConfigError}
        </div>
      {/if}

      <!-- Auth enabled toggle -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div>
          <p class="text-sm font-medium text-foreground">
            {$language === 'es' ? 'Autenticación requerida' : 'Require authentication'}
          </p>
          <p class="text-xs text-foreground-muted">
            {$language === 'es'
              ? 'Proteger la aplicación con inicio de sesión'
              : 'Protect the application with a login screen'}
          </p>
        </div>
        <button
          class="relative w-11 h-6 rounded-full transition-colors {authConfig.authEnabled ? 'bg-primary' : 'bg-background-tertiary'}"
          onclick={() => (authConfig.authEnabled = !authConfig.authEnabled)}
          aria-label={authConfig.authEnabled ? 'Disable auth' : 'Enable auth'}
        >
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {authConfig.authEnabled ? 'translate-x-5' : ''}"></span>
        </button>
      </div>

      <!-- Session timeout -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div class="flex items-center gap-2 mb-2">
          <Clock class="w-4 h-4 text-foreground-muted" />
          <p class="text-sm font-medium text-foreground">
            {$language === 'es' ? 'Tiempo de sesión (horas)' : 'Session timeout (hours)'}
          </p>
        </div>
        <div class="flex gap-2">
          {#each [1, 8, 24, 72, 168] as hours}
            <button
              class="flex-1 px-2 py-1.5 text-sm rounded-lg transition-colors {authConfig.sessionTimeoutSecs === hours * 3600 ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
              onclick={() => (authConfig.sessionTimeoutSecs = hours * 3600)}
            >
              {hours}h
            </button>
          {/each}
        </div>
      </div>

      <!-- Login attempts -->
      <div class="grid grid-cols-2 gap-3">
        <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <p class="text-sm font-medium text-foreground mb-2">
            {$language === 'es' ? 'Intentos máximos' : 'Max login attempts'}
          </p>
          <div class="flex gap-1">
            {#each [3, 5, 10] as n}
              <button
                class="flex-1 py-1.5 text-sm rounded-lg transition-colors {authConfig.maxLoginAttempts === n ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
                onclick={() => (authConfig.maxLoginAttempts = n)}
              >
                {n}
              </button>
            {/each}
          </div>
        </div>

        <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <p class="text-sm font-medium text-foreground mb-2">
            {$language === 'es' ? 'Bloqueo (minutos)' : 'Lockout (minutes)'}
          </p>
          <div class="flex gap-1">
            {#each [5, 15, 30] as min}
              <button
                class="flex-1 py-1.5 text-sm rounded-lg transition-colors {authConfig.lockoutDurationSecs === min * 60 ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
                onclick={() => (authConfig.lockoutDurationSecs = min * 60)}
              >
                {min}m
              </button>
            {/each}
          </div>
        </div>
      </div>

      <!-- Default provider -->
      <div class="p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <p class="text-sm font-medium text-foreground mb-2">
          {$language === 'es' ? 'Proveedor predeterminado' : 'Default provider'}
        </p>
        <div class="flex gap-2">
          {#each [{ id: 'local', label: $language === 'es' ? 'Local' : 'Local' }, { id: 'ldap', label: 'LDAP' }, { id: 'oidc', label: 'OIDC' }] as p}
            <button
              class="flex-1 py-1.5 text-sm rounded-lg transition-colors {authConfig.defaultProvider === p.id ? 'bg-primary text-white' : 'bg-background text-foreground-muted hover:bg-background-tertiary'}"
              onclick={() => (authConfig.defaultProvider = p.id)}
            >
              {p.label}
            </button>
          {/each}
        </div>
      </div>

      <button
        onclick={saveAuthConfig}
        disabled={authConfigLoading}
        class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
      >
        {#if authConfigLoading}
          <RefreshCw class="w-4 h-4 animate-spin" />
          {$language === 'es' ? 'Guardando...' : 'Saving...'}
        {:else}
          {st.save}
        {/if}
      </button>
    </div>

    <hr class="border-border" />
  {/if}

  <!-- LDAP Configuration (admin only) -->
  {#if isAdmin}
    <div class="space-y-4">
      <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
        <Server class="w-5 h-5 text-primary" />
        {$language === 'es' ? 'Configuración LDAP' : 'LDAP Configuration'}
      </h3>

      {#if ldapSaved}
        <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm">
          <Check class="w-4 h-4" />
          {$language === 'es' ? 'Guardado correctamente' : 'Saved successfully'}
        </div>
      {/if}

      {#if ldapError}
        <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
          <AlertTriangle class="w-4 h-4" />
          {ldapError}
        </div>
      {/if}

      {#if ldapTestResult}
        <div class="flex items-center gap-2 p-3 rounded-lg text-sm {ldapTestResult.success ? 'bg-running/10 border border-running/30 text-running' : 'bg-stopped/10 border border-stopped/30 text-stopped'}">
          {#if ldapTestResult.success}
            <Check class="w-4 h-4 shrink-0" />
          {:else}
            <AlertTriangle class="w-4 h-4 shrink-0" />
          {/if}
          {ldapTestResult.message}
        </div>
      {/if}

      <!-- Enable toggle -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div>
          <p class="text-sm font-medium text-foreground">
            {$language === 'es' ? 'Habilitar LDAP' : 'Enable LDAP'}
          </p>
          <p class="text-xs text-foreground-muted">
            {$language === 'es' ? 'Autenticar usuarios mediante un servidor LDAP' : 'Authenticate users via an LDAP server'}
          </p>
        </div>
        <button
          class="relative w-11 h-6 rounded-full transition-colors {ldapConfig.enabled ? 'bg-primary' : 'bg-background-tertiary'}"
          onclick={() => (ldapConfig.enabled = !ldapConfig.enabled)}
          aria-label={ldapConfig.enabled ? 'Disable LDAP' : 'Enable LDAP'}
        >
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {ldapConfig.enabled ? 'translate-x-5' : ''}"></span>
        </button>
      </div>

      {#if ldapConfig.enabled}
        <!-- Server URL -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'URL del servidor' : 'Server URL'}
          </label>
          <input
            type="text"
            bind:value={ldapConfig.serverURL}
            placeholder="ldap://192.168.1.1:389"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Bind DN -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'DN de enlace' : 'Bind DN'}
          </label>
          <input
            type="text"
            bind:value={ldapConfig.bindDN}
            placeholder="cn=admin,dc=example,dc=com"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Bind Password -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'Contraseña de enlace' : 'Bind Password'}
          </label>
          <input
            type="password"
            bind:value={ldapConfig.bindPassword}
            placeholder={$language === 'es' ? 'Dejar vacío para conservar la actual' : 'Leave empty to keep current'}
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Base DN -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'DN base' : 'Base DN'}
          </label>
          <input
            type="text"
            bind:value={ldapConfig.baseDN}
            placeholder="dc=example,dc=com"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
        </div>

        <!-- TLS Enabled toggle -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">
              {$language === 'es' ? 'TLS habilitado' : 'TLS Enabled'}
            </p>
            <p class="text-xs text-foreground-muted">
              {$language === 'es' ? 'Usar ldaps:// para conexión segura' : 'Use ldaps:// for a secure connection'}
            </p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {ldapConfig.tlsEnabled ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (ldapConfig.tlsEnabled = !ldapConfig.tlsEnabled)}
            aria-label={ldapConfig.tlsEnabled ? 'Disable TLS' : 'Enable TLS'}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {ldapConfig.tlsEnabled ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- StartTLS toggle -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">StartTLS</p>
            <p class="text-xs text-foreground-muted">
              {$language === 'es' ? 'Actualizar conexión LDAP sin cifrar a TLS' : 'Upgrade plain LDAP connection to TLS'}
            </p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {ldapConfig.startTLS ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (ldapConfig.startTLS = !ldapConfig.startTLS)}
            aria-label={ldapConfig.startTLS ? 'Disable StartTLS' : 'Enable StartTLS'}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {ldapConfig.startTLS ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Auto-create users toggle -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">
              {$language === 'es' ? 'Crear usuarios automáticamente' : 'Auto-create users'}
            </p>
            <p class="text-xs text-foreground-muted">
              {$language === 'es' ? 'Crear cuenta local al primer inicio de sesión LDAP' : 'Create a local account on first LDAP login'}
            </p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {ldapConfig.autoCreateUsers ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (ldapConfig.autoCreateUsers = !ldapConfig.autoCreateUsers)}
            aria-label={ldapConfig.autoCreateUsers ? 'Disable auto-create' : 'Enable auto-create'}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {ldapConfig.autoCreateUsers ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Advanced fields toggle -->
        <button
          onclick={() => (ldapShowAdvanced = !ldapShowAdvanced)}
          class="w-full py-2 border border-border text-foreground-muted rounded-lg hover:bg-background-tertiary transition-colors text-sm flex items-center justify-center gap-2"
        >
          {ldapShowAdvanced
            ? ($language === 'es' ? 'Ocultar campos avanzados' : 'Hide advanced fields')
            : ($language === 'es' ? 'Mostrar campos avanzados' : 'Show advanced fields')}
        </button>

        {#if ldapShowAdvanced}
          <div class="space-y-3 p-3 rounded-lg bg-background-tertiary/20 border border-border">
            <p class="text-xs font-medium text-foreground-muted uppercase tracking-wide">
              {$language === 'es' ? 'Campos avanzados' : 'Advanced fields'}
            </p>

            <div>
              <label class="block text-sm font-medium text-foreground mb-1">
                {$language === 'es' ? 'Filtro de usuario' : 'User Filter'}
              </label>
              <input
                type="text"
                bind:value={ldapConfig.userFilter}
                placeholder="(uid=%s)"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>

            <div class="grid grid-cols-3 gap-2">
              <div>
                <label class="block text-sm font-medium text-foreground mb-1">
                  {$language === 'es' ? 'Atrib. usuario' : 'Username Attr'}
                </label>
                <input
                  type="text"
                  bind:value={ldapConfig.usernameAttr}
                  placeholder="uid"
                  class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-foreground mb-1">
                  {$language === 'es' ? 'Atrib. email' : 'Email Attr'}
                </label>
                <input
                  type="text"
                  bind:value={ldapConfig.emailAttr}
                  placeholder="mail"
                  class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-foreground mb-1">
                  {$language === 'es' ? 'Atrib. nombre' : 'Display Name Attr'}
                </label>
                <input
                  type="text"
                  bind:value={ldapConfig.displayNameAttr}
                  placeholder="cn"
                  class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
                />
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium text-foreground mb-1">
                {$language === 'es' ? 'DN base de grupos' : 'Group Base DN'}
              </label>
              <input
                type="text"
                bind:value={ldapConfig.groupBaseDN}
                placeholder="ou=groups,dc=example,dc=com"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-foreground mb-1">
                {$language === 'es' ? 'Filtro de grupo' : 'Group Filter'}
              </label>
              <input
                type="text"
                bind:value={ldapConfig.groupFilter}
                placeholder="(member=%s)"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-foreground mb-1">
                {$language === 'es' ? 'Grupo de administradores' : 'Admin Group'}
              </label>
              <input
                type="text"
                bind:value={ldapConfig.adminGroup}
                placeholder="cn=admins,ou=groups,dc=example,dc=com"
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>
          </div>
        {/if}

        <!-- Test Connection + Save buttons -->
        <div class="flex gap-2">
          <button
            onclick={testLdapConnection}
            disabled={ldapTesting || !ldapConfig.serverURL}
            class="flex-1 py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors disabled:opacity-50 flex items-center justify-center gap-2 text-sm"
          >
            {#if ldapTesting}
              <RefreshCw class="w-4 h-4 animate-spin" />
            {:else}
              <Server class="w-4 h-4" />
            {/if}
            {$language === 'es' ? 'Probar conexión' : 'Test Connection'}
          </button>

          <button
            onclick={saveLdapConfig}
            disabled={ldapLoading}
            class="flex-1 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 text-sm"
          >
            {#if ldapLoading}
              <RefreshCw class="w-4 h-4 animate-spin" />
              {$language === 'es' ? 'Guardando...' : 'Saving...'}
            {:else}
              {st.save}
            {/if}
          </button>
        </div>
      {/if}
    </div>

    <hr class="border-border" />
  {/if}

  <!-- OIDC Configuration (admin only) -->
  {#if isAdmin}
    <div class="space-y-4">
      <h3 class="text-lg font-semibold text-foreground flex items-center gap-2">
        <Key class="w-5 h-5 text-primary" />
        {$language === 'es' ? 'Configuración OIDC' : 'OIDC Configuration'}
      </h3>

      {#if oidcSaved}
        <div class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm">
          <Check class="w-4 h-4" />
          {$language === 'es' ? 'Guardado correctamente' : 'Saved successfully'}
        </div>
      {/if}

      {#if oidcError}
        <div class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm">
          <AlertTriangle class="w-4 h-4" />
          {oidcError}
        </div>
      {/if}

      <!-- Enable toggle -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
        <div>
          <p class="text-sm font-medium text-foreground">
            {$language === 'es' ? 'Habilitar OIDC' : 'Enable OIDC'}
          </p>
          <p class="text-xs text-foreground-muted">
            {$language === 'es'
              ? 'Autenticar usuarios mediante un proveedor OIDC (Google, Keycloak, Auth0…)'
              : 'Authenticate users via an OIDC provider (Google, Keycloak, Auth0…)'}
          </p>
        </div>
        <button
          class="relative w-11 h-6 rounded-full transition-colors {oidcConfig.enabled ? 'bg-primary' : 'bg-background-tertiary'}"
          onclick={() => (oidcConfig.enabled = !oidcConfig.enabled)}
          aria-label={oidcConfig.enabled ? 'Disable OIDC' : 'Enable OIDC'}
        >
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {oidcConfig.enabled ? 'translate-x-5' : ''}"></span>
        </button>
      </div>

      {#if oidcConfig.enabled}
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'URL del proveedor' : 'Provider URL'}
          </label>
          <input
            type="url"
            bind:value={oidcConfig.providerURL}
            placeholder="https://accounts.google.com"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
          <p class="text-xs text-foreground-muted mt-1">
            {$language === 'es' ? 'URL base del proveedor (se usará para descubrimiento automático)' : 'Base URL of the provider (used for auto-discovery)'}
          </p>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Client ID</label>
            <input
              type="text"
              bind:value={oidcConfig.clientId}
              placeholder="your-client-id"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">Client Secret</label>
            <input
              type="password"
              bind:value={oidcConfig.clientSecret}
              placeholder={$language === 'es' ? 'Dejar vacío para conservar' : 'Leave empty to keep current'}
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-foreground mb-1">
            {$language === 'es' ? 'URL de redirección (Redirect URI)' : 'Redirect URI'}
          </label>
          <input
            type="url"
            bind:value={oidcConfig.redirectURL}
            placeholder="http://localhost:3007/auth/oidc/callback"
            class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
          />
        </div>

        <!-- Auto-create users -->
        <div class="flex items-center justify-between p-3 rounded-lg bg-background-tertiary/30 border border-border">
          <div>
            <p class="text-sm font-medium text-foreground">
              {$language === 'es' ? 'Crear usuarios automáticamente' : 'Auto-create users'}
            </p>
            <p class="text-xs text-foreground-muted">
              {$language === 'es' ? 'Crear cuenta local al primer inicio de sesión OIDC' : 'Create a local account on first OIDC login'}
            </p>
          </div>
          <button
            class="relative w-11 h-6 rounded-full transition-colors {oidcConfig.autoCreateUsers ? 'bg-primary' : 'bg-background-tertiary'}"
            onclick={() => (oidcConfig.autoCreateUsers = !oidcConfig.autoCreateUsers)}
            aria-label={oidcConfig.autoCreateUsers ? 'Disable auto-create' : 'Enable auto-create'}
          >
            <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform shadow-sm {oidcConfig.autoCreateUsers ? 'translate-x-5' : ''}"></span>
          </button>
        </div>

        <!-- Admin group claim -->
        <div class="grid grid-cols-2 gap-3 p-3 rounded-lg bg-background-tertiary/20 border border-border">
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">
              {$language === 'es' ? 'Claim de grupo admin' : 'Admin group claim'}
            </label>
            <input
              type="text"
              bind:value={oidcConfig.adminGroupClaim}
              placeholder="groups"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1">
              {$language === 'es' ? 'Valor del grupo admin' : 'Admin group value'}
            </label>
            <input
              type="text"
              bind:value={oidcConfig.adminGroupValue}
              placeholder="admins"
              class="w-full px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
            />
          </div>
        </div>

        <button
          onclick={saveOidcConfig}
          disabled={oidcLoading}
          class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {#if oidcLoading}
            <RefreshCw class="w-4 h-4 animate-spin" />
            {$language === 'es' ? 'Guardando...' : 'Saving...'}
          {:else}
            {st.save}
          {/if}
        </button>
      {/if}
    </div>

    <hr class="border-border" />
  {/if}

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
