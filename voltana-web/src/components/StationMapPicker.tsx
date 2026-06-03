import { useMemo } from "react";
import { MapContainer, TileLayer, Marker, useMapEvents, useMap } from "react-leaflet";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
// Same Vite marker-icon fix as pages/Map.tsx (idempotent if Map already merged).
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";

L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

const TEHRAN_CENTER: [number, number] = [35.7219, 51.3347];
const DEFAULT_ZOOM = 12;
const OSM_TILE_URL = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png";
const OSM_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const inLatRange = (n: number) => Number.isFinite(n) && n >= -90 && n <= 90;
const inLngRange = (n: number) => Number.isFinite(n) && n >= -180 && n <= 180;

// Click anywhere on the map to (re)place the marker.
function ClickToPlace({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click: (e) => onPick(e.latlng.lat, e.latlng.lng),
  });
  return null;
}

// Recenter the view when the coords change from outside (typing in the inputs),
// without fighting the user's manual pan/zoom on every render.
function Recenter({ lat, lng }: { lat: number | null; lng: number | null }) {
  const map = useMap();
  useMemo(() => {
    if (lat != null && lng != null && inLatRange(lat) && inLngRange(lng)) {
      map.setView([lat, lng], map.getZoom());
    }
  }, [lat, lng, map]);
  return null;
}

interface StationMapPickerProps {
  latitude: number | null;
  longitude: number | null;
  onChange: (lat: number, lng: number) => void;
  className?: string;
}

// Leaflet/OSM picker reused by the admin add/edit form: click or drag the marker
// to set coordinates; typing valid lat/lng recenters the view. Coordinates are
// rounded to 6 decimals (~0.1 m) to keep the form values tidy.
export function StationMapPicker({ latitude, longitude, onChange, className }: StationMapPickerProps) {
  const hasPos = latitude != null && longitude != null && inLatRange(latitude) && inLngRange(longitude);
  const round = (n: number) => Math.round(n * 1e6) / 1e6;
  const pick = (lat: number, lng: number) => onChange(round(lat), round(lng));

  return (
    <div className={className ?? "w-full h-[300px] rounded-lg overflow-hidden bg-muted"}>
      <MapContainer
        center={hasPos ? [latitude as number, longitude as number] : TEHRAN_CENTER}
        zoom={DEFAULT_ZOOM}
        scrollWheelZoom
        style={{ width: "100%", height: "100%" }}
      >
        <TileLayer url={OSM_TILE_URL} attribution={OSM_ATTRIBUTION} />
        <ClickToPlace onPick={pick} />
        <Recenter lat={latitude} lng={longitude} />
        {hasPos && (
          <Marker
            position={[latitude as number, longitude as number]}
            draggable
            eventHandlers={{
              dragend: (e) => {
                const { lat, lng } = e.target.getLatLng();
                pick(lat, lng);
              },
            }}
          />
        )}
      </MapContainer>
    </div>
  );
}
