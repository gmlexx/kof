package grafana

type GrafanaPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PluginList []GrafanaPlugin

type PluginMap map[string]PluginList
