import { useEffect, useState } from "react";
import { api } from "@/config/api";
import { pb } from "@/config/pb";
import CharacterList from "@/components/CharacterList";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useUIStore } from "@/app/store/uiStore";
import { useShallow } from "zustand/shallow";

type Character = {
  record_id: string;
  character_id: number;
  name: string;
  corp_id: number;
  corp_name: string;
  alliance_id: number;
  alliance_name: string;
  is_main: boolean;
  esi_token_valid: boolean;
  esi_last_error: string;
  esi_last_refresh_at?: string;
};

export default function Profile() {
  const { standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      standaloneAuth: s.standaloneAuth,
    })),
  );
  const setToast = useUIStore((s) => s.setToast);
  const [characters, setCharacters] = useState<Character[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    api
      .get("/auth/profile")
      .then((res) => {
        if (!mounted) return;
        setCharacters(res.data.characters || []);
      })
      .catch(() => {
        if (!mounted) return;
        setToast({ text: "Failed to load profile", color: "error" });
      })
      .finally(() => {
        if (mounted) setLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [setToast]);

  const linkCharacter = async () => {
    const token = pb.authStore.token;
    if (!token) {
      setToast({ text: "Authentication required", color: "error" });
      return;
    }
    try {
      const res = await api.get("/auth/link");
      const url = res.data?.url;
      if (url) {
        window.location.href = url;
        return;
      }
      setToast({ text: "Unable to start link flow", color: "error" });
    } catch {
      setToast({ text: "Unable to start link flow", color: "error" });
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
      <section className="card bg-base-200/70 border border-slate-800">
        <div className="card-body">
          <h2 className="font-display text-2xl">Linked Characters</h2>
          <p className="text-sm text-slate-400">
            Manage the characters tied to your Sentinel account.
          </p>
          {loading ? (
            <p className="text-sm text-slate-400">Loading characters…</p>
          ) : (
            <CharacterList characters={characters} />
          )}
        </div>
      </section>

      <section className="card bg-base-200/70 border border-slate-800">
        <div className="card-body">
          <h2 className="font-display text-2xl">Add Character</h2>
          <p className="text-sm text-slate-400">
            Link another character via EVE SSO. Each character keeps its own
            scopes and tokens.
          </p>
          <button
            className="btn btn-sm btn-success btn-outline mt-3"
            onClick={linkCharacter}
            disabled={!standaloneAuth}
          >
            Link another character
          </button>
        </div>
      </section>
    </div>
  );
}
