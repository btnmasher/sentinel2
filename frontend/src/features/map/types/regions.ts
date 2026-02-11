const normalizeRegionName = (value: string) =>
  value.toLowerCase().replace(/_/g, " ").replace(/\s+/g, " ").trim();

export const REGIONS = [
  { region: "10000014", name: "Catch" },
  { region: "10000039", name: "Esoteria" },
  { region: "10000031", name: "Impass" },
  { region: "10000059", name: "Paragon Soul" },
  { region: "10000054", name: "Aridia" },
  { region: "10000069", name: "Black Rise" },
  { region: "10000055", name: "Branch" },
  { region: "10000007", name: "Cache" },
  { region: "10000051", name: "Cloud Ring" },
  { region: "10000053", name: "Cobalt Edge" },
  { region: "10000012", name: "Curse" },
  { region: "10000035", name: "Deklein" },
  { region: "10000060", name: "Delve" },
  { region: "10000001", name: "Derelik" },
  { region: "10000005", name: "Detorid" },
  { region: "10000036", name: "Devoid" },
  { region: "10000043", name: "Domain" },
  { region: "10000064", name: "Essence" },
  { region: "10000027", name: "Etherium Reach" },
  { region: "10000037", name: "Everyshore" },
  { region: "10000046", name: "Fade" },
  { region: "10000056", name: "Feythabolis" },
  { region: "10000058", name: "Fountain" },
  { region: "10000029", name: "Geminate" },
  { region: "10000067", name: "Genesis" },
  { region: "10000011", name: "Great Wildlands" },
  { region: "10000030", name: "Heimatar" },
  { region: "10000025", name: "Immensea" },
  { region: "10000009", name: "Insmother" },
  { region: "10000052", name: "Kador" },
  { region: "10000049", name: "Khanid" },
  { region: "10000065", name: "Kor-Azor" },
  { region: "10000016", name: "Lonetrek" },
  { region: "10000013", name: "Malpais" },
  { region: "10000042", name: "Metropolis" },
  { region: "10000028", name: "Molden Heath" },
  { region: "10000040", name: "Oasa" },
  { region: "10000062", name: "Omist" },
  { region: "10000021", name: "Outer Passage" },
  { region: "10000057", name: "Outer Ring" },
  { region: "10000063", name: "Period Basis" },
  { region: "10000066", name: "Perrigen Falls" },
  { region: "10000048", name: "Placid" },
  { region: "10000047", name: "Providence" },
  { region: "10000023", name: "Pure Blind" },
  { region: "10000050", name: "Querious" },
  { region: "10000008", name: "Scalding Pass" },
  { region: "10000032", name: "Sinq Laison" },
  { region: "10000044", name: "Solitude" },
  { region: "10000022", name: "Stain" },
  { region: "10000041", name: "Syndicate" },
  { region: "10000020", name: "Tash-Murkon" },
  { region: "10000045", name: "Tenal" },
  { region: "10000061", name: "Tenerifis" },
  { region: "10000038", name: "The Bleak Lands" },
  { region: "10000033", name: "The Citadel" },
  { region: "10000002", name: "The Forge" },
  { region: "10000034", name: "The Kalevala Expanse" },
  { region: "10000018", name: "The Spire" },
  { region: "10000010", name: "Tribute" },
  { region: "10000003", name: "Vale of the Silent" },
  { region: "10000015", name: "Venal" },
  { region: "10000068", name: "Verge Vendor" },
  { region: "10000006", name: "Wicked Creek" },
];

export const REGION_MAP = REGIONS.reduce(
  (map, region) => {
    map[region.region] = region.name;
    return map;
  },
  {} as Record<string, string>,
);

const REGION_NAME_MAP = REGIONS.reduce(
  (map, region) => {
    map[normalizeRegionName(region.name)] = region.region;
    return map;
  },
  {} as Record<string, string>,
);

export const resolveRegionTokens = (input: string) => {
  const tokens = input
    .split(/[,+]/)
    .map((token) => token.trim())
    .filter(Boolean);

  const resolved: string[] = [];
  for (const token of tokens) {
    const numeric = Number.parseInt(token, 10);
    if (!Number.isNaN(numeric)) {
      resolved.push(String(numeric));
      continue;
    }
    const normalized = normalizeRegionName(token);
    const match = REGION_NAME_MAP[normalized];
    if (match) {
      resolved.push(match);
    }
  }
  return Array.from(new Set(resolved));
};
