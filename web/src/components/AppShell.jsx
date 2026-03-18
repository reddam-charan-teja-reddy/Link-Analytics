import { NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useTheme } from '../context/ThemeContext';

export default function AppShell({ children }) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();

  async function onLogout() {
    await logout();
    navigate('/');
  }

  return (
    <div className="app-page">
      <header className="app-header">
        <div>
          <p className="eyebrow">CAMPAIGN LINK HUB</p>
          <h1 className="brand-title">FlowLinks</h1>
        </div>
        <nav className="main-nav" aria-label="Main navigation">
          <NavLink to="/links" className={({ isActive }) => (isActive ? 'nav-item active' : 'nav-item')}>
            Home
          </NavLink>
          <NavLink to="/groups" className={({ isActive }) => (isActive ? 'nav-item active' : 'nav-item')}>
            Groups
          </NavLink>
        </nav>
        <div className="user-zone">
          <button className="btn ghost" onClick={toggleTheme}>{theme === 'dark' ? 'Light mode' : 'Dark mode'}</button>
          <span className="user-email">{user?.email}</span>
          <button className="btn ghost" onClick={onLogout}>Sign out</button>
        </div>
      </header>
      <main className="app-main">{children}</main>
    </div>
  );
}
