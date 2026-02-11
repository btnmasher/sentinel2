const RESET_FLAG_KEY = "intel-map-config/reset-v1";
const STORE_KEYS = [
  "intel-map-config/settings",
  "intel-map-config/intel",
  "intel-map-config/data",
];

export function ensurePersistReset() {
  if (typeof window === "undefined") return;
  try {
    if (window.localStorage.getItem(RESET_FLAG_KEY)) return;
    STORE_KEYS.forEach((key) => window.localStorage.removeItem(key));
    window.localStorage.setItem(RESET_FLAG_KEY, "true");
  } catch {
    // Ignore storage errors (private mode, etc.)
  }
}
