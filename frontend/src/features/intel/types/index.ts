export type IntelReport = {
  recordId?: string;
  id: number;
  time: number;
  author: string;
  text: string;
  channel_id?: string;
  meta?: Record<string, unknown>;
  systems: Array<{
    system: number;
    name: string;
    constellation: number;
    region: number;
  }>;
  regions: number[];
};
