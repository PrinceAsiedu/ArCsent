export interface Finding {
  scanner_name: string;
  severity: string;
  category: string;
  description: string;
  occurred_at: string;
  remediation?: string;
  evidence?: Record<string, string>;
}

export interface Baseline {
  metric: string;
  mean: number;
  min: number;
  max: number;
}

export interface SystemStatus {
  status: string;
  running_as_root: boolean;
  version: string;
}

export interface SignatureSourceStatus {
  source: string;
  url?: string;
  path?: string;
  bytes?: number;
  updated_at?: string;
  duration?: string;
  error?: string;
}

export interface SignatureStatus {
  last_run?: string;
  next_run?: string;
  airgap_mode?: boolean;
  airgap_import_path?: string;
  sources?: Record<string, SignatureSourceStatus>;
}

export const api = {
  getToken: () => localStorage.getItem("arcsent_token"),
  setToken: (token: string) => localStorage.setItem("arcsent_token", token),

  async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const token = this.getToken();
    const headers = {
      Authorization: token || "",
      "Content-Type": "application/json",
      ...options?.headers,
    };

    const res = await fetch(`/api${path}`, { ...options, headers });
    if (!res.ok) {
      if (res.status === 401) {
        throw new Error("Unauthorized");
      }
      throw new Error(`Request failed: ${res.statusText}`);
    }
    return res.json();
  },

  getStatus: () => api.fetch<SystemStatus>("/status"),
  getScanners: () => api.fetch<{ plugins: string[]; jobs: any[] }>("/scanners"),
  getFindings: () => api.fetch<Finding[]>("/findings"),
  getBaselines: () => api.fetch<Baseline[]>("/baselines"),
  getLatestResults: () => api.fetch<any[]>("/results/latest"),
  triggerScan: (plugin: string) =>
    api.fetch(`/scanners/trigger/${plugin}`, { method: "POST" }),
  getSignaturesStatus: () => api.fetch<SignatureStatus>("/signatures/status"),
  triggerSignaturesUpdate: () =>
    api.fetch<SignatureStatus>("/signatures/update", { method: "POST" }),
  getMetricsText: async (): Promise<string> => {
    const token = api.getToken();
    const headers = {
      Authorization: token || "",
      "Content-Type": "text/plain",
    };
    const res = await fetch(`/api/metrics`, { headers });
    if (!res.ok) {
      if (res.status === 401) {
        throw new Error("Unauthorized");
      }
      throw new Error(`Request failed: ${res.statusText}`);
    }
    return res.text();
  },
};
