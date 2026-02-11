const COLOR_MAP: Record<string, string> = {
  "grey lighten-2": "#eeeeee",
  "yellow accent-1": "#fff9c4",
  yellow: "#ffeb3b",
  "orange darken-2": "#f57c00",
  "red darken-2": "#d32f2f",
  "green lighten-1": "#66bb6a",
};

export function colorToHex(name: string) {
  return COLOR_MAP[name] || "#ffffff";
}

export function colorForAge(
  elapsedSeconds: number | undefined,
  flashSeconds: number,
  fadeSeconds: number,
) {
  if (elapsedSeconds === undefined) {
    return "#0274b8";
  }
  const flash = Math.max(0, Math.floor(flashSeconds));
  const fade = Math.max(0, Math.floor(fadeSeconds));
  if (elapsedSeconds < flash) {
    return "#d32f2f";
  }
  if (fade <= 0) {
    return "#0274b8";
  }
  const phaseSeconds = elapsedSeconds - flash;
  if (phaseSeconds >= fade) {
    return "#0274b8";
  }
  const bucket = Math.max(1, fade / 4);

  if (phaseSeconds < bucket) return "#d32f2f"; // red
  if (phaseSeconds < bucket * 2) return "#f57c00"; // orange
  if (phaseSeconds < bucket * 3) return "#ffeb3b"; // yellow
  return "#66bb6a"; // green
}

export function hashString(str: string) {
  let hash = 0;
  if (str.length === 0) return hash;
  for (let i = 0; i < str.length; i += 1) {
    const chr = str.charCodeAt(i);
    hash = (hash << 5) - hash + chr;
    hash |= 0;
  }
  return hash;
}

export function transformComponent(
  position?: { x: number; y: number },
  scale?: number,
) {
  const translate = position ? `translate(${position.x} ${position.y})` : "";
  const scalePart = scale ? `scale(${scale})` : "";
  return [translate, scalePart].join(" ");
}

type Point = { x: number; y: number };

type UnloadedMarkerOptions = {
  origin: Point;
  hashSeed: string;
  direction?: Point;
  systems?: Point[];
  minDistance?: number;
  maxDistance?: number;
  avoidRadius?: number;
  bump?: number;
  radialSteps?: number;
  radialStepSize?: number;
  scatterScale?: number;
};

export function placeUnloadedMarker({
  origin,
  hashSeed,
  direction,
  systems = [],
  minDistance = 90,
  maxDistance = 200,
  avoidRadius = 48,
  bump = 20,
  radialSteps = 10,
  radialStepSize = 8,
  scatterScale = 2,
}: UnloadedMarkerOptions): Point {
  const hash = hashString(hashSeed);
  const radial = minDistance + ((hash >> 4) % radialSteps) * radialStepSize;
  const scatter = ((hash >> 8) % 9) - 4;

  let pos: Point;
  if (direction) {
    const len = Math.hypot(direction.x, direction.y) || 1;
    const nx = direction.x / len;
    const ny = direction.y / len;
    const px = -ny;
    const py = nx;
    pos = {
      x: origin.x + nx * radial + px * scatter * scatterScale,
      y: origin.y + ny * radial + py * scatter * scatterScale,
    };
  } else {
    const angle = (hash % 360) * (Math.PI / 180);
    pos = {
      x: origin.x + Math.cos(angle) * radial,
      y: origin.y + Math.sin(angle) * radial,
    };
  }

  for (const systemPos of systems) {
    const sx = pos.x - systemPos.x;
    const sy = pos.y - systemPos.y;
    const dist = Math.hypot(sx, sy) || 1;
    if (dist < avoidRadius) {
      const scale = (avoidRadius - dist + bump) / dist;
      pos = {
        x: pos.x + sx * scale,
        y: pos.y + sy * scale,
      };
    }
  }

  for (const systemPos of systems) {
    const sx = pos.x - systemPos.x;
    const sy = pos.y - systemPos.y;
    const dist = Math.hypot(sx, sy) || 1;
    if (dist < avoidRadius) {
      const scale = (avoidRadius - dist + bump) / dist;
      pos = {
        x: pos.x + sx * scale,
        y: pos.y + sy * scale,
      };
    }
  }

  if (systems.length > 0) {
    let closest = systems[0];
    let closestDist = Math.hypot(pos.x - closest.x, pos.y - closest.y);
    for (let i = 1; i < systems.length; i += 1) {
      const systemPos = systems[i];
      const dist = Math.hypot(pos.x - systemPos.x, pos.y - systemPos.y);
      if (dist < closestDist) {
        closestDist = dist;
        closest = systemPos;
      }
    }
    if (closestDist < avoidRadius) {
      const sx = pos.x - closest.x;
      const sy = pos.y - closest.y;
      const dist = Math.hypot(sx, sy) || 1;
      const scale = (avoidRadius - dist + bump) / dist;
      pos = {
        x: pos.x + sx * scale,
        y: pos.y + sy * scale,
      };
    }
  }

  let dx = pos.x - origin.x;
  let dy = pos.y - origin.y;
  let distance = Math.hypot(dx, dy) || 1;
  if (distance < minDistance) {
    const scale = minDistance / distance;
    pos = {
      x: origin.x + dx * scale,
      y: origin.y + dy * scale,
    };
  }

  dx = pos.x - origin.x;
  dy = pos.y - origin.y;
  distance = Math.hypot(dx, dy) || 1;
  if (distance > maxDistance) {
    const scale = maxDistance / distance;
    pos = {
      x: origin.x + dx * scale,
      y: origin.y + dy * scale,
    };
  }

  return pos;
}
