package helpers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
)

func ManageGCE(ctx context.Context, projectID string, instances map[string]string, zone string) error {
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("compute client init: %w", err)
	}
	defer client.Close()

	for instance, desiredState := range instances {
		desiredState = strings.ToLower(strings.TrimSpace(desiredState))
		log.Printf("[gce] Checking instance=%s desired=%s\n", instance, desiredState)

		// Get current status
		getReq := &computepb.GetInstanceRequest{
			Project:  projectID,
			Zone:     zone,
			Instance: instance,
		}
		vm, err := client.Get(ctx, getReq)
		if err != nil {
			log.Printf("[gce] Get instance=%s error: %v\n", instance, err)
			continue
		}

		current := strings.ToUpper(vm.GetStatus())
		log.Printf("[gce] instance=%s current=%s\n", instance, current)

		switch desiredState {
		case "stopped":
			if current == "RUNNING" {
				log.Printf("[gce] Stopping instance=%s\n", instance)
				op, err := client.Stop(ctx, &computepb.StopInstanceRequest{
					Project:  projectID,
					Zone:     zone,
					Instance: instance,
				})
				if err != nil {
					log.Printf("[gce] Stop instance=%s error: %v\n", instance, err)
					continue
				}
				_ = op // optional: wait operation completion (lihat catatan bawah)
			} else {
				log.Printf("[gce] instance=%s already desired=%s\n", instance, desiredState)
			}

		case "running":
			if current == "TERMINATED" || current == "STOPPED" {
				log.Printf("[gce] Starting instance=%s\n", instance)
				op, err := client.Start(ctx, &computepb.StartInstanceRequest{
					Project:  projectID,
					Zone:     zone,
					Instance: instance,
				})
				if err != nil {
					log.Printf("[gce] Start instance=%s error: %v\n", instance, err)
					continue
				}
				_ = op
			} else {
				log.Printf("[gce] instance=%s already desired=%s\n", instance, desiredState)
			}

		default:
			log.Printf("[gce] instance=%s invalid desired state: %s (use running|stopped)\n", instance, desiredState)
		}

		time.Sleep(200 * time.Millisecond)
	}

	return nil
}
