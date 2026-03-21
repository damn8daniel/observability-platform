const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080';

async function request(path, options = {}) {
  const url = `${API_BASE}${path}`;
  const headers = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': 'default',
    ...options.headers,
  };

  const response = await fetch(url, { ...options, headers });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  // Logs
  queryLogs: (params) => {
    const qs = new URLSearchParams(params).toString();
    return request(`/api/v1/logs?${qs}`);
  },
  ingestLogs: (logs) => request('/api/v1/logs', { method: 'POST', body: JSON.stringify(logs) }),
  aggregateLogs: (params) => {
    const qs = new URLSearchParams(params).toString();
    return request(`/api/v1/logs/aggregate?${qs}`);
  },

  // Traces
  queryTraces: (params) => {
    const qs = new URLSearchParams(params).toString();
    return request(`/api/v1/traces?${qs}`);
  },
  getTrace: (traceId) => request(`/api/v1/traces/${traceId}`),

  // Metrics
  pushMetrics: (metrics) => request('/api/v1/metrics', { method: 'POST', body: JSON.stringify(metrics) }),

  // Correlation
  getCorrelation: (traceId) => request(`/api/v1/correlate/${traceId}`),

  // Dashboards
  listDashboards: () => request('/api/v1/dashboards'),
  createDashboard: (dashboard) => request('/api/v1/dashboards', { method: 'POST', body: JSON.stringify(dashboard) }),
  getDashboard: (id) => request(`/api/v1/dashboards/${id}`),
  updateDashboard: (id, dashboard) => request(`/api/v1/dashboards/${id}`, { method: 'PUT', body: JSON.stringify(dashboard) }),
  deleteDashboard: (id) => request(`/api/v1/dashboards/${id}`, { method: 'DELETE' }),

  // Alerts
  listAlertRules: () => request('/api/v1/alerts/rules'),
  createAlertRule: (rule) => request('/api/v1/alerts/rules', { method: 'POST', body: JSON.stringify(rule) }),
  deleteAlertRule: (id) => request(`/api/v1/alerts/rules/${id}`, { method: 'DELETE' }),
  listAlerts: () => request('/api/v1/alerts'),
};
