package main

import (
	"fmt"
	"os"

	"github.com/masuldev/mcl/cmd"
	"github.com/masuldev/mcl/internal"
)

var mclVersion string

func main() {
	// 새로운 AWS 인증 시스템 초기화
	auth, err := internal.NewAwsAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "AWS 인증 초기화 실패: %v\n", err)
		os.Exit(1)
	}

	// 전역 AWS Config 설정
	cmd.SetGlobalAwsConfig(auth.GetConfig())
	cmd.SetGlobalRegion(auth.GetRegion())

	cmd.Execute(mclVersion)
}
