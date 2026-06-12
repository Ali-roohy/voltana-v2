import { lazy, Suspense, useEffect, useRef } from "react";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "./lib/query-client";
import { BrowserRouter, Routes, Route, useNavigate, useLocation } from "react-router-dom";
import { I18nextProvider } from "react-i18next";
import { LanguageProvider } from "./contexts/LanguageContext";
import { ThemeProvider } from "./contexts/ThemeContext";
import { FontProvider } from "./contexts/FontContext";
import { motion } from "framer-motion";
import i18n from "./i18n/config";
import Index from "./pages/Index";
import Auth from "./pages/Auth";
import VerifyEmail from "./pages/VerifyEmail";
import Cars from "./pages/Cars";
import Charging from "./pages/Charging";
import Settings from "./pages/Settings";
import NotFound from "./pages/NotFound";
import { AdminRoute } from "./components/AdminRoute";
import { BottomNav } from "./components/BottomNav";

// Leaflet pages are lazy-loaded so the Leaflet bundle never enters the main chunk.
// Named MapPage (not Map) to avoid shadowing the native JS Map global — Rollup
// renames the native Map to an alias when a module-level variable called "Map"
// is in scope, which breaks every `new Map()` call elsewhere in the bundle.
const MapPage = lazy(() => import("./pages/Map"));
const AdminStations = lazy(() => import("./pages/AdminStations"));
const AdminUsers = lazy(() => import("./pages/AdminUsers"));
const CatalogPage = lazy(() => import("./features/catalog/CatalogPage"));

const MapFallback = () => (
  <div className="flex items-center justify-center min-h-screen bg-background text-muted-foreground text-sm">
    در حال بارگذاری نقشه…
  </div>
);

const AdminFallback = () => (
  <div className="flex items-center justify-center min-h-screen bg-background text-muted-foreground text-sm">
    در حال بارگذاری…
  </div>
);

const SwipeableRoutes = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const startX = useRef(0);

  const routes = ["/", "/cars", "/catalog", "/charging", "/map", "/settings"];
  const currentIndex = routes.indexOf(location.pathname);

  const handleDragEnd = (_: any, info: { offset: { x: number } }) => {
    const swipeThreshold = 100;
    if (info.offset.x > swipeThreshold && currentIndex > 0) {
      navigate(routes[currentIndex - 1]);
    } else if (info.offset.x < -swipeThreshold && currentIndex < routes.length - 1) {
      navigate(routes[currentIndex + 1]);
    }
  };

  return (
    <>
      <motion.div
        drag="x"
        dragConstraints={{ left: 0, right: 0 }}
        dragElastic={0.2}
        onDragEnd={handleDragEnd}
        className="min-h-screen pb-16 md:pb-0"
      >
        <Routes>
          <Route path="/" element={<Index />} />
          <Route path="/auth" element={<Auth />} />
          <Route path="/verify-email" element={<VerifyEmail />} />
          <Route path="/cars" element={<Cars />} />
          <Route
            path="/catalog"
            element={
              <Suspense fallback={<AdminFallback />}>
                <CatalogPage />
              </Suspense>
            }
          />
          <Route path="/charging" element={<Charging />} />
          <Route path="/map" element={<Suspense fallback={<MapFallback />}><MapPage /></Suspense>} />
          <Route path="/settings" element={<Settings />} />
          <Route
            path="/admin/stations"
            element={
              <AdminRoute>
                <Suspense fallback={<AdminFallback />}>
                  <AdminStations />
                </Suspense>
              </AdminRoute>
            }
          />
          <Route
            path="/admin/users"
            element={
              <AdminRoute>
                <Suspense fallback={<AdminFallback />}>
                  <AdminUsers />
                </Suspense>
              </AdminRoute>
            }
          />
          {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
          <Route path="*" element={<NotFound />} />
        </Routes>
      </motion.div>
      <BottomNav />
    </>
  );
};

const App = () => (
  <ThemeProvider>
    <FontProvider>
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <LanguageProvider>
          <TooltipProvider>
            <Sonner />
            <BrowserRouter>
              <SwipeableRoutes />
            </BrowserRouter>
          </TooltipProvider>
        </LanguageProvider>
      </I18nextProvider>
    </QueryClientProvider>
    </FontProvider>
  </ThemeProvider>
);

export default App;
