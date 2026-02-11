import AdminPage from "./AdminPage";
import NotAuthorized from "@/components/NotAuthorized";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useAuthStore } from "@/app/store/authStore";
import LoadingCard from "@/components/LoadingCard";
import { useShallow } from "zustand/shallow";

export default function AdminRoute() {
  const { loaded, standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      loaded: s.loaded,
      standaloneAuth: s.standaloneAuth,
    })),
  );
  const {
    loaded: authLoaded,
    isAuthenticated,
    isAdmin,
  } = useAuthStore(
    useShallow((s) => ({
      loaded: s.loaded,
      isAuthenticated: s.isAuthenticated,
      isAdmin: s.isAdmin,
    })),
  );

  if (!loaded || !authLoaded) {
    return <LoadingCard subtitle="Preparing admin tools…" />;
  }

  if (!standaloneAuth) {
    return (
      <NotAuthorized message="This page isn’t available with the current login method." />
    );
  }

  if (!isAuthenticated || !isAdmin) {
    return <NotAuthorized message="You don’t have access to this page." />;
  }

  return <AdminPage />;
}
