import { writable, derived } from 'svelte/store';
import type { Container, ContainerStats, Host } from '$lib/api/docker';

// Language store - English by default
export type Language = 'es' | 'en';
export const language = writable<Language>('en');

// Translations
export const translations = {
	es: {
		hosts: 'Hosts',
		containers: 'Contenedores',
		refresh: 'Refrescar',
		settings: 'Configuración',
		search: 'Buscar contenedores...',
		total: 'Total',
		running: 'Activos',
		stopped: 'Detenidos',
		online: 'En línea',
		offline: 'Sin conexión',
		all: 'Todos',
		noContainers: 'No se encontraron contenedores',
		terminal: 'Terminal',
		logs: 'Logs',
		restart: 'Reiniciar',
		stop: 'Detener',
		start: 'Iniciar',
		filterByHost: 'Filtrar por host',
		clearFilter: 'Limpiar filtro',
		language: 'Idioma',
		theme: 'Tema',
		searchResults: 'Resultados de búsqueda',
		noSearchResults: 'No se encontraron contenedores',
		uptime: 'Tiempo activo',
		stoppedContainer: 'Contenedor detenido',
		loading: 'Cargando...',
		connectionError: 'Error de conexión al servidor',
		lightMode: 'Modo claro',
		darkMode: 'Modo oscuro'
	},
	en: {
		hosts: 'Hosts',
		containers: 'Containers',
		refresh: 'Refresh',
		settings: 'Settings',
		search: 'Search containers...',
		total: 'Total',
		running: 'Running',
		stopped: 'Stopped',
		online: 'Online',
		offline: 'Offline',
		all: 'All',
		noContainers: 'No containers found',
		terminal: 'Terminal',
		logs: 'Logs',
		restart: 'Restart',
		stop: 'Stop',
		start: 'Start',
		filterByHost: 'Filter by host',
		clearFilter: 'Clear filter',
		language: 'Language',
		theme: 'Theme',
		searchResults: 'Search results',
		noSearchResults: 'No containers found',
		uptime: 'Uptime',
		stoppedContainer: 'Container stopped',
		loading: 'Loading...',
		connectionError: 'Connection error to server',
		lightMode: 'Light Mode',
		darkMode: 'Dark Mode'
	}
};

// Helper to get translation
export function t(key: keyof typeof translations.es): string {
	let lang: Language = 'es';
	language.subscribe(l => lang = l)();
	return translations[lang][key];
}

// Containers store
export const containers = writable<Container[]>([]);
export const containerStats = writable<Map<string, ContainerStats>>(new Map());
export const hosts = writable<Host[]>([]);

// Filter/search
export const searchQuery = writable('');
export const selectedHost = writable<string | null>(null);
export const stateFilter = writable<string | null>(null);

// UI state
export const isLoading = writable(true);
export const error = writable<string | null>(null);
export const selectedContainer = writable<Container | null>(null);
export const showTerminal = writable(false);
export const showLogs = writable(false);

// Derived stores
export const filteredContainers = derived(
	[containers, searchQuery, selectedHost, stateFilter],
	([$containers, $searchQuery, $selectedHost, $stateFilter]) => {
		let result = $containers;
		
		if ($searchQuery) {
			const query = $searchQuery.toLowerCase();
			result = result.filter(c => 
				c.name.toLowerCase().includes(query) ||
				c.image.toLowerCase().includes(query) ||
				c.hostName.toLowerCase().includes(query)
			);
		}
		
		if ($selectedHost) {
			result = result.filter(c => c.hostId === $selectedHost);
		}
		
		if ($stateFilter) {
			result = result.filter(c => c.state === $stateFilter);
		}
		
		return result;
	}
);

export const containerCount = derived(containers, $containers => ({
	total: $containers.length,
	running: $containers.filter(c => c.state === 'running').length,
	stopped: $containers.filter(c => c.state === 'exited').length,
	other: $containers.filter(c => !['running', 'exited'].includes(c.state)).length
}));

// Get stats for a specific container
export function getContainerStats(containerId: string): ContainerStats | undefined {
	let stats: ContainerStats | undefined;
	containerStats.subscribe(map => {
		stats = map.get(containerId);
	})();
	return stats;
}

// Update stats map
export function updateStats(newStats: ContainerStats[]) {
	containerStats.update(map => {
		for (const stat of newStats) {
			map.set(stat.id, stat);
		}
		return new Map(map);
	});
}

// WebSocket for real-time updates
let ws: WebSocket | null = null;
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null;

export function connectWebSocket() {
	if (ws && ws.readyState === WebSocket.OPEN) return;
	
	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	const wsUrl = `${protocol}//${window.location.host}/ws/events`;
	
	try {
		ws = new WebSocket(wsUrl);
		
		ws.onopen = () => {
			console.log('🔌 WebSocket connected');
			error.set(null);
		};
		
		ws.onmessage = (event) => {
			try {
				const msg = JSON.parse(event.data);
				
				if (msg.type === 'containers' && Array.isArray(msg.data)) {
					containers.set(msg.data);
					isLoading.set(false);
				} else if (msg.type === 'stats' && Array.isArray(msg.data)) {
					updateStats(msg.data);
				} else if (msg.type === 'hosts' && Array.isArray(msg.data)) {
					hosts.set(msg.data);
				}
			} catch (e) {
				console.error('WS message parse error:', e);
			}
		};
		
		ws.onclose = () => {
			console.log('🔌 WebSocket disconnected, reconnecting...');
			ws = null;
			// Reconnect after 3 seconds
			wsReconnectTimer = setTimeout(connectWebSocket, 3000);
		};
		
		ws.onerror = (e) => {
			console.error('WebSocket error:', e);
		};
	} catch (e) {
		console.error('WebSocket connection failed:', e);
		// Fallback to SSE
		wsReconnectTimer = setTimeout(connectWebSocket, 5000);
	}
}

export function disconnectWebSocket() {
	if (wsReconnectTimer) {
		clearTimeout(wsReconnectTimer);
		wsReconnectTimer = null;
	}
	if (ws) {
		ws.close();
		ws = null;
	}
}
