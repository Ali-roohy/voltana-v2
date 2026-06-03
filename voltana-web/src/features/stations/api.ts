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

// Admin write payload. Mirrors the API's stationRequest (TASK-0013): `power_kw`
// (not "max_power_kw"), optional address/operator/connector_types. lat/lng are
// required numbers; the optional fields are omitted when blank so the server's
// `omitempty` validators (power_kw min=1, etc.) don't reject empty values.
export interface StationInput {
  name: string;
  latitude: number;
  longitude: number;
  address?: string | null;
  operator?: string | null;
  connector_types?: string | null;
  power_kw?: number | null;
}

function toPayload(input: StationInput) {
  const trimmed = (v?: string | null) => {
    const t = v?.trim();
    return t ? t : undefined;
  };
  return {
    name: input.name.trim(),
    latitude: input.latitude,
    longitude: input.longitude,
    address: trimmed(input.address),
    operator: trimmed(input.operator),
    connector_types: trimmed(input.connector_types),
    power_kw: input.power_kw && input.power_kw > 0 ? input.power_kw : undefined,
  };
}

export const createStation = (input: StationInput) =>
  api.post<Station>("/v1/stations", toPayload(input));

export const updateStation = (id: string, input: StationInput) =>
  api.put<Station>(`/v1/stations/${id}`, toPayload(input));

export const deleteStation = (id: string) => api.del<void>(`/v1/stations/${id}`);
