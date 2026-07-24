const BASE = '/api/v1';

let token: string | null = localStorage.getItem('aster_token');

export function setToken(t: string | null) {
  token = t;
  if (t) localStorage.setItem('aster_token', t);
  else localStorage.removeItem('aster_token');
}

export function getToken() {
  return token;
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  const data = await res.json();

  if (!res.ok) {
    throw new Error(data.error || data.message || `HTTP ${res.status}`);
  }
  return data as T;
}

// ─── Auth ───

export interface User {
  id: string;
  username: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export const auth = {
  register: (body: { username: string; email: string; password: string }) =>
    request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  login: (body: { username: string; password: string }) =>
    request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  me: () => request<User>('/auth/me'),
};

// ─── Sessions ───

export interface Session {
  id: string;
  title: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export const sessions = {
  list: () => request<Session[]>('/sessions'),
  get: (id: string) => request<Session>(`/sessions/${id}`),
  create: (title: string) =>
    request<Session>('/sessions', {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
  delete: (id: string) =>
    request<void>(`/sessions/${id}`, { method: 'DELETE' }),
};

// ─── Skills ───

export interface Skill {
  id: string;
  name: string;
  description: string;
  category: string;
  content: string;
  created_at: string;
}

export const skills = {
  list: () => request<Skill[]>('/skills'),
  create: (body: { name: string; description: string; category: string; content: string }) =>
    request<Skill>('/skills', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  delete: (id: string) =>
    request<void>(`/skills/${id}`, { method: 'DELETE' }),
};

// ─── Modules ───

export interface Module {
  name: string;
  version: string;
  description: string;
}

export const modules = {
  list: () => request<Module[]>('/modules'),
};

// ─── Health ───

export const health = () =>
  request<{ status: string; version: string }>('/health');
