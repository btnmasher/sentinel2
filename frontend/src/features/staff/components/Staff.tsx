import IntelChannelsCard from "./IntelChannelsCard";
import JumpbridgeListCard from "./JumpbridgeListCard";
import SovereigntyCampaignWatchlistCard from "./SovereigntyCampaignWatchlistCard";
import { useAppConfigStore } from "@/app/store/appConfigStore";

export default function Staff() {
  const timersEnabled = useAppConfigStore((s) => s.timersEnabled);

  return (
    <div className="grid lg:grid-cols-[1fr_1fr] gap-6 items-start">
      <div className="space-y-6">
        <IntelChannelsCard />
        {timersEnabled && <SovereigntyCampaignWatchlistCard />}
      </div>
      <div>
        <JumpbridgeListCard />
      </div>
    </div>
  );
}
