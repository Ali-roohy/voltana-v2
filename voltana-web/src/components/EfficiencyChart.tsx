import {
  ComposedChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ReferenceArea,
  ResponsiveContainer,
} from 'recharts';
import { useLanguage } from '@/contexts/LanguageContext';

export interface EfficiencyPoint {
  date: string;
  efficiency: number;
  carName: string;
}

interface Props {
  data: EfficiencyPoint[];
}

export function EfficiencyChart({ data }: Props) {
  const { language } = useLanguage();
  const fa = language === 'fa';

  if (data.length < 2) {
    return (
      <div className="flex items-center justify-center h-32 text-sm text-muted-foreground text-center px-4">
        {fa
          ? 'داده ناکافی — حداقل ۲ جلسه با اودومتر لازم است'
          : 'Not enough data — at least 2 sessions with odometer readings required'}
      </div>
    );
  }

  const effValues = data.map((d) => d.efficiency);
  const avg = effValues.reduce((s, v) => s + v, 0) / effValues.length;
  const min = Math.min(...effValues);
  const max = Math.max(...effValues);

  const CustomTooltip = ({ active, payload }: any) => {
    if (!active || !payload?.length) return null;
    const p = payload[0].payload as EfficiencyPoint;
    return (
      <div className="bg-card border border-border rounded-md px-3 py-2 text-xs shadow-md">
        <div className="font-medium">{p.date}</div>
        <div className="text-primary">{p.efficiency.toFixed(1)} kWh/100km</div>
        <div className="text-muted-foreground">{p.carName}</div>
      </div>
    );
  };

  return (
    <ResponsiveContainer width="100%" height={180}>
      <ComposedChart data={data} margin={{ top: 8, right: 8, left: -20, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
        <XAxis
          dataKey="date"
          tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
          tickLine={false}
          axisLine={false}
          interval="preserveStartEnd"
        />
        <YAxis
          tick={{ fontSize: 10, fill: 'hsl(var(--muted-foreground))' }}
          tickLine={false}
          axisLine={false}
          unit=" "
        />
        <Tooltip content={<CustomTooltip />} />

        {/* Min/max range band */}
        <ReferenceArea
          y1={min}
          y2={max}
          fill="hsl(var(--primary))"
          fillOpacity={0.06}
          stroke="none"
        />

        {/* Average reference line */}
        <ReferenceLine
          y={avg}
          stroke="hsl(var(--accent))"
          strokeDasharray="5 3"
          strokeWidth={1.5}
          label={{
            value: `avg ${avg.toFixed(1)}`,
            position: 'insideTopRight',
            fontSize: 9,
            fill: 'hsl(var(--accent))',
          }}
        />

        {/* Primary efficiency trend */}
        <Line
          type="monotone"
          dataKey="efficiency"
          stroke="hsl(var(--primary))"
          strokeWidth={2}
          dot={{ fill: 'hsl(var(--primary))', r: 3 }}
          activeDot={{ r: 5 }}
          name={fa ? 'مصرف kWh/100km' : 'kWh/100km'}
          animationDuration={600}
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}
