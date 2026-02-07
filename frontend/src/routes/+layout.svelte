<script lang="ts">
  import "../app.css";
  import { onMount, onDestroy } from "svelte";
  import { browser } from "$app/environment";
  import {
    Search,
    Settings as SettingsIcon,
    RefreshCw,
    Globe,
    X,
    User,
    LogOut,
    ChevronDown,
    Moon,
    Sun,
    Menu,
    Home,
    Shield,
    Bell,
    Palette,
    Info,
    Database,
    Users,
    ArrowUpCircle,
  } from "lucide-svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import Login from "$lib/components/Login.svelte";
  import Settings from "$lib/components/Settings.svelte";
  import {
    language,
    translations,
    selectedHost,
    pendingUpdatesCount,
    checkForUpdates,
  } from "$lib/stores/docker";
  import {
    isAuthenticated,
    isLoading,
    auth,
    currentUser,
    setupActivityTracking,
    cleanupActivityTracking,
  } from "$lib/stores/auth";

  let { children } = $props();
  let showCommandPalette = $state(false);
  let showSettings = $state(false);
  let showUserMenu = $state(false);
  let showSidebar = $state(false);
  let isRefreshing = $state(false);
  let settingsInitialView = $state<string | undefined>(undefined);

  // Theme state - initialize from localStorage if available
  type Theme = "dark" | "light";
  let isDark = $state(
    browser ? localStorage.getItem("dockerverse-theme") !== "light" : true,
  );

  // Get current translations
  let t = $derived(translations[$language]);

  // User menu translations
  const userMenuText = $derived({
    settings: $language === "es" ? "Configuración" : "Settings",
    logout: $language === "es" ? "Cerrar Sesión" : "Sign Out",
  });

  // Sidebar menu items
  const sidebarItems = $derived([
    {
      id: "dashboard",
      icon: Home,
      label: $language === "es" ? "Dashboard" : "Dashboard",
      href: "/",
    },
    ...($currentUser?.roles?.includes("admin")
      ? [
          {
            id: "users",
            icon: Users,
            label: $language === "es" ? "Usuarios" : "Users",
            action: () => openSettingsView("users"),
          },
        ]
      : []),
    {
      id: "notifications",
      icon: Bell,
      label: $language === "es" ? "Notificaciones" : "Notifications",
      action: () => openSettingsView("notifications"),
    },
    {
      id: "appearance",
      icon: Palette,
      label: $language === "es" ? "Apariencia" : "Appearance",
      action: () => openSettingsView("appearance"),
    },
    {
      id: "security",
      icon: Shield,
      label: $language === "es" ? "Seguridad" : "Security",
      action: () => openSettingsView("security"),
    },
    {
      id: "data",
      icon: Database,
      label: $language === "es" ? "Datos" : "Data",
      action: () => openSettingsView("data"),
    },
    {
      id: "about",
      icon: Info,
      label: $language === "es" ? "Acerca de" : "About",
      action: () => openSettingsView("about"),
    },
  ]);

  function handleKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === "k") {
      e.preventDefault();
      showCommandPalette = true;
    }
    if (e.key === "Escape") {
      showCommandPalette = false;
      showSettings = false;
      showUserMenu = false;
      showSidebar = false;
    }
  }

  function openSettingsView(view: string) {
    settingsInitialView = view;
    showSettings = true;
    showSidebar = false;
  }

  function toggleLanguage() {
    language.update((l) => {
      const newLang = l === "es" ? "en" : "es";
      localStorage.setItem("dockerverse-language", newLang);
      return newLang;
    });
  }

  function toggleTheme() {
    isDark = !isDark;
    applyTheme(isDark);
    localStorage.setItem("dockerverse-theme", isDark ? "dark" : "light");
  }

  function applyTheme(dark: boolean) {
    if (typeof document !== "undefined") {
      const root = document.documentElement;
      if (dark) {
        root.classList.remove("light");
      } else {
        root.classList.add("light");
      }
    }
  }

  async function handleRefresh() {
    isRefreshing = true;
    // Dispatch custom event that page can listen to
    window.dispatchEvent(new CustomEvent("dockerverse:refresh"));
    setTimeout(() => (isRefreshing = false), 1000);
  }

  function clearHostFilter() {
    selectedHost.set(null);
  }

  function handleLogout() {
    auth.logout();
    showUserMenu = false;
    showSidebar = false;
  }

  function openSettings() {
    settingsInitialView = undefined;
    showSettings = true;
    showUserMenu = false;
  }

  // Close user menu when clicking outside
  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest(".user-menu-container")) {
      showUserMenu = false;
    }
  }

  // Check for updates periodically
  let updateCheckInterval: ReturnType<typeof setInterval> | null = null;

  function startUpdateCheck() {
    // Check immediately on login
    checkForUpdates();
    // Then check every 5 minutes
    updateCheckInterval = setInterval(checkForUpdates, 5 * 60 * 1000);
  }

  function stopUpdateCheck() {
    if (updateCheckInterval) {
      clearInterval(updateCheckInterval);
      updateCheckInterval = null;
    }
  }

  onMount(() => {
    window.addEventListener("keydown", handleKeydown);
    window.addEventListener("click", handleClickOutside);

    // Load saved theme
    const savedTheme = localStorage.getItem("dockerverse-theme") as Theme;
    if (savedTheme) {
      isDark = savedTheme === "dark";
    }
    // Always apply theme on mount
    applyTheme(isDark);

    // Load saved language
    const savedLang = localStorage.getItem("dockerverse-language");
    if (savedLang === "es" || savedLang === "en") {
      language.set(savedLang);
    }

    // Setup activity tracking for auto-logout
    setupActivityTracking(() => {
      auth.logout();
    });

    // Start update check if already authenticated
    if ($isAuthenticated) {
      startUpdateCheck();
    }

    // Watch for auth changes
    const unsubAuth = isAuthenticated.subscribe((authenticated) => {
      if (authenticated) {
        startUpdateCheck();
      } else {
        stopUpdateCheck();
      }
    });

    return () => {
      window.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("click", handleClickOutside);
      unsubAuth();
    };
  });

  onDestroy(() => {
    cleanupActivityTracking();
    stopUpdateCheck();
  });
