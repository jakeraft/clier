package view

import "github.com/jakeraft/clier/internal/config"

type Config struct {
	ServerURL       string `json:"server_url"`
	DashboardURL    string `json:"dashboard_url"`
	CredentialsPath string `json:"credentials_path"`
	WorkspaceDir    string `json:"workspace_dir"`
}

type ConfigValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DashboardOpenResult struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

func ConfigOf(cfg *config.File) Config {
	return Config{
		ServerURL:       cfg.ServerURL,
		DashboardURL:    cfg.DashboardURL,
		CredentialsPath: cfg.CredentialsPath,
		WorkspaceDir:    cfg.WorkspaceDir,
	}
}

func ConfigValueOf(key, value string) ConfigValue {
	return ConfigValue{Key: key, Value: value}
}

func DashboardOpenOf(url string) DashboardOpenResult {
	return DashboardOpenResult{Status: "opened", URL: url}
}
