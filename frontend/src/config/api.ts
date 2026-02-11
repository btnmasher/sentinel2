import axios from "axios";
import { useUIStore } from "@/app/store/uiStore";
import { useAuthStore } from "@/app/store/authStore";
import { pb } from "@/config/pb";

export const api = axios.create({
  baseURL: "/api",
});

api.interceptors.request.use((config) => {
  const token = pb.authStore.token;
  if (token) {
    config.headers = config.headers ?? {};
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const response = error?.response;
    const authHeader = error?.config?.headers?.["X-Auth-Check"];
    const authHeaderLower = error?.config?.headers?.["x-auth-check"];
    const skipAuthRedirect = Boolean(authHeader || authHeaderLower);
    if (response?.status === 401) {
      if (skipAuthRedirect) {
        useAuthStore.getState().invalidate();
      } else {
        useAuthStore
          .getState()
          .forceLogout("Authentication expired, returning to home.");
      }
      return Promise.reject(error);
    }
    const method = error?.config?.method
      ? String(error.config.method).toUpperCase()
      : undefined;
    const url = error?.config?.url ? String(error.config.url) : undefined;
    const requestMeta = {
      type: "request",
      method,
      url,
      status: response?.status,
      data: response?.data,
    };

    if (response?.status === 403 && response.headers?.refresh_url) {
      useUIStore.getState().setToast({
        timeout: 5000,
        color: "error",
        text: "Authentication expired, redirecting to auth.",
        meta: requestMeta,
      });
      window.location.reload();
    } else if (response?.status === 429) {
      const message =
        typeof response.data === "string"
          ? response.data
          : response?.data?.message || "Rate limited";
      useUIStore.getState().setToast({
        timeout: 5000,
        color: "error",
        text: message,
        meta: requestMeta,
      });
    }
    return Promise.reject(error);
  },
);