</script>

<div class="min-h-screen bg-background flex">
  <!-- Sidebar (visible only when authenticated) -->
  {#if $isAuthenticated}
    <!-- Mobile sidebar overlay -->
    {#if showSidebar}
      <div
        class="fixed inset-0 bg-black/50 z-40 lg:hidden"
        onclick={() => (showSidebar = false)}
        role="button"
        tabindex="0"
        onkeydown={(e) => e.key === "Enter" && (showSidebar = false)}
      ></div>
    {/if}

    <!-- Sidebar -->
    <aside
      class="fixed lg:sticky top-0 left-0 z-50 h-screen w-64 bg-background-secondary border-r border-border transform transition-transform duration-300 lg:translate-x-0 {showSidebar
        ? 'translate-x-0'
        : '-translate-x-full lg:translate-x-0'}"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="p-4 border-b border-border">
          <a
            href="/"
            class="flex items-center gap-3 hover:opacity-80 transition-opacity"
            onclick={() => (showSidebar = false)}
          >
            <span class="text-3xl">🐳</span>
            <div>
              <h1 class="text-xl font-bold text-foreground">DockerVerse</h1>
              <p class="text-xs text-foreground-muted">Multi-Host Management</p>
            </div>
          </a>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
          {#each sidebarItems as item}
            {@const Icon = item.icon}
            {#if item.href}
              <a
                href={item.href}
                class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-foreground-muted hover:text-foreground hover:bg-background-tertiary transition-colors"
                onclick={() => (showSidebar = false)}
              >
                <Icon class="w-5 h-5" />
                <span class="text-sm font-medium">{item.label}</span>
              </a>
            {:else}
              <button
                class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-foreground-muted hover:text-foreground hover:bg-background-tertiary transition-colors"
                onclick={item.action}
              >
                <Icon class="w-5 h-5" />
                <span class="text-sm font-medium">{item.label}</span>
              </button>
            {/if}
          {/each}
        </nav>

        <!-- User section at bottom -->
        <div class="p-4 border-t border-border">
          <div class="flex items-center gap-3 mb-3">
            <div
              class="w-10 h-10 bg-primary/20 rounded-full flex items-center justify-center"
            >
              {#if $currentUser?.avatar}
                <img
                  src={$currentUser.avatar}
                  alt="Avatar"
                  class="w-10 h-10 rounded-full object-cover"
                />
              {:else}
                <User class="w-5 h-5 text-primary" />
              {/if}
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-foreground truncate">
                {$currentUser?.firstName || $currentUser?.username}
              </p>
              <p class="text-xs text-foreground-muted truncate">
                {$currentUser?.email}
              </p>
            </div>
          </div>
          <button
            class="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
            onclick={handleLogout}
          >
            <LogOut class="w-4 h-4" />
            {userMenuText.logout}
          </button>
        </div>
      </div>
    </aside>
  {/if}

  <!-- Main content area -->
  <div
    class="flex-1 flex flex-col min-h-screen {$isAuthenticated
      ? 'lg:ml-0'
      : ''}"
  >
    <!-- Header -->
    <header
      class="sticky top-0 z-40 glass border-b border-background-tertiary/50"
    >
      <div class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex h-16 items-center justify-between">
          <!-- Mobile menu button & Logo -->
          <div class="flex items-center gap-3">
            {#if $isAuthenticated}
              <button
                class="lg:hidden btn btn-ghost btn-icon"
                onclick={() => (showSidebar = !showSidebar)}
              >
                <Menu class="w-5 h-5" />
              </button>
            {/if}

            <!-- Logo (only visible when NOT authenticated or on mobile) -->
            {#if !$isAuthenticated}
              <a
                href="/"
                class="flex items-center gap-3 hover:opacity-80 transition-opacity"
              >
                <span class="text-3xl">🐳</span>
                <div>
                  <h1 class="text-xl font-bold text-foreground">DockerVerse</h1>
                  <p class="text-xs text-foreground-muted">
                    Multi-Host Management
                  </p>
                </div>
              </a>
            {:else}
              <!-- Page title on mobile when sidebar logo is hidden -->
              <h2 class="lg:hidden text-lg font-semibold text-foreground">
                DockerVerse
              </h2>
            {/if}
          </div>

          <!-- Search (only when authenticated) -->
          {#if $isAuthenticated}
            <button
              class="hidden md:flex items-center gap-2 px-4 py-2 bg-background-tertiary/50
                     rounded-lg text-foreground-muted hover:text-foreground
                     border border-background-tertiary hover:border-primary/30
                     transition-all duration-200 min-w-[280px]"
              onclick={() => (showCommandPalette = true)}
            >
              <Search class="w-4 h-4" />
              <span class="text-sm">{t.search}</span>
              <kbd
                class="ml-auto px-2 py-0.5 text-xs bg-background rounded border border-background-tertiary"
              >
                ⌘K
              </kbd>
            </button>
          {/if}

          <!-- Actions -->
          <div class="flex items-center gap-2">
            <!-- Host filter indicator (only when authenticated) -->
            {#if $isAuthenticated && $selectedHost}
              <button
                class="flex items-center gap-1 px-2 py-1 text-xs bg-primary/20 text-primary rounded-lg hover:bg-primary/30 transition-colors"
                onclick={clearHostFilter}
                title={t.clearFilter}
              >
                <span>{$selectedHost}</span>
                <X class="w-3 h-3" />
              </button>
            {/if}

            <!-- Theme Toggle -->
            <button
              class="btn btn-ghost btn-icon"
              title={isDark ? t.lightMode : t.darkMode}
              onclick={toggleTheme}
            >
              {#if isDark}
                <Sun class="w-5 h-5" />
              {:else}
                <Moon class="w-5 h-5" />
              {/if}
            </button>

            <!-- Language Toggle -->
            <button
              class="btn btn-ghost btn-icon"
              title={t.language}
              onclick={toggleLanguage}
            >
              <Globe class="w-5 h-5" />
              <span class="text-xs ml-1">{$language.toUpperCase()}</span>
            </button>

            <!-- Refresh (only when authenticated) -->
            {#if $isAuthenticated}
              <button
                class="btn btn-ghost btn-icon {isRefreshing
                  ? 'animate-spin'
                  : ''}"
                title={t.refresh}
                onclick={handleRefresh}
                disabled={isRefreshing}
              >
                <RefreshCw class="w-5 h-5" />
              </button>

              <!-- Pending Updates Counter -->
              {#if $pendingUpdatesCount > 0}
                <button
                  class="relative btn btn-ghost btn-icon text-paused hover:text-primary"
                  title="{$pendingUpdatesCount} {t.pendingUpdates}"
                  onclick={() => openSettingsView("data")}
                >
                  <ArrowUpCircle class="w-5 h-5" />
                  <span
                    class="absolute -top-1 -right-1 w-5 h-5 bg-paused text-background text-xs font-bold rounded-full flex items-center justify-center"
                  >
                    {$pendingUpdatesCount}
                  </span>
                </button>
              {/if}
            {/if}

            <!-- User Menu -->
            {#if $isAuthenticated}
              <div class="relative user-menu-container">
                <button
                  class="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-background-tertiary transition-colors"
                  onclick={() => (showUserMenu = !showUserMenu)}
                >
                  <div
                    class="w-8 h-8 rounded-full flex items-center justify-center overflow-hidden {$currentUser?.avatar
                      ? ''
                      : 'bg-primary/20'}"
                  >
                    {#if $currentUser?.avatar}
                      <img
                        src={$currentUser.avatar}
                        alt="Avatar"
                        class="w-8 h-8 object-cover"
                      />
                    {:else}
                      <User class="w-4 h-4 text-primary" />
                    {/if}
                  </div>
                  <span class="text-sm text-foreground hidden sm:block"
                    >{$currentUser?.firstName || $currentUser?.username}</span
                  >
                  <ChevronDown
                    class="w-4 h-4 text-foreground-muted transition-transform {showUserMenu
                      ? 'rotate-180'
                      : ''}"
                  />
                </button>

                <!-- Dropdown Menu -->
                {#if showUserMenu}
                  <div
                    class="absolute right-0 top-full mt-2 w-48 bg-background-secondary border border-border rounded-lg shadow-lg py-1 z-50"
                  >
                    <div class="px-4 py-2 border-b border-border">
                      <p class="text-sm font-medium text-foreground">
                        {$currentUser?.firstName}
                        {$currentUser?.lastName}
                      </p>
                      <p class="text-xs text-foreground-muted">
                        {$currentUser?.email}
                      </p>
                    </div>
                    <button
                      class="w-full flex items-center gap-2 px-4 py-2 text-sm text-foreground hover:bg-background-tertiary transition-colors"
                      onclick={openSettings}
                    >
                      <SettingsIcon class="w-4 h-4" />
                      {userMenuText.settings}
                    </button>
                    <button
                      class="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-background-tertiary transition-colors"
                      onclick={handleLogout}
                    >
                      <LogOut class="w-4 h-4" />
                      {userMenuText.logout}
                    </button>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    {#if $isLoading}
      <!-- Auth loading state - prevents login flash -->
      <div class="min-h-[calc(100vh-4rem)] flex items-center justify-center">
        <div class="flex flex-col items-center gap-4">
          <div
            class="w-16 h-16 border-4 border-primary/30 border-t-primary rounded-full animate-spin"
          ></div>
          <span class="text-foreground-muted text-sm">{t.loading}</span>
        </div>
      </div>
    {:else if $isAuthenticated}
      <main class="flex-1 max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {@render children()}
      </main>
    {:else}
      <main class="flex-1 flex items-center justify-center">
        <Login />
      </main>
    {/if}
  </div>
</div>

<!-- Command Palette -->
{#if showCommandPalette && $isAuthenticated}
  <CommandPalette onclose={() => (showCommandPalette = false)} />
{/if}

<!-- Settings Modal -->
{#if showSettings && $isAuthenticated}
  <Settings onclose={() => (showSettings = false)} />
{/if}
