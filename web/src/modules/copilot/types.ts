export type CopilotSession = {
  id: string;
};

export type CopilotModel = {
  id: string;
  provider_id: string;
  name: string;
  enabled: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
};

export type CopilotProvider = {
  id: string;
  name: string;
  type: string;
  base_url: string;
  enabled: boolean;
  has_api_key: boolean;
  api_key_masked: string;
  models: CopilotModel[];
  created_at: string;
  updated_at: string;
};

export type CopilotConfigStatus = {
  configured: boolean;
  source: string;
  provider_id?: string;
  provider_name?: string;
  model_id?: string;
  model_name?: string;
  base_url?: string;
};
