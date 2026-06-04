import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, useNavigate, useLocation } from "react-router-dom";
import { I18nextProvider } from "react-i18next";
import { LanguageProvider } from "./contexts/LanguageContext";
import { ThemeProvider } from "./contexts/ThemeContext";
import { motion } from "framer-motion";
import { useEffect, useRef } from "react";
import i18n from "./i18n/config";
import Index from "./pages/Index";
import Auth from "./pages/Auth";
import VerifyEmail from "./pages/VerifyEmail";
import Cars from "./pages/Cars";
import Charging from "./pages/Charging";
import Map from "./pages/Map";
import Settings from "./pages/Settings";
import AdminStations from "./pages/AdminStations";
import NotFound from "./pages/NotFound";
import { AdminRoute } from "./components/AdminRoute";
import { BottomNav } from "./components/BottomNav";

const queryClient = new QueryClient();

const SwipeableRoutes = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const startX = useRef(0);

  const routes = ["/", "/cars", "/charging", "/map", "/settings"];
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
          <Route path="/charging" element={<Charging />} />
          <Route path="/map" element={<Map />} />
          <Route path="/settings" element={<Settings />} />
          <Route
            path="/admin/stations"
            element={
              <AdminRoute>
                <AdminStations />
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
  </ThemeProvider>
);

export default App;
