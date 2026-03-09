import IntelChannelsCard from "./IntelChannelsCard";
import JumpbridgeListCard from "./JumpbridgeListCard";
import OrganizationStandingsCard from "./OrganizationStandingsCard";

export default function Staff() {
  return (
    <div className="grid gap-6 items-start lg:h-[calc(100dvh-8.5rem)] lg:grid-cols-[1fr_1fr] lg:items-stretch">
      <div className="flex min-h-0 flex-col gap-6 lg:overflow-hidden">
        <IntelChannelsCard />
        <OrganizationStandingsCard />
      </div>
      <div className="min-h-0 lg:overflow-hidden">
        <JumpbridgeListCard />
      </div>
    </div>
  );
}
