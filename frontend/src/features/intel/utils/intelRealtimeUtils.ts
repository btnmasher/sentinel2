export const INTEL_UPLOADER_COUNT_TOPIC = "intel.uploaders_count";

export type IntelUploaderCountMessage = {
  uploaders: number;
};

export const normalizeUploaderCountMessage = (
  input: unknown,
): IntelUploaderCountMessage | null => {
  if (!input || typeof input !== "object") {
    return null;
  }
  const source = input as Record<string, unknown>;
  const raw = source.uploaders;
  const parsed =
    typeof raw === "number" ? raw : typeof raw === "string" ? Number(raw) : NaN;
  if (!Number.isFinite(parsed) || parsed < 0) {
    return null;
  }
  return { uploaders: parsed };
};
