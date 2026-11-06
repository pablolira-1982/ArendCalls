import { apiGet, apiPost, apiPatch, apiDelete, checkStatus } from "@/lib/api";
import { getClientId } from "@/lib/client-id";
import type { SessionInfo, RecordingConfig, RecordingConfigPatch, RecordingsOverview } from "@/types/session";

export const listSessions = () =>
  apiGet<{ sessions: SessionInfo[] }>("/api/sessions").then((r) => r.sessions ?? []);

export const createSession = (name: string) =>
  apiPost<{ id: string }>("/api/sessions", { name });

export const updateSession = (id: string, name: string) =>
  apiPatch<{ status: string }>(`/api/sessions/${id}`, { name });

export const updateWebhookUrl = (id: string, webhook_url: string) =>
  apiPatch<{ status: string }>(`/api/sessions/${id}/webhook`, { webhook_url });

/** Override por conta: força mostrar as chamadas recebidas desta conta no painel
 *  mesmo com o switch global desligado (bypass p/ teste). Nunca esconde. */
export const setPanelInbound = (id: string, enabled: boolean) =>
  apiPatch<{ status: string; panelInbound: boolean }>(`/api/sessions/${id}/panel-inbound`, { enabled });

/** Switch global: o painel mostra chamadas recebidas ou não. */
export const getPanelSettings = () =>
  apiGet<{ inboundCalls: boolean }>(`/api/panel-settings`);

export const setPanelInboundCalls = (inboundCalls: boolean) =>
  apiPatch<{ status: string; inboundCalls: boolean }>(`/api/panel-settings`, { inboundCalls });

export const getRecordingConfig = (id: string) =>
  apiGet<RecordingConfig>(`/api/sessions/${id}/recording-config`);

export const updateRecordingConfig = (id: string, patch: RecordingConfigPatch) =>
  apiPatch<RecordingConfig>(`/api/sessions/${id}/recording-config`, patch);

export const listRecordings = (id: string) =>
  apiGet<RecordingsOverview>(`/api/sessions/${id}/recordings`);

export const deleteSession = (id: string) => apiDelete(`/api/sessions/${id}`);

const postVoid = async (path: string): Promise<void> => {
  const r = await fetch(path, {
    method: "POST",
    headers: { "X-Client-Id": getClientId(), "Content-Type": "application/json" },
    body: "{}",
  });
  await checkStatus(r, path);
};

export const logoutSession = (id: string) => postVoid(`/api/sessions/${id}/logout`);

export const pairSession = (id: string) => postVoid(`/api/sessions/${id}/pair`);

export const startSession = (id: string) => postVoid(`/api/sessions/${id}/start`);

export const stopSession = (id: string) => postVoid(`/api/sessions/${id}/stop`);

export const restartSession = (id: string) => postVoid(`/api/sessions/${id}/restart`);

export const getProxy = (id: string) =>
  apiGet<{ proxy: string; enabled: boolean }>(`/api/sessions/${id}/proxy`);

export const setProxy = async (id: string, proxy: string): Promise<void> => {
  const r = await fetch(`/api/sessions/${id}/proxy`, {
    method: "PUT",
    headers: { "X-Client-Id": getClientId(), "Content-Type": "application/json" },
    body: JSON.stringify({ proxy }),
  });
  await checkStatus(r, `/api/sessions/${id}/proxy`);
};

export type SIPExtConfig = {
  authUser?: string;
  enabled: boolean;
  host: string;
  port: number;
  user: string;
  pass: string;
  dest: string;
  proxy?: string;
  inboundMode?: "originate" | "agent" | "auto";
  agentAutoAnswer?: boolean;
  agentRingTimeoutSeconds?: number;
  status?: string;
  error?: string;
  advertise?: string;
};

export const getSIPExt = (id: string) =>
  apiGet<SIPExtConfig>(`/api/sessions/${id}/sip-ext`);

export const setSIPExt = (id: string, cfg: Omit<SIPExtConfig, "status" | "error" | "advertise">) =>
  apiPost<{ status: string }>(`/api/sessions/${id}/sip-ext`, cfg);

export type NativeFormInfo = {
  flowId: string;
  flowName?: string;
  formType?: string;
  body?: string;
  cta?: string;
  screen?: string;
  data?: Record<string, unknown>;
  fields?: string[];
  sentCount: number;
  lastSentAt?: number;
  responseCount: number;
  lastResponseAt?: number;
};

export const listNativeForms = (id: string) =>
  apiGet<{ forms: NativeFormInfo[]; count: number }>(`/api/sessions/${id}/native-forms`);
