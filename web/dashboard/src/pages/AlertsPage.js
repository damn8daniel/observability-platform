import React, { useState } from 'react';
import { api } from '../utils/api';
import { useApi } from '../hooks/useApi';

function AlertsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({
    name: '',
    type: 'log',
    query: '',
    condition: 'gt',
    threshold: 100,
    duration: '5m',
    channels: ['default-webhook'],
  });

  const { data: rulesData, loading: rulesLoading, refetch: refetchRules } = useApi(
    () => api.listAlertRules(),
    []
  );
  const { data: alertsData, loading: alertsLoading, refetch: refetchAlerts } = useApi(
    () => api.listAlerts(),
    []
  );

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      await api.createAlertRule(form);
      setShowCreate(false);
      setForm({ name: '', type: 'log', query: '', condition: 'gt', threshold: 100, duration: '5m', channels: ['default-webhook'] });
      refetchRules();
    } catch (err) {
      console.error('Failed to create alert rule:', err);
    }
  };

  const handleDelete = async (id) => {
    try {
      await api.deleteAlertRule(id);
      refetchRules();
    } catch (err) {
      console.error('Failed to delete alert rule:', err);
    }
  };

  return (
    <div>
      <div style={styles.header}>
        <h1 style={styles.title}>Alerts</h1>
        <button style={styles.createBtn} onClick={() => setShowCreate(!showCreate)}>
          + New Rule
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} style={styles.form}>
          <input
            placeholder="Rule name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            style={styles.input}
          />
          <select
            value={form.type}
            onChange={(e) => setForm({ ...form, type: e.target.value })}
            style={styles.select}
          >
            <option value="log">Log</option>
            <option value="metric">Metric</option>
          </select>
          <input
            placeholder="Query (e.g., level:ERROR)"
            value={form.query}
            onChange={(e) => setForm({ ...form, query: e.target.value })}
            style={styles.input}
          />
          <select
            value={form.condition}
            onChange={(e) => setForm({ ...form, condition: e.target.value })}
            style={styles.select}
          >
            <option value="gt">Greater than</option>
            <option value="lt">Less than</option>
            <option value="eq">Equals</option>
          </select>
          <input
            type="number"
            placeholder="Threshold"
            value={form.threshold}
            onChange={(e) => setForm({ ...form, threshold: Number(e.target.value) })}
            style={{ ...styles.input, width: 100 }}
          />
          <button type="submit" style={styles.saveBtn}>Create Rule</button>
        </form>
      )}

      <h2 style={styles.sectionTitle}>Alert Rules</h2>
      {rulesLoading && <div style={styles.loading}>Loading...</div>}
      <div style={styles.ruleList}>
        {(rulesData?.rules || []).map((rule) => (
          <div key={rule.id} style={styles.ruleCard}>
            <div style={styles.ruleHeader}>
              <span style={styles.ruleName}>{rule.name}</span>
              <span style={styles.ruleType}>{rule.type}</span>
              <button style={styles.deleteBtn} onClick={() => handleDelete(rule.id)}>
                Delete
              </button>
            </div>
            <div style={styles.ruleDetails}>
              Query: <code>{rule.query}</code> | Condition: {rule.condition} {rule.threshold}
            </div>
          </div>
        ))}
        {rulesData && (!rulesData.rules || rulesData.rules.length === 0) && (
          <div style={styles.empty}>No alert rules configured.</div>
        )}
      </div>

      <h2 style={{ ...styles.sectionTitle, marginTop: 32 }}>Fired Alerts</h2>
      {alertsLoading && <div style={styles.loading}>Loading...</div>}
      <div style={styles.alertList}>
        {(alertsData?.alerts || []).map((alert) => (
          <div key={alert.id} style={styles.alertCard}>
            <div style={styles.alertStatus}>
              <span style={{
                ...styles.statusBadge,
                background: alert.status === 'firing' ? '#ef4444' : '#22c55e',
              }}>
                {alert.status}
              </span>
              <span style={styles.alertTime}>
                {new Date(alert.fired_at).toLocaleString()}
              </span>
            </div>
            <div style={styles.alertMessage}>{alert.message}</div>
          </div>
        ))}
        {alertsData && (!alertsData.alerts || alertsData.alerts.length === 0) && (
          <div style={styles.empty}>No alerts fired.</div>
        )}
      </div>
    </div>
  );
}

const styles = {
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 },
  title: { fontSize: 24, fontWeight: 700, color: '#f1f5f9' },
  sectionTitle: { fontSize: 18, fontWeight: 600, color: '#e2e8f0', marginBottom: 12 },
  createBtn: {
    padding: '10px 20px', background: '#2563eb', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  form: { display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' },
  input: {
    flex: 1, minWidth: 150, padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14, outline: 'none',
  },
  select: {
    padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14,
  },
  saveBtn: {
    padding: '10px 20px', background: '#16a34a', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  loading: { color: '#64748b', padding: 20 },
  empty: { color: '#64748b', padding: 20, textAlign: 'center' },
  ruleList: { display: 'flex', flexDirection: 'column', gap: 8 },
  ruleCard: {
    background: '#1e293b', borderRadius: 6, padding: '12px 16px',
    border: '1px solid #334155',
  },
  ruleHeader: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 6 },
  ruleName: { fontWeight: 600, color: '#f1f5f9', fontSize: 14 },
  ruleType: {
    fontSize: 10, fontWeight: 700, textTransform: 'uppercase',
    color: '#38bdf8', background: '#0f172a', padding: '2px 6px', borderRadius: 3,
  },
  deleteBtn: {
    marginLeft: 'auto', padding: '4px 10px', background: 'transparent',
    color: '#ef4444', border: '1px solid #ef4444', borderRadius: 4,
    cursor: 'pointer', fontSize: 12,
  },
  ruleDetails: { color: '#94a3b8', fontSize: 12 },
  alertList: { display: 'flex', flexDirection: 'column', gap: 8 },
  alertCard: {
    background: '#1e293b', borderRadius: 6, padding: '12px 16px',
    border: '1px solid #334155',
  },
  alertStatus: { display: 'flex', alignItems: 'center', gap: 12, marginBottom: 6 },
  statusBadge: {
    padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 700,
    color: '#fff', textTransform: 'uppercase',
  },
  alertTime: { color: '#64748b', fontSize: 12, fontFamily: 'monospace' },
  alertMessage: { color: '#e2e8f0', fontSize: 13 },
};

export default AlertsPage;
