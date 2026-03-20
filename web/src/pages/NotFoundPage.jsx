import { Link } from 'react-router-dom';

export default function NotFoundPage() {
  return (
    <div className="center-screen">
      <section className="panel" style={{ maxWidth: 560, width: '100%' }}>
        <p className="eyebrow">404</p>
        <h1>Page not found</h1>
        <p className="muted">The page you requested does not exist or may have moved.</p>
        <div className="row-actions" style={{ marginTop: 16 }}>
          <Link to="/" className="btn secondary">Go to landing</Link>
          <Link to="/links" className="btn primary">Go to dashboard</Link>
        </div>
      </section>
    </div>
  );
}
