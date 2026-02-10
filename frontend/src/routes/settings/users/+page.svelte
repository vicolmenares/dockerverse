<script lang="ts">
  import {
    User,
    Plus,
    Pencil,
    Trash2,
    Camera,
    X,
  } from 'lucide-svelte';
  import { language } from '$lib/stores/docker';
  import { currentUser } from '$lib/stores/auth';
  import { API_BASE } from '$lib/api/docker';
  import { settingsText } from '$lib/settings';
  import { goto } from '$app/navigation';

  let st = $derived(settingsText[$language]);

  // Redirect non-admins
  $effect(() => {
    if ($currentUser && !$currentUser.roles?.includes('admin')) {
      goto('/settings');
    }
  });

  let usersList = $state<any[]>([]);
  let usersLoading = $state(false);
  let showUserForm = $state(false);
  let editingUser = $state<any>(null);
  let userForm = $state({
    username: '',
    email: '',
    password: '',
    firstName: '',
    lastName: '',
    role: 'user',
  });
  let avatarPreview = $state<string | null>(null);
  let avatarFile = $state<File | null>(null);
  let uploadingAvatar = $state(false);

  async function loadUsers() {
    if (!$currentUser?.roles?.includes('admin')) return;
    usersLoading = true;
    try {
      const token = localStorage.getItem('auth_access_token');
      const res = await fetch(`${API_BASE}/api/users`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        usersList = await res.json();
      } else {
        console.error('Failed to load users:', res.status);
      }
    } catch (e) {
      console.error(e);
    }
    usersLoading = false;
  }

  async function saveUser() {
    const token = localStorage.getItem('auth_access_token');
    if (!token) {
      alert('Error: No authentication token');
      return;
    }

    const method = editingUser ? 'PATCH' : 'POST';
    const url = editingUser
      ? `${API_BASE}/api/users/${editingUser.username}`
      : `${API_BASE}/api/users`;

    try {
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(userForm),
      });

      if (res.ok) {
        // If we have a new avatar and editing the current user, upload it
        if (avatarPreview && editingUser?.username === $currentUser?.username) {
          await uploadAvatar(avatarPreview);
        }
        showUserForm = false;
        editingUser = null;
        userForm = { username: '', email: '', password: '', firstName: '', lastName: '', role: 'user' };
        avatarPreview = null;
        avatarFile = null;
        await loadUsers();
      } else {
        const error = await res.text();
        alert(`Error: ${error}`);
      }
    } catch (e) {
      alert(`Error: ${e}`);
    }
  }

  async function uploadAvatar(dataUri: string) {
    uploadingAvatar = true;
    try {
      const token = localStorage.getItem('auth_access_token');
      const res = await fetch(`${API_BASE}/api/auth/avatar`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ avatar: dataUri }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({ error: 'Upload failed' }));
        console.error('Avatar upload failed:', data.error);
      }
    } catch (e) {
      console.error('Avatar upload error:', e);
    }
    uploadingAvatar = false;
  }

  function handleAvatarSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 400000) {
      alert($language === 'es' ? 'Imagen muy grande (max 400KB)' : 'Image too large (max 400KB)');
      return;
    }
    avatarFile = file;
    const reader = new FileReader();
    reader.onload = (e) => {
      avatarPreview = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  }

  async function deleteUser(username: string) {
    if (!confirm('Delete user ' + username + '?')) return;
    const token = localStorage.getItem('auth_access_token');
    try {
      await fetch(`${API_BASE}/api/users/${username}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      loadUsers();
    } catch (e) {
      console.error(e);
    }
  }

  // Load users on mount
  $effect(() => {
    loadUsers();
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="p-6 space-y-6 max-w-3xl mx-auto">
  <div class="flex justify-between items-center">
    <h3 class="text-lg font-semibold text-foreground">{st.users}</h3>
    <button
      onclick={() => {
        showUserForm = true;
        editingUser = null;
        userForm = { username: '', email: '', password: '', firstName: '', lastName: '', role: 'user' };
        avatarPreview = null;
        avatarFile = null;
      }}
      class="flex items-center gap-1 px-3 py-1.5 bg-primary text-white rounded-lg text-sm hover:bg-primary/90"
    >
      <Plus class="w-4 h-4" />
      {st.addUser}
    </button>
  </div>

  {#if showUserForm}
    <div class="p-5 bg-background-secondary rounded-xl border border-border space-y-4">
      <div class="flex items-center justify-between">
        <h4 class="text-sm font-semibold text-foreground">
          {editingUser ? ($language === 'es' ? 'Editar Usuario' : 'Edit User') : ($language === 'es' ? 'Nuevo Usuario' : 'New User')}
        </h4>
        <button onclick={() => { showUserForm = false; avatarPreview = null; }} class="p-1 hover:bg-background-tertiary rounded">
          <X class="w-4 h-4 text-foreground-muted" />
        </button>
      </div>

      <!-- Avatar Section (only for editing current user) -->
      {#if editingUser}
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="w-16 h-16 rounded-full bg-primary/20 flex items-center justify-center overflow-hidden border-2 border-border">
              {#if avatarPreview}
                <img src={avatarPreview} alt="Avatar" class="w-full h-full object-cover" />
              {:else if editingUser.avatar}
                <img src={editingUser.avatar} alt="Avatar" class="w-full h-full object-cover" />
              {:else}
                <User class="w-8 h-8 text-primary" />
              {/if}
            </div>
            {#if editingUser.username === $currentUser?.username}
              <label class="absolute -bottom-1 -right-1 p-1.5 bg-primary rounded-full cursor-pointer hover:bg-primary/80 transition-colors">
                <Camera class="w-3 h-3 text-white" />
                <input type="file" accept="image/*" class="hidden" onchange={handleAvatarSelect} />
              </label>
            {/if}
          </div>
          <div>
            <p class="text-sm font-medium text-foreground">{editingUser.username}</p>
            {#if editingUser.username === $currentUser?.username}
              <p class="text-xs text-foreground-muted">{$language === 'es' ? 'Click en la cámara para cambiar avatar' : 'Click camera to change avatar'}</p>
            {:else}
              <p class="text-xs text-foreground-muted">{$language === 'es' ? 'Avatar solo editable por el propio usuario' : 'Avatar only editable by the user themselves'}</p>
            {/if}
          </div>
        </div>
      {/if}

      <div class="grid grid-cols-2 gap-3">
        <input
          type="text"
          placeholder={st.username}
          bind:value={userForm.username}
          disabled={!!editingUser}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground disabled:opacity-50 focus:border-primary focus:outline-none"
        />
        <input
          type="email"
          placeholder={st.email}
          bind:value={userForm.email}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
        />
        <input
          type="text"
          placeholder={st.firstName}
          bind:value={userForm.firstName}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
        />
        <input
          type="text"
          placeholder={st.lastName}
          bind:value={userForm.lastName}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
        />
        <input
          type="password"
          placeholder={editingUser ? ($language === 'es' ? 'Nueva contraseña (dejar vacío para mantener)' : 'New password (leave empty to keep)') : st.newPassword}
          bind:value={userForm.password}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
        />
        <select
          bind:value={userForm.role}
          class="px-3 py-2 bg-background border border-border rounded-lg text-sm text-foreground focus:border-primary focus:outline-none"
        >
          <option value="user">{$language === 'es' ? 'Usuario' : 'User'}</option>
          <option value="admin">Admin</option>
        </select>
      </div>
      <div class="flex gap-2 pt-1">
        <button onclick={saveUser} class="px-4 py-2 bg-primary text-white rounded-lg text-sm hover:bg-primary/90 transition-colors">{st.save}</button>
        <button onclick={() => { showUserForm = false; avatarPreview = null; }} class="px-4 py-2 bg-background-tertiary text-foreground rounded-lg text-sm hover:bg-background-tertiary/80 transition-colors">{st.cancel}</button>
      </div>
    </div>
  {/if}

  {#if usersLoading}
    <p class="text-foreground-muted text-center py-8">{st.loading}</p>
  {:else}
    <div class="space-y-3">
      {#each usersList as user}
        <div class="flex items-center justify-between p-4 bg-background-secondary rounded-xl border border-border hover:border-primary/20 transition-colors">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-full flex items-center justify-center overflow-hidden {user.avatar ? '' : 'bg-primary/20'}">
              {#if user.avatar}
                <img src={user.avatar} alt={user.username} class="w-full h-full object-cover" />
              {:else}
                <User class="w-5 h-5 text-primary" />
              {/if}
            </div>
            <div>
              <div class="flex items-center gap-2">
                <p class="font-medium text-foreground">{user.username}</p>
                <span class="text-[10px] px-1.5 py-0.5 rounded-full {user.role === 'admin' ? 'bg-accent-orange/15 text-accent-orange' : 'bg-primary/15 text-primary'}">
                  {user.role === 'admin' ? 'Admin' : ($language === 'es' ? 'Usuario' : 'User')}
                </span>
              </div>
              <p class="text-sm text-foreground-muted">{user.email}</p>
            </div>
          </div>
          <div class="flex gap-1">
            <button
              onclick={() => {
                editingUser = user;
                showUserForm = true;
                userForm = { ...user, password: '' };
                avatarPreview = null;
                avatarFile = null;
              }}
              class="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
            ><Pencil class="w-4 h-4 text-foreground-muted" /></button>
            {#if user.username !== 'admin'}
              <button
                onclick={() => deleteUser(user.username)}
                class="p-2 hover:bg-stopped/10 rounded-lg transition-colors"
              ><Trash2 class="w-4 h-4 text-stopped" /></button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
