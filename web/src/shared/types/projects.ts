export type TemplateParam = {
  name: string;
  label: string;
  description: string;
  type: "string" | "number" | "boolean" | "secret";
  required: boolean;
  default?: string | number | boolean;
  placeholder?: string;
};

export type TemplateSpec = {
  id: string;
  name: string;
  version: string;
  description: string;
  params: TemplateParam[];
  health_hints: string[];
};

export type ProjectContainer = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  ports: string[];
};

export type Project = {
  id: string;
  name: string;
  template_id: string;
  template_version: string;
  config: Record<string, unknown>;
  compose_path: string;
  status: string;
  last_error: string;
  runtime: {
    status: string;
    containers: ProjectContainer[];
  };
  created_at: string;
  updated_at: string;
};
