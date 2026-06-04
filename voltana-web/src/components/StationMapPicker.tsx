import { useEffect, useRef } from "react";
import L from "leaflet";
import "@/lib/leaflet-setup";

const TEHRAN_CENTER: L.LatLngTuple = [35.7219, 51.3347];
const DEFAULT_ZOOM = 12;
const OSM_TILE_URL = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png";
const OSM_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const inLatRange = (n: number) => Number.isFinite(n) && n >= -90 && n <= 90;
const inLngRange = (n: number) => Number.isFinite(n) && n >= -180 && n <= 180;
const round6 = (n: number) => Math.round(n * 1e6) / 1e6;

export interface StationMapPickerProps {
  latitude: number | null;
  longitude: number | null;
  onChange: (lat: number, lng: number) => void;
  className?: string;
}

// Click or drag the marker to pick coordinates. Typing valid values into the
// parent form recenters the view (unless the change came from a map interaction).
export function StationMapPicker({
  latitude,
  longitude,
  onChange,
  className,
}: StationMapPickerProps) {
  const containerRef    = useRef<HTMLDivElement>(null);
  const mapRef          = useRef<L.Map | null>(null);
  const markerRef       = useRef<L.Marker | null>(null);
  const onChangeRef     = useRef(onChange);
  const skipRecenterRef = useRef(false); // set true when the change came from map itself

  // Keep onChangeRef current so event handlers never have a stale closure.
  onChangeRef.current = onChange;

  const hasPos =
    latitude != null &&
    longitude != null &&
    inLatRange(latitude) &&
    inLngRange(longitude);

  // Initialise the map once on mount.
  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    const center: L.LatLngTuple = hasPos
      ? [latitude as number, longitude as number]
      : TEHRAN_CENTER;

    const map = L.map(containerRef.current, { center, zoom: DEFAULT_ZOOM });
    L.tileLayer(OSM_TILE_URL, { attribution: OSM_ATTRIBUTION }).addTo(map);

    map.on("click", (e: L.LeafletMouseEvent) => {
      skipRecenterRef.current = true;
      onChangeRef.current(round6(e.latlng.lat), round6(e.latlng.lng));
    });

    if (hasPos) {
      const m = L.marker([latitude as number, longitude as number], {
        draggable: true,
      }).addTo(map);
      m.on("dragend", () => {
        skipRecenterRef.current = true;
        const ll = m.getLatLng();
        onChangeRef.current(round6(ll.lat), round6(ll.lng));
      });
      markerRef.current = m;
    }

    mapRef.current = map;
    return () => {
      map.remove();
      mapRef.current  = null;
      markerRef.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync marker position (and optionally the view) when coords change from outside.
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    if (hasPos) {
      const pos: L.LatLngTuple = [latitude as number, longitude as number];

      if (markerRef.current) {
        markerRef.current.setLatLng(pos);
      } else {
        const m = L.marker(pos, { draggable: true }).addTo(map);
        m.on("dragend", () => {
          skipRecenterRef.current = true;
          const ll = m.getLatLng();
          onChangeRef.current(round6(ll.lat), round6(ll.lng));
        });
        markerRef.current = m;
      }

      if (!skipRecenterRef.current) {
        map.setView(pos, map.getZoom(), { animate: true });
      }
    } else {
      markerRef.current?.remove();
      markerRef.current = null;
    }

    skipRecenterRef.current = false;
  }, [latitude, longitude]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div
      ref={containerRef}
      className={className ?? "w-full h-[300px] rounded-lg overflow-hidden bg-muted"}
    />
  );
}
