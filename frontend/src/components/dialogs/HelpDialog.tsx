import useModal from "@/app/hooks/useModal";
import { useModalBody } from "@/components/dialogs/ModalBodyContext";
import {
  defineUIDialogModal,
  useUIStore,
  UI_DIALOG,
} from "@/app/store/uiStore";
import { useSettingsStore } from "@/app/store/settingsStore";

function HelpDialogBody() {
  const { close } = useModalBody();
  const setIntroduction = useSettingsStore((s) => s.setIntroduction);

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
        <h4 className="font-semibold">Map Legend</h4>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line
              x1="4"
              y1="5"
              x2="28"
              y2="5"
              className="map-gate constellation"
            />
          </svg>
          Constellation gate
        </div>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line x1="4" y1="5" x2="28" y2="5" className="map-gate region" />
          </svg>
          Region gate
        </div>
        <div className="flex items-center gap-2">
          <svg width="32" height="10">
            <line
              x1="4"
              y1="5"
              x2="28"
              y2="5"
              className="map-gate jumpbridge"
            />
          </svg>
          Jumpbridge
        </div>
        <div className="mt-2 text-green-300">
          Jita - Shift click systems to filter logs.
        </div>
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
      <div>
        <h4 className="font-semibold">Saved settings</h4>
        <p>
          Map regions, intel log filters, and settings are stored in your
          browser.
        </p>
      </div>
      <div className="modal-action">
        <button
          className="btn btn-sm btn-outline"
          onClick={() => {
            setIntroduction(true);
            close();
          }}
        >
          Show intro
        </button>
        <button className="btn btn-sm btn-outline" onClick={() => close()}>
          Close
        </button>
      </div>
    </>
  );
}

export const AppModalHelp = defineUIDialogModal({
  key: UI_DIALOG.Help,
  useOpen: () => useUIStore((s) => s.dialogs[UI_DIALOG.Help]),
  build: () => ({
    title: "Help",
    body: <HelpDialogBody />,
  }),
});

export default function HelpDialog() {
  useModal(AppModalHelp);

  return null;
}
