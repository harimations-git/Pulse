
import { useEffect, useState } from "react";
import { getMonitorHistory, getMonitorStats } from "../api/monitors";
import type { Monitor, MonitorStats } from "../types/monitor";
import ResponseTimeChart from "./ResponseTimeChart";
import type { CheckRecord } from "../types/checkRecord";
import type { MonitorSnapshot } from "../types/monitorSnapshot";

type MonitorCardProps = {
  monitor: Monitor;
  onSnapshot: (
    monitorId: number,
    snapshot: MonitorSnapshot | null
  ) => void;
};

export default function MonitorCard({
  monitor,
  onSnapshot,
}: MonitorCardProps) {
  const [stats, setStats] = useState<MonitorStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [history, setHistory] = useState<CheckRecord[]>([]);

  const latestCheck = history[0];

  const status = loading ? "Loading..." : !latestCheck ? "Awaiting first check" : latestCheck.healthy ? "Healthy" : "Down";

  useEffect(() => {
    const controller = new AbortController();

    let fetching = false;

    async function refreshData() {
      // Prevent overlapping refresh requests
      if (fetching) return;

      fetching = true;

      try {
        const [newStats, newHistory] = await Promise.all([
          getMonitorStats(monitor.id, controller.signal),
          getMonitorHistory(monitor.id, controller.signal),
        ]);

        if (!controller.signal.aborted) {
          setStats(newStats);
          setHistory(newHistory);
          setError(null);

          onSnapshot(monitor.id, {
            stats: newStats,
            latestCheck: newHistory[0] ?? null,
          });
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to refresh monitor"
          );

          onSnapshot(monitor.id, null);
        }
      } finally {
        fetching = false;

        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    }

    // Fetch immediately
    void refreshData();

    // Refresh every 30 seconds
    const interval = window.setInterval(() => {
      void refreshData();
    }, 30_000);

    return () => {
      window.clearInterval(interval);
      controller.abort();
    };
  }, [monitor.id, onSnapshot]);

  return (
    <article className="rounded-xl border border-slate-800 bg-slate-900 p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-lg font-semibold text-white">
          {monitor.name}
        </h3>

        <span
          className={`rounded-full px-3 py-1 text-xs font-medium ${!latestCheck
            ? "bg-slate-700 text-slate-300"
            : latestCheck.healthy
              ? "bg-emerald-500/15 text-emerald-400"
              : "bg-red-500/15 text-red-400"
            }`}
        >
          {status}
        </span>
      </div>

      <p className="mt-1 break-all text-sm text-slate-400">
        {monitor.url}
      </p>

      {
        loading && (
          <p className="mt-6 text-sm text-slate-400">
            Loading statistics...
          </p>
        )
      }

      {
        error && (
          <p role="alert" className="mt-6 text-sm text-red-400">
            {error}
          </p>
        )
      }

      {
        stats && !loading && (
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
        )
      }

      <p className="mt-6 text-xs text-slate-500">
        Check interval: {monitor.intervalSeconds}s
      </p>

      <ResponseTimeChart history={history} />
    </article >
  );
}
