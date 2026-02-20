import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import RegionRenderer from "./RegionRenderer";
import MapJumpbridge, { buildJumpbridgePath } from "./MapJumpbridge";
import MapRegionGateUnloaded from "./MapRegionGateUnloaded";
import { regionMap as buildRegionMap, useMapStore } from "../store/mapStore";
import { useSettingsStore } from "@/app/store/settingsStore";
import { useUIStore } from "@/app/store/uiStore";
import { useIsCharacterVisible } from "../hooks/useCharacterVisibility";
const ZOOM_FACTOR = 1.25;
const ZOOM_BOUNDS = 8;

export default function MapCanvas() {
  const svgRef = useRef<SVGSVGElement | null>(null);
  const groupRef = useRef<SVGGElement | null>(null);

  const regions = useMapStore((s) => s.regions);
  const systems = useMapStore((s) => s.systems);
  const gates = useMapStore((s) => s.gates);
  const jumpbridges = useMapStore((s) => s.jumpbridges);
  const route = useMapStore((s) => s.route);
  const lastRouteCharacter = useMapStore((s) => s.lastRouteCharacter);
  const isRouteCharacterVisible = useIsCharacterVisible(lastRouteCharacter);
  const mapLayout = useMapStore((s) => s.mapLayout);
  const mapRegions = useMapStore((s) => s.mapRegions);
  const displayJumpbridges = useMapStore((s) => s.displayJumpbridges);
  const systemSearch = useMapStore((s) => s.systemSearch);
  const setSystemSearch = useMapStore((s) => s.setSystemSearch);
  const setMapControls = useMapStore((s) => s.setMapControls);
  const fetchMapTopology = useMapStore((s) => s.fetchMapTopology);
  const fetchMapOverlays = useMapStore((s) => s.fetchMapOverlays);
  const setMapScale = useMapStore((s) => s.setMapScale);
  const mapSettings = useSettingsStore((s) => s.settings.map);
  const setContextMenu = useUIStore((s) => s.setContextMenu);
  const centerMapRef = useRef<() => void>(() => {});
  const zoomByRef = useRef<(delta: number) => void>(() => {});

  const [zoom, setZoom] = useState(0);
  const [matrix, setMatrixState] = useState({
    a: 1,
    b: 0,
    c: 0,
    d: 1,
    e: 0,
    f: 0,
  });
  const [isPanning, setIsPanning] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const panStart = useRef<{ x: number; y: number } | null>(null);
  const hasPointerCapture = useRef(false);
  const [searchClearable, setSearchClearable] = useState(true);
  const hasCentered = useRef(false);
  const [viewportSize, setViewportSize] = useState({ width: 0, height: 0 });

  const regionMap = useMemo(
    () => buildRegionMap(regions, systems),
    [regions, systems],
  );
  const regionEntries = useMemo(() => Object.values(regionMap), [regionMap]);

  useEffect(() => {
    const element = svgRef.current;
    if (!element) return;
    const update = () => {
      const rect = element.getBoundingClientRect();
      setViewportSize({ width: rect.width, height: rect.height });
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const handler = setTimeout(() => {
      fetchMapTopology();
    }, 250);
    return () => clearTimeout(handler);
  }, [fetchMapTopology, mapRegions, mapLayout]);

  useEffect(() => {
    const handler = setTimeout(() => {
      fetchMapOverlays();
    }, 250);
    return () => clearTimeout(handler);
  }, [fetchMapOverlays, mapRegions]);

  const unloadedRegionGates = useMemo(
    () =>
      gates.filter(
        (gate) =>
          gate.type === "region" &&
          (!regions[gate.to_region] || !regions[gate.from_region]),
      ),
    [gates, regions],
  );

  const regionGates = useMemo(() => {
    return gates
      .filter(
        (gate) =>
          gate.type === "region" &&
          regions[gate.to_region] &&
          regions[gate.from_region],
      )
      .map((gate) => {
        const a = systems[gate.from];
        const b = systems[gate.to];
        const regionA = regions[a.region];
        const regionB = regions[b.region];
        return {
          a: {
            x: regionA.position.x + a.position.x,
            y: regionA.position.y + a.position.y,
          },
          b: {
            x: regionB.position.x + b.position.x,
            y: regionB.position.y + b.position.y,
          },
        };
      });
  }, [gates, regions, systems]);

  const jumpbridgeEdges = useMemo(() => {
    const edges = new Map<string, boolean>();
    jumpbridges.forEach((bridge) => {
      const disabled = Boolean(bridge.disabled);
      edges.set(`${bridge.from}:${bridge.to}`, disabled);
      edges.set(`${bridge.to}:${bridge.from}`, disabled);
    });
    return edges;
  }, [jumpbridges]);

  const routeSegments = useMemo(() => {
    if (!isRouteCharacterVisible) {
      return [];
    }
    const pairs = route
      .map((value, idx) => [value, route[idx + 1]] as const)
      .filter((pair) => pair[1]);
    return pairs
      .map(([fromId, toId]) => {
        const from = systems[fromId];
        const to = systems[toId];
        if (!from || !to) return null;
        const jumpbridgeDisabled = jumpbridgeEdges.get(`${fromId}:${toId}`);
        const isJumpbridge = jumpbridgeDisabled !== undefined;
        if (isJumpbridge) {
          return {
            type: "path" as const,
            d: buildJumpbridgePath(from, to, regions),
            disabled: Boolean(jumpbridgeDisabled),
          };
        }
        const a = {
          x: regions[from.region].position.x + from.position.x,
          y: regions[from.region].position.y + from.position.y,
        };
        const b = {
          x: regions[to.region].position.x + to.position.x,
          y: regions[to.region].position.y + to.position.y,
        };
        return { type: "line" as const, a, b };
      })
      .filter(
        (
          segment,
        ): segment is
          | { type: "path"; d: string; disabled: boolean }
          | {
              type: "line";
              a: { x: number; y: number };
              b: { x: number; y: number };
            } => Boolean(segment),
      );
  }, [jumpbridgeEdges, regions, route, systems, isRouteCharacterVisible]);

  const mapTransform = `matrix(${matrix.a},${matrix.b},${matrix.c},${matrix.d},${matrix.e},${matrix.f})`;

  const visibleWorldBounds = useMemo(() => {
    if (viewportSize.width <= 0 || viewportSize.height <= 0) {
      return undefined;
    }
    const scale = matrix.a || 1;
    const left = (0 - matrix.e) / scale;
    const top = (0 - matrix.f) / scale;
    const right = (viewportSize.width - matrix.e) / scale;
    const bottom = (viewportSize.height - matrix.f) / scale;
    const PAD = 240;
    return {
      left: Math.min(left, right) - PAD,
      right: Math.max(left, right) + PAD,
      top: Math.min(top, bottom) - PAD,
      bottom: Math.max(top, bottom) + PAD,
    };
  }, [matrix.a, matrix.e, matrix.f, viewportSize.height, viewportSize.width]);

  const visibleSystemRegions = useMemo(() => {
    if (!hasCentered.current || !visibleWorldBounds) {
      return regionEntries;
    }
    return regionEntries.filter((regionEntry) => {
      if (regionEntry.systems.length === 0) return false;
      let minX = Number.POSITIVE_INFINITY;
      let minY = Number.POSITIVE_INFINITY;
      let maxX = Number.NEGATIVE_INFINITY;
      let maxY = Number.NEGATIVE_INFINITY;

      for (const system of regionEntry.systems) {
        const x = regionEntry.region.position.x + system.position.x;
        const y = regionEntry.region.position.y + system.position.y;
        minX = Math.min(minX, x);
        minY = Math.min(minY, y);
        maxX = Math.max(maxX, x);
        maxY = Math.max(maxY, y);
      }

      return !(
        maxX < visibleWorldBounds.left ||
        minX > visibleWorldBounds.right ||
        maxY < visibleWorldBounds.top ||
        minY > visibleWorldBounds.bottom
      );
    });
  }, [regionEntries, visibleWorldBounds]);

  const systemSearchPosition = useMemo(() => {
    if (!systemSearch) return undefined;
    const system = systems[systemSearch];
    if (!system) return undefined;
    const region = regions[system.region];
    return {
      x: region.position.x + system.position.x,
      y: region.position.y + system.position.y,
    };
  }, [regions, systems, systemSearch]);

  const setMatrix = useCallback(
    (next: Partial<typeof matrix>) => {
      const svgRect = svgRef.current?.getBoundingClientRect();
      const groupRect = groupRef.current?.getBoundingClientRect();
      if (!svgRect || !groupRect) {
        setMatrixState((prev) => ({ ...prev, ...next }));
        return;
      }

      const a = next.a ?? matrix.a;
      const scalar = a / matrix.a || 1;
      const bounds = {
        left: -groupRect.width * scalar + 100,
        top: -groupRect.height * scalar + 100,
        right: svgRect.width - 100,
        bottom: svgRect.height - 100,
      };

      let e = next.e ?? matrix.e;
      let f = next.f ?? matrix.f;

      if (e > bounds.right) e = bounds.right;
      if (e < bounds.left) e = bounds.left;
      if (f > bounds.bottom) f = bounds.bottom;
      if (f < bounds.top) f = bounds.top;

      const updated = {
        a,
        b: next.b ?? matrix.b,
        c: next.c ?? matrix.c,
        d: next.d ?? matrix.d,
        e,
        f,
      };

      setMatrixState(updated);
      if (next.a !== undefined) {
        setMapScale(updated.a);
      }
    },
    [matrix, setMapScale],
  );

  const centerOn = useCallback(
    (x: number, y: number, zoomLevel: number) => {
      if (zoomLevel > ZOOM_BOUNDS || zoomLevel < -ZOOM_BOUNDS) return;
      const scale = Math.pow(ZOOM_FACTOR, Math.round(zoomLevel));
      const svg = svgRef.current?.getBoundingClientRect();
      if (!svg) return;

      const newX = svg.width / 2 - x;
      const newY = svg.height / 2 - y;
      setZoom(zoomLevel);
      setMatrix({ a: scale, b: 0, c: 0, d: scale, e: newX, f: newY });
    },
    [setMatrix],
  );

  const centerMap = useCallback(() => {
    setSystemSearch(undefined);
    const groupBox = groupRef.current?.getBBox();
    const svg = svgRef.current?.getBoundingClientRect();
    if (!groupBox || !svg) return;

    const newScale = Math.min(
      svg.width / groupBox.width,
      svg.height / groupBox.height,
    );
    let zoomLevel = Math.floor(Math.log(newScale) / Math.log(ZOOM_FACTOR));
    zoomLevel = Math.min(zoomLevel, ZOOM_BOUNDS);
    zoomLevel = Math.max(zoomLevel, -ZOOM_BOUNDS);

    const newRoundScale = Math.pow(ZOOM_FACTOR, zoomLevel);
    const x = Math.floor(
      (groupBox.width * newRoundScale) / 2 + groupBox.x * newScale,
    );
    const y = Math.floor(
      (groupBox.height * newRoundScale) / 2 + groupBox.y * newScale,
    );

    centerOn(x, y, zoomLevel);
  }, [centerOn, setSystemSearch]);

  useEffect(() => {
    if (hasCentered.current || Object.keys(regions).length === 0) {
      return;
    }
    const timeout = setTimeout(() => {
      centerMap();
      hasCentered.current = true;
    }, 500);
    return () => clearTimeout(timeout);
  }, [centerMap, regions]);

  useEffect(() => {
    if (systemSearchPosition) {
      centerOn(systemSearchPosition.x, systemSearchPosition.y, 0);
      setSearchClearable(true);
    }
  }, [centerOn, systemSearchPosition]);

  const zoomBy = useCallback(
    (delta: number) => {
      const zoomLevel = zoom + delta;
      if (zoomLevel > ZOOM_BOUNDS || zoomLevel < -ZOOM_BOUNDS) return;

      const zoomScale = Math.pow(ZOOM_FACTOR, zoomLevel);
      const scale = zoomScale / matrix.a;

      const svg = svgRef.current?.getBoundingClientRect();
      if (!svg) return;

      const point = new DOMPointReadOnly(svg.width / 2, svg.height / 2);
      const oldMatrix = new DOMMatrixReadOnly([
        matrix.a,
        matrix.b,
        matrix.c,
        matrix.d,
        matrix.e,
        matrix.f,
      ]);
      const relPoint = point.matrixTransform(oldMatrix.inverse());
      const transform = new DOMMatrixReadOnly()
        .translate(relPoint.x, relPoint.y)
        .scale(scale)
        .translate(-relPoint.x, -relPoint.y);
      const newMatrix = oldMatrix.multiply(transform);

      setZoom(zoomLevel);
      setMatrix({
        a: newMatrix.a,
        b: newMatrix.b,
        c: newMatrix.c,
        d: newMatrix.d,
        e: newMatrix.e,
        f: newMatrix.f,
      });
    },
    [matrix, setMatrix, zoom],
  );

  useEffect(() => {
    setMapControls({
      fit: () => centerMapRef.current(),
      zoomIn: () => zoomByRef.current(1),
      zoomOut: () => zoomByRef.current(-1),
    });
    return () => setMapControls({});
  }, [setMapControls]);

  useEffect(() => {
    centerMapRef.current = centerMap;
  }, [centerMap]);

  useEffect(() => {
    zoomByRef.current = zoomBy;
  }, [zoomBy]);

  const mapScroll = useCallback(
    (event: React.WheelEvent<SVGSVGElement>) => {
      if (event.deltaY === 0) return;
      if (searchClearable && systemSearchPosition) {
        setSystemSearch(undefined);
      }

      const zoomIn = mapSettings.invertZoom
        ? event.deltaY > 1
        : event.deltaY < 1;
      const zoomLevel = zoom + Math.floor(zoomIn ? 1 : -1);
      if (zoomLevel > ZOOM_BOUNDS || zoomLevel < -ZOOM_BOUNDS) return;

      const zoomScale = Math.pow(ZOOM_FACTOR, zoomLevel);
      const scale = zoomScale / matrix.a;

      const svg = svgRef.current?.getBoundingClientRect();
      if (!svg) return;

      const point = new DOMPointReadOnly(
        event.clientX - svg.x,
        event.clientY - svg.y,
      );
      const oldMatrix = new DOMMatrixReadOnly([
        matrix.a,
        matrix.b,
        matrix.c,
        matrix.d,
        matrix.e,
        matrix.f,
      ]);
      const relPoint = point.matrixTransform(oldMatrix.inverse());
      const transform = new DOMMatrixReadOnly()
        .translate(relPoint.x, relPoint.y)
        .scale(scale)
        .translate(-relPoint.x, -relPoint.y);
      const newMatrix = oldMatrix.multiply(transform);

      setZoom(zoomLevel);
      setMatrix({
        a: newMatrix.a,
        b: newMatrix.b,
        c: newMatrix.c,
        d: newMatrix.d,
        e: newMatrix.e,
        f: newMatrix.f,
      });
    },
    [
      matrix,
      zoom,
      setMatrix,
      systemSearchPosition,
      searchClearable,
      setSystemSearch,
      mapSettings.invertZoom,
    ],
  );

  const mapStartPan = (event: React.PointerEvent<SVGSVGElement>) => {
    if (searchClearable && systemSearchPosition) {
      setSystemSearch(undefined);
    }
    panStart.current = { x: event.clientX, y: event.clientY };
  };

  const releasePointer = (event: React.PointerEvent<SVGSVGElement>) => {
    if (hasPointerCapture.current) {
      try {
        event.currentTarget.releasePointerCapture(event.pointerId);
      } catch {
        // ignore if capture wasn't set
      }
      hasPointerCapture.current = false;
    }
    if (isPanning) {
      event.stopPropagation();
      setIsPanning(false);
    }
    panStart.current = null;
    setIsDragging(false);
  };

  const mapDrag = (event: React.PointerEvent<SVGSVGElement>) => {
    if (!panStart.current) return;
    if (!isPanning) {
      const dx = event.clientX - panStart.current.x;
      const dy = event.clientY - panStart.current.y;
      if (Math.hypot(dx, dy) <= 3) return;
      if (!hasPointerCapture.current) {
        try {
          event.currentTarget.setPointerCapture(event.pointerId);
          hasPointerCapture.current = true;
        } catch {
          // ignore pointer capture errors
        }
      }
      setIsPanning(true);
      setIsDragging(true);
    }
    setMatrix({
      e: matrix.e + event.movementX,
      f: matrix.f + event.movementY,
    });
  };

  const showMenu = (event: React.MouseEvent<SVGSVGElement>) => {
    event.preventDefault();
    const rect = event.currentTarget.getBoundingClientRect();
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      anchorRect: {
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height,
      },
      type: "map",
    });
  };

  return (
    <div className="relative h-full">
      {isDragging && <div className="absolute inset-0 z-30" />}
      <svg
        id="intel-map"
        ref={svgRef}
        width="100%"
        height="100%"
        onPointerDown={mapStartPan}
        onPointerUp={releasePointer}
        onPointerOut={releasePointer}
        onPointerCancel={releasePointer}
        onPointerMove={mapDrag}
        onWheel={mapScroll}
        onContextMenu={showMenu}
        className="bg-base-300/30"
      >
        <g ref={groupRef} transform={mapTransform}>
          <g id="map-region-group">
            {regionEntries.map((region) => (
              <RegionRenderer
                key={`region-base-${region.region.region}`}
                region={region.region}
                regionSystems={region.systems}
                showSystems={false}
              />
            ))}
          </g>

          <g id="map-gate-group">
            {unloadedRegionGates.map((gate) => (
              <MapRegionGateUnloaded
                key={`${gate.to}-${gate.from}`}
                gate={gate}
              />
            ))}
            {regionGates.map((gate, idx) => (
              <line
                key={`${gate.a.x}-${gate.a.y}-${idx}`}
                className="map-gate region"
                x1={gate.a.x}
                y1={gate.a.y}
                x2={gate.b.x}
                y2={gate.b.y}
              />
            ))}
          </g>

          {displayJumpbridges && (
            <g>
              {jumpbridges.map((bridge) => (
                <MapJumpbridge
                  key={`${bridge.to}-${bridge.from}`}
                  jumpbridge={bridge}
                />
              ))}
            </g>
          )}

          {routeSegments.length > 0 && (
            <g id="map-route-group">
              {routeSegments.map((segment, idx) => {
                if (!segment) return null;
                if (segment.type === "path") {
                  const routeClass = segment.disabled
                    ? "map-gate route route-disabled"
                    : "map-gate route";
                  const routeBlinkClass = segment.disabled
                    ? "map-gate route route-disabled"
                    : "map-gate route route-blink";
                  return (
                    <g key={`route-path-${idx}`}>
                      <path className={routeClass} d={segment.d} />
                      <path className={routeBlinkClass} d={segment.d} />
                    </g>
                  );
                }
                return (
                  <g key={`route-line-${idx}`}>
                    <line
                      x1={segment.a.x}
                      y1={segment.a.y}
                      x2={segment.b.x}
                      y2={segment.b.y}
                      className="map-gate route"
                    />
                    <line
                      x1={segment.a.x}
                      y1={segment.a.y}
                      x2={segment.b.x}
                      y2={segment.b.y}
                      className="map-gate route route-blink"
                    />
                  </g>
                );
              })}
            </g>
          )}

          <g id="map-system-group">
            {visibleSystemRegions.map((region) => (
              <RegionRenderer
                key={`region-systems-${region.region.region}`}
                region={region.region}
                regionSystems={region.systems}
                showBase={false}
              />
            ))}
          </g>

          {systemSearchPosition && (
            <circle
              fill="none"
              stroke="#3fa739"
              strokeWidth={2}
              opacity={0.6}
              r={30}
              cx={systemSearchPosition.x}
              cy={systemSearchPosition.y}
            />
          )}
        </g>
      </svg>
    </div>
  );
}
