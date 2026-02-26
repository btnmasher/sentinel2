export function organizationBadgeClass(ownerType: string): string {
  return ownerType === "corporation"
    ? "border-sky-400/50 bg-sky-500/20 text-sky-200"
    : "border-violet-400/50 bg-violet-500/20 text-violet-200";
}
