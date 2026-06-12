import { useState } from "react";
import { toast } from "sonner";
import { Loader2, ChevronLeft, ChevronRight, Shield, ShieldOff, Trash2, CheckCircle } from "lucide-react";

import { Header } from "@/components/Header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { ApiError } from "@/lib/api";
import { useMe } from "@/features/auth/hooks";
import { useAdminUsers, useUpdateAdminUser, useDeleteAdminUser } from "@/features/admin-users/hooks";
import type { UserSummary } from "@/features/admin-users/api";

const LIMIT = 20;

export default function AdminUsers() {
  const [offset, setOffset] = useState(0);
  const [toDelete, setToDelete] = useState<UserSummary | null>(null);

  const { data: me } = useMe();
  const { data, isLoading, isError } = useAdminUsers(LIMIT, offset);
  const updateUser = useUpdateAdminUser();
  const deleteUser = useDeleteAdminUser();

  const users = data?.items ?? [];
  const total = data?.total ?? 0;
  const pageStart = offset + 1;
  const pageEnd = Math.min(offset + LIMIT, total);
  const hasPrev = offset > 0;
  const hasNext = offset + LIMIT < total;

  const handleToggleAdmin = (u: UserSummary) => {
    updateUser.mutate(
      { id: u.id, patch: { is_admin: !u.is_admin } },
      {
        onSuccess: () =>
          toast.success(u.is_admin ? "دسترسی ادمین لغو شد" : "دسترسی ادمین اعطا شد"),
        onError: (e) =>
          toast.error(e instanceof ApiError ? e.message : "خطا در به‌روزرسانی"),
      },
    );
  };

  const handleVerify = (u: UserSummary) => {
    updateUser.mutate(
      { id: u.id, patch: { is_email_verified: true } },
      {
        onSuccess: () => toast.success("ایمیل تأیید شد"),
        onError: (e) =>
          toast.error(e instanceof ApiError ? e.message : "خطا در به‌روزرسانی"),
      },
    );
  };

  const handleDeleteConfirm = () => {
    if (!toDelete) return;
    deleteUser.mutate(toDelete.id, {
      onSuccess: () => {
        toast.success("کاربر حذف شد");
        setToDelete(null);
        if (offset > 0 && users.length === 1) setOffset(offset - LIMIT);
      },
      onError: (e) => {
        toast.error(e instanceof ApiError ? e.message : "خطا در حذف کاربر");
        setToDelete(null);
      },
    });
  };

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleDateString("fa-IR", { year: "numeric", month: "short", day: "numeric" });

  return (
    <div className="min-h-screen app-page-bg" dir="rtl">
      <Header />
      <div className="container max-w-5xl mx-auto px-4 py-6 space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-lg font-bold">مدیریت کاربران</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading && (
              <div className="flex justify-center py-12">
                <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
              </div>
            )}

            {isError && (
              <p className="text-center text-destructive py-8">خطا در بارگذاری کاربران</p>
            )}

            {!isLoading && !isError && users.length === 0 && (
              <p className="text-center text-muted-foreground py-8">هیچ کاربری ثبت‌نام نکرده</p>
            )}

            {!isLoading && !isError && users.length > 0 && (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="text-right">نام</TableHead>
                        <TableHead className="text-right">ایمیل</TableHead>
                        <TableHead className="text-right">تلفن</TableHead>
                        <TableHead className="text-center">ادمین</TableHead>
                        <TableHead className="text-center">بات</TableHead>
                        <TableHead className="text-right">تاریخ عضویت</TableHead>
                        <TableHead className="text-center">عملیات</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {users.map((u) => {
                        const isSelf = u.id === me?.id;
                        return (
                          <TableRow key={u.id} className={isSelf ? "bg-muted/30" : ""}>
                            {/* نام */}
                            <TableCell className="text-sm max-w-[140px]">
                              <div className="truncate">
                                {u.full_name ?? <span className="text-muted-foreground">—</span>}
                              </div>
                              {isSelf && (
                                <span className="text-[10px] text-muted-foreground">(شما)</span>
                              )}
                            </TableCell>

                            {/* ایمیل */}
                            <TableCell className="max-w-[180px]">
                              {u.email ? (
                                <div className="space-y-0.5">
                                  <div className="font-mono text-xs truncate">{u.email}</div>
                                  {u.is_email_verified ? (
                                    <Badge variant="outline" className="text-[10px] h-4 px-1 text-green-600 border-green-600">تأیید شده</Badge>
                                  ) : (
                                    <Badge variant="outline" className="text-[10px] h-4 px-1 text-muted-foreground">تأیید نشده</Badge>
                                  )}
                                </div>
                              ) : (
                                <span className="text-muted-foreground text-xs">—</span>
                              )}
                            </TableCell>

                            {/* تلفن */}
                            <TableCell className="font-mono text-xs">
                              {u.phone ?? <span className="text-muted-foreground">—</span>}
                            </TableCell>

                            <TableCell className="text-center">
                              {u.is_admin ? (
                                <Badge variant="default" className="text-xs">ادمین</Badge>
                              ) : (
                                <Badge variant="secondary" className="text-xs">کاربر</Badge>
                              )}
                            </TableCell>

                            <TableCell className="text-center">
                              <div className="flex gap-1 justify-center flex-wrap">
                                {u.bale_linked && <Badge variant="outline" className="text-xs">بله</Badge>}
                                {u.telegram_linked && <Badge variant="outline" className="text-xs">تلگرام</Badge>}
                                {!u.bale_linked && !u.telegram_linked && (
                                  <span className="text-xs text-muted-foreground">—</span>
                                )}
                              </div>
                            </TableCell>

                            <TableCell className="text-xs text-muted-foreground">
                              {formatDate(u.created_at)}
                            </TableCell>

                            <TableCell>
                              <div className="flex gap-1 justify-center flex-wrap">
                                <Button
                                  size="sm"
                                  variant="outline"
                                  className="h-7 px-2 text-xs"
                                  disabled={isSelf || updateUser.isPending}
                                  onClick={() => handleToggleAdmin(u)}
                                  title={u.is_admin ? "لغو ادمین" : "اعطای ادمین"}
                                >
                                  {u.is_admin ? (
                                    <ShieldOff className="h-3 w-3" />
                                  ) : (
                                    <Shield className="h-3 w-3" />
                                  )}
                                </Button>

                                {!u.is_email_verified && (
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="h-7 px-2 text-xs"
                                    disabled={isSelf || updateUser.isPending}
                                    onClick={() => handleVerify(u)}
                                    title="تأیید ایمیل"
                                  >
                                    <CheckCircle className="h-3 w-3" />
                                  </Button>
                                )}

                                <Button
                                  size="sm"
                                  variant="outline"
                                  className="h-7 px-2 text-xs text-destructive hover:text-destructive"
                                  disabled={isSelf || deleteUser.isPending}
                                  onClick={() => setToDelete(u)}
                                  title="حذف کاربر"
                                >
                                  <Trash2 className="h-3 w-3" />
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>

                {/* Pagination */}
                {total > LIMIT && (
                  <div className="flex items-center justify-between mt-4 text-sm text-muted-foreground">
                    <span>
                      نمایش {pageStart}–{pageEnd} از {total}
                    </span>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!hasPrev}
                        onClick={() => setOffset(offset - LIMIT)}
                      >
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!hasNext}
                        onClick={() => setOffset(offset + LIMIT)}
                      >
                        <ChevronLeft className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Delete confirm dialog */}
      <AlertDialog open={!!toDelete} onOpenChange={(open) => !open && setToDelete(null)}>
        <AlertDialogContent dir="rtl">
          <AlertDialogHeader>
            <AlertDialogTitle>حذف کاربر</AlertDialogTitle>
            <AlertDialogDescription>
              آیا مطمئن هستید که می‌خواهید کاربر{" "}
              <span className="font-semibold">{toDelete?.full_name ?? toDelete?.email ?? toDelete?.phone ?? toDelete?.id}</span>{" "}
              را حذف کنید؟ تمام داده‌های این کاربر (خودروها، جلسات شارژ و …) به صورت دائمی حذف می‌شوند.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className="flex-row-reverse gap-2">
            <AlertDialogCancel>لغو</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={handleDeleteConfirm}
            >
              حذف
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
