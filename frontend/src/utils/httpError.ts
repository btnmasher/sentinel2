type HttpErrorResponse = {
  status?: number;
  data?: unknown;
};

type HttpErrorLike = {
  status?: number;
  message?: string;
  response?: HttpErrorResponse;
};

const asHttpError = (error: unknown): HttpErrorLike | null => {
  if (!error || typeof error !== "object") return null;
  return error as HttpErrorLike;
};

export const getHttpStatus = (error: unknown): number | undefined => {
  const parsed = asHttpError(error);
  return parsed?.response?.status ?? parsed?.status;
};

export const getHttpData = (error: unknown): unknown => {
  const parsed = asHttpError(error);
  return parsed?.response?.data;
};

export const getErrorMessage = (error: unknown, fallback: string): string => {
  const parsed = asHttpError(error);
  const data = parsed?.response?.data;
  if (typeof data === "string" && data.trim() !== "") return data;
  if (data && typeof data === "object") {
    const message = (data as { message?: unknown }).message;
    if (typeof message === "string" && message.trim() !== "") return message;
  }
  if (typeof parsed?.message === "string" && parsed.message.trim() !== "") {
    return parsed.message;
  }
  return fallback;
};

export const toErrorMeta = (
  error: unknown,
): { name: string; message: string; stack?: string } | unknown => {
  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
    };
  }
  return error;
};
