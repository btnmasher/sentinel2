import { useAdminStore } from "../store/adminStore";
import AccountActionsSection from "./AccountActionsSection";
import UserSearchSection from "./UserSearchSection";

export default function AdminUserPanel() {
  const user = useAdminStore((s) => s.selectedUser);
  const clearUser = useAdminStore((s) => s.clearUser);

  if (!user) {
    return (
      <div className="h-full min-h-0">
        <UserSearchSection />
      </div>
    );
  }

  return <AccountActionsSection onBack={() => clearUser()} />;
}
