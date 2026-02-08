<script lang="ts">
  import {
    User,
    Shield,
    Bell,
    Palette,
    Globe,
    Info,
    LogOut,
    Database,
    Users,
    ChevronRight,
  } from 'lucide-svelte';
  import { language, type Language } from '$lib/stores/docker';
  import { auth, currentUser } from '$lib/stores/auth';
  import { settingsText } from '$lib/settings';

  let st = $derived(settingsText[$language]);

  let menuItems = $derived([
    ...($currentUser?.roles?.includes('admin')
      ? [{ id: 'users', icon: Users, label: st.users, desc: st.usersDesc }]
      : []),
    { id: 'profile', icon: User, label: st.profile, desc: st.profileDesc },
    { id: 'security', icon: Shield, label: st.security, desc: st.securityDesc },
    { id: 'notifications', icon: Bell, label: st.notifications, desc: st.notificationsDesc },
    { id: 'appearance', icon: Palette, label: st.appearance, desc: st.appearanceDesc },
    { id: 'language', icon: Globe, label: st.language, desc: st.languageDesc },
    { id: 'data', icon: Database, label: st.data, desc: st.dataDesc },
    { id: 'about', icon: Info, label: st.about, desc: st.aboutDesc },
  ] as const);

  function handleLogout() {
    auth.logout();
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="p-2">
  {#each menuItems as item}
    <a
      href="/settings/{item.id}"
      class="w-full flex items-center gap-4 p-3 rounded-lg hover:bg-background-tertiary/50 transition-colors text-left no-underline"
    >
      <div class="p-2 bg-background-tertiary/50 rounded-lg">
        <item.icon class="w-5 h-5 text-primary" />
      </div>
      <div class="flex-1">
        <p class="font-medium text-foreground">{item.label}</p>
        <p class="text-sm text-foreground-muted">{item.desc}</p>
      </div>
      <ChevronRight class="w-5 h-5 text-foreground-muted" />
    </a>
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
