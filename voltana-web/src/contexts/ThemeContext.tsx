import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { ThemeId, applyTheme, getSavedThemeId } from '@/lib/themes';

interface ThemeContextValue {
  themeId: ThemeId;
  setTheme: (id: ThemeId) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeId, setThemeId] = useState<ThemeId>('default');

  // Apply the saved theme once on first mount (before paint if possible).
  useEffect(() => {
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
