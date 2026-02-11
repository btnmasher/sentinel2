import { useEffect } from "react";
import { Route, Routes, Navigate } from "react-router-dom";
import { api } from "@/config/api";
import LoadingCard from "@/components/LoadingCard";
import ErrorBoundary from "@/components/ErrorBoundary";
import IntelAlarm from "@/components/IntelAlarm";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import { useIntelStore, useIntelRealtime } from "@/features/intel";
import { resolveRegionTokens, useMapStore } from "@/features/map";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useShallow } from "zustand/shallow";
import AdminPage from "@/pages/AdminPage";
import AuthCompletePage from "@/pages/AuthCompletePage";
import IntelPage from "@/pages/IntelPage";
import LandingPage from "@/pages/LandingPage";
import NavigationPage from "@/pages/NavigationPage";
import ProfilePage from "@/pages/ProfilePage";
import SettingsPage from "@/pages/SettingsPage";
import StaffPage from "@/pages/StaffPage";
import UploaderPage from "@/pages/UploaderPage";

export default function App() {
  const updateMapConfig = useMapStore((s) => s.updateMapConfig);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const setLogFilters = useIntelStore((s) => s.setLogFilters);
  const loadAppConfig = useAppConfigStore((s) => s.load);
  const { loadAuth, authLoaded, isAuthenticated } = useAuthStore(
    useShallow((s) => ({
      loadAuth: s.load,
      authLoaded: s.loaded,
      isAuthenticated: s.isAuthenticated,
    })),
  );
  const {
    loaded: configLoaded,
    standaloneAuth,
    defaultRegions,
  } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
      defaultRegions: s.defaultRegions,
    })),
  );
  const theme = useSettingsStore((s) => s.settings.theme);
  useIntelRealtime();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const regions = params.get("regions");
    const layout = params.get("layout");
    const filters = params.get("filters");

    if (regions) {
      const parsed = resolveRegionTokens(regions);
      if (parsed.length) {
        updateMapConfig({ mapRegions: parsed });
      }
    }
    if (
      layout === "dotlan" ||
      layout === "metro" ||
      layout === "real" ||
      layout === "eve2d"
    ) {
      updateMapConfig({ mapLayout: layout });
    }
    if (filters) {
      const tokens = filters
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean);
      const numericIds = tokens
        .map((v) => parseInt(v, 10))
        .filter((v) => !Number.isNaN(v));
      const hasNames = tokens.length > numericIds.length;
      if (!hasNames) {
        setLogFilters({ system: numericIds });
        return;
      }
      if (!isAuthenticated) {
        setLogFilters({ system: numericIds });
        return;
      }
      api
        .get(`/map/search?systems=${encodeURIComponent(tokens.join(","))}`)
        .then((res) => {
          const ids = (res.data.systems || []).map(
            (item: { id: number }) => item.id,
          );
          setLogFilters({ system: ids });
        })
        .catch(() => setLogFilters({ system: numericIds }));
    }
  }, [isAuthenticated, setLogFilters, updateMapConfig]);

  useEffect(() => {
    loadAppConfig();
    loadAuth();
  }, [loadAppConfig, loadAuth]);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  useEffect(() => {
    if (!configLoaded) {
      return;
    }
    if (mapRegions.length > 0) {
      return;
    }
    const parsed = resolveRegionTokens(defaultRegions.join(","));
    if (parsed.length > 0) {
      updateMapConfig({ mapRegions: parsed });
    }
  }, [configLoaded, defaultRegions, mapRegions.length, updateMapConfig]);

  if (!authLoaded || !configLoaded) {
    return <LoadingCard full subtitle="Syncing intel stack…" />;
  }

  if (!isAuthenticated) {
    if (window.location.pathname === "/auth/complete") {
      return <AuthCompletePage />;
    }
    return <LandingPage />;
  }

  return (
    <>
      <IntelAlarm />
      <Routes>
        <Route
          path="/"
          element={
            <ErrorBoundary name="Intel">
              <IntelPage />
            </ErrorBoundary>
          }
        />
        <Route
          path="/nav"
          element={
            <ErrorBoundary name="Navigation">
              <NavigationPage />
            </ErrorBoundary>
          }
        />
        <Route
          path="/settings"
          element={
            <ErrorBoundary name="Settings">
              <SettingsPage />
            </ErrorBoundary>
          }
        />
        <Route
          path="/uploader"
          element={
            <ErrorBoundary name="Uploader">
              <UploaderPage />
            </ErrorBoundary>
          }
        />
        {standaloneAuth ? (
          <Route
            path="/profile"
            element={
              <ErrorBoundary name="Profile">
                <ProfilePage />
              </ErrorBoundary>
            }
          />
        ) : (
          <Route path="/profile" element={<Navigate to="/" />} />
        )}
        <Route
          path="/staff"
          element={
            <ErrorBoundary name="Staff">
              <StaffPage />
            </ErrorBoundary>
          }
        />
        {standaloneAuth ? (
          <Route
            path="/admin"
            element={
              <ErrorBoundary name="Admin">
                <AdminPage />
              </ErrorBoundary>
            }
          />
        ) : (
          <Route path="/admin" element={<Navigate to="/" />} />
        )}
        <Route path="*" element={<Navigate to="/" />} />
      </Routes>
    </>
  );
}
