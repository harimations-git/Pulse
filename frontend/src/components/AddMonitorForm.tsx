
import { useState } from "react";
import type { FormEvent } from "react";

import { createMonitor } from "../api/monitors";
import type { Monitor } from "../types/monitor";

type Props = {
    onCreated: (monitor: Monitor) => void;
};

export default function AddMonitorForm({ onCreated }: Props) {
    const [name, setName] = useState("");
    const [url, setUrl] = useState("");
    const [intervalSeconds, setIntervalSeconds] = useState(60);

    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    async function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();

        if (submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);

        try {
            const created = await createMonitor({
                name: name.trim(),
                url: url.trim(),
                intervalSeconds,
            });

            onCreated(created);

            setName("");
            setUrl("");
            setIntervalSeconds(60);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to add monitor");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <form
            onSubmit={handleSubmit}
            className="mb-8 rounded-xl border border-slate-800 bg-slate-900 p-6"
        >
            <h2 className="mb-5 text-lg font-semibold">
                Add monitor
            </h2>

            <div className="grid gap-4 md:grid-cols-3">
                <div>
                    <label
                        htmlFor="monitor-name"
                        className="mb-2 block text-sm text-slate-300"
                    >
                        Name
                    </label>

                    <input
                        id="monitor-name"
                        type="text"
                        required
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        placeholder="My Website"
                        className="w-full rounded-lg border border-slate-700 bg-slate-950 p-3 text-white"
                    />
                </div>

                <div>
                    <label
                        htmlFor="monitor-url"
                        className="mb-2 block text-sm text-slate-300"
                    >
                        Website URL
                    </label>

                    <input
                        id="monitor-url"
                        type="url"
                        required
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="https://example.com"
                        className="w-full rounded-lg border border-slate-700 bg-slate-950 p-3 text-white"
                    />
                </div>

                <div>
                    <label
                        htmlFor="monitor-interval"
                        className="mb-2 block text-sm text-slate-300"
                    >
                        Interval (seconds)
                    </label>

                    <input
                        id="monitor-interval"
                        type="number"
                        min={10}
                        required
                        value={intervalSeconds}
                        onChange={(e) => setIntervalSeconds(Number(e.target.value))}
                        className="w-full rounded-lg border border-slate-700 bg-slate-950 p-3 text-white"
                    />
                </div>
            </div>

            {error && (
                <p role="alert" className="mt-4 text-sm text-red-400">
                    {error}
                </p>
            )}

            <button
            type="submit"
            disabled={submitting}
            className="mt-5 rounded-lg bg-emerald-500 px-5 py-2.5 font-medium text-slate-950 hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"
            >
                {submitting ? "Adding..." : "Add monitor"}
            </button>
        </form>
    );
}