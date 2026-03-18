import { useCallback, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { useAuth } from '../context/AuthContext';
import GoogleSignInButton from '../components/GoogleSignInButton';

export default function RegisterPage() {
  const { signInWithGoogle } = useAuth();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || '';

  const handleGoogleSignIn = useCallback(async (credential) => {
    setLoading(true);
    try {
      await signInWithGoogle(credential);
      toast.success('Account created');
      navigate('/links');
    } catch (err) {
      toast.error(err.message);
    } finally {
      setLoading(false);
    }
  }, [navigate, signInWithGoogle]);

  const handleGoogleError = useCallback((err) => {
    toast.error(err?.message || 'Google Sign-In is unavailable right now');
  }, []);

  return (
    <div className="auth-page">
      <section className="auth-panel">
        <p className="eyebrow">GET STARTED</p>
        <h1>Create your FlowLinks account</h1>
        <p className="muted">Create your account instantly with Google.</p>

        {!googleClientId ? (
          <p className="auth-error">Google Sign-In is not configured. Set VITE_GOOGLE_CLIENT_ID.</p>
        ) : (
          <>
            <GoogleSignInButton
              clientId={googleClientId}
              onCredential={handleGoogleSignIn}
              onError={handleGoogleError}
            />
            {loading ? <p className="tiny muted">Creating account...</p> : null}
          </>
        )}

        <p className="switch-copy">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </section>
    </div>
  );
}
