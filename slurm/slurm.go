package slurm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func GetSlurmNodeStatus(nodeName string) (string, error) {
	command := fmt.Sprintf("scontrol show node %s | grep State", nodeName)
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get node status: %v", err)
	}

	outputStr := string(output)
	logrus.Info("output string :: ", outputStr)
	if strings.Contains(outputStr, "RESUME") {
		return "RESUME", nil
	} else if strings.Contains(outputStr, "DRAIN") {
		return "DRAIN", nil
	} else if strings.Contains(outputStr, "IDLE") {
		return "IDLE", nil
	}
	return "UNKNOWN", nil
}

// Function to allocate resources for Slurm directly on the server
func AllocateSlurmResources() error {
	commands := []string{
		"sinfo",
		"srun hostname",
		"sbatch --wrap='echo $SLURM_JOB_ID; sleep 60'",
		"squeue",
	}

	for _, command := range commands {
		fmt.Printf("Executing command: %s\n", command)
		cmd := exec.Command("bash", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute command '%s': %v", command, err)
		}
	}

	fmt.Println("Slurm resource allocation completed successfully.")
	return nil
}
