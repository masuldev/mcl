package internal

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/fatih/color"
)

type EksCluster struct {
	Name     string
	Arn      string
	Version  string
	Status   string
	Region   string
	Endpoint string
	RoleArn  string
}

// EKS 클러스터 목록 조회
func ListEksClusters(ctx context.Context, cfg aws.Config) ([]EksCluster, error) {
	client := eks.NewFromConfig(cfg)

	clusters := []EksCluster{}

	// 모든 클러스터 조회
	result, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
	}

	// 각 클러스터의 상세 정보 조회
	for _, clusterName := range result.Clusters {
		cluster, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
			Name: aws.String(clusterName),
		})
		if err != nil {
			continue // 개별 클러스터 조회 실패 시 건너뛰기
		}

		eksCluster := EksCluster{
			Name:     aws.ToString(cluster.Cluster.Name),
			Arn:      aws.ToString(cluster.Cluster.Arn),
			Version:  aws.ToString(cluster.Cluster.Version),
			Status:   string(cluster.Cluster.Status),
			Region:   cfg.Region,
			Endpoint: aws.ToString(cluster.Cluster.Endpoint),
			RoleArn:  aws.ToString(cluster.Cluster.RoleArn),
		}

		clusters = append(clusters, eksCluster)
	}

	return clusters, nil
}

// EKS 클러스터 정보 출력
func PrintEksClusters(cmd string, clusters []EksCluster) {
	for _, cluster := range clusters {
		LogEksCluster(cmd, cluster.Region, cluster.Name, cluster.Arn, cluster.Version, cluster.Status, cluster.Endpoint)
	}
}

// EKS 클러스터 로그 출력
func LogEksCluster(cmd, region, name, arn, version, status, endpoint string) {
	fmt.Printf("%s: region: %s, name: %s, arn: %s, version: %s, status: %s, endpoint: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.YellowString(name),
		color.YellowString(arn), color.BlueString(version), color.GreenString(status),
		color.MagentaString(endpoint))
}

// kubectl update-config 실행
func UpdateKubectlConfig(ctx context.Context, clusterName, region string) error {
	// aws eks update-kubeconfig 명령어 실행
	cmd := exec.CommandContext(ctx, "aws", "eks", "update-kubeconfig",
		"--name", clusterName,
		"--region", region)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update kubectl config: %w, output: %s", err, string(output))
	}

	LogSuccess("kubectl config updated for cluster: %s in region: %s", clusterName, region)
	return nil
}

// kubectl get nodes 실행
func GetKubectlNodes(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get kubectl nodes: %w, output: %s", err, string(output))
	}

	fmt.Println(string(output))
	return nil
}

// kubectl get pods 실행
func GetKubectlPods(ctx context.Context, namespace string) error {
	args := []string{"get", "pods"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get kubectl pods: %w, output: %s", err, string(output))
	}

	fmt.Println(string(output))
	return nil
}

// kubectl describe node 실행
func DescribeKubectlNode(ctx context.Context, nodeName string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "describe", "node", nodeName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to describe kubectl node: %w, output: %s", err, string(output))
	}

	fmt.Println(string(output))
	return nil
}

// kubectl logs 실행
func GetKubectlLogs(ctx context.Context, podName, namespace string, follow bool) error {
	args := []string{"logs"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, podName)

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get kubectl logs: %w, output: %s", err, string(output))
	}

	fmt.Println(string(output))
	return nil
}

// kubectl exec 실행
func ExecKubectl(ctx context.Context, podName, namespace, command string) error {
	args := []string{"exec"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, podName, "--", command)

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to exec kubectl command: %w, output: %s", err, string(output))
	}

	fmt.Println(string(output))
	return nil
}

// kubectl apply 실행
func ApplyKubectl(ctx context.Context, filePath string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply kubectl file: %w, output: %s", err, string(output))
	}

	LogSuccess("kubectl apply completed for file: %s", filePath)
	fmt.Println(string(output))
	return nil
}

// kubectl delete 실행
func DeleteKubectl(ctx context.Context, resourceType, resourceName, namespace string) error {
	args := []string{"delete", resourceType, resourceName}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete kubectl resource: %w, output: %s", err, string(output))
	}

	LogSuccess("kubectl delete completed for %s: %s", resourceType, resourceName)
	fmt.Println(string(output))
	return nil
}
