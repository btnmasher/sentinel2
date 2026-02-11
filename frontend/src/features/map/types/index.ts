export type Region = {
  region: number;
  name: string;
  position: { x: number; y: number };
};

export type System = {
  system: number;
  name: string;
  security_status: number;
  region: number;
  constellation: number;
  position: { x: number; y: number };
  absolute: { x: number; y: number; z: number };
};

export type Gate = {
  from: number;
  to: number;
  type: "solarsystem" | "constellation" | "region";
  to_region: number;
  from_region: number;
  to_dotlan_x?: number;
  to_dotlan_y?: number;
  to_metro_x?: number;
  to_metro_y?: number;
  from_dotlan_x?: number;
  from_dotlan_y?: number;
  from_metro_x?: number;
  from_metro_y?: number;
};

export type Jumpbridge = {
  from: number;
  to: number;
  from_region?: number;
  to_region?: number;
  friendly: boolean;
};

export type Character = {
  id: number;
  name: string;
  is_main?: boolean;
};

export type MapLayout = "dotlan" | "metro" | "real" | "eve2d";
