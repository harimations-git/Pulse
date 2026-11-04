import type { Monitor } from "../types/monitor";
import type { MonitorSnapshot } from "../types/monitorSnapshot";

type DashboardSummaryProps = {
    monitors: Monitor[];
    snapshots: Record<number, MonitorSnapshot | null>;
};

export default function DashboardSummary({
    monitors,
    snapshots,
}: DashboardSummaryProps) {
    const current = monitors.map((monitor) => snapshots[monitor.id]);

    const healthy = current.filter((snapshot) => snapshot?.latestCheck?.healthy === true).length;

    const down = current.filter((snapshot) => snapshot?.latestCheck?.healthy === false).length;

    //Only include monitors that have recorded checks
    const withData = current.filter((snapshot): snapshot is MonitorSnapshot => snapshot != null && snapshot.stats.totalChecks > 0);

    const averageUptime = withData.length > 0 ? withData.reduce((total, snapshot) => total + snapshot.stats.uptimePercentage, 0) / withData.length : null;

    const cards = [
        {
            label: "Total monitors",
            value: monitors.length.toString(),
            color: "text-white",
        },
        {
            label: "Healthy",
            value: healthy.toString(),
            color: "text-emerald-400",
        },
        {
            label: "Down",
            value: down.toString(),
            color: "text-red-400",
        },
        {
            label: "Average uptime",
            value:
                averageUptime === null
                    ? "—"
                    : `${averageUptime.toFixed(1)}%`,
            color: "text-white",
        },
    ];

    return (
        <section className="mb-10">
            <h2 className="mb-5 text-xl font-semibold text-white">
                Overview
            </h2>

            <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
                {cards.map((card) => (
                    <div
                        key={card.label}
                        className="rounded-xl border border-slate-800 bg-slate-900 p-5"
                    >
                        <p className="text-sm text-slate-400">
                            {card.label}
                        </p>

                        <p
                            className={`mt-3 text-3xl font-bold ${card.color}`}
                        >
                            {card.value}
                        </p>
                    </div>
                ))}
            </div>

            <p className="mt-3 text-xs text-slate-500">
                Status reflects the latest recorded check.
                Average uptime includes monitors with check history.
            </p>
        </section>
    );
}
