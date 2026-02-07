<script lang="ts">
  import { auth } from "$lib/stores/auth";
  import { language, translations } from "$lib/stores/docker";
  import { API_BASE } from "$lib/api/docker";
  import {
    User,
    Lock,
    Eye,
    EyeOff,
    AlertCircle,
    Loader2,
    Mail,
    ArrowLeft,
    Check,
    Key,
    Shield,
  } from "lucide-svelte";

  let username = $state("");
  let password = $state("");
  let rememberMe = $state(false);
  let showPassword = $state(false);
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  // 2FA State
  let requires2FA = $state(false);
  let totpCode = $state("");
  let useRecoveryCode = $state(false);
  let recoveryCode = $state("");

  // Forgot password state
  type ForgotView =
    | "login"
    | "request"
    | "verify-code"
    | "new-password"
    | "success";
  let forgotView = $state<ForgotView>("login");
  let forgotUsername = $state("");
  let forgotEmail = $state("");
  let maskedEmail = $state("");
  let resetCode = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let codeVerified = $state(false);

  let forgotLoading = $state(false);
  let forgotError = $state<string | null>(null);

  // Get translations
  let t = $derived(translations[$language]);

  // Login translations
  const loginText = {
    es: {
      title: "Bienvenido a DockerVerse",
      subtitle: "Gestión Multi-Host de Docker",
      username: "Usuario",
      password: "Contraseña",
      rememberMe: "Recordarme",
      login: "Iniciar Sesión",
      loggingIn: "Iniciando sesión...",
      forgotPassword: "¿Olvidaste tu contraseña?",
      errorInvalid: "Usuario o contraseña incorrectos",
      errorNetwork: "Error de conexión. Intenta de nuevo.",
      demo: "Demo: admin / admin",
      // Forgot password
      forgotTitle: "Recuperar Contraseña",
      enterUsername: "Ingresa tu nombre de usuario",
      sendCode: "Enviar Código",
      codeSentTo: "Código enviado a",
      enterCode: "Ingresa el código de 6 dígitos",
      verifyCode: "Verificar Código",
      codeVerified: "¡Código verificado!",
      newPassword: "Nueva contraseña",
      confirmPassword: "Confirmar contraseña",
      resetPassword: "Restablecer Contraseña",
      passwordMismatch: "Las contraseñas no coinciden",
      invalidCode: "Código inválido o expirado",
      userNotFound: "Usuario no encontrado o sin email",
      successTitle: "¡Contraseña Actualizada!",
      successMessage: "Ya puedes iniciar sesión con tu nueva contraseña",
      backToLogin: "Volver al inicio de sesión",
      // 2FA
      twoFactorTitle: "Verificación de Dos Factores",
      twoFactorSubtitle: "Ingresa el código de tu app de autenticación",
      enterTotpCode: "Código de 6 dígitos",
      verify: "Verificar",
      useRecoveryCode: "Usar código de recuperación",
      useAuthApp: "Usar app de autenticación",
      recoveryCodePlaceholder: "Código de recuperación",
      invalidTotpCode: "Código de verificación inválido",
    },
    en: {
      title: "Welcome to DockerVerse",
      subtitle: "Multi-Host Docker Management",
      username: "Username",
      password: "Password",
      rememberMe: "Remember me",
      login: "Sign In",
      loggingIn: "Signing in...",
      forgotPassword: "Forgot password?",
      errorInvalid: "Invalid username or password",
      errorNetwork: "Connection error. Please try again.",
      demo: "Demo: admin / admin",
      // Forgot password
      forgotTitle: "Password Recovery",
      enterUsername: "Enter your username",
      sendCode: "Send Code",
      codeSentTo: "Code sent to",
      enterCode: "Enter the 6-digit code",
      verifyCode: "Verify Code",
      codeVerified: "Code verified!",
      newPassword: "New password",
      confirmPassword: "Confirm password",
      resetPassword: "Reset Password",
      passwordMismatch: "Passwords do not match",
      invalidCode: "Invalid or expired code",
      userNotFound: "User not found or no email configured",
      successTitle: "Password Updated!",
      successMessage: "You can now sign in with your new password",
      backToLogin: "Back to login",
      // 2FA
      twoFactorTitle: "Two-Factor Verification",
      twoFactorSubtitle: "Enter the code from your authenticator app",
      enterTotpCode: "6-digit code",
      verify: "Verify",
      useRecoveryCode: "Use recovery code",
      useAuthApp: "Use authenticator app",
      recoveryCodePlaceholder: "Recovery code",
      invalidTotpCode: "Invalid verification code",
    },
  };

  let lt = $derived(loginText[$language]);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = null;
    isLoading = true;

    try {
      const result = await auth.login({
        username,
        password,
        rememberMe,
        totpCode: requires2FA ? totpCode : undefined,
        recoveryCode: requires2FA && useRecoveryCode ? recoveryCode : undefined,
      });

      if (result.requiresTOTP) {
        requires2FA = true;
        isLoading = false;
        return;
      }

      if (!result.success) {
        if (requires2FA) {
          error = lt.invalidTotpCode;
        } else {
          error = result.error || lt.errorInvalid;
        }
      }
    } catch (e) {
      error = lt.errorNetwork;
    } finally {
      isLoading = false;
    }
  }

  function handleBack2FA() {
    requires2FA = false;
    totpCode = "";
    recoveryCode = "";
    useRecoveryCode = false;
    error = null;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && username && password && !isLoading) {
      handleSubmit(e);
    }
  }

  async function handleRequestReset() {
    forgotError = null;
    forgotLoading = true;

    try {
      const res = await fetch(`${API_BASE}/api/auth/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: forgotUsername }),
      });

      const data = await res.json();

      if (data.maskedEmail) {
        maskedEmail = data.maskedEmail;
        forgotEmail = data.email;
        forgotView = "verify-code";
      } else {
        forgotError = lt.userNotFound;
      }
    } catch (e) {
      forgotError = lt.errorNetwork;
    } finally {
      forgotLoading = false;
    }
  }

  async function handleVerifyCode() {
    forgotError = null;
    forgotLoading = true;

    try {
      const res = await fetch(`${API_BASE}/api/auth/verify-code`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: forgotEmail, code: resetCode }),
      });

      const data = await res.json();

      if (data.valid) {
        codeVerified = true;
        forgotView = "new-password";
      } else {
        forgotError = lt.invalidCode;
      }
    } catch (e) {
      forgotError = lt.errorNetwork;
    } finally {
      forgotLoading = false;
    }
  }

  async function handleResetPassword() {
    forgotError = null;

    if (newPassword !== confirmPassword) {
      forgotError = lt.passwordMismatch;
      return;
    }

    forgotLoading = true;

    try {
      const res = await fetch(`${API_BASE}/api/auth/reset-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: forgotEmail,
          code: resetCode,
          newPassword: newPassword,
        }),
      });

      const data = await res.json();

      if (data.success) {
        forgotView = "success";
      } else {
        forgotError = lt.invalidCode;
      }
    } catch (e) {
      forgotError = lt.errorNetwork;
    } finally {
      forgotLoading = false;
    }
  }

  function resetForgotState() {
    forgotView = "login";
    forgotUsername = "";
    forgotEmail = "";
    maskedEmail = "";
    resetCode = "";
    newPassword = "";
    confirmPassword = "";
    forgotError = null;
    codeVerified = false;
  }
