<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import {
    Server, User, KeyRound, Bell, Palette, Database, Info, Users, Settings, Shield
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { isAuthenticated, currentUser } from '$lib/stores/auth';
  import { settingsText } from '$lib/settings';

  let { children } = $props();

  let st = $derived(settingsText[$language]);

  // Auth guard
  $effect(() => {
    if (browser && !$isAuthenticated) {
      goto('/');
    }
  });

  let tabs = $derived([
    { id: 'environments', label: st.environments, icon: Server, href: '/settings/environments' },
    ...($currentUser?.roles?.includes('admin')
      ? [
          { id: 'users', label: st.users, icon: Users, href: '/settings/users' },
          { id: 'audit', label: st.audit, icon: Shield, href: '/settings/audit' },
        ]
      : []),
    { id: 'profile', label: st.profile, icon: User, href: '/settings/profile' },
    { id: 'authentication', label: st.authentication, icon: KeyRound, href: '/settings/authentication' },
    { id: 'notifications', label: st.notifications, icon: Bell, href: '/settings/notifications' },
    { id: 'general', label: st.general, icon: Palette, href: '/settings/general' },
    { id: 'data', label: st.data, icon: Database, href: '/settings/data' },
    { id: 'about', label: st.about, icon: Info, href: '/settings/about' },
  ]);

  let activeTab = $derived.by(() => {
    const path = $page.url.pathname;
    if (path.startsWith('/settings/environments')) return 'environments';
    if (path.startsWith('/settings/users')) return 'users';
    if (path.startsWith('/settings/profile')) return 'profile';
    if (path.startsWith('/settings/authentication') || path.startsWith('/settings/security')) return 'authentication';
    if (path.startsWith('/settings/notifications')) return 'notifications';
    if (path.startsWith('/settings/general') || path.startsWith('/settings/appearance')) return 'general';
    if (path.startsWith('/settings/data')) return 'data';
    if (path.startsWith('/settings/about')) return 'about';
    if (path.startsWith('/settings/audit')) return 'audit';
    return 'environments';
  });
</script>

<div class="max-w-5xl mx-auto w-full">
  <!-- Header -->
  <div class="flex items-center gap-2.5 mb-5">
    <Settings class="w-5 h-5 text-primary flex-shrink-0" />
    <h2 class="text-lg font-semibold text-foreground">{st.settings}</h2>
  </div>

  <!-- Horizontal tab bar -->
  <div class="border-b border-border mb-6">
    <div class="flex gap-0 overflow-x-auto" role="tablist" aria-label="Settings sections">
      {#each tabs as tab}
        {@const Icon = tab.icon}
        {@const isActive = activeTab === tab.id}
        <a
          href={tab.href}
          role="tab"
          aria-selected={isActive}
          class="flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 transition-colors cursor-pointer {isActive
            ? 'border-primary text-primary'
            : 'border-transparent text-foreground-muted hover:text-foreground hover:border-border'}"
        >
          <Icon class="w-3.5 h-3.5" />
          {tab.label}
        </a>
      {/each}
    </div>
  </div>

  <!-- Tab content -->
  {@render children()}
</div>
