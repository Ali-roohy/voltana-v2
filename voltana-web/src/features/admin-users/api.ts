import { api } from "@/lib/api";

export interface UserSummary {
  id: string;
  email: string | null;
  phone: string | null;
  is_admin: boolean;
  is_email_verified: boolean;
  bale_linked: boolean;
  telegram_linked: boolean;
  created_at: string;
}

export interface UsersPage {
  items: UserSummary[];
  total: number;
  limit: number;
  offset: number;
}

export const listAdminUsers = (limit = 20, offset = 0) =>
  api.get<UsersPage>(`/v1/admin/users?limit=${limit}&offset=${offset}`);

export const getAdminUser = (id: string) =>
  api.get<UserSummary>(`/v1/admin/users/${id}`);

export const updateAdminUser = (
  id: string,
  patch: { is_admin?: boolean; is_email_verified?: boolean },
) => api.put<UserSummary>(`/v1/admin/users/${id}`, patch);

export const deleteAdminUser = (id: string) =>
  api.del<void>(`/v1/admin/users/${id}`);
