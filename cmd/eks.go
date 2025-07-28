package cmd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
)

var (
	startEksCommand = &cobra.Command{
		Use:   "eks",
		Short: "EKS (Elastic Kubernetes Service) management",
		Long:  "EKS (Elastic Kubernetes Service) management - list clusters, update kubectl config, and manage Kubernetes resources",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			credential := GetGlobalAwsConfig()
			if credential == nil {
				internal.RealPanic(fmt.Errorf("AWS credentials not configured"))
			}

			// EKS 클러스터 목록 조회
			clusters, err := internal.ListEksClusters(ctx, *credential)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			if len(clusters) == 0 {
				internal.LogWarning("No EKS clusters found in region: %s", credential.Region)
				return
			}

			internal.LogInfo("Found %d EKS clusters in region: %s", len(clusters), credential.Region)
			internal.PrintEksClusters("eks", clusters)

			// kubectl update-config 옵션 제공
			if len(clusters) > 0 {
				var doUpdate bool
				prompt := &survey.Confirm{
					Message: "kubectl config를 업데이트하시겠습니까?",
					Default: false,
				}
				if err := survey.AskOne(prompt, &doUpdate); err != nil {
					internal.RealPanic(err)
				}

				if doUpdate {
					// 클러스터 선택
					var options []string
					for _, cluster := range clusters {
						options = append(options, cluster.Name)
					}

					var selectedCluster string
					clusterPrompt := &survey.Select{
						Message: "업데이트할 클러스터를 선택하세요:",
						Options: options,
					}
					if err := survey.AskOne(clusterPrompt, &selectedCluster); err != nil {
						internal.RealPanic(err)
					}

					// kubectl config 업데이트
					err := internal.UpdateKubectlConfig(ctx, selectedCluster, credential.Region)
					if err != nil {
						internal.RealPanic(internal.WrapError(err))
					}

					// kubectl 명령어 실행 옵션 제공
					var runKubectl bool
					kubectlPrompt := &survey.Confirm{
						Message: "kubectl 명령어를 실행하시겠습니까?",
						Default: false,
					}
					if err := survey.AskOne(kubectlPrompt, &runKubectl); err != nil {
						internal.RealPanic(err)
					}

					if runKubectl {
						runKubectlCommands(ctx)
					}
				}
			}
		},
	}
)

func runKubectlCommands(ctx context.Context) {
	// kubectl 명령어 선택
	var kubectlOptions = []string{
		"get nodes",
		"get pods",
		"get pods -n kube-system",
		"get services",
		"get namespaces",
		"describe nodes",
		"logs (follow)",
		"exec",
		"apply",
		"delete",
	}

	var selectedCommand string
	commandPrompt := &survey.Select{
		Message: "실행할 kubectl 명령어를 선택하세요:",
		Options: kubectlOptions,
	}
	if err := survey.AskOne(commandPrompt, &selectedCommand); err != nil {
		internal.RealPanic(err)
	}

	switch selectedCommand {
	case "get nodes":
		err := internal.GetKubectlNodes(ctx)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "get pods":
		err := internal.GetKubectlPods(ctx, "")
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "get pods -n kube-system":
		err := internal.GetKubectlPods(ctx, "kube-system")
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "get services":
		// kubectl get services 실행
		cmd := exec.CommandContext(ctx, "kubectl", "get", "services")
		output, err := cmd.CombinedOutput()
		if err != nil {
			internal.RealPanic(fmt.Errorf("failed to get services: %w, output: %s", err, string(output)))
		}
		fmt.Println(string(output))
	case "get namespaces":
		// kubectl get namespaces 실행
		cmd := exec.CommandContext(ctx, "kubectl", "get", "namespaces")
		output, err := cmd.CombinedOutput()
		if err != nil {
			internal.RealPanic(fmt.Errorf("failed to get namespaces: %w, output: %s", err, string(output)))
		}
		fmt.Println(string(output))
	case "describe nodes":
		// 노드 이름 입력 받기
		var nodeName string
		nodePrompt := &survey.Input{
			Message: "노드 이름을 입력하세요:",
		}
		if err := survey.AskOne(nodePrompt, &nodeName); err != nil {
			internal.RealPanic(err)
		}
		err := internal.DescribeKubectlNode(ctx, nodeName)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "logs (follow)":
		// 파드 이름과 네임스페이스 입력 받기
		var podName, namespace string
		podPrompt := &survey.Input{
			Message: "파드 이름을 입력하세요:",
		}
		if err := survey.AskOne(podPrompt, &podName); err != nil {
			internal.RealPanic(err)
		}
		nsPrompt := &survey.Input{
			Message: "네임스페이스를 입력하세요 (선택사항):",
		}
		if err := survey.AskOne(nsPrompt, &namespace); err != nil {
			internal.RealPanic(err)
		}
		err := internal.GetKubectlLogs(ctx, podName, namespace, true)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "exec":
		// 파드 이름, 네임스페이스, 명령어 입력 받기
		var podName, namespace, command string
		podPrompt := &survey.Input{
			Message: "파드 이름을 입력하세요:",
		}
		if err := survey.AskOne(podPrompt, &podName); err != nil {
			internal.RealPanic(err)
		}
		nsPrompt := &survey.Input{
			Message: "네임스페이스를 입력하세요 (선택사항):",
		}
		if err := survey.AskOne(nsPrompt, &namespace); err != nil {
			internal.RealPanic(err)
		}
		cmdPrompt := &survey.Input{
			Message: "실행할 명령어를 입력하세요:",
			Default: "ls -la",
		}
		if err := survey.AskOne(cmdPrompt, &command); err != nil {
			internal.RealPanic(err)
		}
		err := internal.ExecKubectl(ctx, podName, namespace, command)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "apply":
		// 파일 경로 입력 받기
		var filePath string
		filePrompt := &survey.Input{
			Message: "적용할 YAML 파일 경로를 입력하세요:",
		}
		if err := survey.AskOne(filePrompt, &filePath); err != nil {
			internal.RealPanic(err)
		}
		err := internal.ApplyKubectl(ctx, filePath)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	case "delete":
		// 리소스 타입, 이름, 네임스페이스 입력 받기
		var resourceType, resourceName, namespace string
		typePrompt := &survey.Input{
			Message: "리소스 타입을 입력하세요 (예: pod, service, deployment):",
		}
		if err := survey.AskOne(typePrompt, &resourceType); err != nil {
			internal.RealPanic(err)
		}
		namePrompt := &survey.Input{
			Message: "리소스 이름을 입력하세요:",
		}
		if err := survey.AskOne(namePrompt, &resourceName); err != nil {
			internal.RealPanic(err)
		}
		nsPrompt := &survey.Input{
			Message: "네임스페이스를 입력하세요 (선택사항):",
		}
		if err := survey.AskOne(nsPrompt, &namespace); err != nil {
			internal.RealPanic(err)
		}
		err := internal.DeleteKubectl(ctx, resourceType, resourceName, namespace)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
	}
}

func init() {
	rootCmd.AddCommand(startEksCommand)
}
