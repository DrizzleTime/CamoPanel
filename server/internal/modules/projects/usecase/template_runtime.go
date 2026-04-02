package usecase

import (
	"path/filepath"

	"camopanel/server/internal/services"
)

func templateRuntime(cfg ProjectConfig, projectName string) services.TemplateRuntime {
	return services.TemplateRuntime{
		ProjectName:          projectName,
		BridgeNetworkName:    cfg.BridgeNetworkName,
		OpenRestyContainer:   cfg.OpenRestyContainer,
		OpenRestyHostConfDir: filepath.Join(cfg.OpenRestyDataDir, "conf.d"),
		OpenRestyHostSiteDir: filepath.Join(cfg.OpenRestyDataDir, "www"),
		OpenRestyHostCertDir: filepath.Join(cfg.OpenRestyDataDir, "certs"),
	}
}
