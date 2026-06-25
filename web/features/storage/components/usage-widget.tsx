"use client";

import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";
import { StorageProvider } from "@/types";
import { formatBytes } from "@/lib/utils";

const colors: Record<string, string> = { r2: "#F38020", s3: "#FF9900", local: "#5E9DD2" };

export function UsageWidget({ providers }: { providers: StorageProvider[] }) {
  const data = providers.map((p) => ({ name: p.name, value: p.usedBytes, id: p.id }));
  const total = data.reduce((sum, d) => sum + d.value, 0);

  return (
    <div className="flex items-center gap-6">
      <div className="relative h-[160px] w-[160px] shrink-0">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie data={data} dataKey="value" innerRadius={52} outerRadius={76} strokeWidth={0}>
              {data.map((d) => (
                <Cell key={d.id} fill={colors[d.id]} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{ background: "#1A1A1E", border: "1px solid #2C2C32", borderRadius: 8, fontSize: 12 }}
              formatter={(value: number) => formatBytes(value)}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="font-mono text-sm font-semibold text-ink">{formatBytes(total)}</span>
          <span className="text-[10px] text-ink-faint">total used</span>
        </div>
      </div>
      <div className="flex flex-1 flex-col gap-2.5">
        {data.map((d) => (
          <div key={d.id} className="flex items-center justify-between text-[13px]">
            <span className="flex items-center gap-2">
              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: colors[d.id] }} />
              {d.name}
            </span>
            <span className="font-mono text-ink-muted">{formatBytes(d.value)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
