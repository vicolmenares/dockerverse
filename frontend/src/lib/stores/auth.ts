import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';
import { API_BASE } from '$lib/api/docker';

// Auth storage key constant
const AUTH_STORAGE_KEY = 'auth';

// =============================================================================
// Activity Tracking for Auto-Logout
// =============================================================================

const INACTIVITY_TIMEOUT = 30 * 60 * 1000; // 30 minutes in milliseconds
const ACTIVITY_CHECK_INTERVAL = 60 * 1000; // Check every minute

let lastActivityTime = Date.now();
let activityCheckInterval: ReturnType<typeof setInterval> | null = null;
let logoutCallback: (() => void) | null = null;

export function updateActivity() {
	lastActivityTime = Date.now();
}

export function setupActivityTracking(onLogout: () => void) {
	if (!browser) return;
	
	logoutCallback = onLogout;
	
	// Track user activity
	const events = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart', 'click'];
	events.forEach(event => {
		document.addEventListener(event, updateActivity, { passive: true });
	});
	
	// Check for inactivity periodically
	activityCheckInterval = setInterval(() => {
		const timeSinceActivity = Date.now() - lastActivityTime;
		if (timeSinceActivity >= INACTIVITY_TIMEOUT) {
			console.log('[Auth] Auto-logout due to inactivity');
			logoutCallback?.();
		}
	}, ACTIVITY_CHECK_INTERVAL);
}

export function cleanupActivityTracking() {
	if (!browser) return;
	
	const events = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart', 'click'];
	events.forEach(event => {
		document.removeEventListener(event, updateActivity);
	});
	
	if (activityCheckInterval) {
		clearInterval(activityCheckInterval);
		activityCheckInterval = null;
	}
}

export function getTimeUntilLogout(): number {
	const remaining = INACTIVITY_TIMEOUT - (Date.now() - lastActivityTime);
	return Math.max(0, remaining);
}

// =============================================================================
// Types - OIDC/Keycloak compatible
// =============================================================================

export interface User {
	id: string;
	username: string;
	email: string;
	firstName: string;
	lastName: string;
	avatar?: string;
	roles: string[];
	createdAt: string;
	lastLogin?: string;
}

export interface AuthTokens {
	accessToken: string;
	refreshToken: string;
	expiresIn: number;
	tokenType: string;
}

export interface AuthState {
	user: User | null;
	tokens: AuthTokens | null;
	isAuthenticated: boolean;
	isLoading: boolean;
	error: string | null;
}

export interface LoginCredentials {
	username: string;
	password: string;
	rememberMe?: boolean;
	totpCode?: string;
	recoveryCode?: string;
}

export interface LoginResult {
	success: boolean;
	requiresTOTP?: boolean;
	tempToken?: string;
	error?: string;
}

export interface AuthConfig {
	// For future Keycloak integration
	provider: 'local' | 'keycloak' | 'oauth2';
	keycloakUrl?: string;
	realm?: string;
	clientId?: string;
}

// =============================================================================
// Configuration
// =============================================================================

export const authConfig = writable<AuthConfig>({
	provider: 'local'
});

// =============================================================================
// Auth Store
// =============================================================================

