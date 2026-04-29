package helpers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

func ManageCloudSQL(
	ctx context.Context,
	projectID string,
	instances map[string]string,
	status map[string]string,
) error {

	if len(instances) == 0 {
		log.Println("[cloudsql] No Cloud SQL instances configured")
		return nil
	}

	svc, err := sqladmin.NewService(ctx, option.WithScopes(sqladmin.CloudPlatformScope))
	if err != nil {
		return fmt.Errorf("sqladmin service init: %w", err)
	}

	for name, desired := range instances {
		desired = strings.ToLower(strings.TrimSpace(desired))

		log.Printf("[cloudsql] Checking instance=%s desired=%s", name, desired)

		ins, err := svc.Instances.Get(projectID, name).Do()
		if err != nil {
			log.Printf("[cloudsql] Get instance=%s error: %v", name, err)
			status[name] = "ERROR"
			continue
		}

		currentPolicy := ins.Settings.ActivationPolicy
		log.Printf("[cloudsql] instance=%s currentActivationPolicy=%s",
			name, currentPolicy)

		var targetPolicy string

		switch desired {
		case "active":
			targetPolicy = "ALWAYS"

		case "inactive":
			targetPolicy = "NEVER"

		default:
			log.Printf("[cloudsql] invalid desired state for %s", name)
			status[name] = "INVALID"
			continue
		}

		if currentPolicy == targetPolicy {
			log.Printf("[cloudsql] instance=%s already desired state", name)
			status[name] = strings.Title(desired)
			continue
		}

		log.Printf("[cloudsql] Updating activationPolicy %s -> %s",
			currentPolicy, targetPolicy)

		_, err = svc.Instances.Patch(projectID, name,
			&sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{
					ActivationPolicy: targetPolicy,
				},
			}).Do()

		if err != nil {
			log.Printf("[cloudsql] patch failed: %v", err)
			status[name] = "FAILED"
			continue
		}

		status[name] = strings.Title(desired)

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}