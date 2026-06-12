export type ThemeId =
  | 'default'
  | 'tesla-red'
  | 'leaf-green'
  | 'ocean-blue'
  | 'midnight-black'
  | 'sunset-orange'
  | 'sakura-pink'
  | 'arctic-white';

export interface Theme {
  id: ThemeId;
  nameEn: string;
  nameFa: string;
  /** CSS color value shown in the Settings swatch (pure presentational). */
  swatchPrimary: string;
  swatchAccent: string;
  /** CSS custom-property overrides applied to :root via inline style. */
  vars: Partial<Record<string, string>>;
}

// The full set of vars any theme may override — cleared before every switch.
// Exported so the dynamic car-color theme (lib/dynamic-theme.ts) clears the
// exact same set when taking over / handing back.
export const OVERRIDEABLE_VARS = [
  '--primary',
  '--primary-foreground',
  '--primary-glow',
  '--accent',
  '--accent-foreground',
  '--ring',
  '--gradient-primary',
  '--shadow-soft',
  '--shadow-glow',
] as const;

export const THEMES: Theme[] = [
  {
    id: 'default',
    nameEn: 'Voltana',
    nameFa: 'پیش‌فرض',
    swatchPrimary: '#007FFF',
    swatchAccent: '#00FFD0',
    vars: {}, // no overrides — CSS-sheet values stay
  },
  {
    id: 'tesla-red',
    nameEn: 'Tesla Red',
    nameFa: 'قرمز تسلا',
    swatchPrimary: '#E32020',
    swatchAccent: '#FF6B35',
    vars: {
      '--primary': '0 88% 55%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '15 100% 62%',
      '--accent': '15 100% 62%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '0 88% 55%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(0 88% 55%), hsl(15 100% 62%))',
      '--shadow-soft': '0 4px 20px -4px hsl(0 88% 55% / 0.25)',
      '--shadow-glow': '0 0 40px hsl(15 100% 62% / 0.4)',
    },
  },
  {
    id: 'leaf-green',
    nameEn: 'Leaf Green',
    nameFa: 'سبز برگ',
    swatchPrimary: '#1DB356',
    swatchAccent: '#00E5AA',
    vars: {
      '--primary': '142 75% 42%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '155 100% 45%',
      '--accent': '155 100% 45%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '142 75% 42%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(142 75% 42%), hsl(155 100% 45%))',
      '--shadow-soft': '0 4px 20px -4px hsl(142 75% 42% / 0.25)',
      '--shadow-glow': '0 0 40px hsl(155 100% 45% / 0.4)',
    },
  },
  {
    id: 'ocean-blue',
    nameEn: 'Ocean Blue',
    nameFa: 'آبی اقیانوس',
    swatchPrimary: '#2E5FE0',
    swatchAccent: '#00C8FF',
    vars: {
      '--primary': '222 90% 55%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '200 100% 52%',
      '--accent': '200 100% 52%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '222 90% 55%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(222 90% 55%), hsl(200 100% 52%))',
      '--shadow-soft': '0 4px 20px -4px hsl(222 90% 55% / 0.25)',
      '--shadow-glow': '0 0 40px hsl(200 100% 52% / 0.4)',
    },
  },
  {
    id: 'midnight-black',
    nameEn: 'Midnight',
    nameFa: 'نیمه‌شب',
    swatchPrimary: '#8855E0',
    swatchAccent: '#CC55FF',
    vars: {
      '--primary': '270 68% 60%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '290 90% 68%',
      '--accent': '290 90% 68%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '270 68% 60%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(270 68% 60%), hsl(290 90% 68%))',
      '--shadow-soft': '0 4px 20px -4px hsl(270 68% 60% / 0.3)',
      '--shadow-glow': '0 0 40px hsl(290 90% 68% / 0.45)',
    },
  },
  {
    id: 'sunset-orange',
    nameEn: 'Sunset',
    nameFa: 'غروب',
    swatchPrimary: '#F07020',
    swatchAccent: '#FFD040',
    vars: {
      '--primary': '25 95% 53%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '45 100% 55%',
      '--accent': '45 100% 55%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '25 95% 53%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(25 95% 53%), hsl(45 100% 55%))',
      '--shadow-soft': '0 4px 20px -4px hsl(25 95% 53% / 0.25)',
      '--shadow-glow': '0 0 40px hsl(45 100% 55% / 0.4)',
    },
  },
  {
    id: 'sakura-pink',
    nameEn: 'Sakura',
    nameFa: 'شکوفه',
    swatchPrimary: '#E83570',
    swatchAccent: '#FF70B0',
    vars: {
      '--primary': '340 80% 60%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '355 90% 68%',
      '--accent': '355 90% 68%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '340 80% 60%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(340 80% 60%), hsl(355 90% 68%))',
      '--shadow-soft': '0 4px 20px -4px hsl(340 80% 60% / 0.25)',
      '--shadow-glow': '0 0 40px hsl(355 90% 68% / 0.4)',
    },
  },
  {
    id: 'arctic-white',
    nameEn: 'Arctic',
    nameFa: 'قطبی',
    swatchPrimary: '#00C8E0',
    swatchAccent: '#80FFEE',
    vars: {
      '--primary': '188 100% 44%',
      '--primary-foreground': '0 0% 100%',
      '--primary-glow': '175 100% 48%',
      '--accent': '175 100% 48%',
      '--accent-foreground': '0 0% 100%',
      '--ring': '188 100% 44%',
      '--gradient-primary': 'linear-gradient(135deg, hsl(188 100% 44%), hsl(175 100% 48%))',
      '--shadow-soft': '0 4px 20px -4px hsl(188 100% 44% / 0.2)',
      '--shadow-glow': '0 0 40px hsl(175 100% 48% / 0.35)',
    },
  },
];

const STORAGE_KEY = 'voltana:theme';

export function applyTheme(id: ThemeId): void {
  const theme = THEMES.find((t) => t.id === id) ?? THEMES[0];
  // Clear every possible override first so switching from one theme to another
  // doesn't leave stale vars from the previous theme.
  OVERRIDEABLE_VARS.forEach((v) => document.documentElement.style.removeProperty(v));
  // Apply the new theme's overrides as inline styles (highest specificity).
  Object.entries(theme.vars).forEach(([k, v]) =>
    document.documentElement.style.setProperty(k, v),
  );
  localStorage.setItem(STORAGE_KEY, id);
}

export function getSavedThemeId(): ThemeId {
  const saved = localStorage.getItem(STORAGE_KEY) as ThemeId | null;
  return THEMES.some((t) => t.id === saved) ? (saved as ThemeId) : 'default';
}
