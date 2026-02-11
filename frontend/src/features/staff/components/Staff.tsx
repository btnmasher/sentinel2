import IntelChannelsCard from "./IntelChannelsCard";
import JumpbridgeListCard from "./JumpbridgeListCard";

export default function Staff() {
  return (
    <div className="grid lg:grid-cols-[1fr_1fr] gap-6">
      <IntelChannelsCard />
      <JumpbridgeListCard />
    </div>
  );
}
