import type { CurrentUser } from "./api/types";

export function hasPermission(user: CurrentUser | null, permission: string): boolean {
  return Boolean(user?.permissions.includes(permission));
}
