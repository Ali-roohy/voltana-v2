import { useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { DatabaseBackup, Download, Upload, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { ApiError } from "@/lib/api";
import { exportAccountData, importAccountData } from "./api";

// «پشتیبان‌گیری و بازگردانی» (TASK-0037 FEAT-4). Import REPLACES the user's
// own data — guarded by an explicit destructive-action dialog.
export function BackupRestoreCard() {
  const queryClient = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const [exporting, setExporting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [pendingBackup, setPendingBackup] = useState<unknown | null>(null);

  const handleExport = async () => {
    setExporting(true);
    try {
      const data = await exportAccountData();
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `voltana-backup-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success("فایل پشتیبان دانلود شد");
    } catch {
      toast.error("خطا در تهیه پشتیبان");
    } finally {
      setExporting(false);
    }
  };

  const handleFilePicked = async (file: File) => {
    try {
      const text = await file.text();
      setPendingBackup(JSON.parse(text));
    } catch {
      toast.error("فایل انتخاب‌شده JSON معتبر نیست");
    }
  };

  const handleImportConfirmed = async () => {
    if (pendingBackup == null) return;
    setImporting(true);
    try {
      const res = await importAccountData(pendingBackup);
      // Imported rows replace everything the cache knows about.
      queryClient.invalidateQueries();
      toast.success(
        `بازگردانی انجام شد — ${res.imported.cars} خودرو، ${res.imported.sessions} جلسه شارژ`,
      );
    } catch (err) {
      toast.error(err instanceof ApiError ? `خطا: ${err.message}` : "خطا در بازگردانی");
    } finally {
      setImporting(false);
      setPendingBackup(null);
      if (fileRef.current) fileRef.current.value = "";
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <DatabaseBackup className="w-5 h-5" />
          پشتیبان‌گیری و بازگردانی
        </CardTitle>
        <CardDescription>
          خروجی JSON از خودروها، جلسات شارژ، تنظیمات و تاریخچه باتری شما
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <Button variant="outline" className="w-full gap-2" disabled={exporting} onClick={handleExport}>
          {exporting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
          دانلود فایل پشتیبان
        </Button>

        <input
          ref={fileRef}
          type="file"
          accept="application/json,.json"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleFilePicked(f);
          }}
        />
        <Button
          variant="outline"
          className="w-full gap-2"
          disabled={importing}
          onClick={() => fileRef.current?.click()}
        >
          {importing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />}
          بازگردانی از فایل
        </Button>
        <p className="text-xs text-muted-foreground">
          بازگردانی، داده‌های فعلی حساب شما را با محتوای فایل جایگزین می‌کند.
        </p>
      </CardContent>

      <AlertDialog open={pendingBackup != null} onOpenChange={(open) => !open && setPendingBackup(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>جایگزینی داده‌های حساب؟</AlertDialogTitle>
            <AlertDialogDescription>
              همه خودروها، جلسات شارژ، تنظیمات و تاریخچه باتری فعلی شما حذف و با محتوای فایل
              پشتیبان جایگزین می‌شود. این عمل قابل بازگشت نیست.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>انصراف</AlertDialogCancel>
            <AlertDialogAction onClick={handleImportConfirmed}>
              جایگزین کن
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}
