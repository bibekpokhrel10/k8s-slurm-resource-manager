# k8s-slurm-resource-manager

## Resource Allocation Application

This application provides a resource allocation mechanism using Kubernetes (Kueue) and Slurm. It allows users to choose between Slurm and Kueue for managing workloads in a cluster environment.

---

## Setting up the Application

Here’s a step-by-step breakdown of how to set up and run your application:

### **1. Kubernetes (K3s) and Kueue Setup**
- Ensure a **Kubernetes (K3s) cluster** is running.
- Install **Kueue** on the cluster.
- Create necessary **queues** up to the `LocalQueue` level.

### **2. Slurm Setup**
- Slurm must be installed and manually configured on the system.

### **3. Environment Configuration**
- Create a `.env` file with the required variables:
  ```sh
  KUBECONFIG=/path/to/kubeconfig
  NODE=node-name
  ```
- Set `KUBECONFIG` in the environment:
  ```sh
  export KUBECONFIG=/path/to/kubeconfig
  ```

### **4. Running the Application**
- The application will **prompt the user** to choose between **Kubernetes (Kueue)** and **Slurm**.
- Based on the selection:
  - If **Kubernetes** is selected, it will use **Kueue** to allocate resources.
  - If **Slurm** is selected, it will perform **Slurm job scheduling**.


## How the Application Works

1. **User Selection**:
   - The application prompts the user to choose between Slurm and Kueue for resource allocation.

2. **Slurm-Based Allocation**:
   - Drains the Kubernetes node using `kubectl drain`.
   - Resumes the node in Slurm using `scontrol`.
   - Runs Slurm commands to allocate resources.

3. **Kueue-Based Allocation**:
   - Drains the node in Slurm if needed.
   - Uncordons the Kubernetes node using `kubectl uncordon`.
   - Waits for Kueue to become available.
   - Submits a Kubernetes job with Kueue scheduling.

4. **Node Resource Management**:
   - Checks the status of Slurm nodes.
   - Performs actions like `resume`, `drain`, `drain-k8s`, and `uncordon-k8s`.

5. **Service Availability Checks**:
   - Ensures Kueue is running before scheduling jobs.

6. **Execution of Jobs**:
   - Submits jobs to either Slurm or Kueue based on user choice.

---

# K3s Installation Guide (Lightweight Kubernetes)

## Prerequisites
- Ubuntu system with sudo access
- At least 2 CPUs and 2GB RAM
- Docker installed (optional but recommended)

## Installation Steps

1. **Download and Install K3s**:
   ```bash
   curl -sfL https://get.k3s.io | sh -
   ```

2. **Verify Installation**:
   ```bash
   kubectl get nodes
   ```

3. **Configure Kubeconfig**:
   ```bash
   export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
   ```

4. **Enable Systemd Service**:
   ```bash
   sudo systemctl enable k3s
   sudo systemctl start k3s
   ```

---

# Slurm Configuration Guide for Ubuntu

## Prerequisites
- Ubuntu OS with sudo access
- Basic Linux command-line knowledge

## Installation Steps

1. **Install Required Packages**:
   ```bash
   sudo apt install munge slurmd slurm-client slurmctld
   ```

2. **Verify Hostname**:
   ```bash
   hostname
   ```

3. **Configure Slurm**:
   - Edit `/etc/slurm/slurm.conf` and add:
     ```
     ClusterName=YourClusterName
     SlurmctldHost=YourHostname
     ProctrackType=proctrack/linuxproc
     NodeName=YourHostname CPUs=4 Sockets=1 CoresPerSocket=2 ThreadsPerCore=2 State=UNKNOWN
     PartitionName=debug Nodes=ALL Default=YES MaxTime=INFINITE State=UP
     ```

4. **Enable and Start Services**:
   ```bash
   sudo systemctl enable slurmctld
   sudo systemctl enable slurmd
   sudo systemctl start slurmctld
   sudo systemctl start slurmd
   ```

5. **Verify Slurm Services**:
   ```bash
   sudo systemctl status slurmctld
   sudo systemctl status slurmd
   ```

6. **Run a Test Job**:
   ```bash
   sbatch --wrap='echo $SLURM_JOB_ID; sleep 60'
   ```

---

# Kueue Setup Guide

## What is Kueue?
Kueue is a Kubernetes-native batch scheduling system that efficiently manages jobs based on available resources.

## Prerequisites
- Kubernetes 1.25+
- Docker installed

## Installation Steps

1. **Install Kueue**:
   ```bash
   kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/v0.9.1/manifests.yaml
   ```

2. **Wait for Kueue Controller to Start**:
   ```bash
   kubectl wait deploy/kueue-controller-manager -nkueue-system --for=condition=available --timeout=5m
   ```

## Setup ClusterQueue, ResourceFlavor, and LocalQueue

1. **Create a ClusterQueue**:
   ```yaml
   apiVersion: kueue.x-k8s.io/v1beta1
   kind: ClusterQueue
   metadata:
     name: "cluster-queue"
   spec:
     namespaceSelector: {}
     resourceGroups:
     - coveredResources: ["cpu", "memory"]
       flavors:
       - name: "default-flavor"
         resources:
         - name: "cpu"
           nominalQuota: 9
         - name: "memory"
           nominalQuota: 36Gi
   ```
   ```bash
   kubectl apply -f cluster-queue.yaml
   ```

2. **Create a ResourceFlavor**:
   ```yaml
   apiVersion: kueue.x-k8s.io/v1beta1
   kind: ResourceFlavor
   metadata:
     name: "default-flavor"
   ```
   ```bash
   kubectl apply -f default-flavor.yaml
   ```

3. **Create a LocalQueue**:
   ```yaml
   apiVersion: kueue.x-k8s.io/v1beta1
   kind: LocalQueue
   metadata:
     namespace: "default"
     name: "user-queue"
   spec:
     clusterQueue: "cluster-queue"
   ```
   ```bash
   kubectl apply -f default-user-queue.yaml
   ```

## Running a Job with Kueue

1. **Create a Job Manifest**:
   ```yaml
   apiVersion: batch/v1
   kind: Job
   metadata:
     generateName: sample-job-
     namespace: default
     labels:
       kueue.x-k8s.io/queue-name: user-queue
   spec:
     parallelism: 3
     completions: 3
     suspend: true
     template:
       spec:
         containers:
         - name: dummy-job
           image: gcr.io/k8s-staging-perf-tests/sleep:v0.1.0
           args: ["30s"]
           resources:
             requests:
               cpu: 1
               memory: "200Mi"
         restartPolicy: Never
   ```
   ```bash
   kubectl apply -f sample-job.yaml
   ```

2. **Check Job Status**:
   ```bash
   kubectl -n default get workloads
   ```

---

## Summary
- **This application automates resource allocation using Slurm and Kueue.**
- **K3s provides a lightweight Kubernetes environment.**
- **Slurm is used for HPC job scheduling.**
- **Kueue optimizes batch scheduling in Kubernetes.**

By following this guide, you can efficiently allocate resources in your cluster environment.

