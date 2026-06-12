import { useState, useEffect, useRef } from "react";
import L from "leaflet";
import "@/lib/leaflet-setup";

import { Header } from "@/components/Header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { MapPin, Zap, Building2, Loader2 } from "lucide-react";
import { useStations, useStation } from "@/features/stations/hooks";

const TEHRAN_CENTER: L.LatLngTuple = [35.7219, 51.3347];
const DEFAULT_ZOOM = 12;
const OSM_TILE_URL = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png";
const OSM_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const MapPage = () => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data: stations, isLoading, isError } = useStations();
  const { data: selected } = useStation(selectedId);

  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef       = useRef<L.Map | null>(null);
  const groupRef     = useRef<L.LayerGroup | null>(null);

  // Initialise the map once on mount.
  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;
    const map = L.map(containerRef.current, { center: TEHRAN_CENTER, zoom: DEFAULT_ZOOM });
    L.tileLayer(OSM_TILE_URL, { attribution: OSM_ATTRIBUTION }).addTo(map);
    groupRef.current = L.layerGroup().addTo(map);
    mapRef.current   = map;
    return () => {
      map.remove();
      mapRef.current  = null;
      groupRef.current = null;
    };
  }, []);

  // Re-render markers whenever the stations list changes.
  useEffect(() => {
    const group = groupRef.current;
    if (!group || !stations) return;
    group.clearLayers();
    stations.forEach((station) => {
      L.marker([station.latitude, station.longitude])
        .bindPopup(
          `<strong>${station.name}</strong>${
            station.power_kw != null ? `<br/>${station.power_kw} kW` : ""
          }`
        )
        .on("click", () => setSelectedId(station.id))
        .addTo(group);
    });
  }, [stations]);

  return (
    <div className="min-h-screen app-page-bg" dir="rtl">
      <Header />
      <main className="container mx-auto p-3 sm:p-4 space-y-3 sm:space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl sm:text-3xl font-bold">نقشه ایستگاه‌های شارژ</h1>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-4">
          <div className="lg:col-span-2">
            <Card>
              <CardContent className="p-0">
                <div
                  className="w-full rounded-lg overflow-hidden bg-muted"
                  style={{ height: "500px" }}
                >
                  <div ref={containerRef} style={{ width: "100%", height: "100%" }} />
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="space-y-3 sm:space-y-4">
            {selected ? (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <MapPin className="w-5 h-5" />
                    {selected.name}
                  </CardTitle>
                  {selected.address && (
                    <CardDescription>{selected.address}</CardDescription>
                  )}
                </CardHeader>
                <CardContent className="space-y-3 sm:space-y-4">
                  {selected.power_kw != null && (
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">قدرت</span>
                      <div className="flex items-center gap-1">
                        <Zap className="w-4 h-4 text-yellow-500" />
                        <span className="font-semibold">{selected.power_kw} kW</span>
                      </div>
                    </div>
                  )}
                  {selected.connector_types && (
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">نوع کانکتور</span>
                      <div className="flex flex-wrap gap-1 justify-end">
                        {selected.connector_types.split(",").map((ct) => (
                          <Badge key={ct} variant="outline" className="font-bold">
                            {ct.trim()}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  {selected.operator && (
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">اپراتور</span>
                      <div className="flex items-center gap-1">
                        <Building2 className="w-4 h-4 text-muted-foreground" />
                        <span className="font-semibold">{selected.operator}</span>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle>راهنما</CardTitle>
                  <CardDescription>
                    روی مارکرها کلیک کنید تا اطلاعات ایستگاه شارژ را ببینید
                  </CardDescription>
                </CardHeader>
              </Card>
            )}

            <Card>
              <CardHeader>
                <CardTitle>لیست ایستگاه‌ها</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 max-h-[300px] sm:max-h-[400px] overflow-y-auto">
                {isLoading && (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    در حال بارگذاری…
                  </div>
                )}
                {isError && (
                  <p className="text-sm text-destructive">خطا در بارگذاری ایستگاه‌ها</p>
                )}
                {!isLoading && !isError && stations?.length === 0 && (
                  <p className="text-sm text-muted-foreground">ایستگاهی ثبت نشده است</p>
                )}
                {stations?.map((station) => (
                  <div
                    key={station.id}
                    className={`p-2 sm:p-3 border rounded-lg cursor-pointer hover:bg-accent transition-colors ${
                      selectedId === station.id ? "bg-accent" : ""
                    }`}
                    onClick={() => setSelectedId(station.id)}
                  >
                    <h3 className="font-semibold text-xs sm:text-sm truncate">
                      {station.name}
                    </h3>
                    <div className="flex items-center gap-2 sm:gap-4 mt-2 text-xs flex-wrap">
                      {station.power_kw != null && (
                        <span className="flex items-center gap-1">
                          <Zap className="w-3 h-3" />
                          {station.power_kw} kW
                        </span>
                      )}
                      {station.connector_types && (
                        <span>{station.connector_types}</span>
                      )}
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  );
};

export default MapPage;
