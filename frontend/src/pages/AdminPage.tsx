import MainLayout from "@/layouts/MainLayout";
import { AdminRoute } from "@/features/admin";

export default function AdminPage() {
  return (
    <MainLayout>
      <AdminRoute />
    </MainLayout>
  );
}
