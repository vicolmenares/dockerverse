// API Configuration
// Uses PUBLIC_API_URL from environment, falls back to localhost in dev
const getApiBase = (): string => {
	if (import.meta.env.DEV) return 'http://localhost:3001';
	// In production, use the PUBLIC_API_URL passed at build time
	return import.meta.env.PUBLIC_API_URL || '';
};
export const API_BASE = getApiBase();

// Auth helper
function getAuthHeaders(): Record<string, string> {
	const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_access_token') : null;
	const headers: Record<string, string> = { 'Content-Type': 'application/json' };
	if (token) headers['Authorization'] = `Bearer ${token}`;
	return headers;
}

// Get auth token for SSE/EventSource (which doesn't support headers)
function getAuthToken(): string | null {
	return typeof localStorage !== 'undefined' ? localStorage.getItem('auth_access_token') : null;
}

export interface Container {
	id: string;
	name: string;
	image: string;
	status: string;
	state: string;
	created: number;
	hostId: string;
	hostName: string;
	ports: PortMapping[];
	labels: Record<string, string>;
	health: string;
	networks: Record<string, string>;
	volumes: number;
}

export interface PortMapping {
	private: number;
	public: number;
	type: string;
}

export interface ContainerStats {
	id: string;
	name: string;
	hostId: string;
	cpuPercent: number;
	memoryUsage: number;
	memoryLimit: number;
	memoryPercent: number;
	networkRx: number;
	networkTx: number;
	blockRead: number;
	blockWrite: number;
}

export interface DiskInfo {
	mountPoint: string;
	device: string;
	totalBytes: number;
	usedBytes: number;
	freeBytes: number;
}

export interface Host {
	id: string;
	name: string;
	containerCount: number;
	runningCount: number;
	cpuPercent: number;
	memoryPercent: number;
	memoryUsed: number;
	memoryTotal: number;
	online: boolean;
	disks: DiskInfo[];
	sshHost?: string;
}

export interface ImageUpdate {
	containerId: string;
	containerName: string;
	image: string;
	hostId: string;
	currentDigest: string;
	latestDigest?: string;
	currentTag: string;
	latestTag?: string;
	hasUpdate: boolean;
	checkedAt: number;
}

export interface BulkUpdateResultItem {
	containerId: string;
	containerName: string;
	hostId: string;
	success: boolean;
	error?: string;
}

export interface BulkUpdateResult {
	matched: number;
	updated: number;
	failed: number;
	results: BulkUpdateResultItem[];
}

// Fetch with timeout utility
async function fetchWithTimeout(url: string, options: RequestInit, timeoutMs = 8000): Promise<Response> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), timeoutMs);
	try {
		return await fetch(url, { ...options, signal: controller.signal });
	} finally {
		clearTimeout(timer);
	}
}

// API Functions
export async function fetchContainers(): Promise<Container[]> {
	const res = await fetchWithTimeout(`${API_BASE}/api/containers`, { headers: getAuthHeaders() });
	if (!res.ok) throw new Error('Failed to fetch containers');
	return res.json();
}

export async function fetchHosts(): Promise<Host[]> {
	const res = await fetchWithTimeout(`${API_BASE}/api/hosts`, { headers: getAuthHeaders() }, 15000);
	if (!res.ok) throw new Error('Failed to fetch hosts');
	return res.json();
}

export async function searchContainers(query: string): Promise<Container[]> {
	const res = await fetchWithTimeout(`${API_BASE}/api/search?q=${encodeURIComponent(query)}`, { headers: getAuthHeaders() });
	if (!res.ok) throw new Error('Failed to search');
	return res.json();
}

export async function containerAction(hostId: string, containerId: string, action: 'start' | 'stop' | 'restart'): Promise<void> {
	const res = await fetchWithTimeout(`${API_BASE}/api/containers/${hostId}/${containerId}/${action}`, {
		method: 'POST',
		headers: getAuthHeaders()
	}, 15000);
	if (!res.ok) throw new Error(`Failed to ${action} container`);
}

// Image updates API
export async function fetchImageUpdates(): Promise<ImageUpdate[]> {
	const res = await fetchWithTimeout(`${API_BASE}/api/updates`, { headers: getAuthHeaders() }, 30000);
	if (!res.ok) throw new Error('Failed to fetch image updates');
	return res.json();
}

export async function checkImageUpdate(hostId: string, containerId: string): Promise<ImageUpdate> {
	const res = await fetchWithTimeout(`${API_BASE}/api/updates/${hostId}/${containerId}/check`, {
		method: 'POST',
		headers: getAuthHeaders()
	}, 15000);
	if (!res.ok) throw new Error('Failed to check for image update');
	return res.json();
}

// Trigger Watchtower update for a specific container
export async function triggerContainerUpdate(hostId: string, containerId: string): Promise<{ success: boolean; message: string }> {
	const res = await fetchWithTimeout(`${API_BASE}/api/containers/${hostId}/${containerId}/update`, {
		method: 'POST',
		headers: getAuthHeaders()
	}, 30000);
	if (!res.ok) {
		const data = await res.json().catch(() => ({ error: 'Update failed' }));
		throw new Error(data.error || 'Failed to trigger update');
	}
	return res.json();
}

