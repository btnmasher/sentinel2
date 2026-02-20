import MainLayout from "@/layouts/MainLayout";
import { TimerBoard } from "@/features/timers";

export default function TimersPage() {
  return (
    <MainLayout>
      <TimerBoard />
    </MainLayout>
  );
}
