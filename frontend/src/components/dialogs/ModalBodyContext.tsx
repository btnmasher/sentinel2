import { createContext, useContext } from "react";
import type { ReactNode } from "react";
import type { ModalCloseReason } from "@/app/store/uiStore";

export type ModalBodyHelpers = {
  close: (reason?: ModalCloseReason) => void;
};

const ModalBodyContext = createContext<ModalBodyHelpers | null>(null);

export function ModalBodyProvider({
  close,
  children,
}: {
  close: (reason?: ModalCloseReason) => void;
  children: ReactNode;
}) {
  return (
    <ModalBodyContext.Provider value={{ close }}>
      {children}
    </ModalBodyContext.Provider>
  );
}

export function useModalBody() {
  const context = useContext(ModalBodyContext);
  if (!context) {
    throw new Error("useModalBody must be used within ModalBodyProvider");
  }
  return context;
}
