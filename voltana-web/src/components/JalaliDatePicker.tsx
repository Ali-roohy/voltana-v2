import * as React from "react";
import { format as formatJalali, parse as parseJalali } from "date-fns-jalali";
import { format as formatGregorian } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useLanguage } from "@/contexts/LanguageContext";
import { Calendar } from "@/components/ui/calendar";

interface JalaliDatePickerProps {
  date: Date | undefined;
  onDateChange: (date: Date | undefined) => void;
  placeholder?: string;
  className?: string;
}

export function JalaliDatePicker({ 
  date, 
  onDateChange, 
  placeholder = "انتخاب تاریخ",
  className 
}: JalaliDatePickerProps) {
  const { language } = useLanguage();
  const [isOpen, setIsOpen] = React.useState(false);

  const formatDate = (d: Date) => {
    if (language === 'fa') {
      return formatJalali(d, 'yyyy/MM/dd');
    }
    return formatGregorian(d, 'yyyy/MM/dd');
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "w-full justify-start text-right font-normal",
            !date && "text-muted-foreground",
            className
          )}
        >
          <CalendarIcon className="h-4 w-4" />
          {date ? formatDate(date) : <span>{placeholder}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        {language === 'fa' ? (
          <JalaliCalendar
            selected={date}
            onSelect={(newDate) => {
              onDateChange(newDate);
              setIsOpen(false);
            }}
          />
        ) : (
          <Calendar
            mode="single"
            selected={date}
            onSelect={(newDate) => {
              onDateChange(newDate);
              setIsOpen(false);
            }}
            initialFocus
            className="pointer-events-auto"
          />
        )}
      </PopoverContent>
    </Popover>
  );
}

interface JalaliCalendarProps {
  selected: Date | undefined;
  onSelect: (date: Date | undefined) => void;
}

function JalaliCalendar({ selected, onSelect }: JalaliCalendarProps) {
  const [currentMonth, setCurrentMonth] = React.useState(() => {
    const now = selected || new Date();
    return {
      year: parseInt(formatJalali(now, 'yyyy')),
      month: parseInt(formatJalali(now, 'M'))
    };
  });

  const jalaliMonths = [
    'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 
    'مرداد', 'شهریور', 'مهر', 'آبان', 
    'آذر', 'دی', 'بهمن', 'اسفند'
  ];

  const jalaliWeekDays = ['ش', 'ی', 'د', 'س', 'چ', 'پ', 'ج'];

  const getDaysInJalaliMonth = (year: number, month: number) => {
    if (month <= 6) return 31;
    if (month <= 11) return 30;
    // Check for leap year (simplified)
    const isLeap = ((year % 33) * 8 + 29) % 33 < 8;
    return isLeap ? 30 : 29;
  };

  const jalaliToGregorian = (jYear: number, jMonth: number, jDay: number): Date => {
    // Create a jalali date string and parse it with date-fns-jalali
    const jalaliDateStr = `${jYear}/${jMonth.toString().padStart(2, '0')}/${jDay.toString().padStart(2, '0')}`;
    try {
      return parseJalali(jalaliDateStr, 'yyyy/MM/dd', new Date());
    } catch {
      return new Date();
    }
  };

  const generateCalendarDays = () => {
    const daysInMonth = getDaysInJalaliMonth(currentMonth.year, currentMonth.month);
    const firstDay = jalaliToGregorian(currentMonth.year, currentMonth.month, 1);
    const startWeekDay = (firstDay.getDay() + 1) % 7; // Convert to Saturday=0

    const days: (number | null)[] = [];
    
    // Add empty cells for days before the first day of month
    for (let i = 0; i < startWeekDay; i++) {
      days.push(null);
    }
    
    // Add the days of the month
    for (let day = 1; day <= daysInMonth; day++) {
      days.push(day);
    }
    
    return days;
  };

  const handleDayClick = (day: number) => {
    const gregorianDate = jalaliToGregorian(currentMonth.year, currentMonth.month, day);
    onSelect(gregorianDate);
  };

  const nextMonth = () => {
    if (currentMonth.month === 12) {
      setCurrentMonth({ year: currentMonth.year + 1, month: 1 });
    } else {
      setCurrentMonth({ ...currentMonth, month: currentMonth.month + 1 });
    }
  };

  const prevMonth = () => {
    if (currentMonth.month === 1) {
      setCurrentMonth({ year: currentMonth.year - 1, month: 12 });
    } else {
      setCurrentMonth({ ...currentMonth, month: currentMonth.month - 1 });
    }
  };

  const isSelectedDay = (day: number) => {
    if (!selected) return false;
    const selectedJalali = formatJalali(selected, 'yyyy/M/d').split('/');
    return (
      parseInt(selectedJalali[0]) === currentMonth.year &&
      parseInt(selectedJalali[1]) === currentMonth.month &&
      parseInt(selectedJalali[2]) === day
    );
  };

  const days = generateCalendarDays();

  return (
    <div className="p-3 pointer-events-auto" dir="rtl">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Button variant="ghost" size="sm" onClick={prevMonth}>
            &gt;
          </Button>
          <div className="font-semibold">
            {jalaliMonths[currentMonth.month - 1]} {currentMonth.year}
          </div>
          <Button variant="ghost" size="sm" onClick={nextMonth}>
            &lt;
          </Button>
        </div>
        <div className="grid grid-cols-7 gap-1">
          {jalaliWeekDays.map(day => (
            <div key={day} className="text-center text-sm font-medium p-2">
              {day}
            </div>
          ))}
          {days.map((day, idx) => (
            <div key={idx} className="flex items-center justify-center">
              {day !== null ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDayClick(day)}
                  className={cn(
                    "h-9 w-9 p-0 font-normal",
                    isSelectedDay(day) && "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground"
                  )}
                >
                  {day}
                </Button>
              ) : (
                <div className="h-9 w-9" />
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}