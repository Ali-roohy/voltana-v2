import { useRef, useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface OTPInput6Props {
  value: string;
  onChange: (val: string) => void;
  /** Called once when all 6 digits are filled. */
  onComplete?: (val: string) => void;
  disabled?: boolean;
  /** When true the slots are replaced by a spinner. */
  loading?: boolean;
}

export function OTPInput6({
  value,
  onChange,
  onComplete,
  disabled = false,
  loading = false,
}: OTPInput6Props) {
  const inputRefs = useRef<Array<HTMLInputElement | null>>(Array(6).fill(null));
  const [focusedIdx, setFocusedIdx] = useState<number | null>(null);
  const isComplete = value.length === 6;

  // Fire onComplete exactly once when all 6 digits are present.
  const prevComplete = useRef(false);
  useEffect(() => {
    if (isComplete && !prevComplete.current && onComplete) {
      onComplete(value);
    }
    prevComplete.current = isComplete;
  }, [isComplete, value, onComplete]);

  // Auto-focus the next empty slot when the component first mounts or resets.
  useEffect(() => {
    if (disabled || loading) return;
    const first = value.length < 6 ? value.length : 5;
    inputRefs.current[first]?.focus();
  }, [disabled, loading]); // only on mount / disabled/loading change

  const focusAt = (idx: number) => {
    inputRefs.current[Math.max(0, Math.min(5, idx))]?.focus();
  };

  const handleChange = (idx: number, e: React.ChangeEvent<HTMLInputElement>) => {
    const char = e.target.value.replace(/[^0-9]/g, "").slice(-1);
    if (!char) return;
    // Replace at position idx; keep all digits before and after.
    const next = (value.slice(0, idx) + char + value.slice(idx + 1)).slice(0, 6);
    onChange(next);
    if (idx < 5) focusAt(idx + 1);
  };

  const handleKeyDown = (idx: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Backspace") {
      e.preventDefault();
      if (value[idx]) {
        onChange(value.slice(0, idx) + value.slice(idx + 1));
      } else if (idx > 0) {
        onChange(value.slice(0, idx - 1) + value.slice(idx));
        focusAt(idx - 1);
      }
    } else if (e.key === "ArrowLeft") {
      focusAt(idx - 1);
    } else if (e.key === "ArrowRight") {
      focusAt(idx + 1);
    }
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text").replace(/[^0-9]/g, "").slice(0, 6);
    onChange(pasted);
    focusAt(Math.min(pasted.length, 5));
  };

  return (
    <div dir="ltr" className="flex items-center justify-center min-h-[3.5rem]">
      <AnimatePresence mode="wait">
        {loading ? (
          <motion.div
            key="loading"
            initial={{ opacity: 0, scale: 0.85 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.85 }}
            transition={{ duration: 0.18 }}
            className="flex items-center gap-2 py-2"
          >
            <Loader2 className="w-5 h-5 animate-spin text-primary" />
            <span className="text-sm text-muted-foreground">در حال تأیید...</span>
          </motion.div>
        ) : (
          <motion.div
            key="slots"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.12 }}
            className="flex gap-2"
          >
            {Array.from({ length: 6 }, (_, idx) => {
              const char = value[idx] ?? "";
              const isFocused = focusedIdx === idx;

              return (
                <motion.div
                  key={idx}
                  // Bounce on focus
                  animate={
                    isFocused
                      ? { scale: 1.12, y: -2 }
                      : { scale: 1, y: 0 }
                  }
                  transition={{ type: "spring", stiffness: 500, damping: 22 }}
                  className="relative"
                >
                  {/* Entry pop — remounts when char changes to trigger animation */}
                  <AnimatePresence mode="popLayout">
                    {char && (
                      <motion.span
                        key={char + idx}
                        initial={{ scale: 0.5, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        exit={{ scale: 0.5, opacity: 0 }}
                        transition={{ type: "spring", stiffness: 600, damping: 20 }}
                        className="absolute inset-0 flex items-center justify-center text-xl font-bold pointer-events-none select-none z-10 text-foreground"
                      >
                        {char}
                      </motion.span>
                    )}
                  </AnimatePresence>

                  {/* Success checkmark overlay */}
                  <AnimatePresence>
                    {isComplete && char && (
                      <motion.div
                        key="check"
                        initial={{ scale: 0, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        exit={{ scale: 0, opacity: 0 }}
                        transition={{ type: "spring", stiffness: 600, damping: 20, delay: idx * 0.04 }}
                        className="absolute inset-0 flex items-center justify-center z-20 rounded-xl bg-green-500/90"
                      >
                        <Check className="w-5 h-5 text-white" strokeWidth={3} />
                      </motion.div>
                    )}
                  </AnimatePresence>

                  <input
                    ref={(el) => { inputRefs.current[idx] = el; }}
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    maxLength={2}
                    value=""              /* visual content drawn by the span overlay */
                    disabled={disabled || loading}
                    onChange={(e) => handleChange(idx, e)}
                    onKeyDown={(e) => handleKeyDown(idx, e)}
                    onPaste={handlePaste}
                    onFocus={() => setFocusedIdx(idx)}
                    onBlur={() => setFocusedIdx(null)}
                    aria-label={`رقم ${idx + 1}`}
                    className={cn(
                      "w-11 h-12 rounded-xl border-2 bg-background text-transparent",
                      "outline-none transition-colors duration-150 cursor-text",
                      "caret-transparent select-none",
                      // border colour: complete → green, focused → primary, filled → primary/60, empty → border
                      isComplete
                        ? "border-green-500 shadow-[0_0_12px_hsl(142_76%_36%/0.45)]"
                        : isFocused
                          ? "border-[hsl(var(--primary))] shadow-[0_0_0_3px_hsl(var(--ring)/0.25)]"
                          : char
                            ? "border-[hsl(var(--primary)/0.55)]"
                            : "border-border",
                      "disabled:opacity-40 disabled:cursor-not-allowed",
                    )}
                  />
                </motion.div>
              );
            })}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
