
import { useEffect, useState } from "react";
import { getMonitors } from "./api/monitors";
import type { Monitor } from "./types/monitor";
import MonitorCard from "./components/MonitorCard";
import AddMonitorForm from "./components/AddMonitorForm";

function App() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getMonitors()
      .then(setMonitors)
      .catch((err: unknown) => {
        setError(
          err instanceof Error ? err.message : "Unknown error"
        );
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <main className="min-h-screen bg-slate-950 p-8 text-white">
      <div className="mx-auto max-w-5xl">

        <header className="mb-10">
          <h1 className="text-4xl font-bold text-emerald-400">
            Pulse
          </h1>
          <p className="mt-2 text-slate-400">
            Website uptime monitoring dashboard
          </p>
        </header>

        <h2 className="mb-5 text-xl font-semibold">
          Your monitors
        </h2>

        {loading && <p>Loading monitors...</p>}

        {error && (
          <p role="alert" className="text-red-400">
            {error}
          </p>
        )}

        {!loading && !error && monitors.length === 0 && (
          <p className="text-slate-400">
            No monitors registered yet.
          </p>
        )}

        <AddMonitorForm
          onCreated={(created) => {
            setMonitors((current) => [...current, created]);
          }}
        />

        <div className="grid gap-4 md:grid-cols-2">
          {!loading &&
            !error &&
            monitors.map((monitor) => (
              <MonitorCard
                key={monitor.id}
                monitor={monitor}
              />
            ))}
        </div>

      </div>
    </main>
  );
}

export default App;
