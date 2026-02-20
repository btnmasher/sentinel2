import { useCallback, useEffect, useRef } from "react";
import useModalControls from "@/app/hooks/useModalControls";
import type { ModalCloseReason, ModalConfig } from "@/app/store/uiStore";

export type ModalDefinition<K extends string = string> = {
  key: K;
  useOpen: () => boolean;
  setOpen: (key: K, open: boolean) => void;
  build: (handleClose: (reason?: ModalCloseReason) => void) => ModalConfig;
  beforeDismiss?: (
    reason: ModalCloseReason,
  ) => boolean | void | Promise<boolean | void>;
  afterDismiss?: (reason: ModalCloseReason) => void | Promise<void>;
};

type UseModalOptions<K extends string = string> = {
  open: boolean;
  onDismiss?: () => void;
  modalKey?: K;
  setOpenByKey?: (key: K, open: boolean) => void;
  beforeDismiss?: (
    reason: ModalCloseReason,
  ) => boolean | void | Promise<boolean | void>;
  afterDismiss?: (reason: ModalCloseReason) => void | Promise<void>;
  build: (handleClose: (reason?: ModalCloseReason) => void) => ModalConfig;
};

const isModalDefinition = (value: unknown): value is ModalDefinition => {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.key === "string" &&
    typeof candidate.useOpen === "function" &&
    typeof candidate.setOpen === "function" &&
    typeof candidate.build === "function"
  );
};

export default function useModal<K extends string>(
  input: UseModalOptions<K> | ModalDefinition<K>,
  definitionOptions?: Omit<
    UseModalOptions<K>,
    "open" | "build" | "modalKey" | "setOpenByKey"
  >,
) {
  const definition = isModalDefinition(input)
    ? (input as ModalDefinition<K>)
    : null;
  let normalized: UseModalOptions<K>;
  if (definition) {
    normalized = {
      open: definition.useOpen(),
      modalKey: definition.key,
      setOpenByKey: definition.setOpen,
      build: definition.build,
      beforeDismiss:
        definitionOptions?.beforeDismiss ?? definition.beforeDismiss,
      afterDismiss: definitionOptions?.afterDismiss ?? definition.afterDismiss,
      onDismiss: definitionOptions?.onDismiss,
    };
  } else {
    normalized = input as UseModalOptions<K>;
  }
  const {
    open,
    onDismiss,
    modalKey,
    setOpenByKey,
    beforeDismiss,
    afterDismiss,
    build,
  } = normalized;
  const { openModal, closeModal } = useModalControls();
  const openedRef = useRef(false);
  const onDismissRef = useRef(onDismiss);
  const modalKeyRef = useRef(modalKey);
  const setOpenByKeyRef = useRef(setOpenByKey);
  const beforeDismissRef = useRef(beforeDismiss);
  const afterDismissRef = useRef(afterDismiss);
  const buildRef = useRef(build);
  const afterOpenCloseRef = useRef<ModalConfig["onClose"]>(null);

  useEffect(() => {
    onDismissRef.current = onDismiss;
  }, [onDismiss]);

  useEffect(() => {
    modalKeyRef.current = modalKey;
    setOpenByKeyRef.current = setOpenByKey;
  }, [modalKey, setOpenByKey]);

  useEffect(() => {
    beforeDismissRef.current = beforeDismiss;
  }, [beforeDismiss]);

  useEffect(() => {
    afterDismissRef.current = afterDismiss;
  }, [afterDismiss]);

  useEffect(() => {
    buildRef.current = build;
  }, [build]);

  const dismissInternal = useCallback(async (reason: ModalCloseReason) => {
    if (beforeDismissRef.current) {
      const shouldContinue = await beforeDismissRef.current(reason);
      if (shouldContinue === false) return false;
    }

    if (onDismissRef.current) {
      onDismissRef.current();
    } else if (setOpenByKeyRef.current && modalKeyRef.current) {
      setOpenByKeyRef.current(modalKeyRef.current, false);
    } else {
      throw new Error(
        "useModal requires either onDismiss or modalKey/setOpenByKey for close state ownership",
      );
    }

    if (afterOpenCloseRef.current) {
      const shouldContinue = await afterOpenCloseRef.current(reason);
      if (shouldContinue === false) return false;
    }

    if (afterDismissRef.current) {
      await afterDismissRef.current(reason);
    }

    return true;
  }, []);

  const handleClose = useCallback(
    (reason: ModalCloseReason = "button") => {
      void (async () => {
        const shouldClose = await dismissInternal(reason);
        if (!shouldClose) return;
        closeModal();
        openedRef.current = false;
        afterOpenCloseRef.current = null;
      })();
    },
    [closeModal, dismissInternal],
  );

  const openFromKey = useCallback(() => {
    if (!setOpenByKeyRef.current || !modalKeyRef.current) return;
    setOpenByKeyRef.current(modalKeyRef.current, true);
  }, []);

  useEffect(() => {
    if (open && !openedRef.current) {
      const config = buildRef.current(handleClose);
      afterOpenCloseRef.current = config.onClose ?? null;
      openModal({
        ...config,
        onClose: async (reason = "programmatic") => {
          const shouldClose = await dismissInternal(reason);
          if (shouldClose) {
            openedRef.current = false;
            afterOpenCloseRef.current = null;
          }
          return shouldClose;
        },
      });
      openedRef.current = true;
      return;
    }
    if (!open && openedRef.current) {
      closeModal();
      openedRef.current = false;
      afterOpenCloseRef.current = null;
    }
  }, [closeModal, dismissInternal, handleClose, open, openModal]);

  useEffect(() => {
    if (!open || !openedRef.current) return;
    const config = buildRef.current(handleClose);
    afterOpenCloseRef.current = config.onClose ?? null;
    openModal({
      ...config,
      onClose: async (reason = "programmatic") => {
        const shouldClose = await dismissInternal(reason);
        if (shouldClose) {
          openedRef.current = false;
          afterOpenCloseRef.current = null;
        }
        return shouldClose;
      },
    });
  }, [build, dismissInternal, handleClose, open, openModal]);

  return { handleClose, open: openFromKey };
}
