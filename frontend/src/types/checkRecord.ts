export type CheckRecord = {
  id: number;
  monitorId: number;
  statusCode: number;
  responseTimeMs: number;
  healthy: boolean;
  errorMessage: string | null;
  checkedAt: string;
};