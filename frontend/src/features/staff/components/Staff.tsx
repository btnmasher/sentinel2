import IntelChannelsCard from "./IntelChannelsCard";
import JumpbridgeListCard from "./JumpbridgeListCard";
import OrganizationStandingsCard from "./OrganizationStandingsCard";

export default function Staff() {
  return (
    <div className="grid lg:grid-cols-[1fr_1fr] gap-6 items-start">
      <div className="space-y-6">
        <IntelChannelsCard />
        <OrganizationStandingsCard />
      </div>
      <div>
        <JumpbridgeListCard />
      </div>
    </div>
  );
}
