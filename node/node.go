package node

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/k8s-slurm-resource-manager/slurm"
)

func ManageNodeResources(nodeName string, action string) error {
	status := ""
	if action == "resume" || action == "drain" {
		var err error
		status, err = slurm.GetSlurmNodeStatus(nodeName)
		if err != nil {
			return err
		}
	}
	var command string
	switch action {
	case "resume":
		if status == "RESUME" || status == "IDLE" {
			fmt.Printf("Node %s is already in RESUME/IDLE state, skipping.\n", nodeName)
			return nil
		}
		command = fmt.Sprintf("scontrol update NodeName=%s state=RESUME", nodeName)
	case "drain":
		if status == "DRAIN" {
			fmt.Printf("Node %s is already in DRAIN state, skipping.\n", nodeName)
			return nil
		}
		command = fmt.Sprintf("scontrol update NodeName=%s state=DRAIN reason=\"k8s\"", nodeName)
	case "drain-k8s":
		command = fmt.Sprintf("kubectl drain %s --ignore-daemonsets --delete-emptydir-data", nodeName)
	case "uncordon-k8s":
		command = fmt.Sprintf("kubectl uncordon %s", nodeName)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	fmt.Printf("Executing command: %s\n", command)
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
