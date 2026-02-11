import { useEffect, useState } from "react";
import Modal from "./Modal";
import { useSettingsStore } from "@/app/store/settingsStore";

export default function IntroductionDialog() {
  const introduction = useSettingsStore((s) => s.settings.introduction);
  const setIntroduction = useSettingsStore((s) => s.setIntroduction);
  const [introCountdown, setIntroCountdown] = useState(15);

  useEffect(() => {
    if (!introduction) return;
    setIntroCountdown(15);
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
  }, [introduction]);

  return (
    <Modal
      open={introduction}
      title="Welcome to Sentinel"
      actions={
        <button
          className="btn btn-sm btn-error btn-outline"
          disabled={introCountdown > 0}
          onClick={() => setIntroduction(false)}
        >
          {introCountdown > 0 ? introCountdown : "close"}
        </button>
      }
    >
      <div>
        <h4 className="font-semibold">Basic controls</h4>
        <ul className="list-disc ml-4">
          <li>click and drag - move around the map</li>
          <li>mouse scroll wheel - zoom in/out (do not hold control/command)</li>
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
    </Modal>
  );
}
