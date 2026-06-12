import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { ThemeId, applyTheme, getSavedThemeId } from '@/lib/themes';
import { applyDynamicTheme, getSavedDynamicColor } from '@/lib/dynamic-theme';

interface ThemeContextValue {
  themeId: ThemeId;
  setTheme: (id: ThemeId) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeId, setThemeId] = useState<ThemeId>('default');

  // Apply the saved theme once on first mount (before paint if possible).
  // A persisted dynamic car-color theme (TASK-0033) wins over the preset id;
  // themeId stays 'default' so Settings shows no preset as selected-ish state.
  useEffect(() => {
    const dynamicColor = getSavedDynamicColor();
    if (dynamicColor !== null) {
      applyDynamicTheme(dynamicColor);
      return;
    }
    const saved = getSavedThemeId();
    applyTheme(saved);
    setThemeId(saved);
  }, []);

  const setTheme = (id: ThemeId) => {
    applyTheme(id);
    setThemeId(id);
  };

  return (
    <ThemeContext.Provider value={{ themeId, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useAppTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useAppTheme must be used within ThemeProvider');
  return ctx;
}
