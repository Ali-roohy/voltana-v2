import { api } from "@/lib/api";

// Lightweight marker projection from GET /v1/stations.
export interface StationMarker {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
  connector_types: string | null;
  power_kw: number | null;
}

// Full station detail from GET /v1/stations/:id.
export interface Station extends StationMarker {
  address: string | null;
  operator: string | null;
  created_at: string;
  updated_at: string;
}

interface ListResponse<T> {
  items: T[];
  limit: number;
  offset: number;
  total: number;
}

export async function listStations(): Promise<StationMarker[]> {
  const res = await api.get<ListResponse<StationMarker>>("/v1/stations");
  return res.items;
}

export const getStation = (id: string) => api.get<Station>(`/v1/stations/${id}`);
