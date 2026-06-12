import { motion } from "framer-motion";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { cssColorFor } from "@/lib/dynamic-theme";

interface ColorPickerProps {
  colors: string[];
  selected: string | null;
  onSelect: (color: string) => void;
}

// Swatch buttons for a car's exterior colors. Selecting one drives the
// dynamic app theme (see CarDetail / lib/dynamic-theme.ts).
export const ColorPicker = ({ colors, selected, onSelect }: ColorPickerProps) => {
  if (colors.length === 0) {
    return <p className="text-sm text-muted-foreground">رنگی ثبت نشده است</p>;
  }
  return (
    <div className="flex flex-wrap gap-3">
      {colors.map((color) => {
        const isSelected = color === selected;
        const css = cssColorFor(color);
        return (
          <motion.button
            key={color}
            type="button"
            whileTap={{ scale: 0.9 }}
            whileHover={{ scale: 1.05 }}
            onClick={() => onSelect(color)}
            className={cn(
              "flex items-center gap-2 rounded-full border px-3 py-2 text-sm transition-colors",
              isSelected ? "border-primary bg-primary/10 font-semibold" : "border-border bg-background",
            )}
          >
            <span
              className="inline-flex h-6 w-6 items-center justify-center rounded-full border border-black/10 shadow-sm"
              style={{ backgroundColor: css }}
            >
              {isSelected && (
                <Check className="h-4 w-4" style={{ color: css === "#f4f4f5" || css === "#c0c4cc" || css === "#d6c49a" ? "#27272a" : "#ffffff" }} />
              )}
            </span>
            <span>{color}</span>
          </motion.button>
        );
      })}
    </div>
  );
};
