import { useQuery } from "@tanstack/react-query";
import { getDashboard, getBattery, getBatteryHistory } from "./api";

// Keyed ["dashboard"] — charging mutations already invalidate this prefix so the
// stats refetch after a session is added/edited/removed.
export function useDashboard() {
  return useQuery({ queryKey: ["dashboard"], queryFn: getDashboard });
}

export function useBattery(carId?: string) {
  return useQuery({
    queryKey: ["battery", carId],
    queryFn: () => getBattery(carId!),
    enabled: !!carId,
  });
}

export function useBatteryHistory(carId?: string) {
  return useQuery({
    queryKey: ["battery-history", carId],
    queryFn: () => getBatteryHistory(carId!),
    enabled: !!carId,
  });
}