export async function triggerBulkUpdate(
	hostId?: string,
	nameFilter?: string,
	dryRun = false
): Promise<BulkUpdateResult> {
	const containers = await fetchContainers();
	const filter = nameFilter?.toLowerCase().trim() || '';
	const matched = containers.filter((container) => {
		if (hostId && container.hostId !== hostId) return false;
		if (filter && !container.name.toLowerCase().includes(filter)) return false;
		return true;
	});

	if (dryRun) {
		return {
			matched: matched.length,
			updated: 0,
			failed: 0,
			results: []
		};
	}

	const results = await Promise.all(
		matched.map(async (container) => {
			try {
				await triggerContainerUpdate(container.hostId, container.id);
				return {
					containerId: container.id,
					containerName: container.name,
					hostId: container.hostId,
					success: true
				};
			} catch (error) {
				return {
					containerId: container.id,
					containerName: container.name,
					hostId: container.hostId,
					success: false,
					error: error instanceof Error ? error.message : 'Update failed'
				};
			}
		})
	);

	const updated = results.filter((r) => r.success).length;
	const failed = results.length - updated;

	return {
		matched: matched.length,
		updated,
		failed,
		results
	};
}

// SSE Event Source with callbacks for all message types
export interface SSECallbacks {
	onStats?: (stats: ContainerStats[]) => void;
	onContainers?: (containers: Container[]) => void;
	onHosts?: (hosts: Host[]) => void;
	onError?: (error: Event) => void;
}

// SSE Event Sources - uses /api/events endpoint with token auth
export function createStatsStream(onMessage: (stats: ContainerStats[]) => void, onError?: (error: Event) => void): EventSource {
	const token = getAuthToken();
	// Backend uses /events endpoint, not /stats/stream
	const url = token ? `${API_BASE}/api/events?token=${encodeURIComponent(token)}` : `${API_BASE}/api/events`;
	const eventSource = new EventSource(url);
	
	eventSource.onmessage = (event) => {
		try {
			const data = JSON.parse(event.data);
			// Handle both 'stats' type and direct stats data
			if (data.type === 'stats' && data.data) {
				onMessage(data.data);
			}
		} catch (e) {
			console.error('Error parsing SSE message:', e);
		}
	};
	
	eventSource.onerror = (error) => {
		console.error('SSE connection error - will retry:', error);
		onError?.(error);
	};
	
	return eventSource;
}

// New SSE stream with all event types
export function createEventStream(callbacks: SSECallbacks): EventSource {
	const token = getAuthToken();
	const url = token ? `${API_BASE}/api/events?token=${encodeURIComponent(token)}` : `${API_BASE}/api/events`;
	const eventSource = new EventSource(url);
	
	eventSource.onmessage = (event) => {
		try {
			const data = JSON.parse(event.data);
			if (data.type === 'stats' && data.data && callbacks.onStats) {
				callbacks.onStats(data.data);
			} else if (data.type === 'containers' && data.data && callbacks.onContainers) {
				callbacks.onContainers(data.data);
			} else if (data.type === 'hosts' && data.data && callbacks.onHosts) {
				callbacks.onHosts(data.data);
			}
		} catch (e) {
			console.error('Error parsing SSE message:', e);
		}
	};
	
	eventSource.onerror = (error) => {
		console.error('SSE connection error - will retry:', error);
		callbacks.onError?.(error);
	};
	
	return eventSource;
}

export function createLogStream(hostId: string, containerId: string, onLog: (line: string) => void): EventSource {
	const token = getAuthToken();
	const baseUrl = `${API_BASE}/api/logs/${hostId}/${containerId}/stream`;
	const url = token ? `${baseUrl}?token=${encodeURIComponent(token)}` : baseUrl;
	const eventSource = new EventSource(url);
	
	eventSource.onmessage = (event) => {
		try {
			const data = JSON.parse(event.data);
			if (data.type === 'log') {
				onLog(data.data);
			}
		} catch (e) {
			console.error('Error parsing log message:', e);
		}
	};
	
	return eventSource;
}

// WebSocket for Terminal
export function createTerminalConnection(hostId: string, containerId: string): WebSocket {
	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	// In production, use the API_BASE host for WebSocket
	let wsBase: string;
	if (import.meta.env.DEV) {
		wsBase = 'ws://localhost:3001';
	} else if (API_BASE) {
		// Extract host from API_BASE (e.g., http://192.168.1.145:3006 -> ws://192.168.1.145:3006)
		wsBase = API_BASE.replace(/^http/, 'ws');
	} else {
		wsBase = `${protocol}//${window.location.host}`;
	}
	return new WebSocket(`${wsBase}/ws/terminal/${hostId}/${containerId}`);
}

// Alias for backwards compatibility
export const createTerminalWebSocket = createTerminalConnection;

// Utility functions
export function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function formatUptime(status: string): string {
	const match = status.match(/Up\s+(.+?)(?:\s*\(|$)/);
	return match ? match[1] : status;
}

export function getStateColor(state: string): string {
	switch (state.toLowerCase()) {
		case 'running': return 'running';
		case 'exited':
		case 'dead': return 'stopped';
		case 'paused': return 'paused';
		case 'restarting': return 'restarting';
		default: return 'foreground-muted';
	}
}
