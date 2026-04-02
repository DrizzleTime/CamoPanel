import { useEffect, useState } from "react";
import type { Project, TemplateSpec } from "../../../shared/types";
import { loadWebsitesDataBundle } from "../api";
import type { Certificate, OpenRestyStatus, Website } from "../types";

export function useWebsitesData() {
  const [status, setStatus] = useState<OpenRestyStatus | null>(null);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [templates, setTemplates] = useState<TemplateSpec[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = async () => {
    setLoading(true);
    try {
      const data = await loadWebsitesDataBundle();
      setStatus(data.status);
      setWebsites(data.websites);
      setCertificates(data.certificates);
      setProjects(data.projects);
      setTemplates(data.templates);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refresh();
  }, []);

  return {
    status,
    websites,
    certificates,
    projects,
    templates,
    loading,
    refresh,
  };
}
