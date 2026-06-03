import { useLanguage } from "@/contexts/LanguageContext";
import { AlertTriangle, Battery, BatteryWarning } from "lucide-react";
import { motion } from "framer-motion";
import { cn } from "@/lib/utils";
import { useEffect } from "react";
import { toast } from "sonner";
interface SOCAnalysisProps {
  startSoc: number | null;
  endSoc: number | null;
}
export const SOCAnalysis = ({
  startSoc,
  endSoc
}: SOCAnalysisProps) => {
  const {
    language
  } = useLanguage();
  useEffect(() => {
    if (startSoc !== null && startSoc < 20) {
      toast.error(
        language === "fa"
          ? "⚠️ شارژ باتری به زیر ۲۰٪ رسیده است. لطفاً در اسرع وقت شارژ کنید."
          : "⚠️ Battery charge is below 20%. Please charge as soon as possible.",
      );
    }
  }, [startSoc, language]);
  if (!startSoc && startSoc !== 0 || !endSoc && endSoc !== 0) {
    return <div className="text-sm text-muted-foreground">
        {language === 'fa' ? 'اطلاعات SOC ناقص است' : 'SOC data incomplete'}
      </div>;
  }
  const min = Math.min(startSoc, endSoc);
  const max = Math.max(startSoc, endSoc);

  // Check for warnings
  const warnings = [];
  if (min < 20) {
    warnings.push(language === 'fa' ? '⚠️ شارژ به زیر ۲۰٪ رسیده - خطر برای باتری!' : '⚠️ Charge below 20% - Battery danger!');
  }
  if (max > 85) {
    warnings.push(language === 'fa' ? '⚠️ شارژ به بالای ۸۵٪ رسیده - توصیه نمی‌شود!' : '⚠️ Charge above 85% - Not recommended!');
  }
  const getIcon = () => {
    if (min < 20) return BatteryWarning;
    return Battery;
  };
  const Icon = getIcon();
  return <motion.div className="space-y-4" initial={{
    opacity: 0,
    y: 20
  }} animate={{
    opacity: 1,
    y: 0
  }} transition={{
    duration: 0.5
  }}>
      {/* Header with icon and percentage */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <motion.div animate={{
          scale: [1, 1.1, 1],
          rotate: [0, 5, -5, 0]
        }} transition={{
          duration: 2,
          repeat: Infinity,
          repeatDelay: 1
        }}>
            <Icon className={cn("h-8 w-8", min < 20 ? "text-red-500" : "text-primary")} />
          </motion.div>
          <div>
            <div className="text-3xl font-bold flex items-baseline gap-2">
              {/* bug #6 fix: show start → end (was reversed) */}
              <motion.span key={startSoc} initial={{
              opacity: 0,
              scale: 0.8
            }} animate={{
              opacity: 1,
              scale: 1
            }} className="bg-lime-50 text-lime-900 text-sm">
                {startSoc}%
              </motion.span>
              <span className="text-lg bg-lime-50 text-lime-900">→</span>
              <motion.span key={endSoc} initial={{
              opacity: 0,
              scale: 0.8
            }} animate={{
              opacity: 1,
              scale: 1
            }} className="bg-lime-50 text-lime-900 text-sm">
                {endSoc}%
              </motion.span>
            </div>
            <p className="text-xs text-center bg-lime-50 text-lime-700">
              {language === 'fa' ? 'محدوده شارژ' : 'Charge Range'}
            </p>
          </div>
        </div>
        <motion.div initial={{
        opacity: 0
      }} animate={{
        opacity: 1
      }} transition={{
        delay: 0.3
      }} className="text-3xl font-bold text-blue-900 bg-blue-50">
          {Math.abs(endSoc - startSoc)}%
        </motion.div>
      </div>
      
      {/* Legend */}
      <div className="flex gap-3 text-[10px] text-muted-foreground justify-center">
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-red-600 border border-red-800" />
          <span>{language === 'fa' ? '< ۲۰٪ خطر' : '< 20% Danger'}</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-gradient-to-r from-primary to-accent border border-primary" />
          <span>{language === 'fa' ? 'محدوده شارژ' : 'Charge Range'}</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-orange-500 border border-orange-700" />
          <span>{language === 'fa' ? '> ۸۵٪ هشدار' : '> 85% Warning'}</span>
        </div>
      </div>
      
      <div className="relative h-12 w-full bg-muted/30 rounded-2xl overflow-hidden border border-border/50 shadow-inner">
        {/* Danger zone: 0-20% */}
        <motion.div className="absolute top-0 left-0 h-full bg-red-600 border-r-2 border-red-800 rounded-l-2xl" style={{
        width: '20%'
      }} initial={{
        opacity: 0
      }} animate={{
        opacity: 1
      }} transition={{
        delay: 0.1
      }} />
        
        {/* Warning zone: 85-100% */}
        <motion.div className="absolute top-0 left-[85%] h-full bg-orange-500 border-l-2 border-orange-700 rounded-r-2xl" style={{
        width: '15%'
      }} initial={{
        opacity: 0
      }} animate={{
        opacity: 1
      }} transition={{
        delay: 0.2
      }} />
        
        {/* Charging range with animation */}
        <motion.div className="absolute top-0 h-full bg-gradient-to-r from-primary via-accent to-primary shadow-lg rounded-2xl" style={{
        left: `${min}%`,
        width: `${max - min}%`,
        backgroundSize: '200% 100%'
      }} initial={{
        scaleX: 0
      }} animate={{
        scaleX: 1,
        backgroundPosition: ['0% 50%', '100% 50%', '0% 50%']
      }} transition={{
        scaleX: {
          duration: 0.8,
          ease: "easeOut"
        },
        backgroundPosition: {
          duration: 3,
          repeat: Infinity,
          ease: "linear"
        }
      }} />
        
        {/* Start SOC marker */}
        
        
        {/* End SOC marker */}
        
      </div>
      
      {/* Zone labels below chart */}
      <div className="relative w-full h-8 flex items-center justify-between px-1">
        {/* Danger zone arrow and label */}
        <motion.div className="absolute left-0 flex flex-col items-center" style={{
        width: '20%'
      }} initial={{
        opacity: 0,
        y: -10
      }} animate={{
        opacity: 1,
        y: 0
      }} transition={{
        delay: 0.3
      }}>
          <div className="flex items-center gap-1 w-full justify-center">
            <div className="text-[10px] font-bold text-red-700 dark:text-red-400">0%</div>
            <div className="flex-1 flex items-center">
              <div className="h-0.5 bg-red-600 flex-1" />
              <div className="w-1.5 h-1.5 bg-red-600 rotate-45 transform translate-x-[-2px]" />
            </div>
            <div className="flex-1 flex items-center">
              <div className="w-1.5 h-1.5 bg-red-600 rotate-45 transform translate-x-[2px]" />
              <div className="h-0.5 bg-red-600 flex-1" />
            </div>
            <div className="text-[10px] font-bold text-red-700 dark:text-red-400">20%</div>
          </div>
          <div className="text-[9px] text-red-600 dark:text-red-400 mt-0.5 font-medium">
            {language === 'fa' ? 'خطر' : 'Danger'}
          </div>
        </motion.div>
        
        {/* Warning zone arrow and label */}
        <motion.div className="absolute right-0 flex flex-col items-center" style={{
        width: '15%'
      }} initial={{
        opacity: 0,
        y: -10
      }} animate={{
        opacity: 1,
        y: 0
      }} transition={{
        delay: 0.4
      }}>
          <div className="flex items-center gap-1 w-full justify-center">
            <div className="text-[10px] font-bold text-orange-700 dark:text-orange-400">100%</div>
            <div className="flex-1 flex items-center">
              <div className="h-0.5 bg-orange-500 flex-1" />
              <div className="w-1.5 h-1.5 bg-orange-500 rotate-45 transform translate-x-[-2px]" />
            </div>
            <div className="flex-1 flex items-center">
              <div className="w-1.5 h-1.5 bg-orange-500 rotate-45 transform translate-x-[2px]" />
              <div className="h-0.5 bg-orange-500 flex-1" />
            </div>
            <div className="text-[10px] font-bold text-orange-700 dark:text-orange-400">85%</div>
          </div>
          <div className="text-[9px] text-orange-600 dark:text-orange-400 mt-0.5 font-medium">
            {language === 'fa' ? 'هشدار' : 'Warning'}
          </div>
        </motion.div>
      </div>
      
      {/* Calibration note */}
      <motion.div className="text-[10px] text-muted-foreground text-center italic" initial={{
      opacity: 0
    }} animate={{
      opacity: 1
    }} transition={{
      delay: 0.6
    }}>
        {language === 'fa' ? '💡 توصیه: ماهی یکبار تا 100% شارژ کنید تا باتری کالیبره شود' : '💡 Tip: Charge to 100% once a month to calibrate the battery'}
      </motion.div>
      
      {/* Warnings with animation */}
      {warnings.length > 0 && <motion.div className="space-y-2" initial={{
      opacity: 0,
      height: 0
    }} animate={{
      opacity: 1,
      height: "auto"
    }} transition={{
      delay: 0.6
    }}>
          {warnings.map((warning, idx) => <motion.div key={idx} className="flex items-center gap-2 text-sm text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-950/30 px-3 py-2 rounded-lg border border-orange-200 dark:border-orange-800 shadow-sm" initial={{
        opacity: 0,
        x: -20
      }} animate={{
        opacity: 1,
        x: 0
      }} transition={{
        delay: 0.7 + idx * 0.1
      }}>
              <AlertTriangle className="w-4 h-4 flex-shrink-0" />
              <span>{warning}</span>
            </motion.div>)}
        </motion.div>}
    </motion.div>;
};