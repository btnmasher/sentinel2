export const markdownToPlainText = (input: string): string =>
  input
    .replaceAll(/\[([^\]]+)\]\(([^)]+)\)/g, "$1")
    .replaceAll(/`([^`]+)`/g, "$1")
    .replaceAll(/\*\*([^*]+)\*\*/g, "$1")
    .replaceAll(/\*([^*]+)\*/g, "$1")
    .replaceAll(/^#{1,6}\s+/gm, "")
    .replaceAll(/^[-*]\s+/gm, "")
    .replaceAll(/\r\n/g, "\n")
    .replaceAll(/\n+/g, " ")
    .trim();
