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
  // 后端无单条查询路由，从列表中取标题
  get: async (id: string) => {
    const all = await request<Session[]>('/sessions');
    const found = all.find((s) => s.id === id);
    if (!found) throw new Error('session not found');
    return found;
  },
  create: (title: string) =>
    request<Session>('/sessions', {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
  delete: (id: string) =>
    request<void>(`/sessions/${id}`, { method: 'DELETE' }),
};

// ─── Skills ───
// 字段与后端 skillItem 对齐:name 为主键，无 id/category/content

export interface Skill {
  name: string;
  description: string;
  agent: string;
  tags: string[];
  source: string;
}

export const skills = {
  // 后端返回 { skills: [...] }，需解包
  list: () =>
    request<{ skills: Skill[] }>('/skills').then((r) => r.skills ?? []),
  create: (body: { name: string; description: string; instructions: string; tags?: string[] }) =>
    request<Skill>('/skills/custom', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  delete: (name: string) =>
    request<void>(`/skills/custom/${encodeURIComponent(name)}`, { method: 'DELETE' }),
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

// health 端点在 /api/health（无 v1 前缀），故绕过 BASE 直接调用
export const health = () =>
  fetch('/api/health').then((r) => r.json() as Promise<{ ok: boolean; service: string }>);
