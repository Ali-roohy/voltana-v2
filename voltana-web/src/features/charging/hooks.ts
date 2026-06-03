import { useQuery, useMutation, useQueryClient, keepPreviousData } from "@tanstack/react-query";
import {
  listChargingSessions,
  createChargingSession,
  updateChargingSession,
  deleteChargingSession,
  type ChargingInput,
  type ChargingListFilter,
} from "./api";

const KEY = ["charging-sessions"];

// Single source query for sessions (fixes bug #5 — the old page fetched 4×).
// An optional filter is folded into the query key so each range/car is cached
// separately; a no-arg call keeps the base key (dashboard stays unchanged).
// Mutations invalidate the base key, which prefix-matches every filtered key.
export function useChargingSessions(filter?: ChargingListFilter) {
  return useQuery({
    queryKey: filter ? [...KEY, filter] : KEY,
    queryFn: () => listChargingSessions(filter),
    placeholderData: keepPreviousData, // avoid an empty flash when the filter changes
  });
}

function useInvalidate() {
  const qc = useQueryClient();
  return () => {
    qc.invalidateQueries({ queryKey: KEY });
    qc.invalidateQueries({ queryKey: ["dashboard"] });
    // A session write changes battery SOH (server recomputes); refetch the chart/card.
    qc.invalidateQueries({ queryKey: ["battery"] });
    qc.invalidateQueries({ queryKey: ["battery-history"] });
  };
}

export function useCreateSession() {
  const invalidate = useInvalidate();
  return useMutation({ mutationFn: (input: ChargingInput) => createChargingSession(input), onSuccess: invalidate });
}

export function useUpdateSession() {
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ChargingInput }) => updateChargingSession(id, input),
    onSuccess: invalidate,
  });
}

export function useDeleteSession() {
  const invalidate = useInvalidate();
  return useMutation({ mutationFn: (id: string) => deleteChargingSession(id), onSuccess: invalidate });
}
