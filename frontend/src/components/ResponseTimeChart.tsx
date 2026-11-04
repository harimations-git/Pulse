import {
    CartesianGrid,
    Line,
    LineChart,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
} from "recharts";

import type { CheckRecord } from "../types/checkRecord";

type ResponseTimeChartProps = {
    history: CheckRecord[];
};

export default function ResponseTimeChart({ history, }: ResponseTimeChartProps) {

    const chartData = history
        .toReversed()
        .map((record) => ({
            time: record.checkedAt,
            responseTimeMs: record.responseTimeMs,
        }));

    return (
        <section className="mt-6 border-t border-slate-800 pt-5">
            <h4 className="mb-4 text-sm font-medium text-slate-300">
                Response time history
            </h4>

            {history.length === 0 ? (
                <p className="text-sm text-slate-400">
                    No check history available yet.
                </p>
            ) : (
                <div className="h-48 w-full min-w-0">
                    <ResponsiveContainer width="100%" height="100%">
                        <LineChart
                            data={chartData}
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
    );
}