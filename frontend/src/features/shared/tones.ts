export enum Tone {
  Blue = "blue",
  Yellow = "yellow",
  Green = "green",
  Purple = "purple",
  Gray = "gray",
  Red = "red",
  LightBlue = "lightblue",
  Orange = "orange",
}

export function toneButtonClass(tone: string, active: boolean): string {
  return active
    ? `tone-btn-${tone}-active`
    : `btn-outline tone-btn-${tone}-idle`;
}

export function badgeToneClass(tone: Tone): string {
  return `badge-tone-${tone}`;
}

export function toToneOrDefault(
  value: string | undefined,
  fallback: Tone = Tone.Gray,
): Tone {
  switch (value) {
    case Tone.LightBlue:
      return Tone.LightBlue;
    case Tone.Blue:
      return Tone.Blue;
    case Tone.Yellow:
      return Tone.Yellow;
    case Tone.Green:
      return Tone.Green;
    case Tone.Purple:
      return Tone.Purple;
    case Tone.Gray:
      return Tone.Gray;
    case Tone.Red:
      return Tone.Red;
    case Tone.Orange:
      return Tone.Orange;
    default:
      return fallback;
  }
}
