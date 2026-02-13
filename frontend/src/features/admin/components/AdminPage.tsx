import JobActionsSection from "./JobActionsSection";
import JobRunsSection from "./JobRunsSection";
import AdminLayout from "./AdminLayout";
import SectionErrorBoundary from "./SectionErrorBoundary";
import AdminUserPanel from "./AdminUserPanel";
import { useAppConfigStore } from "@/app/store/appConfigStore";
import { useShallow } from "zustand/shallow";

export default function AdminPage() {
  const { standaloneAuth } = useAppConfigStore(
    useShallow((s) => ({
      standaloneAuth: s.standaloneAuth,
    })),
  );

  return (
    <AdminLayout
      left={
        <>
          <div className="grid grid-rows-[minmax(0,1fr)_auto] gap-6 h-full min-h-0">
            <div className="min-h-0">
              <SectionErrorBoundary fallbackTitle="Job Runs">
                <JobRunsSection />
              </SectionErrorBoundary>
            </div>
            <SectionErrorBoundary fallbackTitle="Job Actions">
              <JobActionsSection />
            </SectionErrorBoundary>
          </div>
        </>
      }
      right={
        <div className="h-full min-h-0">
          <SectionErrorBoundary fallbackTitle="User Admin">
            {standaloneAuth ? (
              <AdminUserPanel />
            ) : (
              <div className="h-full rounded-xl border border-slate-800 bg-slate-950/40 p-6 text-sm text-slate-300">
                Account and character administration is only available in
                standalone auth mode.
              </div>
            )}
          </SectionErrorBoundary>
        </div>
      }
      modals={null}
    />
  );
}
