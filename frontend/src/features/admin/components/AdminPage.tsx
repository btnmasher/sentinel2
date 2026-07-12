import JobActionsSection from "./JobActionsSection";
import JobRunsSection from "./JobRunsSection";
import AdminLayout from "./AdminLayout";
import SectionErrorBoundary from "./SectionErrorBoundary";
import AdminUserPanel from "./AdminUserPanel";

export default function AdminPage() {
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
            <SectionErrorBoundary fallbackTitle="Admin Actions">
              <JobActionsSection />
            </SectionErrorBoundary>
          </div>
        </>
      }
      right={
        <div className="h-full min-h-0">
          <SectionErrorBoundary fallbackTitle="User Admin">
            <AdminUserPanel />
          </SectionErrorBoundary>
        </div>
      }
      modals={null}
    />
  );
}
