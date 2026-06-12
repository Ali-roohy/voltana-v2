import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { authStore } from "@/lib/auth-store";
import { deleteAccount } from "./api";

const CONFIRM_WORD = "حذف";

// «حذف حساب کاربری» (TASK-0037 FEAT-5). Type-to-confirm; the backend refuses
// to delete the last admin regardless of what the UI shows.
export function DeleteAccountCard() {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteAccount();
      // authStore.clear() triggers the query-cache wipe (lib/query-client.ts).
      authStore.clear();
      toast.success("حساب کاربری شما حذف شد");
      navigate("/auth");
    } catch (err) {
      if (err instanceof ApiError && err.code === "LAST_ADMIN") {
        toast.error("حساب ادمین اصلی قابل حذف نیست");
      } else {
        toast.error("خطا در حذف حساب");
      }
      setDeleting(false);
      setOpen(false);
      setConfirmText("");
    }
  };

  return (
    <Card className="border-destructive/40">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-destructive">
          <Trash2 className="w-5 h-5" />
          حذف حساب کاربری
        </CardTitle>
        <CardDescription>
          همه داده‌های شما (خودروها، جلسات شارژ، تنظیمات، تاریخچه باتری) برای همیشه حذف می‌شود.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Button variant="destructive" className="w-full" onClick={() => setOpen(true)}>
          حذف حساب
        </Button>
      </CardContent>

      <AlertDialog
        open={open}
        onOpenChange={(o) => {
          setOpen(o);
          if (!o) setConfirmText("");
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>حذف دائمی حساب</AlertDialogTitle>
            <AlertDialogDescription>
              این عمل قابل بازگشت نیست. برای تأیید، کلمه «{CONFIRM_WORD}» را تایپ کنید:
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Input
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder={CONFIRM_WORD}
            autoFocus
          />
          <AlertDialogFooter>
            <AlertDialogCancel>انصراف</AlertDialogCancel>
            <Button
              variant="destructive"
              disabled={confirmText.trim() !== CONFIRM_WORD || deleting}
              onClick={handleDelete}
            >
              {deleting && <Loader2 className="w-4 h-4 ml-2 animate-spin" />}
              حذف برای همیشه
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}
