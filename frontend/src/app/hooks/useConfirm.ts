import { useUIStore } from "@/app/store/uiStore";

export default function useConfirm() {
  return useUIStore((s) => s.requestConfirm);
}
