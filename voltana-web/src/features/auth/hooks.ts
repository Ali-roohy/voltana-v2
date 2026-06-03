import { useQuery } from "@tanstack/react-query";
import { getMe } from "./api";

// useMe fetches the authenticated user's identity (incl. is_admin) from GET /v1/me.
// `enabled` lets callers defer the query until the auth session has settled.
export function useMe(enabled = true) {
  return useQuery({
    queryKey: ["me"],
    queryFn: getMe,
    enabled,
    staleTime: 5 * 60 * 1000, // identity rarely changes within a session
    retry: false, // a 401/403 shouldn't be retried — surface it to the guard
  });
}
