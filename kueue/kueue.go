package kueue

import (
	"context"
	"log"
	"time"

	"github.com/k8s-slurm-resource-manager/helper"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Function to allocate resources for Kueue (using Job resource)
func AllocateKueueResources(clientset *kubernetes.Clientset, namespace string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sample-job-kueue",
			Namespace:    namespace,
			Labels: map[string]string{
				"kueue.x-k8s.io/queue-name": "user-queue",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism: helper.Int32Ptr(1),
			Completions: helper.Int32Ptr(1),
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

func IsKueueServiceAvailable(clientset *kubernetes.Clientset) bool {
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
func WaitForKueueService(clientset *kubernetes.Clientset, timeout int) {
	for i := 0; i < timeout; i++ {
		if IsKueueServiceAvailable(clientset) {
			return
		}
		log.Printf("Waiting for Kueue service to become available... (%d/%d)", i+1, timeout)
		time.Sleep(5 * time.Second) // Wait for 5 seconds before retrying
	}
	log.Fatal("Timed out waiting for Kueue service to become available.")
}
