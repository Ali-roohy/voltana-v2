import { useQuery } from "@tanstack/react-query";
import { listStations, getStation } from "./api";

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
