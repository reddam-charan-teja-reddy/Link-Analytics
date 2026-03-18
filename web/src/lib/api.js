const API_BASE = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

function clearAuthState() {
  window.dispatchEvent(new Event('auth:unauthorized'));
}

async function refreshAccessToken() {
  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
    });

    if (!res.ok) {
      clearAuthState();
      return false;
    }

    const payload = await res.json().catch(() => null);
    if (!payload?.user) {
      clearAuthState();
      return false;
    }
    return true;
  } catch {
    clearAuthState();
    return false;
  }
}

async function request(path, options = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  let res;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
        credentials: 'include',
      signal: controller.signal,
    });
  } catch (err) {
    clearTimeout(timeout);
    if (err.name === 'AbortError') throw new Error('Request timed out.');
    throw new Error('Server is unreachable.');
  }
  clearTimeout(timeout);

  if (res.status === 204) return null;

  const contentType = res.headers.get('content-type') || '';
  let payload = null;
  if (contentType.includes('application/json')) {
    payload = await res.json().catch(() => null);
  } else {
    payload = await res.text().catch(() => '');
  }

  if (res.status === 401 && options._retry !== true && path !== '/auth/google' && path !== '/auth/refresh') {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      return request(path, { ...options, _retry: true });
    }
    throw new Error('Session expired. Please sign in again.');
  }

  if (!res.ok) {
    const message = payload?.error || payload?.message || `Request failed (${res.status})`;
    if (res.status === 401) {
      clearAuthState();
    }
    throw new Error(message);
  }

  return payload;
}

export function get(path) {
  return request(path);
}

export function post(path, body) {
  return request(path, { method: 'POST', body: JSON.stringify(body) });
}

export function put(path, body) {
  return request(path, { method: 'PUT', body: JSON.stringify(body) });
}

export function del(path) {
  return request(path, { method: 'DELETE' });
}
