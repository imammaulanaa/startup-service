package helpers

import (
	"log"
	"os/exec"
)

func Authenticate(cfg AuthConfig) {
	if cfg.WorkloadIdentity {
		log.Println("[auth] Using Workload Identity / ADC (no service account key needed)")
		return
	}

	if cfg.ServiceAccountKeyFile == "" {
		log.Fatal("[auth] workload_identity=false but service_account_key_file is empty")
	}

	log.Println("[auth] Using service account key file (legacy mode)")
	cmd := exec.Command("gcloud", "auth", "activate-service-account", "--key-file", cfg.ServiceAccountKeyFile)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to authenticate using service account: %v", err)
	}
}
