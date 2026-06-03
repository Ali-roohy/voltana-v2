import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useLanguage } from "@/contexts/LanguageContext";

interface ChargingChartProps {
  startSoc: number;
  endSoc: number;
  energyKwh: number;
  durationMinutes: number;
  batteryCapacity: number;
}

export function ChargingChart({
  startSoc,
  endSoc,
  energyKwh,
  durationMinutes,
  batteryCapacity
}: ChargingChartProps) {
  const { language } = useLanguage();
  const isRTL = language === 'fa';

  // Calculate charging power (kW)
  const avgPowerKw = (energyKwh / (durationMinutes / 60)).toFixed(2);
  
  // Calculate actual SOC change based on energy
  const socChange = ((energyKwh / batteryCapacity) * 100).toFixed(1);
  const calculatedEndSoc = (startSoc + parseFloat(socChange)).toFixed(1);

  const texts = {
    title: isRTL ? 'تحلیل جلسه شارژ' : 'Charging Session Analysis',
    socProgress: isRTL ? 'پیشرفت شارژ باتری' : 'Battery Charge Progress',
    energyReceived: isRTL ? 'انرژی دریافتی' : 'Energy Received',
    calculatedSoc: isRTL ? 'SOC محاسبه شده' : 'Calculated SOC',
  };

  // Calculate the height of the charging bar based on power (normalized to 0-100kW range)
  const barHeight = Math.min((parseFloat(avgPowerKw) / 100) * 100, 100);

  return (
    <Card className="mt-6 overflow-hidden">
      <CardHeader className="bg-gradient-to-r from-primary/10 to-primary/5">
        <CardTitle className="flex items-center gap-2">
          <span>{texts.title}</span>
        </CardTitle>
      </CardHeader>
      <CardContent className="p-6 space-y-6">
        {/* SOC Progress Bar */}
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium">{texts.socProgress}</span>
            <span className="text-sm text-muted-foreground">
              {texts.energyReceived}: {energyKwh} kWh
            </span>
          </div>
          
          <div className="relative h-16 bg-muted/30 rounded-lg overflow-hidden">
            {/* Start SOC */}
            <div
              className="absolute top-0 left-0 h-full bg-muted/50 flex items-center justify-center transition-all duration-500"
              style={{ width: `${startSoc}%` }}
            >
              {startSoc > 15 && (
                <span className="text-xs font-medium">{startSoc}%</span>
              )}
            </div>
            
            {/* Charged amount */}
            <div
              className="absolute top-0 h-full bg-gradient-to-r from-primary to-primary/70 flex items-center justify-center transition-all duration-500 animate-in slide-in-from-left"
              style={{ 
                left: `${startSoc}%`,
                width: `${Math.min(parseFloat(calculatedEndSoc) - startSoc, 100 - startSoc)}%` 
              }}
            >
              {parseFloat(calculatedEndSoc) - startSoc > 10 && (
                <span className="text-xs font-medium text-primary-foreground">
                  +{socChange}%
                </span>
              )}
            </div>

            {/* SOC markers */}
            <div className="absolute top-0 left-0 w-full h-full pointer-events-none">
              {[startSoc, parseFloat(calculatedEndSoc)].map((soc, idx) => (
                <div
                  key={idx}
                  className="absolute top-0 h-full border-r-2 border-foreground/20"
                  style={{ left: `${Math.min(soc, 100)}%` }}
                >
                  <div className={`absolute -top-6 ${idx === 0 ? '-left-4' : '-right-4'} text-xs font-semibold`}>
                    {Math.min(soc, 100).toFixed(0)}%
                  </div>
                </div>
              ))}
            </div>
          </div>
          
          {endSoc !== parseFloat(calculatedEndSoc) && (
            <div className="text-xs text-muted-foreground text-center">
              {texts.calculatedSoc}: {calculatedEndSoc}% 
              {endSoc && ` (${isRTL ? 'ثبت شده' : 'Recorded'}: ${endSoc}%)`}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}