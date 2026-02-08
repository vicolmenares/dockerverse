<script lang="ts">
  import {
    X,
    User,
    Lock,
    Bell,
    Palette,
    Globe,
    Info,
    LogOut,
    ChevronRight,
    Moon,
    Sun,
    Monitor,
    Check,
    Shield,
    Database,
    Activity,
    HardDrive,
    Trash2,
    Download,
    Mail,
    Key,
    Camera,
    Users,
    Plus,
    Pencil,
    Send,
    Settings2,
    Cpu,
    MemoryStick,
    RefreshCw,
    Upload,
  } from "lucide-svelte";
  import { language, translations, type Language } from "$lib/stores/docker";
  import {
    auth,
    currentUser,
    uploadAvatar,
    deleteAvatar,
    getAutoLogoutMinutes,
    setAutoLogoutMinutes,
    AUTO_LOGOUT_OPTIONS,
  } from "$lib/stores/auth";
  import { API_BASE } from "$lib/api/docker";

  // Avatar state
  let avatarState = $state({
    loading: false,
    error: null as string | null,
  });
  let avatarInput: HTMLInputElement | null = $state(null);

  let { onclose }: { onclose: () => void } = $props();

  // Current view
  type SettingsView =
    | "main"
    | "profile"
    | "security"
    | "notifications"
    | "appearance"
    | "language"
    | "data"
    | "about"
    | "users";
  let currentView = $state<SettingsView>("main");

  // TOTP/2FA State
  let totpState = $state({
    enabled: false,
    setupMode: false,
    secret: "",
    qrUrl: "",
    verifyCode: "",
    recoveryCodes: [] as string[],
    recoveryCount: 0,
    loading: false,
    error: null as string | null,
    showRecoveryCodes: false,
    confirmDisable: false,
    disablePassword: "",
  });

  // Users list (loaded from API)
  let usersList = $state<any[]>([]);
  let usersLoading = $state(false);
  let showUserForm = $state(false);
  let editingUser = $state<any>(null);
  let userForm = $state({
    username: "",
    email: "",
    password: "",
    firstName: "",
    lastName: "",
    role: "user",
  });

  // App settings from backend
  let appSettings = $state({
    cpuThreshold: 80,
    memoryThreshold: 80,
    appriseUrl: "https://apprise.nerdslabs.com",
    appriseKey: "dockerverse",
    telegramEnabled: false,
    telegramUrl: "",
    emailEnabled: false,
    notifyOnStop: true,
    notifyOnStart: true,
    notifyOnHighCpu: true,
    notifyOnHighMem: true,
    notifyTags: [] as string[],
  });
  let testingNotification = $state(false);
  let testChannel = $state<"telegram" | "email" | "both">("both");

  // Auto-logout setting
  let autoLogoutMinutes = $state(getAutoLogoutMinutes());

  // Theme
  type Theme = "dark" | "light" | "system";
  let theme = $state<Theme>("dark");

  // Notification settings
  let notifications = $state({
    containerAlerts: true,
    hostOffline: true,
    highCpu: true,
    highMemory: true,
    emailNotifications: false,
    sound: true,
  });

  // Password change form
  let passwordForm = $state({
    current: "",
    new: "",
    confirm: "",
    error: null as string | null,
    success: false,
    loading: false,
  });

  // Profile form state
  let profileForm = $state({
    firstName: "",
    lastName: "",
    email: "",
    loading: false,
    success: false,
    error: null as string | null,
  });

  // Load profile data
  $effect(() => {
    if ($currentUser) {
      profileForm.firstName = $currentUser.firstName || "";
      profileForm.lastName = $currentUser.lastName || "";
      profileForm.email = $currentUser.email || "";
    }
  });

  // Get translations
  let t = $derived(translations[$language]);

  // Settings translations - Complete i18n
  const settingsText = {
    es: {
      settings: "Configuración",
      profile: "Mi Perfil",
      profileDesc: "Gestiona tu información personal",
      password: "Cambiar Contraseña",
      passwordDesc: "Actualiza tu contraseña de acceso",
      notifications: "Notificaciones",
      notificationsDesc: "Configura alertas y umbrales",
      appearance: "Apariencia",
      appearanceDesc: "Personaliza el tema visual",
      language: "Idioma",
      languageDesc: "Selecciona el idioma de la interfaz",
      data: "Datos y Almacenamiento",
      dataDesc: "Gestiona caché y datos locales",
      about: "Acerca de",
      aboutDesc: "Información de la aplicación",
      logout: "Cerrar Sesión",
      back: "Volver",
      save: "Guardar",
      cancel: "Cancelar",
      // Users
      users: "Usuarios",
      usersDesc: "Gestionar usuarios del sistema",
      addUser: "+ Agregar Usuario",
      loading: "Cargando...",
      // Profile
      firstName: "Nombre",
      lastName: "Apellido",
      email: "Correo electrónico",
      username: "Nombre de usuario",
      changeAvatar: "Cambiar avatar",
      // Password
      currentPassword: "Contraseña actual",
      newPassword: "Nueva contraseña",
      confirmPassword: "Confirmar contraseña",
      passwordMismatch: "Las contraseñas no coinciden",
      passwordChanged: "Contraseña actualizada correctamente",
      passwordRequirements: "Mínimo 8 caracteres, una mayúscula y un número",
      // Notifications
      alertThresholds: "Umbrales de Alerta",
      cpuThreshold: "Umbral de CPU",
      memoryThreshold: "Umbral de Memoria",
      containerStopped: "Contenedor detenido",
      containerStoppedDesc: "Notificar cuando un contenedor se detenga",
      containerStarted: "Contenedor iniciado",
      containerStartedDesc: "Notificar cuando un contenedor se inicie",
      highCpu: "CPU Alta",
      highCpuDesc: "Alertar cuando CPU supere el umbral",
      highMemory: "Memoria Alta",
      highMemoryDesc: "Alertar cuando memoria supere el umbral",
      testNotification: "Probar Notificación",
      sending: "Enviando...",
      // Apprise / Notification Channels
      apprise: "Apprise",
      appriseDesc: "Servicios de notificación",
      appriseServer: "Servidor Apprise",
      appriseUrl: "URL de Apprise",
      appriseKey: "Clave (para notificaciones con estado)",
      appriseHelp:
        "Configura las URLs de notificación en tu servidor Apprise. Soporta Telegram, Email, Discord, Slack y más de 100 servicios.",
      sendTestNotification: "Enviar Notificación de Prueba",
      notificationChannels: "Canales de Notificación",
      emailNotifications: "Notificaciones por Email",
      emailNotificationsDesc: "Recibir alertas por correo electrónico",
      telegramNotifications: "Notificaciones de Telegram",
      telegramNotificationsDesc: "Recibir alertas por Telegram",
      telegramUrl: "URL de Telegram (Apprise)",
      telegramUrlPlaceholder: "tgram://token/chat_id",
      telegramUrlHelp: "Formato: tgram://BOT_TOKEN/CHAT_ID",
      testChannelLabel: "Canal de Prueba",
      testTelegram: "Telegram",
      testEmail: "Email",
      testBoth: "Ambos",
      // Appearance
      dark: "Oscuro",
      light: "Claro",
      system: "Sistema",
      themeSelect: "Selecciona un tema",
      // Data
      clearCache: "Limpiar caché",
      clearCacheDesc: "Elimina datos temporales almacenados",
      exportData: "Exportar datos",
      exportDataDesc: "Descarga tu configuración",
      deleteAccount: "Eliminar cuenta",
      deleteAccountDesc: "Elimina permanentemente tu cuenta",
      cacheCleared: "Caché eliminada correctamente",
      // About
      version: "Versión",
      buildDate: "Fecha de build",
      developer: "Desarrollado por",
      license: "Licencia",
      documentation: "Documentación",
      reportBug: "Reportar un problema",
      // Security / 2FA
      security: "Seguridad",
      securityDesc: "Contraseña y autenticación de dos factores",
      twoFactorAuth: "Autenticación de Dos Factores (2FA)",
      twoFactorDesc: "Añade una capa extra de seguridad a tu cuenta",
      twoFactorEnabled: "2FA Activado",
      twoFactorDisabled: "2FA Desactivado",
      enable2FA: "Activar 2FA",
      disable2FA: "Desactivar 2FA",
      setup2FA: "Configurar 2FA",
      verify2FA: "Verificar código",
      scanQRCode: "Escanea el código QR con tu app de autenticación",
      manualEntry: "O ingresa manualmente este código:",
      enterCode: "Ingresa el código de 6 dígitos",
      recoveryCodes: "Códigos de Recuperación",
      recoveryCodesDesc:
        "Guarda estos códigos en un lugar seguro. Úsalos si pierdes acceso a tu app de autenticación.",
      regenerateCodes: "Regenerar Códigos",
      codesRemaining: "códigos restantes",
      invalidCode: "Código inválido",
      confirmDisableTitle: "Desactivar 2FA",
      confirmDisableDesc:
        "¿Estás seguro de que deseas desactivar la autenticación de dos factores?",
      enterPassword: "Ingresa tu contraseña para confirmar",
      // Auto-logout
      autoLogout: "Cierre de sesión automático",
      autoLogoutDesc: "Cerrar sesión después de un período de inactividad",
      autoLogoutDisabled: "Desactivado",
      autoLogoutMinutes: "minutos",
      autoLogoutHour: "hora",
      autoLogoutHours: "horas",
    },
    en: {
      settings: "Settings",
      profile: "My Profile",
      profileDesc: "Manage your personal information",
      password: "Change Password",
      passwordDesc: "Update your access password",
      notifications: "Notifications",
      notificationsDesc: "Configure alerts and thresholds",
      appearance: "Appearance",
      appearanceDesc: "Customize the visual theme",
      language: "Language",
      languageDesc: "Select the interface language",
      data: "Data & Storage",
      dataDesc: "Manage cache and local data",
      about: "About",
      aboutDesc: "Application information",
      logout: "Sign Out",
      back: "Back",
      save: "Save",
      cancel: "Cancel",
      // Users
      users: "Users",
      usersDesc: "Manage system users",
      addUser: "+ Add User",
      loading: "Loading...",
      // Profile
      firstName: "First Name",
      lastName: "Last Name",
      email: "Email",
      username: "Username",
      changeAvatar: "Change avatar",
      // Password
      currentPassword: "Current password",
      newPassword: "New password",
      confirmPassword: "Confirm password",
      passwordMismatch: "Passwords do not match",
      passwordChanged: "Password updated successfully",
      passwordRequirements:
        "Minimum 8 characters, one uppercase and one number",
      // Notifications
      alertThresholds: "Alert Thresholds",
      cpuThreshold: "CPU Threshold",
      memoryThreshold: "Memory Threshold",
      containerStopped: "Container stopped",
      containerStoppedDesc: "Notify when a container stops",
      containerStarted: "Container started",
      containerStartedDesc: "Notify when a container starts",
      highCpu: "High CPU",
      highCpuDesc: "Alert when CPU exceeds threshold",
      highMemory: "High Memory",
      highMemoryDesc: "Alert when memory exceeds threshold",
      testNotification: "Test Notification",
      sending: "Sending...",
      // Apprise / Notification Channels
      apprise: "Apprise",
      appriseDesc: "Notification services",
      appriseServer: "Apprise Server",
      appriseUrl: "Apprise URL",
      appriseKey: "Key (for stateful notifications)",
      appriseHelp:
        "Configure notification URLs in your Apprise server. Supports Telegram, Email, Discord, Slack, and 100+ services.",
      sendTestNotification: "Send Test Notification",
      notificationChannels: "Notification Channels",
      emailNotifications: "Email Notifications",
      emailNotificationsDesc: "Receive alerts via email",
      telegramNotifications: "Telegram Notifications",
      telegramNotificationsDesc: "Receive alerts via Telegram",
      telegramUrl: "Telegram URL (Apprise)",
      telegramUrlPlaceholder: "tgram://token/chat_id",
      telegramUrlHelp: "Format: tgram://BOT_TOKEN/CHAT_ID",
      testChannelLabel: "Test Channel",
      testTelegram: "Telegram",
      testEmail: "Email",
      testBoth: "Both",
      // Appearance
      dark: "Dark",
      light: "Light",
      system: "System",
      themeSelect: "Select a theme",
      // Data
      clearCache: "Clear cache",
      clearCacheDesc: "Delete stored temporary data",
      exportData: "Export data",
      exportDataDesc: "Download your configuration",
      deleteAccount: "Delete account",
      deleteAccountDesc: "Permanently delete your account",
      cacheCleared: "Cache cleared successfully",
      // About
      version: "Version",
      buildDate: "Build date",
      developer: "Developed by",
      license: "License",
      documentation: "Documentation",
      reportBug: "Report a bug",
      // Security / 2FA
      security: "Security",
      securityDesc: "Password and two-factor authentication",
      twoFactorAuth: "Two-Factor Authentication (2FA)",
      twoFactorDesc: "Add an extra layer of security to your account",
      twoFactorEnabled: "2FA Enabled",
      twoFactorDisabled: "2FA Disabled",
      enable2FA: "Enable 2FA",
      disable2FA: "Disable 2FA",
      setup2FA: "Setup 2FA",
      verify2FA: "Verify code",
      scanQRCode: "Scan the QR code with your authenticator app",
      manualEntry: "Or manually enter this code:",
      enterCode: "Enter the 6-digit code",
      recoveryCodes: "Recovery Codes",
      recoveryCodesDesc:
        "Save these codes in a safe place. Use them if you lose access to your authenticator app.",
      regenerateCodes: "Regenerate Codes",
      codesRemaining: "codes remaining",
      invalidCode: "Invalid code",
      confirmDisableTitle: "Disable 2FA",
      confirmDisableDesc:
        "Are you sure you want to disable two-factor authentication?",
      enterPassword: "Enter your password to confirm",
      // Auto-logout
      autoLogout: "Auto Logout",
      autoLogoutDesc: "Sign out after a period of inactivity",
      autoLogoutDisabled: "Disabled",
      autoLogoutMinutes: "minutes",
      autoLogoutHour: "hour",
      autoLogoutHours: "hours",
    },
  };

  let st = $derived(settingsText[$language]);

  // Menu items (security replaces password, includes 2FA)
  let menuItems = $derived([
    ...($currentUser?.roles?.includes("admin")
      ? [
          {
            id: "users",
            icon: Users,
            label: st.users,
            desc: st.usersDesc,
          },
        ]
      : []),
    { id: "profile", icon: User, label: st.profile, desc: st.profileDesc },
    { id: "security", icon: Shield, label: st.security, desc: st.securityDesc },
    {
      id: "notifications",
      icon: Bell,
      label: st.notifications,
      desc: st.notificationsDesc,
    },
    {
      id: "appearance",
      icon: Palette,
      label: st.appearance,
      desc: st.appearanceDesc,
    },
    { id: "language", icon: Globe, label: st.language, desc: st.languageDesc },
    { id: "data", icon: Database, label: st.data, desc: st.dataDesc },
    { id: "about", icon: Info, label: st.about, desc: st.aboutDesc },
  ] as const);

  function goBack() {
    currentView = "main";
  }

  function handleLogout() {
    auth.logout();
    onclose();
  }

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
      const success = await auth.changePassword(
        passwordForm.current,
        passwordForm.new,
      );

      if (success) {
        passwordForm.success = true;
        passwordForm.current = "";
        passwordForm.new = "";
        passwordForm.confirm = "";
        setTimeout(() => (passwordForm.success = false), 3000);
      } else {
        passwordForm.error =
          $language === "es"
            ? "Error al cambiar la contraseña. Verifica tu contraseña actual."
            : "Failed to change password. Check your current password.";
      }
    } catch (err) {
      passwordForm.error =
        $language === "es"
          ? "Error de conexión. Intenta de nuevo."
          : "Connection error. Please try again.";
    } finally {
      passwordForm.loading = false;
    }
  }

  // Avatar functions
  function triggerAvatarUpload() {
    avatarInput?.click();
  }

  async function handleAvatarChange(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith("image/")) {
      avatarState.error =
        $language === "es"
          ? "Solo se permiten imágenes"
          : "Only images are allowed";
      return;
    }

    // Validate file size (500KB max)
    if (file.size > 500 * 1024) {
      avatarState.error =
        $language === "es"
          ? "La imagen no puede superar 500KB"
          : "Image must be under 500KB";
      return;
    }

    avatarState.loading = true;
    avatarState.error = null;

    try {
      await uploadAvatar(file);
    } catch (err) {
      avatarState.error =
        err instanceof Error
          ? err.message
          : $language === "es"
            ? "Error al subir avatar"
            : "Failed to upload avatar";
    } finally {
      avatarState.loading = false;
      // Reset input for re-selection
      input.value = "";
    }
  }

  async function handleDeleteAvatar() {
    avatarState.loading = true;
    avatarState.error = null;

    try {
      await deleteAvatar();
    } catch (err) {
      avatarState.error =
        err instanceof Error
          ? err.message
          : $language === "es"
            ? "Error al eliminar avatar"
            : "Failed to delete avatar";
    } finally {
      avatarState.loading = false;
    }
  }

  async function saveProfile() {
    profileForm.error = null;
    profileForm.success = false;
    profileForm.loading = true;

    const token = localStorage.getItem("auth_access_token");

    try {
      const res = await fetch(`${API_BASE}/api/auth/profile`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
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
        // Update local user state
        auth.refreshUser();
        setTimeout(() => (profileForm.success = false), 3000);
      } else {
        const err = await res.json();
        profileForm.error =
          err.error ||
          ($language === "es" ? "Error al guardar" : "Failed to save");
      }
    } catch (err) {
      profileForm.error =
        $language === "es" ? "Error de conexión" : "Connection error";
    } finally {
      profileForm.loading = false;
    }
  }

  // TOTP Functions
  async function loadTOTPStatus() {
    const token = localStorage.getItem("auth_access_token");
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
      console.error("Failed to load TOTP status:", err);
    }
  }

  async function setupTOTP() {
    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem("auth_access_token");

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/setup`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        totpState.secret = data.secret;
        totpState.qrUrl = data.url;
        totpState.setupMode = true;
      } else {
        const err = await res.json();
        totpState.error = err.error || "Failed to setup 2FA";
      }
    } catch (err) {
      totpState.error =
        $language === "es" ? "Error de conexión" : "Connection error";
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
    const token = localStorage.getItem("auth_access_token");

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/enable`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
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
        totpState.verifyCode = "";
        auth.refreshUser();
      } else {
        const err = await res.json();
        totpState.error = err.error || st.invalidCode;
      }
    } catch (err) {
      totpState.error =
        $language === "es" ? "Error de conexión" : "Connection error";
    } finally {
      totpState.loading = false;
    }
  }

  async function disableTOTP() {
    if (!totpState.disablePassword) {
      totpState.error =
        $language === "es" ? "Ingresa tu contraseña" : "Enter your password";
      return;
    }

    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem("auth_access_token");

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/disable`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ password: totpState.disablePassword }),
      });

      if (res.ok) {
        totpState.enabled = false;
        totpState.confirmDisable = false;
        totpState.disablePassword = "";
        totpState.recoveryCodes = [];
        totpState.recoveryCount = 0;
        auth.refreshUser();
      } else {
        const err = await res.json();
        totpState.error = err.error || "Failed to disable 2FA";
      }
    } catch (err) {
      totpState.error =
        $language === "es" ? "Error de conexión" : "Connection error";
    } finally {
      totpState.loading = false;
    }
  }

  async function regenerateRecoveryCodes() {
    if (!totpState.disablePassword) {
      totpState.error =
        $language === "es" ? "Ingresa tu contraseña" : "Enter your password";
      return;
    }

    totpState.loading = true;
    totpState.error = null;
    const token = localStorage.getItem("auth_access_token");

    try {
      const res = await fetch(`${API_BASE}/api/auth/totp/regenerate-recovery`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ password: totpState.disablePassword }),
      });

      if (res.ok) {
        const data = await res.json();
        totpState.recoveryCodes = data.recoveryCodes || [];
        totpState.recoveryCount = data.recoveryCodes?.length || 0;
        totpState.showRecoveryCodes = true;
        totpState.disablePassword = "";
      } else {
        const err = await res.json();
        totpState.error = err.error || "Failed to regenerate codes";
      }
    } catch (err) {
      totpState.error =
        $language === "es" ? "Error de conexión" : "Connection error";
    } finally {
      totpState.loading = false;
    }
  }

  // Generate QR code data URL from otpauth URL
  function generateQRCodeUrl(otpauthUrl: string): string {
    // Use a QR code API service
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpauthUrl)}`;
  }

  // Load TOTP status when security view is opened
  $effect(() => {
    if (currentView === "security") {
      loadTOTPStatus();
    }
  });

  function clearCache() {
    localStorage.clear();
    sessionStorage.clear();
    // Show success notification
  }

  function setLanguage(lang: Language) {
    language.set(lang);
  }

  function setTheme(newTheme: Theme) {
    theme = newTheme;
    localStorage.setItem("dockerverse-theme", newTheme);
    applyTheme(newTheme);
  }

  function applyTheme(t: Theme) {
    const root = document.documentElement;
    let effectiveTheme = t;

    if (t === "system") {
      effectiveTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
    }

    // Use CSS class on root element
    if (effectiveTheme === "light") {
      root.classList.add("light");
      root.classList.remove("dark");
    } else {
      root.classList.remove("light");
      root.classList.add("dark");
    }
  }

  // Listen for system theme changes
  $effect(() => {
    if (theme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const handleChange = () => applyTheme("system");
      mediaQuery.addEventListener("change", handleChange);
      return () => mediaQuery.removeEventListener("change", handleChange);
    }
  });

  // Load saved theme on mount
  $effect(() => {
    const saved =
      (localStorage.getItem("dockerverse-theme") as Theme) || "dark";
    theme = saved;
    applyTheme(saved);
  });

  // Load users (admin only)
  async function loadUsers() {
    if (!$currentUser?.roles?.includes("admin")) return;
    usersLoading = true;
    try {
      const token = localStorage.getItem("auth_access_token");
      const res = await fetch(`${API_BASE}/api/users`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        usersList = await res.json();
      } else {
        console.error("Failed to load users:", res.status);
      }
    } catch (e) {
      console.error(e);
    }
    usersLoading = false;
  }

  async function saveUser() {
    const token = localStorage.getItem("auth_access_token");
    if (!token) {
      console.error("No auth token found");
      alert("Error: No authentication token");
      return;
    }

    const method = editingUser ? "PATCH" : "POST";
    const url = editingUser
      ? `${API_BASE}/api/users/${editingUser.username}`
      : `${API_BASE}/api/users`;

    console.log("Saving user:", { method, url, userForm });

    try {
      const res = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(userForm),
      });

      console.log("Response status:", res.status);

      if (res.ok) {
        showUserForm = false;
        editingUser = null;
        userForm = {
          username: "",
          email: "",
          password: "",
          firstName: "",
          lastName: "",
          role: "user",
        };
        await loadUsers();
      } else {
        const error = await res.text();
        console.error("Save user failed:", error);
        alert(`Error: ${error}`);
      }
    } catch (e) {
      console.error("Save user error:", e);
      alert(`Error: ${e}`);
    }
  }

  async function deleteUser(username: string) {
    if (!confirm("Delete user " + username + "?")) return;
    const token = localStorage.getItem("auth_access_token");
    try {
      await fetch(`${API_BASE}/api/users/${username}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });
      loadUsers();
    } catch (e) {
      console.error(e);
    }
  }

  // Load app settings
  async function loadSettings() {
    const token = localStorage.getItem("auth_access_token");
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
    const token = localStorage.getItem("auth_access_token");
    try {
      await fetch(`${API_BASE}/api/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(appSettings),
      });
    } catch (e) {
      console.error(e);
    }
  }

  async function testNotification(channel?: "telegram" | "email" | "both") {
    testingNotification = true;
    const token = localStorage.getItem("auth_access_token");
    const selectedChannel = channel || testChannel;
    try {
      const res = await fetch(`${API_BASE}/api/notify/test`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          title: "🧪 DockerVerse Test",
          body: `Test notification via ${selectedChannel.toUpperCase()} - If you see this, notifications are working!`,
          type: "info",
          channel: selectedChannel,
        }),
      });

      const result = await res.json();
      console.log("Notification test result:", result);

      if (result.success) {
        const parts = [];
        if (result.telegram === "sent") parts.push("✅ Telegram");
        if (result.email === "sent") parts.push("✅ Email");
        if (result.telegram === "failed") parts.push("❌ Telegram failed");
        if (result.email === "failed") parts.push("❌ Email failed");
        if (result.email === "no_email") parts.push("⚠️ No email configured");

        alert(
          $language === "es"
            ? `Resultado: ${parts.join(", ") || "Enviado"}`
            : `Result: ${parts.join(", ") || "Sent"}`,
        );
      } else {
        const errors = result.errors?.join("\n") || "Unknown error";
        alert(
          $language === "es"
            ? `Parcialmente enviado:\n${errors}`
            : `Partially sent:\n${errors}`,
        );
      }
    } catch (e) {
      console.error(e);
      alert($language === "es" ? "Error de conexión" : "Connection error");
    }
    testingNotification = false;
  }

  $effect(() => {
    if (currentView === "users") loadUsers();
  });
  $effect(() => {
    if (currentView === "notifications") loadSettings();
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_label_has_associated_control -->
<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
  onclick={(e) => e.target === e.currentTarget && onclose()}
>
  <div
    class="w-full max-w-lg rounded-xl shadow-2xl overflow-hidden max-h-[85vh] flex flex-col animate-fade-in"
    style="background-color: rgb(var(--color-background-secondary)); border: 1px solid rgb(var(--color-background-tertiary));"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between p-4 border-b border-background-tertiary flex-shrink-0"
    >
      {#if currentView !== "main"}
        <button
          onclick={goBack}
          class="flex items-center gap-2 text-foreground-muted hover:text-foreground transition-colors"
        >
          <ChevronRight class="w-5 h-5 rotate-180" />
          <span class="text-sm">{st.back}</span>
        </button>
      {:else}
        <h2 class="text-lg font-semibold text-foreground">{st.settings}</h2>
      {/if}
      <button
        class="text-foreground-muted hover:text-foreground transition-colors"
        onclick={onclose}
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto">
      {#if currentView === "main"}
        <!-- Main Menu -->
        <div class="p-2">
          {#each menuItems as item}
            <button
              onclick={() => (currentView = item.id as SettingsView)}
              class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 transition-colors text-left"
            >
              <div class="p-2 bg-background-tertiary/50 rounded-lg">
                <item.icon class="w-5 h-5 text-primary" />
              </div>
              <div class="flex-1">
                <p class="font-medium text-foreground">{item.label}</p>
                <p class="text-sm text-foreground-muted">{item.desc}</p>
              </div>
              <ChevronRight class="w-5 h-5 text-foreground-muted" />
            </button>
          {/each}

          <!-- Logout -->
          <div class="border-t border-background-tertiary mt-2 pt-2">
            <button
              onclick={handleLogout}
              class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-stopped/10 transition-colors text-left"
            >
              <div class="p-2 bg-stopped/10 rounded-lg">
                <LogOut class="w-5 h-5 text-stopped" />
              </div>
              <div class="flex-1">
                <p class="font-medium text-stopped">{st.logout}</p>
              </div>
            </button>
          </div>
        </div>
      {:else if currentView === "profile"}
        <!-- Profile -->
        <div class="p-4 space-y-6">
          <!-- Success/Error Messages -->
          {#if profileForm.success}
            <div
              class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm"
            >
              <Check class="w-4 h-4" />
              <span
                >{$language === "es"
                  ? "Perfil actualizado"
                  : "Profile updated"}</span
              >
            </div>
          {/if}
          {#if profileForm.error}
            <div
              class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
            >
              <span>{profileForm.error}</span>
            </div>
          {/if}

          <!-- Avatar -->
          <div class="flex flex-col items-center">
            <!-- Hidden file input -->
            <input
              type="file"
              accept="image/*"
              class="hidden"
              bind:this={avatarInput}
              onchange={handleAvatarChange}
            />

            <!-- Avatar display -->
            <button
              onclick={triggerAvatarUpload}
              disabled={avatarState.loading}
              class="relative w-24 h-24 rounded-full mb-3 group overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-background disabled:opacity-50"
            >
              {#if $currentUser?.avatar}
                <img
                  src={$currentUser.avatar}
                  alt="Avatar"
                  class="w-full h-full object-cover"
                />
              {:else}
                <div
                  class="w-full h-full bg-primary/20 flex items-center justify-center"
                >
                  <User class="w-12 h-12 text-primary" />
                </div>
              {/if}

              <!-- Hover overlay -->
              <div
                class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
              >
                {#if avatarState.loading}
                  <RefreshCw class="w-6 h-6 text-white animate-spin" />
                {:else}
                  <Camera class="w-6 h-6 text-white" />
                {/if}
              </div>
            </button>

            <!-- Avatar actions -->
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
                  {$language === "es" ? "Eliminar" : "Remove"}
                </button>
              {/if}
            </div>

            <!-- Avatar error -->
            {#if avatarState.error}
              <p class="mt-2 text-sm text-stopped">{avatarState.error}</p>
            {/if}

            <!-- Size hint -->
            <p class="mt-1 text-xs text-foreground-muted">
              {$language === "es" ? "Máximo 500KB" : "Max 500KB"}
            </p>
          </div>

          <!-- Form -->
          <div class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-foreground mb-1"
                  >{st.firstName}</label
                >
                <input
                  type="text"
                  bind:value={profileForm.firstName}
                  class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-foreground mb-1"
                  >{st.lastName}</label
                >
                <input
                  type="text"
                  bind:value={profileForm.lastName}
                  class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
                />
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1"
                >{st.email}</label
              >
              <input
                type="email"
                bind:value={profileForm.email}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1"
                >{st.username}</label
              >
              <input
                type="text"
                value={$currentUser?.username ?? "admin"}
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
      {:else if currentView === "security"}
        <!-- Security: Auto-Logout + Password + 2FA -->
        <div class="p-4 space-y-6">
          <!-- Auto-Logout Section -->
          <div class="space-y-4">
            <h3
              class="text-lg font-semibold text-foreground flex items-center gap-2"
            >
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

          <!-- Divider -->
          <hr class="border-border" />

          <!-- Password Section -->
          <div class="space-y-4">
            <h3
              class="text-lg font-semibold text-foreground flex items-center gap-2"
            >
              <Lock class="w-5 h-5 text-primary" />
              {st.password}
            </h3>

            {#if passwordForm.success}
              <div
                class="flex items-center gap-2 p-3 bg-running/10 border border-running/30 rounded-lg text-running text-sm"
              >
                <Check class="w-4 h-4" />
                {st.passwordChanged}
              </div>
            {/if}

            {#if passwordForm.error}
              <div
                class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
              >
                {passwordForm.error}
              </div>
            {/if}

            <div>
              <label class="block text-sm font-medium text-foreground mb-1"
                >{st.currentPassword}</label
              >
              <input
                type="password"
                bind:value={passwordForm.current}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1"
                >{st.newPassword}</label
              >
              <input
                type="password"
                bind:value={passwordForm.new}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
              />
              <p class="text-xs text-foreground-muted mt-1">
                {st.passwordRequirements}
              </p>
            </div>
            <div>
              <label class="block text-sm font-medium text-foreground mb-1"
                >{st.confirmPassword}</label
              >
              <input
                type="password"
                bind:value={passwordForm.confirm}
                class="w-full px-3 py-2 bg-background border border-border rounded-lg text-foreground"
              />
            </div>

            <button
              onclick={handlePasswordChange}
              disabled={!passwordForm.current ||
                !passwordForm.new ||
                !passwordForm.confirm ||
                passwordForm.loading}
              class="w-full py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {#if passwordForm.loading}
                <RefreshCw class="w-4 h-4 animate-spin" />
                {$language === "es" ? "Guardando..." : "Saving..."}
              {:else}
                {st.save}
              {/if}
            </button>
          </div>

          <!-- Divider -->
          <hr class="border-border" />

          <!-- Two-Factor Authentication Section -->
          <div class="space-y-4">
            <h3
              class="text-lg font-semibold text-foreground flex items-center gap-2"
            >
              <Shield class="w-5 h-5 text-primary" />
              {st.twoFactorAuth}
            </h3>
            <p class="text-sm text-foreground-muted">{st.twoFactorDesc}</p>

            {#if totpState.error}
              <div
                class="flex items-center gap-2 p-3 bg-stopped/10 border border-stopped/30 rounded-lg text-stopped text-sm"
              >
                {totpState.error}
              </div>
            {/if}

            <!-- Status Badge -->
            <div
              class="flex items-center justify-between p-4 bg-background-tertiary rounded-lg"
            >
              <div class="flex items-center gap-3">
                {#if totpState.enabled}
                  <div
                    class="w-10 h-10 bg-running/20 rounded-full flex items-center justify-center"
                  >
                    <Check class="w-5 h-5 text-running" />
                  </div>
                  <div>
                    <p class="font-medium text-foreground">
                      {st.twoFactorEnabled}
                    </p>
                    <p class="text-xs text-foreground-muted">
                      {totpState.recoveryCount}
                      {st.codesRemaining}
                    </p>
                  </div>
                {:else}
                  <div
                    class="w-10 h-10 bg-foreground-muted/20 rounded-full flex items-center justify-center"
                  >
                    <Shield class="w-5 h-5 text-foreground-muted" />
                  </div>
                  <div>
                    <p class="font-medium text-foreground">
                      {st.twoFactorDisabled}
                    </p>
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
                <div
                  class="space-y-4 p-4 bg-background rounded-lg border border-border"
                >
                  <p class="text-sm text-foreground-muted text-center">
                    {st.scanQRCode}
                  </p>

                  <div class="flex justify-center">
                    <img
                      src={generateQRCodeUrl(totpState.qrUrl)}
                      alt="QR Code"
                      class="w-48 h-48 rounded-lg bg-white p-2"
                    />
                  </div>

                  <div class="text-center">
                    <p class="text-xs text-foreground-muted mb-2">
                      {st.manualEntry}
                    </p>
                    <div
                      class="inline-flex items-center gap-2 px-3 py-2 bg-background-tertiary rounded-lg"
                    >
                      <code class="text-sm font-mono text-primary select-all"
                        >{totpState.secret}</code
                      >
                    </div>
                  </div>

                  <div>
                    <label
                      class="block text-sm font-medium text-foreground mb-1"
                      >{st.enterCode}</label
                    >
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
                      disabled={totpState.loading ||
                        totpState.verifyCode.length !== 6}
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
                    onclick={() =>
                      (totpState.showRecoveryCodes =
                        !totpState.showRecoveryCodes)}
                    class="w-full py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors"
                  >
                    {st.regenerateCodes}
                  </button>
                {:else}
                  <!-- Confirm Disable -->
                  <div
                    class="p-4 bg-stopped/5 border border-stopped/30 rounded-lg space-y-3"
                  >
                    <p class="text-sm text-stopped font-medium">
                      {st.confirmDisableTitle}
                    </p>
                    <p class="text-sm text-foreground-muted">
                      {st.confirmDisableDesc}
                    </p>

                    <div>
                      <label
                        class="block text-sm font-medium text-foreground mb-1"
                        >{st.enterPassword}</label
                      >
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
                          totpState.disablePassword = "";
                          totpState.error = null;
                        }}
                        class="flex-1 py-2 border border-border text-foreground rounded-lg hover:bg-background-tertiary transition-colors"
                      >
                        {st.cancel}
                      </button>
                      <button
                        onclick={disableTOTP}
                        disabled={totpState.loading ||
                          !totpState.disablePassword}
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

            <!-- Recovery Codes Modal/Section -->
            {#if totpState.showRecoveryCodes && totpState.recoveryCodes.length > 0}
              <div
                class="p-4 bg-primary/5 border border-primary/30 rounded-lg space-y-3"
              >
                <h4 class="font-medium text-foreground">{st.recoveryCodes}</h4>
                <p class="text-xs text-foreground-muted">
                  {st.recoveryCodesDesc}
                </p>

                <div class="grid grid-cols-2 gap-2">
                  {#each totpState.recoveryCodes as code}
                    <div
                      class="px-3 py-2 bg-background rounded border border-border font-mono text-sm text-center select-all"
                    >
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
                  {$language === "es"
                    ? "Entendido, los guardé"
                    : "Got it, I saved them"}
                </button>
              </div>
            {/if}
          </div>
        </div>
      {:else if currentView === "users"}
        <!-- Users Management (Admin Only) -->
        <div class="p-4 space-y-4">
          <div class="flex justify-between items-center">
            <h3 class="text-lg font-semibold text-foreground">{st.users}</h3>
            <button
              onclick={() => {
                showUserForm = true;
                editingUser = null;
                userForm = {
                  username: "",
                  email: "",
                  password: "",
                  firstName: "",
                  lastName: "",
                  role: "user",
                };
              }}
              class="flex items-center gap-1 px-3 py-1.5 bg-primary text-white rounded-lg text-sm hover:bg-primary/90"
            >
              <Plus class="w-4 h-4" />
              {st.addUser}
            </button>
          </div>
          {#if showUserForm}
            <div
              class="p-4 bg-background rounded-lg border border-border space-y-3"
            >
              <div class="grid grid-cols-2 gap-3">
                <input
                  type="text"
                  placeholder={st.username}
                  bind:value={userForm.username}
                  disabled={!!editingUser}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground disabled:opacity-50"
                />
                <input
                  type="email"
                  placeholder={st.email}
                  bind:value={userForm.email}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
                />
                <input
                  type="text"
                  placeholder={st.firstName}
                  bind:value={userForm.firstName}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
                />
                <input
                  type="text"
                  placeholder={st.lastName}
                  bind:value={userForm.lastName}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
                />
                <input
                  type="password"
                  placeholder={st.newPassword}
                  bind:value={userForm.password}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
                />
                <select
                  bind:value={userForm.role}
                  class="px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
                >
                  <option value="user"
                    >{$language === "es" ? "Usuario" : "User"}</option
                  >
                  <option value="admin">Admin</option>
                </select>
              </div>
              <div class="flex gap-2">
                <button
                  onclick={saveUser}
                  class="px-4 py-2 bg-primary text-white rounded-lg text-sm"
                  >{st.save}</button
                >
                <button
                  onclick={() => (showUserForm = false)}
                  class="px-4 py-2 bg-background-tertiary text-foreground rounded-lg text-sm"
                  >{st.cancel}</button
                >
              </div>
            </div>
          {/if}
          {#if usersLoading}
            <p class="text-foreground-muted text-center">{st.loading}</p>
          {:else}
            <div class="space-y-2">
              {#each usersList as user}
                <div
                  class="flex items-center justify-between p-3 bg-background rounded-lg border border-border"
                >
                  <div class="flex items-center gap-3">
                    <div
                      class="w-10 h-10 bg-primary/20 rounded-full flex items-center justify-center"
                    >
                      <User class="w-5 h-5 text-primary" />
                    </div>
                    <div>
                      <p class="font-medium text-foreground">{user.username}</p>
                      <p class="text-sm text-foreground-muted">
                        {user.email} • {user.role === "admin"
                          ? "Admin"
                          : $language === "es"
                            ? "Usuario"
                            : "User"}
                      </p>
                    </div>
                  </div>
                  <div class="flex gap-2">
                    <button
                      onclick={() => {
                        editingUser = user;
                        showUserForm = true;
                        userForm = { ...user, password: "" };
                      }}
                      class="p-2 hover:bg-background-tertiary rounded-lg"
                      ><Pencil class="w-4 h-4 text-foreground-muted" /></button
                    >
                    {#if user.username !== "admin"}
                      <button
                        onclick={() => deleteUser(user.username)}
                        class="p-2 hover:bg-stopped/10 rounded-lg"
                        ><Trash2 class="w-4 h-4 text-stopped" /></button
                      >
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {:else if currentView === "notifications"}
        <!-- Notifications with Thresholds -->
        <div class="p-4 space-y-4">
          <!-- Threshold sliders -->
          <div
            class="p-4 bg-background rounded-lg border border-border space-y-4"
          >
            <h4 class="font-medium text-foreground flex items-center gap-2">
              <Settings2 class="w-4 h-4" />
              {st.alertThresholds}
            </h4>
            <div class="space-y-3">
              <div>
                <div class="flex justify-between text-sm mb-1">
                  <span class="text-foreground-muted flex items-center gap-1"
                    ><Cpu class="w-4 h-4" /> {st.cpuThreshold}</span
                  >
                  <span class="text-primary font-medium"
                    >{appSettings.cpuThreshold}%</span
                  >
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
                  <span class="text-foreground-muted flex items-center gap-1"
                    ><MemoryStick class="w-4 h-4" /> {st.memoryThreshold}</span
                  >
                  <span class="text-primary font-medium"
                    >{appSettings.memoryThreshold}%</span
                  >
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
          <!-- Toggle switches with i18n -->
          <div
            class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
          >
            <div class="flex items-center gap-3">
              <Activity class="w-5 h-5 text-foreground-muted" />
              <div>
                <p class="font-medium text-foreground">{st.containerStopped}</p>
                <p class="text-sm text-foreground-muted">
                  {st.containerStoppedDesc}
                </p>
              </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={appSettings.notifyOnStop}
                onchange={saveSettings}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
              ></div>
            </label>
          </div>
          <div
            class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
          >
            <div class="flex items-center gap-3">
              <Activity class="w-5 h-5 text-foreground-muted" />
              <div>
                <p class="font-medium text-foreground">{st.containerStarted}</p>
                <p class="text-sm text-foreground-muted">
                  {st.containerStartedDesc}
                </p>
              </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={appSettings.notifyOnStart}
                onchange={saveSettings}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
              ></div>
            </label>
          </div>
          <div
            class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
          >
            <div class="flex items-center gap-3">
              <Cpu class="w-5 h-5 text-foreground-muted" />
              <div>
                <p class="font-medium text-foreground">{st.highCpu}</p>
                <p class="text-sm text-foreground-muted">{st.highCpuDesc}</p>
              </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={appSettings.notifyOnHighCpu}
                onchange={saveSettings}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
              ></div>
            </label>
          </div>
          <div
            class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
          >
            <div class="flex items-center gap-3">
              <MemoryStick class="w-5 h-5 text-foreground-muted" />
              <div>
                <p class="font-medium text-foreground">{st.highMemory}</p>
                <p class="text-sm text-foreground-muted">{st.highMemoryDesc}</p>
              </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                bind:checked={appSettings.notifyOnHighMem}
                onchange={saveSettings}
                class="sr-only peer"
              />
              <div
                class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
              ></div>
            </label>
          </div>

          <!-- Notification Channels Section -->
          <div class="border-t border-border mt-4 pt-4">
            <h4
              class="font-medium text-foreground flex items-center gap-2 mb-4"
            >
              <Send class="w-4 h-4" />
              {st.notificationChannels}
            </h4>

            <!-- Email Notifications -->
            <div
              class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
            >
              <div class="flex items-center gap-3">
                <Mail class="w-5 h-5 text-foreground-muted" />
                <div>
                  <p class="font-medium text-foreground">
                    {st.emailNotifications}
                  </p>
                  <p class="text-sm text-foreground-muted">
                    {st.emailNotificationsDesc}
                  </p>
                </div>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={appSettings.emailEnabled}
                  onchange={saveSettings}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
                ></div>
              </label>
            </div>

            <!-- Telegram Notifications -->
            <div
              class="flex items-center justify-between p-3 rounded-lg hover:bg-background-tertiary/50"
            >
              <div class="flex items-center gap-3">
                <Send class="w-5 h-5 text-foreground-muted" />
                <div>
                  <p class="font-medium text-foreground">
                    {st.telegramNotifications}
                  </p>
                  <p class="text-sm text-foreground-muted">
                    {st.telegramNotificationsDesc}
                  </p>
                </div>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={appSettings.telegramEnabled}
                  onchange={saveSettings}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-background-tertiary peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
                ></div>
              </label>
            </div>
          </div>

          <!-- Apprise Configuration -->
          <div
            class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4"
          >
            <h4 class="font-medium text-foreground">{st.appriseServer}</h4>
            <div>
              <label class="block text-sm text-foreground-muted mb-1"
                >{st.appriseUrl}</label
              >
              <input
                type="url"
                bind:value={appSettings.appriseUrl}
                onchange={saveSettings}
                placeholder="https://apprise.example.com"
                class="w-full px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground"
              />
            </div>
            <div>
              <label class="block text-sm text-foreground-muted mb-1"
                >{st.appriseKey}</label
              >
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
            <div
              class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4"
            >
              <h4 class="font-medium text-foreground flex items-center gap-2">
                <Send class="w-4 h-4" />
                Telegram
              </h4>
              <div>
                <label class="block text-sm text-foreground-muted mb-1"
                  >{st.telegramUrl}</label
                >
                <input
                  type="text"
                  bind:value={appSettings.telegramUrl}
                  onchange={saveSettings}
                  placeholder={st.telegramUrlPlaceholder}
                  class="w-full px-3 py-2 bg-background-secondary border border-border rounded-lg text-foreground font-mono text-sm"
                />
                <p class="text-xs text-foreground-muted mt-1">
                  {st.telegramUrlHelp}
                </p>
              </div>
            </div>
          {/if}

          <!-- Test Channel Selection -->
          <div
            class="p-4 bg-background rounded-lg border border-border space-y-4 mt-4"
          >
            <h4 class="font-medium text-foreground">{st.testChannelLabel}</h4>
            <div class="grid grid-cols-3 gap-2">
              {#each [{ id: "telegram", label: st.testTelegram, icon: Send }, { id: "email", label: st.testEmail, icon: Mail }, { id: "both", label: st.testBoth, icon: Bell }] as channel}
                <button
                  onclick={() =>
                    (testChannel = channel.id as "telegram" | "email" | "both")}
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
      {:else if currentView === "appearance"}
        <!-- Appearance -->
        <div class="p-4 space-y-4">
          <p class="text-sm text-foreground-muted">{st.themeSelect}</p>
          <div class="grid grid-cols-3 gap-3">
            {#each [{ id: "dark", icon: Moon, label: st.dark }, { id: "light", icon: Sun, label: st.light }, { id: "system", icon: Monitor, label: st.system }] as item}
              <button
                onclick={() => setTheme(item.id as Theme)}
                class="flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all
									   {theme === item.id
                  ? 'border-primary bg-primary/10'
                  : 'border-border hover:border-foreground-muted'}"
              >
                <item.icon
                  class="w-6 h-6 {theme === item.id
                    ? 'text-primary'
                    : 'text-foreground-muted'}"
                />
                <span
                  class="text-sm {theme === item.id
                    ? 'text-primary'
                    : 'text-foreground'}">{item.label}</span
                >
                {#if theme === item.id}
                  <Check class="w-4 h-4 text-primary" />
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {:else if currentView === "language"}
        <!-- Language -->
        <div class="p-4 space-y-2">
          {#each [{ id: "es", flag: "🇪🇸", label: "Español" }, { id: "en", flag: "🇬🇧", label: "English" }] as item}
            <button
              onclick={() => setLanguage(item.id as Language)}
              class="w-full flex items-center gap-4 p-3 rounded-lg border-2 transition-all text-left
								   {$language === item.id
                ? 'border-primary bg-primary/10'
                : 'border-transparent hover:bg-background-tertiary/50'}"
            >
              <span class="text-2xl">{item.flag}</span>
              <span class="flex-1 font-medium text-foreground"
                >{item.label}</span
              >
              {#if $language === item.id}
                <Check class="w-5 h-5 text-primary" />
              {/if}
            </button>
          {/each}
        </div>
      {:else if currentView === "data"}
        <!-- Data & Storage -->
        <div class="p-4 space-y-2">
          <button
            onclick={clearCache}
            class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 text-left"
          >
            <Trash2 class="w-5 h-5 text-foreground-muted" />
            <div class="flex-1">
              <p class="font-medium text-foreground">{st.clearCache}</p>
              <p class="text-sm text-foreground-muted">{st.clearCacheDesc}</p>
            </div>
          </button>
          <button
            class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 text-left"
          >
            <Download class="w-5 h-5 text-foreground-muted" />
            <div class="flex-1">
              <p class="font-medium text-foreground">{st.exportData}</p>
              <p class="text-sm text-foreground-muted">{st.exportDataDesc}</p>
            </div>
          </button>

          <div class="border-t border-background-tertiary my-4"></div>

          <button
            class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-stopped/10 text-left"
          >
            <Trash2 class="w-5 h-5 text-stopped" />
            <div class="flex-1">
              <p class="font-medium text-stopped">{st.deleteAccount}</p>
              <p class="text-sm text-foreground-muted">
                {st.deleteAccountDesc}
              </p>
            </div>
          </button>
        </div>
      {:else if currentView === "about"}
        <!-- About -->
        <div class="p-4 space-y-4">
          <div class="text-center mb-6">
            <span class="text-5xl">🐳</span>
            <h3 class="text-xl font-bold text-foreground mt-2">DockerVerse</h3>
            <p class="text-foreground-muted">Multi-Host Docker Management</p>
          </div>

          <div class="space-y-3">
            <div class="flex justify-between p-3 bg-background rounded-lg">
              <span class="text-foreground-muted">{st.version}</span>
              <span class="text-foreground font-medium">2.1.0</span>
            </div>
            <div class="flex justify-between p-3 bg-background rounded-lg">
              <span class="text-foreground-muted">{st.buildDate}</span>
              <span class="text-foreground font-medium">2026-02-08</span>
            </div>
            <div class="flex justify-between p-3 bg-background rounded-lg">
              <span class="text-foreground-muted">{st.license}</span>
              <span class="text-foreground font-medium">MIT</span>
            </div>
          </div>

          <div class="flex gap-2 pt-4">
            <button
              class="flex-1 py-2 px-4 bg-background text-foreground rounded-lg hover:bg-background-tertiary transition-colors text-sm"
            >
              {st.documentation}
            </button>
            <button
              class="flex-1 py-2 px-4 bg-background text-foreground rounded-lg hover:bg-background-tertiary transition-colors text-sm"
            >
              {st.reportBug}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
