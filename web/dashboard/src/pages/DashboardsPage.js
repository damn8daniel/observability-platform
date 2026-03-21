import React, { useState } from 'react';
import { api } from '../utils/api';
import { useApi } from '../hooks/useApi';

function DashboardsPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');

  const { data, loading, error, refetch } = useApi(() => api.listDashboards(), []);

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newName.trim()) return;

    try {
      await api.createDashboard({
        name: newName,
        panels: [
          {
            id: 'panel-1',
            title: 'Log Volume',
            type: 'logs',
            query: '*',
            position: { x: 0, y: 0, width: 6, height: 4 },
          },
          {
            id: 'panel-2',
            title: 'Error Rate',
            type: 'metrics',
            query: 'level:ERROR',
            position: { x: 6, y: 0, width: 6, height: 4 },
          },
          {
            id: 'panel-3',
            title: 'Trace Latency',
            type: 'traces',
            query: '',
            position: { x: 0, y: 4, width: 12, height: 4 },
          },
        ],
      });
      setNewName('');
      setShowCreate(false);
      refetch();
    } catch (err) {
      console.error('Failed to create dashboard:', err);
    }
  };

  const handleDelete = async (id) => {
    try {
      await api.deleteDashboard(id);
      refetch();
    } catch (err) {
      console.error('Failed to delete dashboard:', err);
    }
  };

  return (
    <div>
      <div style={styles.header}>
        <h1 style={styles.title}>Dashboards</h1>
        <button style={styles.createBtn} onClick={() => setShowCreate(!showCreate)}>
          + New Dashboard
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} style={styles.createForm}>
          <input
            type="text"
            placeholder="Dashboard name..."
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            style={styles.input}
            autoFocus
          />
          <button type="submit" style={styles.saveBtn}>Create</button>
        </form>
      )}

      {loading && <div style={styles.loading}>Loading...</div>}
      {error && <div style={styles.error}>Error: {error}</div>}

      <div style={styles.grid}>
        {(data?.dashboards || []).map((dash) => (
          <div key={dash.id} style={styles.card}>
            <div style={styles.cardHeader}>
              <h3 style={styles.cardTitle}>{dash.name}</h3>
              <button
                style={styles.deleteBtn}
                onClick={() => handleDelete(dash.id)}
              >
                Delete
              </button>
            </div>
            <div style={styles.panelCount}>
              {(dash.panels || []).length} panels
            </div>
            <div style={styles.panelList}>
              {(dash.panels || []).map((panel) => (
                <div key={panel.id} style={styles.panelChip}>
                  <span style={styles.panelType}>{panel.type}</span>
                  {panel.title}
                </div>
              ))}
            </div>
            <div style={styles.cardFooter}>
              Created: {new Date(dash.created_at).toLocaleDateString()}
            </div>
          </div>
        ))}
      </div>

      {data && (!data.dashboards || data.dashboards.length === 0) && (
        <div style={styles.empty}>
          No dashboards yet. Create one to get started.
        </div>
      )}
    </div>
  );
}

const styles = {
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 },
  title: { fontSize: 24, fontWeight: 700, color: '#f1f5f9' },
  createBtn: {
    padding: '10px 20px', background: '#2563eb', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  createForm: { display: 'flex', gap: 8, marginBottom: 20 },
  input: {
    flex: 1, padding: '10px 14px', background: '#1e293b', border: '1px solid #334155',
    borderRadius: 6, color: '#e2e8f0', fontSize: 14, outline: 'none',
  },
  saveBtn: {
    padding: '10px 20px', background: '#16a34a', color: '#fff', border: 'none',
    borderRadius: 6, cursor: 'pointer', fontWeight: 600,
  },
  loading: { color: '#64748b', padding: 20 },
  error: { color: '#ef4444', padding: 20 },
  empty: { color: '#64748b', textAlign: 'center', padding: 40 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340, 1fr))', gap: 16 },
  card: {
    background: '#1e293b', borderRadius: 8, padding: 20,
    border: '1px solid #334155',
  },
  cardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 },
  cardTitle: { fontSize: 16, fontWeight: 600, color: '#f1f5f9' },
  deleteBtn: {
    padding: '4px 10px', background: 'transparent', color: '#ef4444', border: '1px solid #ef4444',
    borderRadius: 4, cursor: 'pointer', fontSize: 12,
  },
  panelCount: { color: '#64748b', fontSize: 13, marginBottom: 12 },
  panelList: { display: 'flex', flexDirection: 'column', gap: 4, marginBottom: 12 },
  panelChip: {
    display: 'flex', alignItems: 'center', gap: 8,
    fontSize: 12, color: '#94a3b8', padding: '4px 8px',
    background: '#0f172a', borderRadius: 4,
  },
  panelType: {
    fontSize: 10, fontWeight: 700, textTransform: 'uppercase',
    color: '#38bdf8', background: '#1e293b', padding: '2px 6px', borderRadius: 3,
  },
  cardFooter: { color: '#475569', fontSize: 11 },
};

export default DashboardsPage;
