import { useState } from "react";
import { MapContainer, TileLayer, Marker, Popup } from "react-leaflet";

import { Header } from "@/components/Header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { MapPin, Zap, Building2, Loader2 } from "lucide-react";
import { useStations, useStation } from "@/features/stations/hooks";

// Tehran — same default view the previous iframe map centered on.
const TEHRAN_CENTER: [number, number] = [35.7219, 51.3347];
const DEFAULT_ZOOM = 12;
const OSM_TILE_URL = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png";
const OSM_ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const Map = () => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data: stations, isLoading, isError } = useStations();
  // Fetch full detail (address/operator) for the selected marker.
  const { data: selected } = useStation(selectedId);

  return (
    <div className="min-h-screen bg-background" dir="rtl">
      <Header />
      <main className="container mx-auto p-3 sm:p-4 space-y-3 sm:space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl sm:text-3xl font-bold">نقشه ایستگاه‌های شارژ</h1>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-4">
          <div className="lg:col-span-2">
            <Card>
              <CardContent className="p-0">
                <div className="w-full h-[400px] sm:h-[500px] lg:h-[600px] rounded-lg overflow-hidden bg-muted">
                  <MapContainer
                    center={TEHRAN_CENTER}
                    zoom={DEFAULT_ZOOM}
                    scrollWheelZoom
                    style={{ width: "100%", height: "100%" }}
                  >
                    <TileLayer url={OSM_TILE_URL} attribution={OSM_ATTRIBUTION} />
                    {stations?.map((station) => (
                      <Marker
                        key={station.id}
                        position={[station.latitude, station.longitude]}
                        eventHandlers={{ click: () => setSelectedId(station.id) }}
                      >
                        <Popup>
                          <div className="font-semibold">{station.name}</div>
                          {station.power_kw != null && <div>{station.power_kw} kW</div>}
                        </Popup>
                      </Marker>
                    ))}
                  </MapContainer>
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
                  {selected.address && <CardDescription>{selected.address}</CardDescription>}
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
                    <h3 className="font-semibold text-xs sm:text-sm truncate">{station.name}</h3>
                    <div className="flex items-center gap-2 sm:gap-4 mt-2 text-xs flex-wrap">
                      {station.power_kw != null && (
                        <span className="flex items-center gap-1">
                          <Zap className="w-3 h-3" />
                          {station.power_kw} kW
                        </span>
                      )}
                      {station.connector_types && <span>{station.connector_types}</span>}
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

export default Map;
