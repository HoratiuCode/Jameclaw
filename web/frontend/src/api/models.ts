import { refreshGatewayState } from "@/store/gateway"

// API client for model list management.

export interface ModelInfo {
  index: number
  model_name: string
  model: string
  api_base?: string
  api_key: string
  proxy?: string
  auth_method?: string
  // Advanced fields
  connect_mode?: string
  workspace?: string
  rpm?: number
  max_tokens_field?: string
  request_timeout?: number
  thinking_level?: string
  extra_body?: Record<string, unknown>
  // Meta
  configured: boolean
  is_default: boolean
  is_image_default: boolean
  is_voice_default: boolean
}

export interface ModelMutationPayload extends Partial<ModelInfo> {
  clear_api_key?: boolean
}

export interface ProviderAuthMethod {
  id: string
  label: string
  description?: string
}

export interface ModelPreset {
  id: string
  name: string
  model_name: string
  model: string
  api_base?: string
  requires_api_key: boolean
  key_label?: string
  request_timeout?: number
  thinking_level?: string
  description?: string
}

export interface ProviderCatalogEntry {
  id: string
  name: string
  category: string
  description: string
  docs_url?: string
  protocols: string[]
  default_api_base?: string
  requires_api_key: boolean
  key_label?: string
  auth_methods?: ProviderAuthMethod[]
  recommended_models: ModelPreset[]
  setup_hint?: string
  local_runtime_hint?: string
  configured: boolean
  default: boolean
  configured_models?: string[]
}

interface ModelsListResponse {
  models: ModelInfo[]
  total: number
  default_model: string
  default_image_model: string
  default_voice_model: string
	model_fallbacks: string[]
}

interface ModelActionResponse {
  status: string
  index?: number
  default_model?: string
  default_image_model?: string
  default_voice_model?: string
}

interface ModelCatalogResponse {
  providers: ProviderCatalogEntry[]
  default_model: string
}

export interface AddModelFromCatalogPayload {
  provider_id: string
  preset_id: string
  model_name?: string
  api_key?: string
  set_default?: boolean
}

export type ModelRole = "chat" | "image" | "voice"

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getModels(): Promise<ModelsListResponse> {
  return request<ModelsListResponse>("/api/models")
}

export async function getModelCatalog(): Promise<ModelCatalogResponse> {
  return request<ModelCatalogResponse>("/api/models/catalog")
}

export async function addModel(
  model: ModelMutationPayload,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>("/api/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function addModelFromCatalog(
  payload: AddModelFromCatalogPayload,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>("/api/models/from-catalog", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
}

export async function updateModel(
  index: number,
  model: ModelMutationPayload,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function deleteModel(index: number): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "DELETE",
  })
}

export async function setDefaultModel(
  modelName: string,
  role: ModelRole = "chat",
): Promise<ModelActionResponse> {
  const response = await request<ModelActionResponse>("/api/models/default", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model_name: modelName, role }),
  })

  await refreshGatewayState()
  return response
}

export interface ModelFailoverResponse {
	status: string
	primary_model: string
	secondary_model: string
	gateway_restarted: boolean
}

export async function setModelFailover(
	primaryModel: string,
	secondaryModel: string,
): Promise<ModelFailoverResponse> {
	const response = await request<ModelFailoverResponse>("/api/models/failover", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({
			primary_model: primaryModel,
			secondary_model: secondaryModel,
		}),
	})
	await refreshGatewayState()
	return response
}

export type { ModelsListResponse, ModelActionResponse, ModelCatalogResponse }
