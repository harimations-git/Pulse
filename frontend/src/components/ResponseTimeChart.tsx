import { useEffect, useState } from "react";
import {
    CartesianGrid,
    Line,
    LineChart,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
} from "recharts";

import { getMonitorHistory } from "../api/monitors";
import type { CheckRecord } from "../types/checkRecord";

type ResponseTimeChartProps = {
    monitorId: number;
};

export default function ResponseTimeChart({ monitorId, }: ResponseTimeChartProps) {
    const [history, setHistory] = useState<CheckRecord[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        getMonitorHistory(monitorId, controller.signal).then(setHistory).catch((err: unknown) => {
            if (!controller.signal.aborted) {
                setError(err instanceof Error ? err.message : "Unknown error");
            }
        }).finally(() => {
            if (!controller.signal.aborted) {
                setLoading(false);
            }
        });

        return () => controller.abort();
    }, [monitorId]);


    const chartData = history
        .slice(0, 30)
        .reverse()
        .map((record) => ({
            time: record.checkedAt,
            responseTimeMs: record.responseTimeMs,
        }));

    return (
        <section className="mt-6 border-t border-slate-800 pt-5">
            <h4 className="mb-4 text-sm font-medium text-slate-300">
                Response time history
            </h4>

            {loading && (
                <p className="text-sm text-slate-400">
                    Loading history...
                </p>
            )}

            {error && (
                <p role="alert" className="text-sm text-red-400">
                    {error}
                </p>
            )}

            {!loading && !error && history.length === 0 && (
                <p className="text-sm text-slate-400">
                    No check history available yet.
                </p>
            )}

            {!loading && !error && history.length > 0 && (
                <div className="h-48 w-full min-w-0">
                    <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={chartData}
                            margin={{ top: 10, right: 10, left: 0, bottom: 5 }}
                        >
                            <CartesianGrid
                                stroke="#334155"
                                strokeDasharray="3 3"
                                vertical={false}
                            />

                            <XAxis
                                dataKey="time"
                                tickFormatter={(value: string) =>
                                    new Date(value).toLocaleTimeString([], {
                                        hour: "2-digit",
                                        minute: "2-digit",
                                    })
                                }
                                tick={{ fill: "#94a3b8", fontSize: 11 }}
                                tickLine={false}
                                axisLine={false}
                                minTickGap={25}
                            />

                            <YAxis
                                domain={[0, "auto"]}
                                tick={{ fill: "#94a3b8", fontSize: 11 }}
                                tickLine={false}
                                axisLine={false}
                                width={40}
                            />

                            <Tooltip
                                labelFormatter={(value) =>
                                    new Date(String(value)).toLocaleString()
                                }
                                formatter={(value) => [
                                    `${Number(value).toFixed(0)} ms`,
                                    "Response time",
                                ]}
                                contentStyle={{
                                    backgroundColor: "#0f172a",
                                    border: "1px solid #334155",
                                    borderRadius: "8px",
                                    color: "#f8fafc",
                                }}
                            />

                            <Line
                                type="monotone"
                                dataKey="responseTimeMs"
                                stroke="#34d399"
                                strokeWidth={2}
                                dot={false}
                                activeDot={{ r: 4 }}
                                isAnimationActive={false}
                            />
                        </LineChart>
                    </ResponsiveContainer>
                </div>
            )}
        </section>
    )
}