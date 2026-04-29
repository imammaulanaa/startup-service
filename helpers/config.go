package helpers

import (
	"encoding/json"
	"os"
)

type AuthConfig struct {
	WorkloadIdentity      bool   `json:"workload_identity"`
	ServiceAccountKeyFile string `json:"service_account_key_file"`
}

type ProjectConfig struct {
	ProjectID         string            `json:"project_id"`
	Zone              string            `json:"zone"`
	GCEInstances      map[string]string `json:"gce_instances"`
	CloudSQLInstances map[string]string `json:"cloud_sql_instances"`
}

type Config struct {
	Auth     				AuthConfig               	`json:"auth"`
	Projects 				map[string]ProjectConfig 	`json:"projects"`
	GoogleChatWebhooks		[]string 					`json:"google_chat_webhooks"`
}

func ReadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
