/**
 * API service layer — typed wrappers for all backend endpoints.
 *
 * All paths are relative (/api/...) — Vite dev server proxies /api to
 * http://localhost:8080. In production the reverse proxy handles it.
 */

// ─── Token management ───────────────────────────────────────────────

const TOKEN_KEY = 'text2midi_token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

// ─── Helpers ────────────────────────────────────────────────────────

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  opts?: { auth?: boolean; raw?: boolean },
): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined && !opts?.raw) {
    headers['Content-Type'] = 'application/json';
  }
  if (opts?.auth) {
    const token = getToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? (opts?.raw ? (body as BodyInit) : JSON.stringify(body)) : undefined,
  });

  if (!res.ok) {
    let errMsg: string;
    try {
      const errBody = await res.json();
      errMsg = errBody.error || errBody.message || `HTTP ${res.status}`;
    } catch {
      errMsg = `HTTP ${res.status}`;
    }
    throw new Error(errMsg);
  }

  // For binary downloads
  if (opts?.raw) {
    return res as unknown as T;
  }

  // 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

// ─── Auth ───────────────────────────────────────────────────────────

export interface User {
  id: number;
  username: string;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

export function register(username: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/api/user/register', { username, password });
}

export function login(username: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/api/user/login', { username, password });
}

export function logout(): Promise<{ status: string }> {
  return request<{ status: string }>('POST', '/api/user/logout', undefined, { auth: true });
}

// ─── Preferences ────────────────────────────────────────────────────

export interface Prefs {
  style: string;
  key: string;
  bpm: number;
  bars: number;
  flatVel: number;
  chaos: number;
  progression: string;
  mode: string;
  loopable: boolean;
  pentatonic: boolean;
}

export function getPrefs(): Promise<Prefs> {
  return request<Prefs>('GET', '/api/user/prefs', undefined, { auth: true });
}

export function savePrefs(prefs: Prefs): Promise<{ status: string }> {
  return request<{ status: string }>('PUT', '/api/user/prefs', prefs, { auth: true });
}

// ─── Generation History ─────────────────────────────────────────────

export interface HistoryEntry {
  id: number;
  prompt?: string;
  style: string;
  key: string;
  bpm: number;
  bars: number;
  midiPath?: string;
  duration?: number;
  noteCount?: number;
  createdAt: string;
}

export interface HistoryResponse {
  history: HistoryEntry[];
}

export function getHistory(): Promise<HistoryResponse> {
  return request<HistoryResponse>('GET', '/api/user/history', undefined, { auth: true });
}

// ─── Info ───────────────────────────────────────────────────────────

export interface InfoResponse {
  styles: string[];
  tiers: Record<string, number>;
}

export function getInfo(): Promise<InfoResponse> {
  return request<InfoResponse>('GET', '/api/info');
}

// ─── Generate ───────────────────────────────────────────────────────

export interface GenerateRequest {
  prompt: string;
  style: string;
  bpm: number;
  bars: number;
  key: string;
  tier?: string;
  seed?: number;
}

export interface GenerateResponse {
  success: boolean;
  fileId?: string;
  fileName?: string;
  fileSize?: number;
  durationSeconds: number;
  tracks: number;
  meta?: {
    output_path: string;
    ticks_per_beat: number;
    total_tracks: number;
    total_note_events: number;
    duration_seconds: number;
  };
  error?: string;
}

export function generateMidi(req: GenerateRequest): Promise<GenerateResponse> {
  return request<GenerateResponse>('POST', '/api/generate', req);
}

// ─── Files ──────────────────────────────────────────────────────────

export interface FileRecord {
  id: string;
  file_name: string;
  file_path: string;
  file_size: number;
  created_at: string;
  render_meta?: {
    output_path: string;
    ticks_per_beat: number;
    total_tracks: number;
    total_note_events: number;
    duration_seconds: number;
  };
}

export function listFiles(): Promise<FileRecord[]> {
  return request<FileRecord[]>('GET', '/api/files');
}

/**
 * Get the download URL for a MIDI file by its ID.
 * This is a URL string, not a fetch — use it as an <a> href or window.open.
 */
export function getDownloadUrl(fileId: string): string {
  return `/api/files/${encodeURIComponent(fileId)}`;
}

/** Download a MIDI file as a Blob (e.g. for programmatic use). */
export async function downloadFile(fileId: string): Promise<Blob> {
  const res = await request<Response>('GET', `/api/files/${encodeURIComponent(fileId)}`, undefined, {
    auth: false,
    raw: true,
  });
  return res.blob();
}

// ─── DNA ────────────────────────────────────────────────────────────

export interface DNAExtractRequest {
  events_by_track: Record<string, Array<{ pitch: number; start_beat: number; duration_beat: number; velocity: number }>>;
  total_bars: number;
  key: string;
}

export interface DNAExtractResponse {
  dna: unknown;  // MusicDNA structure
  quality: number;
}

export function extractDNA(req: DNAExtractRequest): Promise<DNAExtractResponse> {
  return request<DNAExtractResponse>('POST', '/api/dna/extract', req);
}

export interface DNATemplate {
  name: string;
  style: string;
  dna: unknown;
  quality: number;
  source: string;
}

export function listDNALibrary(): Promise<DNATemplate[]> {
  return request<DNATemplate[]>('GET', '/api/dna/library');
}

export function getDNATemplate(name: string): Promise<DNATemplate> {
  return request<DNATemplate>('GET', `/api/dna/library/${encodeURIComponent(name)}`);
}
