package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/fatih/color"
)

type (
	CloudFrontTarget struct {
		Name    string
		Id      string
		Domain  string
		Aliases []string
		Status  string
		Comment string
	}
)

func FindCloudFrontDistribution(ctx context.Context, cfg aws.Config) (map[string]*CloudFrontTarget, error) {
	client := cloudfront.NewFromConfig(cfg)
	table := make(map[string]*CloudFrontTarget)

	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{
		MaxItems: aws.Int32(maxOutputResults),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, distribution := range output.DistributionList.Items {
			name := aws.ToString(distribution.Comment)
			if name == "" {
				name = aws.ToString(distribution.Id)
			}

			// 대체 도메인(CNAME) 추출
			var aliases []string
			if distribution.Aliases != nil && distribution.Aliases.Items != nil {
				for _, alias := range distribution.Aliases.Items {
					aliases = append(aliases, alias)
				}
			}

			distributionId := aws.ToString(distribution.Id)
			table[distributionId] = &CloudFrontTarget{
				Name:    name,
				Id:      distributionId,
				Domain:  aws.ToString(distribution.DomainName),
				Aliases: aliases,
				Status:  aws.ToString(distribution.Status),
				Comment: aws.ToString(distribution.Comment),
			}
		}
	}

	return table, nil
}

func AskCloudFrontTarget(ctx context.Context, cfg aws.Config) (*CloudFrontTarget, error) {
	table, err := FindCloudFrontDistribution(ctx, cfg)
	if err != nil {
		return nil, err
	}

	displayMap := make(map[string]*CloudFrontTarget, len(table))
	options := make([]string, 0, len(table))
	for _, target := range table {
		// 대체도메인이 있으면 우선 표시, 없으면 기본 도메인 표시
		primaryDomain := target.Domain
		if len(target.Aliases) > 0 {
			primaryDomain = target.Aliases[0]
		}

		// name과 id가 같으면 id만 표시
		displayName := target.Name
		if target.Name == target.Id {
			displayName = target.Id
		} else {
			displayName = fmt.Sprintf("%s (%s)", target.Name, target.Id)
		}

		option := fmt.Sprintf("%s - %s", displayName, primaryDomain)
		options = append(options, option)
		displayMap[option] = target
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("CloudFront 배포를 찾을 수 없습니다")
	}

	prompt := &survey.Select{
		Message: "CloudFront 배포를 선택하세요:",
		Options: options,
	}

	selectKey := ""
	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return displayMap[selectKey], nil
}

func CreateCloudFrontInvalidation(ctx context.Context, cfg aws.Config, distributionId string) error {
	client := cloudfront.NewFromConfig(cfg)

	_, err := client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distributionId),
		InvalidationBatch: &types.InvalidationBatch{
			Paths: &types.Paths{
				Quantity: aws.Int32(1),
				Items:    []string{"/*"},
			},
			CallerReference: aws.String(fmt.Sprintf("mcl-invalidation-%d", time.Now().Unix())),
		},
	})

	return err
}

func PrintCloudFront(cmd, region, name, id, domain, status, comment string, aliases []string) {
	// 대체도메인이 있으면 우선 표시, 없으면 기본 도메인 표시
	primaryDomain := domain
	if len(aliases) > 0 {
		primaryDomain = aliases[0]
	}

	// name과 id가 같으면 id만 표시
	displayName := name
	if name == id {
		displayName = id
	}

	aliasInfo := ""
	if len(aliases) > 1 {
		aliasInfo = fmt.Sprintf(", additional aliases: %s", strings.Join(aliases[1:], ", "))
	}

	fmt.Printf("%s: region: %s, name: %s, domain: %s%s, status: %s, comment: %s\n",
		color.CyanString(cmd), color.YellowString(region), color.YellowString(displayName),
		color.BlueString(primaryDomain), color.CyanString(aliasInfo), color.GreenString(status),
		color.MagentaString(comment))
}

func PrintCloudFrontInvalidation(cmd, distributionId string) {
	fmt.Printf("%s: distribution id: %s, invalidation: /*\n",
		color.CyanString(cmd), color.YellowString(distributionId))
}

// 페이징을 지원하는 CloudFront 배포 조회
func FindCloudFrontDistributionWithPaging(ctx context.Context, cfg aws.Config, page int) (map[string]*CloudFrontTarget, error) {
	client := cloudfront.NewFromConfig(cfg)
	table := make(map[string]*CloudFrontTarget)

	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{
		MaxItems: aws.Int32(maxOutputResults),
	})

	// 지정된 페이지까지 스킵
	currentPage := 1
	for paginator.HasMorePages() && currentPage < page {
		_, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		currentPage++
	}

	// 현재 페이지 처리
	if paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, distribution := range output.DistributionList.Items {
			name := aws.ToString(distribution.Comment)
			if name == "" {
				name = aws.ToString(distribution.Id)
			}

			// 대체 도메인(CNAME) 추출
			var aliases []string
			if distribution.Aliases != nil && distribution.Aliases.Items != nil {
				for _, alias := range distribution.Aliases.Items {
					aliases = append(aliases, alias)
				}
			}

			distributionId := aws.ToString(distribution.Id)
			table[distributionId] = &CloudFrontTarget{
				Name:    name,
				Id:      distributionId,
				Domain:  aws.ToString(distribution.DomainName),
				Aliases: aliases,
				Status:  aws.ToString(distribution.Status),
				Comment: aws.ToString(distribution.Comment),
			}
		}
	}

	return table, nil
}