const initialState: AuthState = {
	user: null,
	tokens: null,
	isAuthenticated: false,
	isLoading: true,
	error: null
};

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);

	// Load from localStorage or sessionStorage on init
	if (browser) {
		const stored = localStorage.getItem('auth') || sessionStorage.getItem('auth');
		if (stored) {
			try {
				const data = JSON.parse(stored);
				// Check if token is still valid
				if (data.tokens && isTokenValid(data.tokens)) {
					// Also restore individual tokens for API calls
					if (data.tokens.accessToken) {
						localStorage.setItem('auth_access_token', data.tokens.accessToken);
					}
					if (data.tokens.refreshToken) {
						localStorage.setItem('auth_refresh_token', data.tokens.refreshToken);
					}
					set({ ...data, isLoading: false });
				} else {
					localStorage.removeItem('auth');
					sessionStorage.removeItem('auth');
					localStorage.removeItem('auth_access_token');
					localStorage.removeItem('auth_refresh_token');
					set({ ...initialState, isLoading: false });
				}
			} catch {
				set({ ...initialState, isLoading: false });
			}
		} else {
			set({ ...initialState, isLoading: false });
		}
	}

	return {
		subscribe,
		
		// Login with credentials
		login: async (credentials: LoginCredentials): Promise<LoginResult> => {
			update(s => ({ ...s, isLoading: true, error: null }));
			
			try {
				const config = get(authConfig);
				
				if (config.provider === 'keycloak') {
					// Future Keycloak integration
					const success = await loginWithKeycloak(credentials, config);
					return { success };
				}
				
				// Local authentication
				const response = await fetch(`${API_BASE}/api/auth/login`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						username: credentials.username,
						password: credentials.password,
						rememberMe: credentials.rememberMe,
						totpCode: credentials.totpCode,
						recoveryCode: credentials.recoveryCode
					})
				});

				if (!response.ok) {
					const error = await response.json();
					update(s => ({ ...s, isLoading: false }));
					return { 
						success: false, 
						error: error.error || error.message || 'Invalid credentials' 
					};
				}

				const data = await response.json();
				
				// Check if 2FA is required
				if (data.requiresTOTP) {
					update(s => ({ ...s, isLoading: false }));
					return {
						success: false,
						requiresTOTP: true,
						tempToken: data.tempToken
					};
				}
				
				// Map backend role (string) to frontend roles (array)
				const user = {
					...data.user,
					roles: data.user.role ? [data.user.role] : ['user']
				};
				
				const newState: AuthState = {
					user,
					tokens: data.tokens,
					isAuthenticated: true,
					isLoading: false,
					error: null
				};

				set(newState);
				
				if (browser) {
					// Store individual tokens for API helper
					localStorage.setItem('auth_access_token', data.tokens.accessToken);
					localStorage.setItem('auth_refresh_token', data.tokens.refreshToken);
					// Always persist auth state to localStorage for page refresh support
					localStorage.setItem('auth', JSON.stringify(newState));
				}

				return { success: true };
			} catch (e) {
				const error = e instanceof Error ? e.message : 'Login failed';
				update(s => ({ ...s, isLoading: false, error }));
				return { success: false, error };
			}
		},

		// Logout
		logout: async () => {
			try {
				const state = get({ subscribe });
				if (state.tokens) {
					await fetch(`${API_BASE}/api/auth/logout`, {
						method: 'POST',
						headers: {
							'Authorization': `Bearer ${state.tokens.accessToken}`
						}
					});
				}
			} catch {
				// Ignore logout API errors
			}

			set({ ...initialState, isLoading: false });
			
			if (browser) {
				localStorage.removeItem('auth');
				localStorage.removeItem('auth_access_token');
				localStorage.removeItem('auth_refresh_token');
				sessionStorage.removeItem('auth');
			}
		},

		// Refresh token
		refreshToken: async (): Promise<boolean> => {
			const state = get({ subscribe });
			if (!state.tokens?.refreshToken) return false;

			try {
				const response = await fetch(`${API_BASE}/api/auth/refresh`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ refreshToken: state.tokens.refreshToken })
				});

				if (!response.ok) throw new Error('Token refresh failed');

				const data = await response.json();
				update(s => ({
					...s,
					tokens: data.tokens
				}));

				if (browser) {
					const stored = localStorage.getItem('auth') || sessionStorage.getItem('auth');
					if (stored) {
						const storage = localStorage.getItem('auth') ? localStorage : sessionStorage;
						storage.setItem('auth', JSON.stringify({ ...JSON.parse(stored), tokens: data.tokens }));
					}
				}

				return true;
			} catch {
				// Force logout on refresh failure
				set({ ...initialState, isLoading: false });
				if (browser) {
					localStorage.removeItem('auth');
					sessionStorage.removeItem('auth');
				}
				return false;
			}
		},

		// Update user profile
		updateProfile: async (updates: Partial<User>): Promise<boolean> => {
			const state = get({ subscribe });
			if (!state.tokens) return false;

			try {
				const response = await fetch('/api/auth/profile', {
					method: 'PATCH',
					headers: {
						'Content-Type': 'application/json',
						'Authorization': `Bearer ${state.tokens.accessToken}`
					},
					body: JSON.stringify(updates)
				});

				if (!response.ok) throw new Error('Profile update failed');

				const data = await response.json();
				update(s => ({
					...s,
					user: { ...s.user!, ...data.user }
				}));

				return true;
			} catch {
				return false;
			}
		},

		// Change password
		changePassword: async (currentPassword: string, newPassword: string): Promise<boolean> => {
			const state = get({ subscribe });
			if (!state.tokens) return false;

			try {
				const response = await fetch(`${API_BASE}/api/auth/password`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
						'Authorization': `Bearer ${state.tokens.accessToken}`
					},
					body: JSON.stringify({ currentPassword, newPassword })
				});

				if (!response.ok) {
					const error = await response.json();
					throw new Error(error.message || 'Password change failed');
				}

				return true;
			} catch (err) {
				console.error('Password change error:', err);
				return false;
			}
		},

		// Get access token for API calls
		getAccessToken: (): string | null => {
			const state = get({ subscribe });
			return state.tokens?.accessToken || null;
		},

		// Check if user has role
		hasRole: (role: string): boolean => {
			const state = get({ subscribe });
			return state.user?.roles.includes(role) || false;
		},

		// Clear error
		clearError: () => {
			update(s => ({ ...s, error: null }));
		},

		// Refresh current user data from the API
		refreshUser: async (): Promise<void> => {
			const state = get({ subscribe });
			if (!state.tokens || !state.user) return;

			try {
				const response = await fetch(`${API_BASE}/api/users/${state.user.username}`, {
					headers: {
						'Authorization': `Bearer ${state.tokens.accessToken}`
					}
				});

				if (response.ok) {
					const userData = await response.json();
					const user = {
						...userData,
						roles: userData.role ? [userData.role] : ['user']
					};
					
					update(s => ({ ...s, user }));
					
					if (browser) {
						const stored = localStorage.getItem('auth') || sessionStorage.getItem('auth');
						if (stored) {
							const parsed = JSON.parse(stored);
							const storage = localStorage.getItem('auth') ? localStorage : sessionStorage;
							storage.setItem('auth', JSON.stringify({ ...parsed, user }));
						}
					}
				}
			} catch (err) {
				console.error('Failed to refresh user:', err);
			}
		}
	};
}

