export type Monitor = {
  id: number;
  name: string;
  url: string;
  intervalSeconds: number;
  createdAt: string;
};

export type MonitorStats = {
  monitorId: number;
  totalChecks: number;
  successfulChecks: number;
  uptimePercentage: number;
  averageResponseTimeMs: number;
};