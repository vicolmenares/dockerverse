<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth';
  import { API_BASE } from '$lib/api/docker';
  import { Loader2, AlertCircle } from 'lucide-svelte';

  let status = $state<'loading' | 'error'>('loading');
  let errorMsg = $state('');

  onMount(async () => {
    const params = new URLSearchParams(window.location.search);
    const state = params.get('state');
    const code = params.get('code');
    const errorParam = params.get('error');

    if (errorParam) {
      status = 'error';
      errorMsg = params.get('error_description') || errorParam;
      return;
    }

    if (!state || !code) {
      status = 'error';
      errorMsg = 'Missing state or code in callback URL';
      return;
    }

    try {
      const res = await fetch(`${API_BASE}/api/auth/oidc/callback?state=${encodeURIComponent(state)}&code=${encodeURIComponent(code)}`);
      const data = await res.json();

      if (!res.ok) {
        status = 'error';
        errorMsg = data.error || 'Authentication failed';
        return;
      }

      // Save tokens and user via auth store
      await auth.handleOidcCallback(data);
      goto('/');
    } catch (e) {
      status = 'error';
      errorMsg = 'Connection error. Please try again.';
    }
  });
</script>

<div class="min-h-screen bg-background flex items-center justify-center p-4">
  <div class="text-center space-y-4">
    {#if status === 'loading'}
      <Loader2 class="w-12 h-12 text-primary animate-spin mx-auto" />
      <p class="text-foreground-muted">Completing sign in…</p>
    {:else}
      <AlertCircle class="w-12 h-12 text-stopped mx-auto" />
      <p class="text-lg font-medium text-foreground">Authentication failed</p>
      <p class="text-sm text-foreground-muted">{errorMsg}</p>
      <a href="/" class="inline-block mt-4 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors">
        Back to login
      </a>
    {/if}
  </div>
</div>
