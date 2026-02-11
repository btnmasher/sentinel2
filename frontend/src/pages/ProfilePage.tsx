import MainLayout from "@/layouts/MainLayout";
import { ProfileRoute } from "@/features/profile";

export default function ProfilePage() {
  return (
    <MainLayout>
      <ProfileRoute />
    </MainLayout>
  );
}
