import { useEffect, useRef } from 'react';

const GIS_SCRIPT_ID = 'google-identity-services';

function loadGoogleScript() {
  return new Promise((resolve, reject) => {
    if (window.google?.accounts?.id) {
      resolve();
      return;
    }

    const existing = document.getElementById(GIS_SCRIPT_ID);
    if (existing) {
      existing.addEventListener('load', () => resolve());
      existing.addEventListener('error', () => reject(new Error('Failed to load Google Sign-In script')));
      return;
    }

    const script = document.createElement('script');
    script.id = GIS_SCRIPT_ID;
    script.src = 'https://accounts.google.com/gsi/client';
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load Google Sign-In script'));
    document.head.appendChild(script);
  });
}

export default function GoogleSignInButton({ clientId, onCredential, onError }) {
  const buttonRef = useRef(null);

  useEffect(() => {
    let isMounted = true;

    async function initialize() {
      try {
        await loadGoogleScript();
        if (!isMounted || !buttonRef.current || !window.google?.accounts?.id) {
          return;
        }

        window.google.accounts.id.initialize({
          client_id: clientId,
          callback: (response) => {
            if (!response?.credential) {
              onError?.(new Error('No Google credential returned'));
              return;
            }
            onCredential(response.credential);
          },
          ux_mode: 'popup',
          auto_select: false,
        });

        buttonRef.current.innerHTML = '';
        window.google.accounts.id.renderButton(buttonRef.current, {
          type: 'standard',
          shape: 'pill',
          theme: 'outline',
          text: 'continue_with',
          size: 'large',
          logo_alignment: 'left',
          width: 320,
        });
      } catch (err) {
        onError?.(err);
      }
    }

    initialize();

    return () => {
      isMounted = false;
    };
  }, [clientId, onCredential, onError]);

  return <div className="google-signin" ref={buttonRef} />;
}
