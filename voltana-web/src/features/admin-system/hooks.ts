import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getSystemSettings, updateSystemSettings, type SystemSettings } from "./api";

export const SYSTEM_SETTINGS_KEY = ["admin", "system-settings"] as const;

export function useSystemSettings() {
  return useQuery({
    queryKey: SYSTEM_SETTINGS_KEY,
    queryFn: () => getSystemSettings(),
    staleTime: 30_000,
  });
}

export function useUpdateSystemSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (patch: Partial<SystemSettings>) => updateSystemSettings(patch),
    onSuccess: (updated) => {
      qc.setQueryData(SYSTEM_SETTINGS_KEY, updated);
    },
  });
}
