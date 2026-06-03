import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import fa from './locales/fa.json';
import en from './locales/en.json';

i18n
  .use(initReactI18next)
  .init({
    resources: {
      fa: { translation: fa },
      en: { translation: en },
    },
    lng: 'fa',
    fallbackLng: 'fa',
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
