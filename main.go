package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/k8s-slurm-resource-manager/kueue"
	"github.com/k8s-slurm-resource-manager/node"
	"github.com/k8s-slurm-resource-manager/slurm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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
	nodeName := "srv697719"

	// Prompt user to select between Slurm and Kueue
	var choice string
	fmt.Println("Select the resource allocation method:")
	fmt.Println("1. Slurm")
	fmt.Println("2. k8s")
	fmt.Print("Enter 1 or 2: ")
	fmt.Scanln(&choice)

	switch strings.TrimSpace(choice) {
	case "1":
		if err := node.ManageNodeResources(nodeName, "drain-k8s"); err != nil {
			log.Fatalf("Failed to drain node: %v", err)
		}
		if err := node.ManageNodeResources(nodeName, "resume"); err != nil {
			log.Fatalf("Failed to resume node: %v", err)
		}
		if err := slurm.AllocateSlurmResources(); err != nil {
			log.Fatalf("Failed to allocate Slurm resources: %v", err)
		}
	case "2":
		if err := node.ManageNodeResources(nodeName, "drain"); err != nil {
			log.Fatalf("Failed to drain node: %v", err)
		}
		if err := node.ManageNodeResources(nodeName, "uncordon-k8s"); err != nil {
			log.Fatalf("Failed to uncordon node: %v", err)
		}
		kueue.WaitForKueueService(clientset, 10)
		if err := kueue.AllocateKueueResources(clientset, namespace); err != nil {
			log.Fatalf("Failed to allocate Slurm resources: %v", err)
		}
	default:
		log.Fatalf("Invalid choice: %v", choice)
	}

	fmt.Println("Resource allocation completed.")
}
