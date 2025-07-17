package internal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AwsProfile struct {
	Name         string
	Region       string
	AccessKey    string
	SecretKey    string
	SessionToken string
}

// ~/.aws/config 파일에서 프로파일별 region을 읽어오는 함수
func ListAwsProfilesWithRegion() ([]AwsProfile, error) {
	credPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
	configPath := filepath.Join(os.Getenv("HOME"), ".aws", "config")

	// credentials에서 프로파일 이름만 추출
	credFile, err := os.Open(credPath)
	if err != nil {
		return nil, fmt.Errorf("credentials 파일 열기 실패: %w", err)
	}
	defer credFile.Close()

	credProfiles := make(map[string]struct{})
	scanner := bufio.NewScanner(credFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profile := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			credProfiles[profile] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("credentials 파일 읽기 실패: %w", err)
	}

	// config에서 region 추출
	configFile, err := os.Open(configPath)
	if err != nil {
		// config 파일이 없으면 region 없이 반환
		var result []AwsProfile
		for name := range credProfiles {
			result = append(result, AwsProfile{Name: name, Region: ""})
		}
		return result, nil
	}
	defer configFile.Close()

	profileRegion := make(map[string]string)
	var currentProfile string
	scanner = bufio.NewScanner(configFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if strings.HasPrefix(section, "profile ") {
				currentProfile = strings.TrimPrefix(section, "profile ")
			} else {
				currentProfile = section
			}
		} else if strings.HasPrefix(line, "region") && currentProfile != "" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				profileRegion[currentProfile] = strings.TrimSpace(parts[1])
			}
		}
	}

	// 결과 조합
	var result []AwsProfile
	for name := range credProfiles {
		region := profileRegion[name]
		result = append(result, AwsProfile{Name: name, Region: region})
	}
	return result, nil
}

// AWS credentials 파일 파싱
func ParseAwsCredentials() (map[string]AwsProfile, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	credPath := filepath.Join(homeDir, ".aws", "credentials")
	file, err := os.Open(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	profiles := make(map[string]AwsProfile)
	var currentProfile string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 빈 줄이나 주석 건너뛰기
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 프로파일 섹션 확인
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentProfile = strings.Trim(line, "[]")
			profiles[currentProfile] = AwsProfile{Name: currentProfile}
			continue
		}

		// 키-값 쌍 파싱
		if currentProfile != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				profile := profiles[currentProfile]
				switch key {
				case "aws_access_key_id":
					profile.AccessKey = value
				case "aws_secret_access_key":
					profile.SecretKey = value
				case "aws_session_token":
					profile.SessionToken = value
				}
				profiles[currentProfile] = profile
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading credentials file: %w", err)
	}

	return profiles, nil
}

// AWS config 파일 파싱
func ParseAwsConfig() (map[string]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	regions := make(map[string]string)
	var currentProfile string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 빈 줄이나 주석 건너뛰기
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 프로파일 섹션 확인
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			if strings.HasPrefix(section, "profile ") {
				currentProfile = strings.TrimPrefix(section, "profile ")
			} else if section == "default" {
				currentProfile = "default"
			}
			continue
		}

		// 키-값 쌍 파싱
		if currentProfile != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if key == "region" {
					regions[currentProfile] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return regions, nil
}

// AWS 프로파일 목록과 리전 정보 가져오기 (credentials 정보 포함)
func GetAwsProfilesWithRegion() ([]AwsProfile, error) {
	credentials, err := ParseAwsCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	regions, err := ParseAwsConfig()
	if err != nil {
		// config 파일이 없어도 credentials만으로는 동작 가능
		fmt.Printf("[DEBUG] Config file not found or error: %v\n", err)
	}

	var profiles []AwsProfile
	for name, profile := range credentials {
		// 리전 정보 추가
		if region, exists := regions[name]; exists {
			profile.Region = region
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// 선택된 프로파일/리전을 환경변수로 설정
func SetAwsProfileEnv(profile, region string) {
	os.Setenv("AWS_PROFILE", profile)
	if region != "" {
		os.Setenv("AWS_REGION", region)
		os.Setenv("AWS_DEFAULT_REGION", region)
	}
}

func NewConfig(ctx context.Context, key, secret, session, region, roleArn string) (aws.Config, error) {
	fmt.Println("[DEBUG] NewConfig called")
	var (
		opts []func(*config.LoadOptions) error
		cfg  aws.Config
		err  error
	)

	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParam)
	}

	// region이 비어있으면 환경변수에서 읽어오기
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
	}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	// 자격 증명 옵션 추가: key와 secret이 모두 제공된 경우에만 추가
	if key != "" && secret != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, session)))
	}

	// 한 번의 호출로 기본 구성 로드
	cfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	// 디버깅: 환경변수와 SDK config 값 출력
	fmt.Printf("[DEBUG] AWS_PROFILE=%s\n", os.Getenv("AWS_PROFILE"))
	fmt.Printf("[DEBUG] AWS_REGION=%s\n", os.Getenv("AWS_REGION"))
	fmt.Printf("[DEBUG] AWS_DEFAULT_REGION=%s\n", os.Getenv("AWS_DEFAULT_REGION"))
	fmt.Printf("[DEBUG] SDK config.Region=%s\n", cfg.Region)

	// access key, secret key 디버깅
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		fmt.Printf("[DEBUG] Credentials.Retrieve() error: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] AccessKeyID=%s\n", creds.AccessKeyID)
		fmt.Printf("[DEBUG] SecretAccessKey=%s\n", creds.SecretAccessKey)
	}

	// roleArn이 제공된 경우 AssumeRole Provider 적용
	if roleArn != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(stsClient, roleArn))
	}

	return cfg, nil
}

func NewSharedConfig(ctx context.Context, profile string, sharedConfigFiles, sharedCredentialsFiles []string) (aws.Config, error) {
	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParam)
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithSharedConfigFiles(sharedConfigFiles),
		config.WithSharedCredentialsFiles(sharedCredentialsFiles),
	)

	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	return cfg, nil
}
