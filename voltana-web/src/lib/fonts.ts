export type FontId = 'vazirmatn' | 'inter' | 'noto-arabic' | 'samim' | 'system';

export interface AppFont {
  id: FontId;
  nameEn: string;
  nameFa: string;
  /** CSS font-family stack applied to document.documentElement. */
  stack: string;
  /** Preview text shown in the selector button. */
  previewFa: string;
  previewEn: string;
}

export const FONTS: AppFont[] = [
  {
    id: 'vazirmatn',
    nameEn: 'Vazirmatn',
    nameFa: 'وزیرمتن',
    stack: "'Vazirmatn', sans-serif",
    previewFa: 'ولتانا',
    previewEn: 'Voltana',
  },
  {
    id: 'inter',
    nameEn: 'Inter',
    nameFa: 'اینتر',
    stack: "'Inter', sans-serif",
    previewFa: 'ولتانا',
    previewEn: 'Voltana',
  },
  {
    id: 'noto-arabic',
    nameEn: 'Noto Arabic',
    nameFa: 'نوتو عربی',
    stack: "'Noto Sans Arabic', 'Noto Sans', sans-serif",
    previewFa: 'ولتانا',
    previewEn: 'Voltana',
  },
  {
    id: 'samim',
    nameEn: 'Samim',
    nameFa: 'صمیم',
    stack: "'Samim', sans-serif",
    previewFa: 'ولتانا',
    previewEn: 'Voltana',
  },
  {
    id: 'system',
    nameEn: 'System',
    nameFa: 'سیستم',
    stack: 'system-ui, -apple-system, sans-serif',
    previewFa: 'ولتانا',
    previewEn: 'Voltana',
  },
];

const STORAGE_KEY = 'voltana:font';

export function applyFont(id: FontId): void {
  const font = FONTS.find((f) => f.id === id) ?? FONTS[0];
  document.documentElement.style.fontFamily = font.stack;
  localStorage.setItem(STORAGE_KEY, id);
}

export function getSavedFontId(): FontId {
  const saved = localStorage.getItem(STORAGE_KEY) as FontId | null;
  return FONTS.some((f) => f.id === saved) ? (saved as FontId) : 'vazirmatn';
}
