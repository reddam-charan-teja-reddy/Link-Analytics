import { del, get, post, put } from './api';

export const authApi = {
  google: (data) => post('/auth/google', data),
  refresh: () => post('/auth/refresh', {}),
  logout: () => post('/auth/logout', {}),
  me: () => get('/auth/me'),
};

export const linksApi = {
  list: ({ groupId, source } = {}) => {
    const query = new URLSearchParams();
    if (groupId) query.set('group_id', groupId);
    if (source) query.set('source', source);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return get(`/api/links${suffix}`);
  },
  create: (data) => post('/api/links', data),
  getOne: (id) => get(`/api/links/${id}`),
  update: (id, data) => put(`/api/links/${id}`, data),
  remove: (id) => del(`/api/links/${id}`),
  listSources: (linkId) => get(`/api/links/${linkId}/sources`),
  createSource: (linkId, data) => post(`/api/links/${linkId}/sources`, data),
  removeSource: (linkId, sourceId) => del(`/api/links/${linkId}/sources/${sourceId}`),
  batchCreateSources: (data) => post('/api/sources/batch', data),
};

export const groupsApi = {
  list: () => get('/api/groups'),
  create: (data) => post('/api/groups', data),
  update: (id, data) => put(`/api/groups/${id}`, data),
  remove: (id) => del(`/api/groups/${id}`),
  forLink: (linkId) => get(`/api/links/${linkId}/groups`),
  addLink: (groupId, linkId) => post(`/api/groups/${groupId}/links`, { link_id: linkId }),
  removeLink: (groupId, linkId) => del(`/api/groups/${groupId}/links/${linkId}`),
};

export const analyticsApi = {
  summary: (linkId, params = {}) => {
    const query = new URLSearchParams(params);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return get(`/api/links/${linkId}/analytics${suffix}`);
  },
  clicks: (linkId, params = {}) => {
    const query = new URLSearchParams(params);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return get(`/api/links/${linkId}/analytics/clicks${suffix}`);
  },
  sources: (linkId) => get(`/api/links/${linkId}/analytics/sources`),
  referrers: (linkId) => get(`/api/links/${linkId}/analytics/referrers`),
  locations: (linkId) => get(`/api/links/${linkId}/analytics/locations`),
  browsers: (linkId) => get(`/api/links/${linkId}/analytics/browsers`),
  recent: (linkId, limit = 10) => get(`/api/links/${linkId}/analytics/recent?limit=${limit}`),
};
