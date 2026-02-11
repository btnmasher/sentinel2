import type { SyntheticEvent } from "react";
import ThemeToggle from "@/components/ThemeToggle";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useShallow } from "zustand/shallow";

export default function Landing() {
  const { loaded, authBackend } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      authBackend: s.authBackend,
    })),
  );

  const isEveAuth = authBackend === "eve";
  const eveLoginImage =
    "https://web.ccpgamescdn.com/eveonlineassets/developers/eve-sso-login-white-large.png";
  const handleEveImageError = (event: SyntheticEvent<HTMLImageElement>) => {
    const target = event.currentTarget;
    if (target.dataset.fallback === "true") {
      return;
    }
    target.dataset.fallback = "true";
    target.src = "/eve-login-button-large-white.svg";
  };

  return (
    <div className="min-h-screen landing-shell">
      <div className="landing-theme-toggle">
        <ThemeToggle className="btn btn-sm btn-ghost btn-square" />
      </div>
      <div className="landing-backdrop" />
      <div className="landing-frame">
        <div className="landing-card">
          <div className="landing-eyebrow">Sentinel 2</div>
          <h1 className="landing-title">Intel. Navigation. Control.</h1>
          <p className="landing-subtitle">
            Authenticate to access alliance intel streams, route planning, and
            character tools tailored for fast-moving ops.
          </p>
          {!loaded && (
            <div className="landing-skeleton">
              <div className="skeleton-line skeleton-line-lg" />
              <div className="skeleton-line" />
              <div className="skeleton-line" />
              <div className="skeleton-button" />
              <div className="skeleton-line skeleton-line-sm" />
            </div>
          )}
          {loaded && (
            <div className="landing-actions">
              {isEveAuth ? (
                <a className="eve-login-image" href="/api/auth/login">
                  <img
                    src={eveLoginImage}
                    alt="Log in with EVE Online"
                    loading="lazy"
                    onError={handleEveImageError}
                  />
                </a>
              ) : (
                <a
                  className="btn btn-primary btn-outline landing-auth-button"
                  href="/api/auth/login"
                >
                  Log in with TEST Auth
                </a>
              )}
            </div>
          )}
          {loaded && (
            <div className="landing-hint">
              {isEveAuth
                ? "Use your EVE account to continue."
                : "Use your TEST Auth account to continue."}
            </div>
          )}
        </div>
        <div className="landing-panel">
          <div className="landing-panel-header">Operational Summary</div>
          <ul className="landing-panel-list">
            <li>Live intel feed with filtering and alerting.</li>
            <li>Route planning per character or fleet movement.</li>
            <li>Map overlays for bridges, gate statuses, and positions.</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
