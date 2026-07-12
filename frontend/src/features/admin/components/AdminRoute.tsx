import AdminPage from "./AdminPage";
import NotAuthorized from "@/components/NotAuthorized";
import { useAuthStore } from "@/app/store/authStore";
import LoadingCard from "@/components/LoadingCard";
import { useShallow } from "zustand/shallow";

export default function AdminRoute() {
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

  if (!authLoaded) {
    return <LoadingCard subtitle="Preparing admin tools…" />;
  }

  if (!isAuthenticated || !isAdmin) {
    return <NotAuthorized message="You don’t have access to this page." />;
  }

  return <AdminPage />;
}
