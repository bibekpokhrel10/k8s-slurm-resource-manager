package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Function to allocate resources for Kueue (using Job resource)
func allocateKueueResources(clientset *kubernetes.Clientset, namespace string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sample-job-kueue",
			Namespace:    namespace,
			Labels: map[string]string{
				"kueue.x-k8s.io/queue-name": "user-queue",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: int32Ptr(1),
			Completions: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sleep-count-container",
							Image: "busybox",
							Command: []string{
								"/bin/sh", "-c",
								"for i in $(seq 1 10); do echo \"Sleeping for $i seconds...\"; sleep 1; done; echo \"Job completed!\"",
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	_, err := clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	return err
}

func getSlurmNodeStatus(nodeName string) (string, error) {
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

func manageNodeResources(nodeName string, action string) error {
	status := ""
	if action == "resume" || action == "drain" {
		var err error
		status, err = getSlurmNodeStatus(nodeName)
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

func isKueueServiceAvailable(clientset *kubernetes.Clientset) bool {
	// Check if the service exists
	_, err := clientset.CoreV1().Services("kueue-system").Get(context.TODO(), "kueue-webhook-service", metav1.GetOptions{})
	if err != nil {
		log.Printf("Kueue service not found: %v", err)
		return false
	}

	// Check if the endpoints exist and are not empty
	endpoints, err := clientset.CoreV1().Endpoints("kueue-system").Get(context.TODO(), "kueue-webhook-service", metav1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get endpoints for Kueue service: %v", err)
		return false
	}

	// If there are subsets and addresses, the service is available
	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			log.Println("Kueue service is available.")
			return true
		}
	}

	log.Println("Kueue service endpoints are not ready yet.")
	return false
}

// Function to wait until Kueue service is available
func waitForKueueService(clientset *kubernetes.Clientset, timeout int) {
	for i := 0; i < timeout; i++ {
		if isKueueServiceAvailable(clientset) {
			return
		}
		log.Printf("Waiting for Kueue service to become available... (%d/%d)", i+1, timeout)
		time.Sleep(5 * time.Second) // Wait for 5 seconds before retrying
	}
	log.Fatal("Timed out waiting for Kueue service to become available.")
}

// Function to allocate resources for Slurm directly on the server
func allocateSlurmResources() error {
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

func int32Ptr(i int32) *int32 {
	return &i
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		log.Fatal("KUBECONFIG environment variable not set")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes clientset: %v", err)
	}

	namespace := "default" // Change as needed

	// Prompt user to select between Slurm and Kueue
	var choice string
	fmt.Println("Select the resource allocation method:")
	fmt.Println("1. Slurm")
	fmt.Println("2. k8s")
	fmt.Print("Enter 1 or 2: ")
	fmt.Scanln(&choice)

	switch strings.TrimSpace(choice) {
	case "1":
		if err := manageNodeResources("srv697719", "drain-k8s"); err != nil {
			log.Fatalf("Failed to drain node: %v", err)
		}
		if err := manageNodeResources("srv697719", "resume"); err != nil {
			log.Fatalf("Failed to resume node: %v", err)
		}
		if err := allocateSlurmResources(); err != nil {
			log.Fatalf("Failed to allocate Slurm resources: %v", err)
		}
	case "2":
		if err := manageNodeResources("srv697719", "drain"); err != nil {
			log.Fatalf("Failed to drain node: %v", err)
		}
		if err := manageNodeResources("srv697719", "uncordon-k8s"); err != nil {
			log.Fatalf("Failed to uncordon node: %v", err)
		}
		waitForKueueService(clientset, 10)
		if err := allocateKueueResources(clientset, namespace); err != nil {
			log.Fatalf("Failed to allocate Slurm resources: %v", err)
		}
	default:
		log.Fatalf("Invalid choice: %v", choice)
	}

	fmt.Println("Resource allocation completed.")
}
