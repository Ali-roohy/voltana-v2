import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { FontId, applyFont, getSavedFontId } from '@/lib/fonts';

interface FontContextValue {
  fontId: FontId;
  setFont: (id: FontId) => void;
}

const FontContext = createContext<FontContextValue | undefined>(undefined);

export function FontProvider({ children }: { children: ReactNode }) {
  const [fontId, setFontId] = useState<FontId>('vazirmatn');

  useEffect(() => {
    const saved = getSavedFontId();
    applyFont(saved);
    setFontId(saved);
  }, []);

  const setFont = (id: FontId) => {
    applyFont(id);
    setFontId(id);
  };

  return (
    <FontContext.Provider value={{ fontId, setFont }}>
      {children}
    </FontContext.Provider>
  );
}

export function useAppFont() {
  const ctx = useContext(FontContext);
  if (!ctx) throw new Error('useAppFont must be used within FontProvider');
  return ctx;
}
