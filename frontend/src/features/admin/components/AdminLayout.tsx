import type { ReactNode } from "react";

type AdminLayoutProps = {
  left: ReactNode;
  right: ReactNode;
  modals?: ReactNode;
};

export default function AdminLayout({ left, right, modals }: AdminLayoutProps) {
  return (
    <div className="grid lg:grid-cols-2 gap-6 h-[calc(100vh-5rem-3rem)] items-stretch">
      <div className="space-y-6 h-full min-h-0 overflow-hidden">{left}</div>
      <div className="space-y-6 h-full min-h-0 overflow-hidden">{right}</div>
      {modals}
    </div>
  );
}
