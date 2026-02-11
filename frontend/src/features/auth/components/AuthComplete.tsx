import { useEffect, useState } from "react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useShallow } from "zustand/shallow";

export default function AuthComplete() {
  const [error, setError] = useState("");
  const { loaded, authBackend } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      authBackend: s.authBackend,
    })),
  );

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("code");
    if (!code) {
      setError("Missing auth exchange code.");
      return;
    }

    api
      .get("/auth/exchange", {
        params: { code },
        headers: { "X-Auth-Check": "1" },
      })
      .then((res) => {
        if (res?.data?.token) {
          pb.authStore.save(res.data.token, res.data.record);
          useAuthStore.getState().syncFromPB();
        }
        window.location.href = "/";
      })
      .catch(() => {
        setError("Authentication failed. Please try logging in again.");
      });
  }, []);

  const loginLabel =
    loaded && authBackend === "eve" ? "Log in with EVE Online" : "Log in";

  return (
    <div className="min-h-screen flex items-center justify-center text-slate-300">
      <div className="card bg-base-200/70 border border-slate-800">
        <div className="card-body">
          <h2 className="font-display text-2xl">Completing login</h2>
          <p className="text-sm text-slate-400">
            {error || "Finalizing your session…"}
          </p>
          {error && (
            <div className="mt-4 flex flex-wrap gap-2">
              <a className="btn btn-primary btn-outline" href="/api/auth/login">
                {loginLabel}
              </a>
              <a className="btn btn-outline" href="/">
                Back to landing
              </a>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
