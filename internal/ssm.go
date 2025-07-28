package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// SSM 클라이언트 설치 여부 확인
func CheckSSMClientInstalled() (bool, []string) {
	missing := []string{}
	if _, err := exec.LookPath("aws"); err != nil {
		missing = append(missing, "awscli")
	}
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		missing = append(missing, "session-manager-plugin")
	}
	return len(missing) == 0, missing
}

// SSM 클라이언트 설치 안내 메시지
func PrintSSMInstallGuide(missing []string) {
	fmt.Println("[SSM 클라이언트 설치 안내]")
	for _, m := range missing {
		switch m {
		case "awscli":
			fmt.Println("- awscli: https://docs.aws.amazon.com/ko_kr/cli/latest/userguide/getting-started-install.html")
			fmt.Println("  macOS: brew install awscli")
		case "session-manager-plugin":
			fmt.Println("- session-manager-plugin: https://docs.aws.amazon.com/ko_kr/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
			fmt.Println("  macOS: brew install --cask session-manager-plugin")
		}
	}
}

// SSM 세션 연결
func StartSSMSession(ctx context.Context, instanceId, region string) error {
	cmd := exec.CommandContext(ctx, "aws", "ssm", "start-session", "--target", instanceId, "--region", region)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