</script>

<!-- svelte-ignore a11y_label_has_associated_control -->
<div class="min-h-screen bg-background flex items-center justify-center p-4">
  <!-- Background decoration -->
  <div class="fixed inset-0 overflow-hidden pointer-events-none">
    <div
      class="absolute -top-40 -right-40 w-80 h-80 bg-primary/10 rounded-full blur-3xl"
    ></div>
    <div
      class="absolute -bottom-40 -left-40 w-80 h-80 bg-accent-cyan/10 rounded-full blur-3xl"
    ></div>
  </div>

  <!-- Login Card -->
  <div class="w-full max-w-md relative z-10">
    <!-- Logo -->
    <div class="text-center mb-8">
      <div
        class="inline-flex items-center justify-center w-20 h-20 bg-background-secondary rounded-2xl shadow-xl mb-4"
      >
        <span class="text-5xl">🐳</span>
      </div>
      <h1 class="text-2xl font-bold text-foreground">
        {forgotView === "login" ? lt.title : lt.forgotTitle}
      </h1>
      <p class="text-foreground-muted mt-1">{lt.subtitle}</p>
    </div>

    <!-- Form Card -->
    <div class="card p-8 shadow-2xl">
      {#if forgotView === "login"}
        {#if requires2FA}
          <!-- 2FA Verification Form -->
          <form onsubmit={handleSubmit} class="space-y-6">
            <!-- Back button -->
            <button
              type="button"
              onclick={handleBack2FA}
              class="flex items-center gap-2 text-foreground-muted hover:text-foreground transition-colors"
            >
              <ArrowLeft class="w-4 h-4" />
              {lt.backToLogin}
            </button>

            <!-- Header -->
            <div class="text-center">
              <div
                class="w-16 h-16 mx-auto bg-primary/10 rounded-full flex items-center justify-center mb-4"
              >
                <Shield class="w-8 h-8 text-primary" />
              </div>
              <h2 class="text-xl font-bold text-foreground">
                {lt.twoFactorTitle}
              </h2>
              <p class="text-foreground-muted text-sm mt-1">
                {lt.twoFactorSubtitle}
              </p>
            </div>

            <!-- Error Message -->
            {#if error}
              <div
                class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
              >
                <AlertCircle class="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            {/if}

            {#if !useRecoveryCode}
              <!-- TOTP Code Input -->
              <div class="space-y-2">
                <label
                  for="totpCode"
                  class="block text-sm font-medium text-foreground"
                >
                  {lt.enterTotpCode}
                </label>
                <div class="relative">
                  <Key
                    class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
                  />
                  <input
                    id="totpCode"
                    type="text"
                    bind:value={totpCode}
                    maxlength="6"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
                           text-foreground text-center text-2xl tracking-[0.5em] font-mono
                           placeholder:text-foreground-muted placeholder:tracking-normal placeholder:text-base
                           focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary
                           transition-all duration-200"
                    placeholder="000000"
                    required
                  />
                </div>
              </div>

              <!-- Use Recovery Code Link -->
              <button
                type="button"
                onclick={() => {
                  useRecoveryCode = true;
                  error = null;
                }}
                class="text-sm text-primary hover:text-primary/80 transition-colors"
              >
                {lt.useRecoveryCode}
              </button>
            {:else}
              <!-- Recovery Code Input -->
              <div class="space-y-2">
                <label
                  for="recoveryCode"
                  class="block text-sm font-medium text-foreground"
                >
                  {lt.recoveryCodePlaceholder}
                </label>
                <div class="relative">
                  <Key
                    class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
                  />
                  <input
                    id="recoveryCode"
                    type="text"
                    bind:value={recoveryCode}
                    class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
                           text-foreground text-center font-mono tracking-wider
                           placeholder:text-foreground-muted
                           focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary
                           transition-all duration-200"
                    placeholder="xxxxxxxxxxxxxxxx"
                    required
                  />
                </div>
              </div>

              <!-- Use Auth App Link -->
              <button
                type="button"
                onclick={() => {
                  useRecoveryCode = false;
                  error = null;
                }}
                class="text-sm text-primary hover:text-primary/80 transition-colors"
              >
                {lt.useAuthApp}
              </button>
            {/if}

            <!-- Submit Button -->
            <button
              type="submit"
              disabled={isLoading ||
                (!totpCode && !recoveryCode) ||
                (totpCode.length !== 6 && !useRecoveryCode)}
              class="w-full py-3 bg-primary text-white rounded-lg font-medium
                     hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-all duration-200 flex items-center justify-center gap-2"
            >
              {#if isLoading}
                <Loader2 class="w-5 h-5 animate-spin" />
              {:else}
                <Shield class="w-5 h-5" />
              {/if}
              {isLoading ? lt.loggingIn : lt.verify}
            </button>
          </form>
        {:else}
          <!-- Login Form -->
          <form onsubmit={handleSubmit} class="space-y-6">
            <!-- Error Message -->
            {#if error}
              <div
                class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
              >
                <AlertCircle class="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            {/if}

            <!-- Username -->
            <div class="space-y-2">
              <label
                for="username"
                class="block text-sm font-medium text-foreground"
              >
                {lt.username}
              </label>
              <div class="relative">
                <User
                  class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
                />
                <input
                  id="username"
                  type="text"
                  bind:value={username}
                  onkeydown={handleKeydown}
                  class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
							       text-foreground placeholder:text-foreground-muted
							       focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary
							       transition-all duration-200"
                  placeholder={lt.username.toLowerCase()}
                  required
                  autocomplete="username"
                />
              </div>
            </div>

            <!-- Password -->
            <div class="space-y-2">
              <label
                for="password"
                class="block text-sm font-medium text-foreground"
              >
                {lt.password}
              </label>
              <div class="relative">
                <Lock
                  class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
                />
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  bind:value={password}
                  onkeydown={handleKeydown}
                  class="w-full pl-10 pr-12 py-3 bg-background-secondary border border-border rounded-lg
							       text-foreground placeholder:text-foreground-muted
							       focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary
							       transition-all duration-200"
                  placeholder="••••••••"
                  required
                  autocomplete="current-password"
                />
                <button
                  type="button"
                  onclick={() => (showPassword = !showPassword)}
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-foreground-muted hover:text-foreground transition-colors"
                >
                  {#if showPassword}
                    <EyeOff class="w-5 h-5" />
                  {:else}
                    <Eye class="w-5 h-5" />
                  {/if}
                </button>
              </div>
            </div>

            <!-- Remember Me & Forgot Password -->
            <div class="flex items-center justify-between">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={rememberMe}
                  class="w-4 h-4 rounded border-border text-primary focus:ring-primary/50 bg-background-secondary"
                />
                <span class="text-sm text-foreground-muted"
                  >{lt.rememberMe}</span
                >
              </label>
              <button
                type="button"
                onclick={() => {
                  forgotView = "request";
                }}
                class="text-sm text-primary hover:text-primary/80 transition-colors"
              >
                {lt.forgotPassword}
              </button>
            </div>

            <!-- Submit Button -->
            <button
              type="submit"
              disabled={isLoading || !username || !password}
              class="w-full py-3 px-4 bg-primary hover:bg-primary/90 disabled:bg-primary/50
						     text-white font-medium rounded-lg
						     flex items-center justify-center gap-2
						     transition-all duration-200 disabled:cursor-not-allowed"
            >
              {#if isLoading}
                <Loader2 class="w-5 h-5 animate-spin" />
                <span>{lt.loggingIn}</span>
              {:else}
                <span>{lt.login}</span>
              {/if}
            </button>
          </form>
        {/if}

        <!-- Demo credentials hint -->
        <div class="mt-6 text-center">
          <p class="text-xs text-foreground-muted">{lt.demo}</p>
        </div>
      {:else if forgotView === "request"}
        <!-- Request Reset Code -->
        <div class="space-y-6">
          <button
            onclick={resetForgotState}
            class="flex items-center gap-1 text-sm text-foreground-muted hover:text-foreground transition-colors"
          >
            <ArrowLeft class="w-4 h-4" />
            {lt.backToLogin}
          </button>

          {#if forgotError}
            <div
              class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
            >
              <AlertCircle class="w-4 h-4 flex-shrink-0" />
              <span>{forgotError}</span>
            </div>
          {/if}

          <div class="space-y-2">
            <label class="block text-sm font-medium text-foreground">
              {lt.enterUsername}
            </label>
            <div class="relative">
              <User
                class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
              />
              <input
                type="text"
                bind:value={forgotUsername}
                class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
							     text-foreground placeholder:text-foreground-muted
							     focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                placeholder={lt.username.toLowerCase()}
              />
            </div>
          </div>

          <button
            onclick={handleRequestReset}
            disabled={forgotLoading || !forgotUsername}
            class="w-full py-3 px-4 bg-primary hover:bg-primary/90 disabled:bg-primary/50
						     text-white font-medium rounded-lg flex items-center justify-center gap-2
						     transition-all duration-200 disabled:cursor-not-allowed"
          >
            {#if forgotLoading}
              <Loader2 class="w-5 h-5 animate-spin" />
            {:else}
              <Mail class="w-5 h-5" />
            {/if}
            <span>{lt.sendCode}</span>
          </button>
        </div>
      {:else if forgotView === "verify-code"}
        <!-- Step 2: Verify Reset Code -->
        <div class="space-y-6">
          <button
            onclick={resetForgotState}
            class="flex items-center gap-1 text-sm text-foreground-muted hover:text-foreground transition-colors"
          >
            <ArrowLeft class="w-4 h-4" />
            {lt.backToLogin}
          </button>

          <!-- Masked Email Display -->
          <div
            class="p-4 bg-background rounded-lg border border-border text-center"
          >
            <Mail class="w-8 h-8 text-primary mx-auto mb-2" />
            <p class="text-sm text-foreground-muted">{lt.codeSentTo}</p>
            <p class="text-lg font-mono font-medium text-foreground">
              {maskedEmail}
            </p>
          </div>

          {#if forgotError}
            <div
              class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
            >
              <AlertCircle class="w-4 h-4 flex-shrink-0" />
              <span>{forgotError}</span>
            </div>
          {/if}

          <!-- Reset Code Input -->
          <div class="space-y-2">
            <label class="block text-sm font-medium text-foreground">
              {lt.enterCode}
            </label>
            <input
              type="text"
              bind:value={resetCode}
              maxlength="6"
              class="w-full py-3 text-center text-2xl font-mono tracking-widest bg-background-secondary
                     border border-border rounded-lg text-foreground
                     focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
              placeholder="000000"
            />
          </div>

          <button
            onclick={handleVerifyCode}
            disabled={forgotLoading || resetCode.length !== 6}
            class="w-full py-3 px-4 bg-primary hover:bg-primary/90 disabled:bg-primary/50
						     text-white font-medium rounded-lg flex items-center justify-center gap-2
						     transition-all duration-200 disabled:cursor-not-allowed"
          >
            {#if forgotLoading}
              <Loader2 class="w-5 h-5 animate-spin" />
            {:else}
              <Key class="w-5 h-5" />
            {/if}
            <span>{lt.verifyCode}</span>
          </button>
        </div>
      {:else if forgotView === "new-password"}
        <!-- Step 3: Enter New Password (after code verified) -->
        <div class="space-y-6">
          <button
            onclick={resetForgotState}
            class="flex items-center gap-1 text-sm text-foreground-muted hover:text-foreground transition-colors"
          >
            <ArrowLeft class="w-4 h-4" />
            {lt.backToLogin}
          </button>

          <!-- Code Verified Success -->
          <div
            class="p-4 bg-running/10 rounded-lg border border-running/30 text-center"
          >
            <Check class="w-8 h-8 text-running mx-auto mb-2" />
            <p class="text-sm font-medium text-running">{lt.codeVerified}</p>
          </div>

          {#if forgotError}
            <div
              class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
            >
              <AlertCircle class="w-4 h-4 flex-shrink-0" />
              <span>{forgotError}</span>
            </div>
          {/if}

          <!-- New Password -->
          <div class="space-y-2">
            <label class="block text-sm font-medium text-foreground">
              {lt.newPassword}
            </label>
            <div class="relative">
              <Lock
                class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
              />
              <input
                type="password"
                bind:value={newPassword}
                class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
							     text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                placeholder="••••••••"
              />
            </div>
          </div>

          <!-- Confirm Password -->
          <div class="space-y-2">
            <label class="block text-sm font-medium text-foreground">
              {lt.confirmPassword}
            </label>
            <div class="relative">
              <Lock
                class="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted"
              />
              <input
                type="password"
                bind:value={confirmPassword}
                class="w-full pl-10 pr-4 py-3 bg-background-secondary border border-border rounded-lg
							     text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                placeholder="••••••••"
              />
            </div>
          </div>

          <button
            onclick={handleResetPassword}
            disabled={forgotLoading || !newPassword || !confirmPassword}
            class="w-full py-3 px-4 bg-primary hover:bg-primary/90 disabled:bg-primary/50
						     text-white font-medium rounded-lg flex items-center justify-center gap-2
						     transition-all duration-200 disabled:cursor-not-allowed"
          >
            {#if forgotLoading}
              <Loader2 class="w-5 h-5 animate-spin" />
            {/if}
            <span>{lt.resetPassword}</span>
          </button>
        </div>
      {:else if forgotView === "success"}
        <!-- Success -->
        <div class="space-y-6 text-center">
          <div
            class="inline-flex items-center justify-center w-16 h-16 bg-running/20 rounded-full mx-auto"
          >
            <Check class="w-8 h-8 text-running" />
          </div>
          <div>
            <h3 class="text-lg font-semibold text-foreground">
              {lt.successTitle}
            </h3>
            <p class="text-sm text-foreground-muted mt-1">
              {lt.successMessage}
            </p>
          </div>
          <button
            onclick={resetForgotState}
            class="w-full py-3 px-4 bg-primary hover:bg-primary/90 text-white font-medium rounded-lg"
          >
            {lt.backToLogin}
          </button>
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="mt-8 text-center text-sm text-foreground-muted">
      <p>DockerVerse v1.0.0</p>
    </div>
  </div>
</div>

<style>
  /* Custom checkbox styling */
  input[type="checkbox"] {
    -webkit-appearance: none;
    appearance: none;
    background-color: var(--color-background-secondary);
    margin: 0;
    font: inherit;
    width: 1.15em;
    height: 1.15em;
    border: 1px solid var(--color-border);
    border-radius: 0.25em;
    display: grid;
    place-content: center;
    cursor: pointer;
  }

  input[type="checkbox"]::before {
    content: "";
    width: 0.65em;
    height: 0.65em;
    transform: scale(0);
    transition: 120ms transform ease-in-out;
    box-shadow: inset 1em 1em var(--color-primary);
    transform-origin: bottom left;
    clip-path: polygon(14% 44%, 0 65%, 50% 100%, 100% 16%, 80% 0%, 43% 62%);
  }

  input[type="checkbox"]:checked::before {
    transform: scale(1);
  }
</style>
