import type { CheckRecord } from "./checkRecord";
import type { MonitorStats } from "./monitor";

export type MonitorSnapshot = {
    stats: MonitorStats;
    latestCheck: CheckRecord | null;
}