import React, { useState } from 'react';
import { api } from '../utils/api';
import { useApi } from '../hooks/useApi';

const levelColors = {
  ERROR: '#ef4444',
  WARN: '#f59e0b',
  INFO: '#3b82f6',
  DEBUG: '#6b7280',
};

function LogsPage() {
  const [query, setQuery] = useState('');
  const [level, setLevel] = useState('');
  const [service, setService] = useState('');

  const { data, loading, error, refetch } = useApi(
    () => api.queryLogs({ q: query, level, service, limit: 100 }),
    [query, level, service]
  );

  const handleSearch = (e) => {
    e.preventDefault();
    refetch();
  };

  return (
    <div>
      <h1 style={styles.title}>Log Explorer</h1>

      <form onSubmit={handleSearch} style={styles.searchBar}>
        <input
          type="text"
          placeholder="Search logs..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          style={styles.searchInput}
        />
        <select value={level} onChange={(e) => setLevel(e.target.value)} style={styles.select}>
          <option value="">All Levels</option>
          <option value="ERROR">ERROR</option>
          <option value="WARN">WARN</option>
          <option value="INFO">INFO</option>
          <option value="DEBUG">DEBUG</option>
        </select>
        <input
          type="text"
          placeholder="Service..."
          value={service}
          onChange={(e) => setService(e.target.value)}
          style={styles.serviceInput}
        />
        <button type="submit" style={styles.searchBtn}>Search</button>
      </form>

      {loading && <div style={styles.loading}>Loading...</div>}
      {error && <div style={styles.error}>Error: {error}</div>}

      {data && (
        <div>
          <div style={styles.stats}>
            Total: {data.total || 0} logs
          </div>
          <div style={styles.logList}>
            {(data.logs || []).map((log, i) => (
              <div key={log.id || i} style={styles.logEntry}>
                <div style={styles.logHeader}>
                  <span style={{ ...styles.level, color: levelColors[log.level] || '#6b7280' }}>
                    {log.level}
                  </span>
                  <span style={styles.timestamp}>
                    {new Date(log.timestamp).toLocaleString()}
                  </span>
                  <span style={styles.service}>{log.service}</span>
                  {log.trace_id && (
                    <span style={styles.traceLink} title={log.trace_id}>
                      trace:{log.trace_id.substring(0, 8)}...
                    </span>
                  )}
                </div>
                <div style={styles.message}>{log.message}</div>
                {log.attributes && Object.keys(log.attributes).length > 0 && (
                  <div style={styles.attributes}>
                    {Object.entries(log.attributes).map(([k, v]) => (
                      <span key={k} style={styles.attr}>{k}={v}</span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

const styles = {
  title: { fontSize: 24, fontWeight: 700, marginBottom: 24, color: '#f1f5f9' },
  searchBar: { display: 'flex', gap: 8, marginBottom: 20 },
  searchInput: {
    flex: 1, padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14, outline: 'none',
  },
  select: {
    padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14,
  },
  serviceInput: {
    width: 160, padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14, outline: 'none',
  },
  searchBtn: {
    padding: '10px 20px', background: '#2563eb', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  loading: { color: '#64748b', padding: 20 },
  error: { color: '#ef4444', padding: 20 },
  stats: { color: '#64748b', marginBottom: 12, fontSize: 13 },
  logList: { display: 'flex', flexDirection: 'column', gap: 2 },
  logEntry: {
    padding: '10px 14px', background: '#1e293b', borderRadius: 4,
    borderLeft: '3px solid #334155',
  },
  logHeader: { display: 'flex', gap: 12, alignItems: 'center', marginBottom: 4 },
  level: { fontWeight: 700, fontSize: 11, textTransform: 'uppercase', minWidth: 45 },
  timestamp: { color: '#64748b', fontSize: 12, fontFamily: 'monospace' },
  service: { color: '#38bdf8', fontSize: 12, fontWeight: 600 },
  traceLink: { color: '#a78bfa', fontSize: 11, cursor: 'pointer', fontFamily: 'monospace' },
  message: { color: '#e2e8f0', fontSize: 13, fontFamily: 'monospace', lineHeight: 1.5 },
  attributes: { display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 6 },
  attr: {
    fontSize: 11, color: '#94a3b8', background: '#0f172a', padding: '2px 6px',
    borderRadius: 3, fontFamily: 'monospace',
  },
};

export default LogsPage;
