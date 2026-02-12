export type ModalMap<K extends string> = Record<K, boolean>;

type SetState<S> = (partial: Partial<S> | ((state: S) => Partial<S>)) => void;

export const createModalMap = <K extends string>(
  keys: readonly K[],
): ModalMap<K> => {
  const next = {} as ModalMap<K>;
  keys.forEach((key) => {
    next[key] = false;
  });
  return next;
};

export type ModalRegistryActions<K extends string> = {
  setModal: (modal: K, open: boolean) => void;
  openModalKey: (modal: K) => void;
  closeModalKey: (modal: K) => void;
  resetModals: () => void;
};

export const createModalRegistryActions = <
  S extends { modals: ModalMap<K> },
  K extends string,
>(
  set: SetState<S>,
  keys: readonly K[],
): ModalRegistryActions<K> => ({
  setModal: (modal, open) =>
    set(
      (state) =>
        ({
          modals: {
            ...state.modals,
            [modal]: open,
          },
        }) as Partial<S>,
    ),
  openModalKey: (modal) =>
    set(
      (state) =>
        ({
          modals: {
            ...state.modals,
            [modal]: true,
          },
        }) as Partial<S>,
    ),
  closeModalKey: (modal) =>
    set(
      (state) =>
        ({
          modals: {
            ...state.modals,
            [modal]: false,
          },
        }) as Partial<S>,
    ),
  resetModals: () =>
    set({
      modals: createModalMap(keys),
    } as Partial<S>),
});

export const bindStoreModalSetter = <
  K extends string,
  S extends { setModal: (modal: K, open: boolean) => void },
>(store: {
  getState: () => S;
}) => {
  return (modal: K, open: boolean) => {
    store.getState().setModal(modal, open);
  };
};

type ModalDefinitionInput<K extends string> = {
  key: K;
  useOpen: () => boolean;
  build: (
    handleClose: (
      reason?: import("@/app/store/uiStore").ModalCloseReason,
    ) => void,
  ) => import("@/app/store/uiStore").ModalConfig;
  beforeDismiss?: (
    reason: import("@/app/store/uiStore").ModalCloseReason,
  ) => boolean | void | Promise<boolean | void>;
  afterDismiss?: (
    reason: import("@/app/store/uiStore").ModalCloseReason,
  ) => void | Promise<void>;
};

export const createModalRegistry = <K extends string>(keys: readonly K[]) => {
  return {
    initial: () => createModalMap(keys),
    actions: <S extends { modals: ModalMap<K> }>(set: SetState<S>) =>
      createModalRegistryActions<S, K>(set, keys),
    bind: <S extends { setModal: (modal: K, open: boolean) => void }>(store: {
      getState: () => S;
    }) => bindStoreModalSetter<K, S>(store),
    defineForStore:
      <S extends { setModal: (modal: K, open: boolean) => void }>(store: {
        getState: () => S;
      }) =>
      <KK extends K>(definition: ModalDefinitionInput<KK>) => {
        const setOpen = bindStoreModalSetter<K, S>(store);
        return {
          ...definition,
          setOpen: (key: KK, open: boolean) => setOpen(key, open),
        };
      },
  };
};
