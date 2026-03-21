import React, { useState } from 'react';
import { api } from '../utils/api';
import { useApi } from '../hooks/useApi';

function WaterfallView({ spans }) {
  if (!spans || spans.length === 0) return null;

  const minStart = Math.min(...spans.map((s) => new Date(s.start_time).getTime()));
  const maxEnd = Math.max(...spans.map((s) => new Date(s.end_time).getTime()));
  const totalDuration = maxEnd - minStart || 1;

  return (
    <div style={styles.waterfall}>
      <div style={styles.waterfallHeader}>
        <span style={styles.waterfallLabel}>Waterfall View</span>
        <span style={styles.waterfallDuration}>
          Total: {(totalDuration / 1000).toFixed(2)}ms
        </span>
      </div>
      {spans.map((span, i) => {
        const start = new Date(span.start_time).getTime();
        const end = new Date(span.end_time).getTime();
        const left = ((start - minStart) / totalDuration) * 100;
        const width = Math.max(((end - start) / totalDuration) * 100, 0.5);

        const statusColor = span.status === 2 ? '#ef4444' : span.status === 1 ? '#22c55e' : '#3b82f6';

        return (
          <div key={span.span_id || i} style={styles.spanRow}>
            <div style={styles.spanInfo}>
              <span style={styles.spanService}>{span.service}</span>
              <span style={styles.spanOp}>{span.operation}</span>
            </div>
            <div style={styles.spanBar}>
              <div
                style={{
                  position: 'absolute',
                  left: `${left}%`,
                  width: `${width}%`,
                  height: 20,
                  background: statusColor,
                  borderRadius: 3,
                  minWidth: 2,
                }}
                title={`${span.operation} (${((end - start) / 1000).toFixed(2)}ms)`}
              />
            </div>
            <span style={styles.spanDuration}>
              {((end - start) / 1000).toFixed(2)}ms
            </span>
          </div>
        );
      })}
    </div>
  );
}

function TracesPage() {
  const [traceId, setTraceId] = useState('');
  const [service, setService] = useState('');
  const [selectedTrace, setSelectedTrace] = useState(null);

  const { data, loading, error, refetch } = useApi(
    () => api.queryTraces({ service, limit: 50 }),
    [service]
  );

  const loadTrace = async (id) => {
    try {
      const result = await api.getTrace(id);
      setSelectedTrace(result);
    } catch (err) {
      console.error('Failed to load trace:', err);
    }
  };

  const handleSearch = (e) => {
    e.preventDefault();
    if (traceId) {
      loadTrace(traceId);
    } else {
      refetch();
    }
  };

  return (
    <div>
      <h1 style={styles.title}>Distributed Traces</h1>

      <form onSubmit={handleSearch} style={styles.searchBar}>
        <input
          type="text"
          placeholder="Trace ID..."
          value={traceId}
          onChange={(e) => setTraceId(e.target.value)}
          style={styles.input}
        />
        <input
          type="text"
          placeholder="Service..."
          value={service}
          onChange={(e) => setService(e.target.value)}
          style={styles.input}
        />
        <button type="submit" style={styles.btn}>Search</button>
      </form>

      {selectedTrace && (
        <div style={{ marginBottom: 24 }}>
          <h2 style={styles.subtitle}>Trace: {selectedTrace.trace_id}</h2>
          <WaterfallView spans={selectedTrace.spans} />
        </div>
      )}

      {loading && <div style={styles.loading}>Loading...</div>}
      {error && <div style={styles.error}>Error: {error}</div>}

      {data && !selectedTrace && (
        <div style={styles.traceList}>
          {(data.spans || []).map((span, i) => (
            <div
              key={span.span_id || i}
              style={styles.traceRow}
              onClick={() => loadTrace(span.trace_id)}
            >
              <div style={styles.traceInfo}>
                <span style={styles.traceId}>{span.trace_id.substring(0, 16)}...</span>
                <span style={styles.traceService}>{span.service}</span>
                <span style={styles.traceOp}>{span.operation}</span>
              </div>
              <span style={styles.traceDuration}>
                {(span.duration / 1000000).toFixed(2)}ms
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const styles = {
  title: { fontSize: 24, fontWeight: 700, marginBottom: 24, color: '#f1f5f9' },
  subtitle: { fontSize: 16, fontWeight: 600, marginBottom: 12, color: '#e2e8f0' },
  searchBar: { display: 'flex', gap: 8, marginBottom: 20 },
  input: {
    flex: 1, padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14, outline: 'none',
  },
  btn: {
    padding: '10px 20px', background: '#2563eb', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  loading: { color: '#64748b', padding: 20 },
  error: { color: '#ef4444', padding: 20 },
  waterfall: { background: '#1e293b', borderRadius: 8, padding: 16 },
  waterfallHeader: { display: 'flex', justifyContent: 'space-between', marginBottom: 12 },
  waterfallLabel: { color: '#94a3b8', fontSize: 13, fontWeight: 600 },
  waterfallDuration: { color: '#64748b', fontSize: 12, fontFamily: 'monospace' },
  spanRow: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 4, height: 28 },
  spanInfo: { width: 200, display: 'flex', flexDirection: 'column' },
  spanService: { fontSize: 11, color: '#38bdf8', fontWeight: 600 },
  spanOp: { fontSize: 11, color: '#94a3b8' },
  spanBar: { flex: 1, height: 20, background: '#0f172a', borderRadius: 3, position: 'relative' },
  spanDuration: { fontSize: 11, color: '#64748b', fontFamily: 'monospace', minWidth: 60, textAlign: 'right' },
  traceList: { display: 'flex', flexDirection: 'column', gap: 4 },
  traceRow: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '10px 14px', background: '#1e293b', borderRadius: 4, cursor: 'pointer',
  },
  traceInfo: { display: 'flex', gap: 12, alignItems: 'center' },
  traceId: { color: '#a78bfa', fontSize: 12, fontFamily: 'monospace' },
  traceService: { color: '#38bdf8', fontSize: 12, fontWeight: 600 },
  traceOp: { color: '#94a3b8', fontSize: 12 },
  traceDuration: { color: '#f59e0b', fontSize: 12, fontFamily: 'monospace' },
};

export default TracesPage;
