import { Home, Car, BookOpen, Zap, Map, Settings } from "lucide-react";
import { NavLink } from "./NavLink";
import { useTranslation } from "react-i18next";

export const BottomNav = () => {
  const { t } = useTranslation();

  const navItems = [
    { to: "/", icon: Home, label: t("nav.dashboard") },
    { to: "/cars", icon: Car, label: t("nav.cars") },
    { to: "/catalog", icon: BookOpen, label: t("nav.catalog") },
    { to: "/charging", icon: Zap, label: t("nav.charging") },
    { to: "/map", icon: Map, label: t("nav.map") },
    { to: "/settings", icon: Settings, label: t("nav.settings") },
  ];

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-background/95 backdrop-blur-sm border-t border-border z-50 md:hidden">
      <div className="flex justify-around items-center h-16 px-2">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className="flex flex-col items-center justify-center gap-1 py-2 px-3 rounded-lg transition-colors text-muted-foreground"
            activeClassName="text-primary bg-primary/10"
          >
            <item.icon className="h-5 w-5" />
            <span className="text-[10px] font-medium">{item.label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  );
};
