package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/fatih/color"
)

type AuthMethod string

const (
	AuthMethodEnv   AuthMethod = "env"   // Environment Variables (aws-vault)
	AuthMethodLocal AuthMethod = "local" // ~/.aws/credentials, ~/.aws/config
	AuthMethodNone  AuthMethod = "none"  // No credentials found
)

type AwsAuth struct {
	Method  AuthMethod
	Profile string
	Region  string
	Config  aws.Config
}

// 인증 방식 자동 감지
func DetectAuthMethod() AuthMethod {
	// 1. Environment Variables 확인 (aws-vault 등)
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return AuthMethodEnv
	}

	// 2. 로컬 credentials 파일 확인
	if hasLocalCredentials() {
		return AuthMethodLocal
	}

	return AuthMethodNone
}

// 로컬 credentials 파일 존재 여부 확인
func hasLocalCredentials() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	credPath := filepath.Join(homeDir, ".aws", "credentials")
	_, err = os.Stat(credPath)
	return err == nil
}

// 새로운 AWS 인증 초기화
func NewAwsAuth() (*AwsAuth, error) {
	method := DetectAuthMethod()

	auth := &AwsAuth{
		Method: method,
	}

	switch method {
	case AuthMethodEnv:
		return auth.initFromEnv()
	case AuthMethodLocal:
		return auth.initFromLocal()
	case AuthMethodNone:
		return auth.initInteractive()
	default:
		return nil, fmt.Errorf("unknown auth method: %s", method)
	}
}

// Environment Variables로 초기화 (aws-vault 등)
func (a *AwsAuth) initFromEnv() (*AwsAuth, error) {
	// AWS SDK의 기본 설정 사용
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	a.Config = cfg
	a.Region = cfg.Region

	color.Green("✓ Using AWS credentials from environment variables")
	return a, nil
}

// 로컬 credentials로 초기화 (파일에서 직접 파싱)
func (a *AwsAuth) initFromLocal() (*AwsAuth, error) {
	profiles, err := GetAwsProfilesWithRegion()
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found in ~/.aws/credentials")
	}

	// 인터랙티브 선택
	var options []string
	for _, p := range profiles {
		label := p.Name
		if p.Region != "" {
			label = fmt.Sprintf("%s (%s)", p.Name, p.Region)
		}
		options = append(options, label)
	}

	var selected string
	prompt := &survey.Select{
		Message: "AWS 프로파일을 선택하세요:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, fmt.Errorf("profile selection failed: %w", err)
	}

	// 선택된 프로파일 찾기
	var selectedProfile AwsProfile
	for _, p := range profiles {
		label := p.Name
		if p.Region != "" {
			label = fmt.Sprintf("%s (%s)", p.Name, p.Region)
		}
		if label == selected {
			selectedProfile = p
			break
		}
	}

	// aws-vault와 동일한 환경변수 설정
	os.Setenv("AWS_ACCESS_KEY_ID", selectedProfile.AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", selectedProfile.SecretKey)
	if selectedProfile.SessionToken != "" {
		os.Setenv("AWS_SESSION_TOKEN", selectedProfile.SessionToken)
	}
	if selectedProfile.Region != "" {
		os.Setenv("AWS_REGION", selectedProfile.Region)
		os.Setenv("AWS_DEFAULT_REGION", selectedProfile.Region)
	}

	// AWS SDK의 기본 설정 사용
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load config for profile %s: %w", selectedProfile.Name, err)
	}

	a.Config = cfg
	a.Profile = selectedProfile.Name
	a.Region = cfg.Region

	color.Green("✓ Using AWS profile: %s (region: %s)", selectedProfile.Name, a.Region)
	return a, nil
}

// 인터랙티브 초기화 (인증 정보가 없는 경우)
func (a *AwsAuth) initInteractive() (*AwsAuth, error) {
	color.Yellow("⚠️  No AWS credentials found")
	fmt.Println("Please provide AWS credentials:")

	var accessKey, secretKey, region string

	prompt := &survey.Input{
		Message: "AWS Access Key ID:",
	}
	if err := survey.AskOne(prompt, &accessKey); err != nil {
		return nil, fmt.Errorf("access key input failed: %w", err)
	}

	secretPrompt := &survey.Password{
		Message: "AWS Secret Access Key:",
	}
	if err := survey.AskOne(secretPrompt, &secretKey); err != nil {
		return nil, fmt.Errorf("secret key input failed: %w", err)
	}

	prompt = &survey.Input{
		Message: "AWS Region:",
		Default: "ap-northeast-2",
	}
	if err := survey.AskOne(prompt, &region); err != nil {
		return nil, fmt.Errorf("region input failed: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	a.Config = cfg
	a.Region = region

	color.Green("✓ Using manually entered AWS credentials")
	return a, nil
}

// AWS Config 반환
func (a *AwsAuth) GetConfig() aws.Config {
	return a.Config
}

// Region 반환
func (a *AwsAuth) GetRegion() string {
	return a.Region
}
