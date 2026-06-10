import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteAdminUser, listAdminUsers, updateAdminUser } from "./api";

export function useAdminUsers(limit = 20, offset = 0) {
  return useQuery({
    queryKey: ["admin", "users", limit, offset],
    queryFn: () => listAdminUsers(limit, offset),
  });
}

export function useUpdateAdminUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      patch,
    }: {
      id: string;
      patch: { is_admin?: boolean; is_email_verified?: boolean };
    }) => updateAdminUser(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
  });
}

export function useDeleteAdminUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteAdminUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
  });
}
