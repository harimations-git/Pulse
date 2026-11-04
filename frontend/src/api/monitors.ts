
import type { CheckRecord } from "../types/checkRecord";
import type { Monitor, MonitorStats } from "../types/monitor";

export type CreateMonitorRequest = {
  name: string;
  url: string;
  intervalSeconds: number;
};


export async function getMonitors(): Promise<Monitor[]> {
  const response = await fetch("/api/monitors");

  if (!response.ok) {
    throw new Error("Failed to fetch monitors");
  }

  const data: Monitor[] | null = await response.json();

  return data ?? [];
}

export async function getMonitorStats(
  monitorId: number,
  signal?: AbortSignal
): Promise<MonitorStats> {
  const response = await fetch(
    `/api/monitors/${monitorId}/stats`,
    { signal }
  );

  if (!response.ok) {
    throw new Error("Failed to fetch monitor statistics");
  }

  return response.json();
}

export async function getMonitorHistory(
  monitorId: number,
  signal?: AbortSignal
): Promise<CheckRecord[]> {
  const response = await fetch(
    `/api/monitors/${monitorId}/results`,
    { signal }
  );

  if (!response.ok) {
    throw new Error("Failed to fetch monitoring history");
  }

  const data: CheckRecord[] | null = await response.json();

  return data ?? [];
}

export async function createMonitor(
  monitor: CreateMonitorRequest
): Promise<Monitor> {
  const response = await fetch("/api/monitors", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(monitor),
  });

  if(!response.ok){
    const message = await response.text();
    throw new Error(message || "Failed to create monitor");
  }

  return response.json();
}


export async function checkMonitorNow(
  monitorId: number
): Promise<void> {
  const response = await fetch(
    `/api/monitors/${monitorId}/check`,
    {
      method: "POST",
    }
  );

  if (response.status === 409) {
    throw new Error(
      "This monitor is already being checked. Try again shortly."
    );
  }

  if (!response.ok) {
    throw new Error("Failed to check monitor");
  }
}
