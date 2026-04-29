package main

import (
	"context"
	"log"

	"startup-service/helpers"
)

func main() {
	ctx := context.Background()

	cfg, err := helpers.ReadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	helpers.Authenticate(cfg.Auth)

	for projectName, projectCfg := range cfg.Projects {
		log.Printf("=== Processing project: %s (project_id=%s) ===\n", projectName, projectCfg.ProjectID)

		log.Println("Managing Cloud SQL instances...")
		if err := helpers.ManageCloudSQL(ctx, projectCfg.ProjectID, projectCfg.CloudSQLInstances); err != nil {
			log.Printf("ManageCloudSQL error (project=%s): %v\n", projectCfg.ProjectID, err)
		}

		log.Println("Managing GCE instances...")
		if err := helpers.ManageGCE(ctx, projectCfg.ProjectID, projectCfg.GCEInstances, projectCfg.Zone); err != nil {
			log.Printf("ManageGCE error (project=%s): %v\n", projectCfg.ProjectID, err)
		}
	}
	if len(config.GoogleChatWebhooks) > 0 { 
		message := formatStatusSummary(statusSummary)
		if err := helpers.SendGoogleChatNotification(config.GoogleChatWebhooks, message); err != nil {
			log.Printf("Failed to send notification: %v", err)
		}
	}
}

// appendMaps combines two maps into one.
func appendMaps(m1, m2 map[string]string) map[string]string {
	combined := make(map[string]string)
	for k, v := range m1 {
		combined[k] = v
	}
	for k, v := range m2 {
		combined[k] = v
	}
	return combined
}

// formatStatusSummary formats the status summary for Google Chat.
func formatStatusSummary(statusSummary map[string]map[string]string) string {
	message := "Process completed. Status summary:\n"
	for project, instances := range statusSummary {
		message += fmt.Sprintf("Project: %s\n", project)
		for instance, status := range instances {
			message += fmt.Sprintf("  - %s: %s\n", instance, status)
		}
		message += "\n"
	}
	return message
}
