<script lang="ts">
  import "../app.css";
  import { onMount } from "svelte";
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
  } from "lucide-svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import Login from "$lib/components/Login.svelte";
  import Settings from "$lib/components/Settings.svelte";
  import { language, translations, selectedHost } from "$lib/stores/docker";
  import {
    isAuthenticated,
    isLoading,
    auth,
    currentUser,
  } from "$lib/stores/auth";

  let { children } = $props();
  let showCommandPalette = $state(false);
  let showSettings = $state(false);
  let showUserMenu = $state(false);
  let isRefreshing = $state(false);

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

  function handleKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === "k") {
      e.preventDefault();
      showCommandPalette = true;
    }
    if (e.key === "Escape") {
      showCommandPalette = false;
      showSettings = false;
      showUserMenu = false;
    }
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
  }

  function openSettings() {
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

    return () => {
      window.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("click", handleClickOutside);
    };
  });
</script>

<div class="min-h-screen bg-background">
  <!-- Header -->
  <header
    class="sticky top-0 z-40 glass border-b border-background-tertiary/50"
  >
    <div class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex h-16 items-center justify-between">
        <!-- Logo - Click to go home -->
        <a
          href="/"
          class="flex items-center gap-3 hover:opacity-80 transition-opacity"
        >
          <span class="text-3xl">🐳</span>
          <div>
            <h1 class="text-xl font-bold text-foreground">DockerVerse</h1>
            <p class="text-xs text-foreground-muted">Multi-Host Management</p>
          </div>
        </a>

        <!-- Search -->
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

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <!-- Host filter indicator -->
          {#if $selectedHost}
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

          <!-- Refresh -->
          <button
            class="btn btn-ghost btn-icon {isRefreshing ? 'animate-spin' : ''}"
            title={t.refresh}
            onclick={handleRefresh}
            disabled={isRefreshing}
          >
            <RefreshCw class="w-5 h-5" />
          </button>

          <!-- User Menu -->
          {#if $isAuthenticated}
            <div class="relative user-menu-container">
              <button
                class="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-background-tertiary transition-colors"
                onclick={() => (showUserMenu = !showUserMenu)}
              >
                <div
                  class="w-8 h-8 bg-primary/20 rounded-full flex items-center justify-center"
                >
                  <User class="w-4 h-4 text-primary" />
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
    <main class="max-w-[1800px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
      {@render children()}
    </main>
  {:else}
    <Login />
  {/if}
</div>

<!-- Command Palette -->
{#if showCommandPalette && $isAuthenticated}
  <CommandPalette onclose={() => (showCommandPalette = false)} />
{/if}

<!-- Settings Modal -->
{#if showSettings && $isAuthenticated}
  <Settings onclose={() => (showSettings = false)} />
{/if}
