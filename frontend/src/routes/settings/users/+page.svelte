<script lang="ts">
  import {
    User,
    Plus,
    Pencil,
    Trash2,
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
      console.error('No auth token found');
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
        showUserForm = false;
        editingUser = null;
        userForm = { username: '', email: '', password: '', firstName: '', lastName: '', role: 'user' };
        await loadUsers();
      } else {
        const error = await res.text();
        console.error('Save user failed:', error);
        alert(`Error: ${error}`);
      }
    } catch (e) {
      console.error('Save user error:', e);
      alert(`Error: ${e}`);
    }
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
<div class="p-4 space-y-4">
  <div class="flex justify-between items-center">
    <h3 class="text-lg font-semibold text-foreground">{st.users}</h3>
    <button
      onclick={() => {
        showUserForm = true;
        editingUser = null;
        userForm = { username: '', email: '', password: '', firstName: '', lastName: '', role: 'user' };
      }}
      class="flex items-center gap-1 px-3 py-1.5 bg-primary text-white rounded-lg text-sm hover:bg-primary/90"
    >
      <Plus class="w-4 h-4" />
      {st.addUser}
    </button>
  </div>
  {#if showUserForm}
    <div class="p-4 bg-background rounded-lg border border-border space-y-3">
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
          <option value="user">{$language === 'es' ? 'Usuario' : 'User'}</option>
          <option value="admin">Admin</option>
        </select>
      </div>
      <div class="flex gap-2">
        <button onclick={saveUser} class="px-4 py-2 bg-primary text-white rounded-lg text-sm">{st.save}</button>
        <button onclick={() => (showUserForm = false)} class="px-4 py-2 bg-background-tertiary text-foreground rounded-lg text-sm">{st.cancel}</button>
      </div>
    </div>
  {/if}
  {#if usersLoading}
    <p class="text-foreground-muted text-center">{st.loading}</p>
  {:else}
    <div class="space-y-2">
      {#each usersList as user}
        <div class="flex items-center justify-between p-3 bg-background rounded-lg border border-border">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 bg-primary/20 rounded-full flex items-center justify-center">
              <User class="w-5 h-5 text-primary" />
            </div>
            <div>
              <p class="font-medium text-foreground">{user.username}</p>
              <p class="text-sm text-foreground-muted">
                {user.email} • {user.role === 'admin' ? 'Admin' : ($language === 'es' ? 'Usuario' : 'User')}
              </p>
            </div>
          </div>
          <div class="flex gap-2">
            <button
              onclick={() => {
                editingUser = user;
                showUserForm = true;
                userForm = { ...user, password: '' };
              }}
              class="p-2 hover:bg-background-tertiary rounded-lg"
            ><Pencil class="w-4 h-4 text-foreground-muted" /></button>
            {#if user.username !== 'admin'}
              <button
                onclick={() => deleteUser(user.username)}
                class="p-2 hover:bg-stopped/10 rounded-lg"
              ><Trash2 class="w-4 h-4 text-stopped" /></button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
