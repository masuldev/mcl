package cmd

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
)

var ssmCmd = &cobra.Command{
	Use:   "ssm",
	Short: "EC2 인스턴스에 SSM으로 접속",
	Long:  "EC2 인스턴스 목록에서 선택 후 SSM(Session Manager)으로 접속합니다.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		cfg := GetGlobalAwsConfig()
		if cfg == nil {
			internal.RealPanic(fmt.Errorf("AWS credentials not configured"))
		}

		// SSM 클라이언트 설치 여부 확인
		ok, missing := internal.CheckSSMClientInstalled()
		if !ok {
			internal.LogWarning("SSM 클라이언트가 설치되어 있지 않습니다.")
			internal.PrintSSMInstallGuide(missing)
			return
		}

		// EC2 인스턴스 목록 조회
		instances, err := internal.FindInstance(ctx, *cfg)
		if err != nil {
			internal.RealPanic(internal.WrapError(err))
		}
		if len(instances) == 0 {
			internal.LogWarning("실행 중인 EC2 인스턴스가 없습니다.")
			return
		}

		// 인스턴스 선택
		var options []string
		idToInstance := map[string]*internal.Target{}
		for _, inst := range instances {
			label := fmt.Sprintf("%s (%s, %s)", inst.Name, inst.Id, inst.PrivateIp)
			options = append(options, label)
			idToInstance[label] = inst
		}
		var selected string
		prompt := &survey.Select{
			Message: "SSM으로 접속할 인스턴스를 선택하세요:",
			Options: options,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			internal.RealPanic(err)
		}
		inst := idToInstance[selected]

		// SSM 세션 연결
		internal.LogInfo("SSM 세션을 시작합니다: %s (%s)", inst.Name, inst.Id)
		err = internal.StartSSMSession(ctx, inst.Id, cfg.Region)
		if err != nil {
			internal.RealPanic(fmt.Errorf("SSM 세션 연결 실패: %w", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(ssmCmd)
}
