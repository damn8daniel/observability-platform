import React from 'react';
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom';
import LogsPage from './pages/LogsPage';
import TracesPage from './pages/TracesPage';
import DashboardsPage from './pages/DashboardsPage';
import AlertsPage from './pages/AlertsPage';

const navItems = [
  { path: '/', label: 'Logs', icon: '\u2630' },
  { path: '/traces', label: 'Traces', icon: '\u21C4' },
  { path: '/dashboards', label: 'Dashboards', icon: '\u25A6' },
  { path: '/alerts', label: 'Alerts', icon: '\u26A0' },
];

function Sidebar() {
  const location = useLocation();

  return (
    <nav style={styles.sidebar}>
      <div style={styles.logo}>
        <span style={styles.logoIcon}>{'\u25C9'}</span>
        <span style={styles.logoText}>Observability</span>
      </div>
      {navItems.map((item) => (
        <Link
          key={item.path}
          to={item.path}
          style={{
            ...styles.navLink,
            ...(location.pathname === item.path ? styles.navLinkActive : {}),
          }}
        >
          <span style={styles.navIcon}>{item.icon}</span>
          {item.label}
        </Link>
      ))}
    </nav>
  );
}

function App() {
  return (
    <BrowserRouter>
      <div style={styles.layout}>
        <Sidebar />
        <main style={styles.main}>
          <Routes>
            <Route path="/" element={<LogsPage />} />
            <Route path="/traces" element={<TracesPage />} />
            <Route path="/dashboards" element={<DashboardsPage />} />
            <Route path="/alerts" element={<AlertsPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

const styles = {
  layout: {
    display: 'flex',
    minHeight: '100vh',
  },
  sidebar: {
    width: 220,
    background: '#1e293b',
    padding: '20px 0',
    display: 'flex',
    flexDirection: 'column',
    borderRight: '1px solid #334155',
  },
  logo: {
    display: 'flex',
    alignItems: 'center',
    padding: '0 20px 24px',
    borderBottom: '1px solid #334155',
    marginBottom: 16,
  },
  logoIcon: {
    fontSize: 24,
    marginRight: 10,
    color: '#38bdf8',
  },
  logoText: {
    fontSize: 16,
    fontWeight: 700,
    color: '#f1f5f9',
  },
  navLink: {
    display: 'flex',
    alignItems: 'center',
    padding: '10px 20px',
    color: '#94a3b8',
    textDecoration: 'none',
    fontSize: 14,
    transition: 'all 0.15s',
  },
  navLinkActive: {
    color: '#38bdf8',
    background: '#0f172a',
    borderLeft: '3px solid #38bdf8',
  },
  navIcon: {
    marginRight: 10,
    fontSize: 16,
  },
  main: {
    flex: 1,
    padding: 32,
    overflow: 'auto',
  },
};

export default App;
