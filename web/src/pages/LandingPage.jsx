import { useCallback, useRef, useState } from 'react';
import { Navigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { useAuth } from '../context/AuthContext';
import GoogleSignInButton from '../components/GoogleSignInButton';

export default function LandingPage() {
  const { isAuthenticated, signInWithGoogle } = useAuth();
  const [loading, setLoading] = useState(false);
  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || '';

  if (isAuthenticated) return <Navigate to="/links" replace />;

  const handleGoogleSignIn = useCallback(async (credential) => {
    setLoading(true);
    try {
      await signInWithGoogle(credential);
      toast.success('Signed in');
    } catch (err) {
      toast.error(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  }, [signInWithGoogle]);

  const handleGoogleError = useCallback((err) => {
    toast.error(err?.message || 'Google Sign-In is unavailable right now');
  }, []);

  const authCardRef = useRef(null);

  const handleScrollToAuth = () => {
    authCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <div className="landing-page">
      <section className="ad-strip ad-strip-strong" aria-label="Positioning statement">
        <p>Built for teams that need campaign-grade link tracking without analytics clutter.</p>
        <p>One short URL can be shared cleanly or with optional source tags like <code>?src=email</code>.</p>
      </section>

      <section className="landing-hero">
        <div className="landing-copy landing-copy-strong">
          <p className="eyebrow">Growth Tracking, Simplified</p>
          <h1>Short links that ship faster and report cleaner.</h1>
          <p className="muted">
            FlowLinks helps teams launch campaign links quickly, organize by group, and compare real performance
            without opening five different dashboards.
          </p>

          <div className="landing-cta-row">
            <button type="button" className="btn primary" onClick={handleScrollToAuth}>Continue with Google</button>
            <a className="btn secondary" href="/login">Open login page</a>
          </div>

          <div className="proof-grid">
            <article>
              <h3>Campaign-ready in minutes</h3>
              <p>Create and distribute a short link immediately, then monitor results from one workspace.</p>
            </article>
            <article>
              <h3>Unique hash by default</h3>
              <p>Every link works as-is. Add <code>?src=...</code> only when you want source segmentation.</p>
            </article>
            <article>
              <h3>Group workspace flow</h3>
              <p>Create and manage links inside campaign groups so your team stays organized.</p>
            </article>
          </div>

          <ul className="idea-list">
            <li>Create links from Home or directly in a group page.</li>
            <li>Track trends, recent activity, and quality signals in one dashboard.</li>
            <li>Use source tags for attribution only, not redirect correctness.</li>
            <li>Switch from launch to insights without context switching.</li>
          </ul>

          <div className="landing-stats">
            <article>
              <strong>3x faster</strong>
              <span>campaign setup and tracking</span>
            </article>
            <article>
              <strong>One unified workspace</strong>
              <span>for links, groups, sources, and analytics</span>
            </article>
            <article>
              <strong>Less guesswork</strong>
              <span>for your next campaign decision</span>
            </article>
          </div>
        </div>

        <aside className="auth-card auth-card-strong" aria-label="Authentication" ref={authCardRef}>
          <p className="eyebrow">Start free</p>
          <p className="muted small">Sign in with Google to create your account and continue.</p>

          {!googleClientId ? (
            <p className="auth-error">Google Sign-In is not configured. Set VITE_GOOGLE_CLIENT_ID.</p>
          ) : (
            <>
              <GoogleSignInButton
                clientId={googleClientId}
                onCredential={handleGoogleSignIn}
                onError={handleGoogleError}
              />
              {loading ? <p className="tiny muted">Please wait...</p> : null}
            </>
          )}

          <p className="muted tiny">By continuing, you agree to use FlowLinks for lawful traffic tracking only.</p>
        </aside>
      </section>
    </div>
  );
}
