"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { ProviderComparisonPoint } from "@/types";

export function ProviderComparisonChart({ data }: { data: ProviderComparisonPoint[] }) {
  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid stroke="#1F1F23" vertical={false} />
        <XAxis dataKey="provider" stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} />
        <YAxis stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} width={40} />
        <Tooltip
          contentStyle={{ background: "#1A1A1E", border: "1px solid #2C2C32", borderRadius: 8, fontSize: 12 }}
          formatter={(value: number) => [`${value}ms`, "Avg latency"]}
        />
        <Bar dataKey="avgLatencyMs" radius={[4, 4, 0, 0]} fill="#5E6AD2" />
      </BarChart>
    </ResponsiveContainer>
  );
}
