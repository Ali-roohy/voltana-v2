import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getSettings, updateSettings, type SettingsUpdate } from "./api";

export function useSettings() {
  // GET /v1/settings auto-creates a default row on first call.
  return useQuery({ queryKey: ["settings"], queryFn: getSettings });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: SettingsUpdate) => updateSettings(body),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings"] }),
  });
}
