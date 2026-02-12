import { useEffect, useState } from "react";
import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import { useSettingsStore } from "@/app/store/settingsStore";

const INTRO_MODAL = {
  Welcome: "welcome",
} as const;
type IntroModalKey = (typeof INTRO_MODAL)[keyof typeof INTRO_MODAL];

function IntroductionDialogBody() {
  const { close } = useModalBody();
  const setIntroduction = useSettingsStore((s) => s.setIntroduction);
  const [introCountdown, setIntroCountdown] = useState(3);

  useEffect(() => {
    setIntroCountdown(3);
    const timer = setInterval(() => {
      setIntroCountdown((value) => {
        if (value <= 1) {
          clearInterval(timer);
          return 0;
        }
        return value - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const handleClose = () => {
    setIntroduction(false);
    close();
  };

  return (
    <>
      <div>
        <h4 className="font-semibold">Basic controls</h4>
        <ul className="list-disc ml-4">
          <li>click and drag - move around the map</li>
          <li>
            mouse scroll wheel - zoom in/out (do not hold control/command)
          </li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">Intel log filters</h4>
        <ul className="list-disc ml-4">
          <li>shift-click a system - filter logs by system</li>
          <li>Additional settings located above the intel panel</li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">Navigation bar</h4>
        <ul className="list-disc ml-4">
          <li>Search for systems and load new regions</li>
          <li>Use the toolbar to hide jumpbridges or open help</li>
        </ul>
      </div>
      <div>
        <h4 className="font-semibold">Context menus</h4>
        <ul className="list-disc ml-4">
          <li>Right click on the map - set jumprange settings, clear route</li>
          <li>
            Right click on a system - show jumpranges, set route, show
            zkill/dotlan
          </li>
          <li>
            Right click on selected text in intel logs - search player names
          </li>
        </ul>
      </div>
      <p className="text-sm">
        This map can get laggy at times, so it is not guaranteed to run on old
        computers. If you have issues, ask in it-office on TEST discord.
      </p>
      <div className="modal-action">
        <button
          className="btn btn-sm btn-error btn-outline"
          disabled={introCountdown > 0}
          onClick={() => handleClose()}
        >
          {introCountdown > 0 ? introCountdown : "close"}
        </button>
      </div>
    </>
  );
}

export default function IntroductionDialog() {
  const introduction = useSettingsStore((s) => s.settings.introduction);
  const setIntroductionModal = (_modal: IntroModalKey, open: boolean) => {
    useSettingsStore.getState().setIntroduction(open);
  };

  useModal({
    open: introduction,
    modalKey: INTRO_MODAL.Welcome,
    setOpenByKey: setIntroductionModal,
    build: () => ({
      title: "Welcome to Sentinel",
      dismissible: false,
      body: <IntroductionDialogBody />,
    }),
  });

  return null;
}
