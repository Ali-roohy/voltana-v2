import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  listStations,
  getStation,
  createStation,
  updateStation,
  deleteStation,
  type StationInput,
} from "./api";

export function useStations() {
  return useQuery({ queryKey: ["stations"], queryFn: listStations });
}

export function useStation(id: string | null) {
  return useQuery({
    queryKey: ["stations", id],
    queryFn: () => getStation(id as string),
    enabled: !!id,
  });
}

// ── admin mutations ───────────────────────────────────────────────────────────
// Each invalidates the ["stations"] list (and detail) so the table + map refresh.

export function useCreateStation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: StationInput) => createStation(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["stations"] }),
  });
}

export function useUpdateStation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: StationInput }) => updateStation(id, input),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ["stations"] });
      qc.invalidateQueries({ queryKey: ["stations", id] });
    },
  });
}

export function useDeleteStation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteStation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["stations"] }),
  });
}
