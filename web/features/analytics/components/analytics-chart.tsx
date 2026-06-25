"use client";

import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from "recharts";
import { AnalyticsPoint } from "@/types";
import { formatBytes, formatDate } from "@/lib/utils";

interface AnalyticsChartProps {
  data: AnalyticsPoint[];
  metric: "transfers" | "bytes" | "successRate";
}

export function AnalyticsChart({ data, metric }: AnalyticsChartProps) {
  if (metric === "successRate") {
    return (
      <ResponsiveContainer width="100%" height={260}>
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid stroke="#1F1F23" vertical={false} />
          <XAxis dataKey="date" tickFormatter={(d) => formatDate(d)} stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} minTickGap={30} />
          <YAxis domain={[90, 100]} stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} width={40} />
          <Tooltip
            contentStyle={{ background: "#1A1A1E", border: "1px solid #2C2C32", borderRadius: 8, fontSize: 12 }}
            labelFormatter={(d) => formatDate(d)}
            formatter={(value: number) => [`${value.toFixed(1)}%`, "Success rate"]}
          />
          <Line type="monotone" dataKey="successRate" stroke="#4CB782" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    );
  }

  if (metric === "bytes") {
    return (
      <ResponsiveContainer width="100%" height={260}>
        <AreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="upBytes" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#5E6AD2" stopOpacity={0.4} />
              <stop offset="100%" stopColor="#5E6AD2" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="downBytes" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#5E9DD2" stopOpacity={0.4} />
              <stop offset="100%" stopColor="#5E9DD2" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="#1F1F23" vertical={false} />
          <XAxis dataKey="date" tickFormatter={(d) => formatDate(d)} stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} minTickGap={30} />
          <YAxis tickFormatter={(v) => formatBytes(v)} stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} width={70} />
          <Tooltip
            contentStyle={{ background: "#1A1A1E", border: "1px solid #2C2C32", borderRadius: 8, fontSize: 12 }}
            labelFormatter={(d) => formatDate(d)}
            formatter={(value: number, name: string) => [formatBytes(value), name === "uploadBytes" ? "Uploaded" : "Downloaded"]}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} formatter={(value) => (value === "uploadBytes" ? "Uploaded" : "Downloaded")} />
          <Area type="monotone" dataKey="uploadBytes" stroke="#5E6AD2" strokeWidth={2} fill="url(#upBytes)" />
          <Area type="monotone" dataKey="downloadBytes" stroke="#5E9DD2" strokeWidth={2} fill="url(#downBytes)" />
        </AreaChart>
      </ResponsiveContainer>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id="upCount" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#5E6AD2" stopOpacity={0.4} />
            <stop offset="100%" stopColor="#5E6AD2" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="downCount" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#5E9DD2" stopOpacity={0.4} />
            <stop offset="100%" stopColor="#5E9DD2" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid stroke="#1F1F23" vertical={false} />
        <XAxis dataKey="date" tickFormatter={(d) => formatDate(d)} stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} minTickGap={30} />
        <YAxis stroke="#5C5F66" fontSize={11} tickLine={false} axisLine={false} width={40} />
        <Tooltip
          contentStyle={{ background: "#1A1A1E", border: "1px solid #2C2C32", borderRadius: 8, fontSize: 12 }}
          labelFormatter={(d) => formatDate(d)}
          formatter={(value: number, name: string) => [value, name === "uploads" ? "Uploads" : "Downloads"]}
        />
        <Legend wrapperStyle={{ fontSize: 12 }} formatter={(value) => (value === "uploads" ? "Uploads" : "Downloads")} />
        <Area type="monotone" dataKey="uploads" stroke="#5E6AD2" strokeWidth={2} fill="url(#upCount)" />
        <Area type="monotone" dataKey="downloads" stroke="#5E9DD2" strokeWidth={2} fill="url(#downCount)" />
      </AreaChart>
    </ResponsiveContainer>
  );
}
