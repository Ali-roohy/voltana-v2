import { useQuery } from "@tanstack/react-query";
import { getCatalog } from "./api";

// The catalog only changes via migrations (server caches it for 1 h), so a
// long staleTime avoids refetching on every page visit.
export function useCatalog() {
  return useQuery({
    queryKey: ["catalog"],
    queryFn: getCatalog,
    staleTime: 30 * 60 * 1000,
  });
}
