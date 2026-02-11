import NavigationRouteControlCard from "./NavigationRouteControlCard";
import NavigationFavoritesCard from "./NavigationFavoritesCard";
import NavigationTopRoutesCard from "./NavigationTopRoutesCard";
import PanelContainer from "@/components/PanelContainer";

export default function NavigationPanel() {
  return (
    <PanelContainer
      title="Navigation Matrix"
      subtitle="Map overlays and route control"
    >
      <NavigationRouteControlCard />
      <NavigationFavoritesCard />
      <NavigationTopRoutesCard />
    </PanelContainer>
  );
}
