
import { useEffect, useState } from "react";
import { getMonitorStats } from "../api/monitors";
import type { Monitor, MonitorStats } from "../types/monitor";
import ResponseTimeChart from "./ResponseTimeChart";

type MonitorCardProps = {
  monitor: Monitor;
};

export default function MonitorCard({
  monitor,
}: MonitorCardProps) {
  const [stats, setStats] = useState<MonitorStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    getMonitorStats(monitor.id, controller.signal)
      .then(setStats)
      .catch((err: unknown) => {
        if (!controller.signal.aborted) {
          setError(
            err instanceof Error ? err.message : "Unknown error"
          );
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [monitor.id]);

  return (
    <article className="rounded-xl border border-slate-800 bg-slate-900 p-6">
      <h3 className="text-lg font-semibold text-white">
        {monitor.name}
      </h3>

      <p className="mt-1 break-all text-sm text-slate-400">
        {monitor.url}
      </p>

      {loading && (
        <p className="mt-6 text-sm text-slate-400">
          Loading statistics...
        </p>
      )}

      {error && (
        <p role="alert" className="mt-6 text-sm text-red-400">
          {error}
        </p>
      )}

      {stats && !loading && (
        <>
          {stats.totalChecks === 0 ? (
            <p className="mt-6 text-sm text-slate-400">
              No monitoring data available yet.
            </p>
          ) : (
            <>
              <div className="mt-6 grid grid-cols-3 gap-4">
                <div>
                  <p className="text-xs text-slate-400">
                    Uptime
                  </p>
                  <p className="mt-1 text-xl font-semibold text-emerald-400">
                    {stats.uptimePercentage.toFixed(1)}%
                  </p>
                </div>

                <div>
                  <p className="text-xs text-slate-400">
                    Avg. response
                  </p>
                  <p className="mt-1 text-xl font-semibold text-white">
                    {stats.averageResponseTimeMs.toFixed(0)}ms
                  </p>
                </div>

                <div>
                  <p className="text-xs text-slate-400">
                    Checks
                  </p>
                  <p className="mt-1 text-xl font-semibold text-white">
                    {stats.totalChecks}
                  </p>
                </div>
              </div>

              <div className="mt-5">
                <div className="mb-2 flex justify-between text-xs">
                  <span className="text-slate-400">
                    Successful checks
                  </span>
                  <span className="text-white">
                    {stats.successfulChecks} / {stats.totalChecks}
                  </span>
                </div>

                <div className="h-2 overflow-hidden rounded-full bg-slate-700">
                  <div
                    className="h-full rounded-full bg-emerald-400"
                    style={{
                      width: `${Math.max(
                        0,
                        Math.min(100, stats.uptimePercentage)
                      )}%`,
                    }}
                  />
                </div>
              </div>
            </>
          )}
        </>
      )}

      <p className="mt-6 text-xs text-slate-500">
        Check interval: {monitor.intervalSeconds}s
      </p>

      <ResponseTimeChart monitorId={monitor.id} />
    </article>
  );
}