// Helper function to check token validity
function isTokenValid(tokens: AuthTokens): boolean {
	try {
		// Decode JWT payload (base64)
		const payload = JSON.parse(atob(tokens.accessToken.split('.')[1]));
		const exp = payload.exp * 1000; // Convert to milliseconds
		return Date.now() < exp;
	} catch {
		return false;
	}
}

// Future Keycloak integration placeholder
async function loginWithKeycloak(credentials: LoginCredentials, config: AuthConfig): Promise<boolean> {
	// This will be implemented when integrating with Keycloak
	// Uses OIDC password grant or redirect flow
	console.log('Keycloak login not yet implemented', credentials, config);
	throw new Error('Keycloak integration not configured');
}

// Export the store
export const auth = createAuthStore();

// Derived stores for convenience
export const isAuthenticated = derived(auth, $auth => $auth.isAuthenticated);
export const currentUser = derived(auth, $auth => $auth.user);
export const isLoading = derived(auth, $auth => $auth.isLoading);
export const authError = derived(auth, $auth => $auth.error);

// Export auth header helper
export function getAuthHeaders(): Record<string, string> {
	const token = auth.getAccessToken();
	return token ? { 'Authorization': `Bearer ${token}` } : {};
}

// Avatar API functions
export async function uploadAvatar(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = async () => {
			const base64Data = reader.result as string;
			
			try {
				const response = await fetch(`${API_BASE}/api/auth/avatar`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
						...getAuthHeaders()
					},
					body: JSON.stringify({ avatar: base64Data })
				});
				
				if (!response.ok) {
					const error = await response.json();
					throw new Error(error.error || 'Failed to upload avatar');
				}
				
				const data = await response.json();
				
				// Update user in store
				auth.update(state => {
					if (state.user) {
						return {
							...state,
							user: { ...state.user, avatar: data.avatar }
						};
					}
					return state;
				});
				
				// Update in localStorage
				const stored = localStorage.getItem(AUTH_STORAGE_KEY);
				if (stored) {
					const parsed = JSON.parse(stored);
					if (parsed.user) {
						parsed.user.avatar = data.avatar;
						localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(parsed));
					}
				}
				
				resolve(data.avatar);
			} catch (e) {
				reject(e);
			}
		};
		reader.onerror = () => reject(new Error('Failed to read file'));
		reader.readAsDataURL(file);
	});
}

export async function deleteAvatar(): Promise<void> {
	const response = await fetch(`${API_BASE}/api/auth/avatar`, {
		method: 'DELETE',
		headers: getAuthHeaders()
	});
	
	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to delete avatar');
	}
	
	// Update user in store
	auth.update(state => {
		if (state.user) {
			return {
				...state,
				user: { ...state.user, avatar: undefined }
			};
		}
		return state;
	});
	
	// Update in localStorage
	const stored = localStorage.getItem(AUTH_STORAGE_KEY);
	if (stored) {
		const parsed = JSON.parse(stored);
		if (parsed.user) {
			delete parsed.user.avatar;
			localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(parsed));
		}
	}
}
