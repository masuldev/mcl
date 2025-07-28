package internal

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type (
	RdsTarget struct {
		Name     string
		Endpoint string
		Id       string
		Status   string
		Engine   string
	}
)

func FindRdsInstance(ctx context.Context, cfg aws.Config) (map[string]*RdsTarget, error) {
	client := rds.NewFromConfig(cfg)
	table := make(map[string]*RdsTarget)

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(maxOutputResults),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, dbInstance := range output.DBInstances {
			var name string
			for _, tag := range dbInstance.TagList {
				if aws.ToString(tag.Key) == "Name" {
					name = aws.ToString(tag.Value)
					break
				}
			}
			if name == "" {
				name = aws.ToString(dbInstance.DBInstanceIdentifier)
			}

			instanceId := aws.ToString(dbInstance.DBInstanceIdentifier)
			table[instanceId] = &RdsTarget{
				Name:     name,
				Endpoint: aws.ToString(dbInstance.Endpoint.Address),
				Id:       instanceId,
				Status:   aws.ToString(dbInstance.DBInstanceStatus),
				Engine:   aws.ToString(dbInstance.Engine),
			}
		}
	}

	return table, nil
}

func FindDBInstancesIds(ctx context.Context, cfg aws.Config) ([]string, error) {
	table, err := FindRdsInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, target := range table {
		ids = append(ids, target.Id)
	}
	return ids, nil
}

func AskRdsTarget(ctx context.Context, cfg aws.Config) (*RdsTarget, error) {
	table, err := FindRdsInstance(ctx, cfg)
	if err != nil {
		return nil, err
	}

	displayMap := make(map[string]*RdsTarget, len(table))
	options := make([]string, 0, len(table))
	for _, target := range table {
		option := fmt.Sprintf("%s (%s) - %s", target.Name, target.Id, target.Engine)
		options = append(options, option)
		displayMap[option] = target
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("RDS 인스턴스를 찾을 수 없습니다")
	}

	prompt := &survey.Select{
		Message: "RDS 인스턴스를 선택하세요:",
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

func PrintRds(cmd, region, name, id, endpoint, status, engine string) {
	LogAwsServiceDetail(cmd, region, name, id, endpoint, status, engine)
}

// 페이징을 지원하는 RDS 인스턴스 조회
func FindRdsInstanceWithPaging(ctx context.Context, cfg aws.Config, page int) (map[string]*RdsTarget, error) {
	client := rds.NewFromConfig(cfg)
	table := make(map[string]*RdsTarget)

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(maxOutputResults),
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

		for _, dbInstance := range output.DBInstances {
			var name string
			for _, tag := range dbInstance.TagList {
				if aws.ToString(tag.Key) == "Name" {
					name = aws.ToString(tag.Value)
					break
				}
			}
			if name == "" {
				name = aws.ToString(dbInstance.DBInstanceIdentifier)
			}

			instanceId := aws.ToString(dbInstance.DBInstanceIdentifier)
			table[instanceId] = &RdsTarget{
				Name:     name,
				Endpoint: aws.ToString(dbInstance.Endpoint.Address),
				Id:       instanceId,
				Status:   aws.ToString(dbInstance.DBInstanceStatus),
				Engine:   aws.ToString(dbInstance.Engine),
			}
		}
	}

	return table, nil
}

//func FindRdsInstance(ctx context.Context, cfg aws.Config) (map[string]*RdsTarget, error) {
//	var (
//		client = rds.NewFromConfig(cfg)
//		table = make(map[string]*RdsTarget)
//		outputFunc = func(table map[string]*RdsTarget, output *rds.DescribeDBInstancesOutput) {
//			for _, DBInstance := range output.DBInstances {
//				DBInstance.
//			}
//		}
//	)
//
//	DBInstances, err :=
//}
//
//func FindDBInstancesIds(ctx context.Context, cfg aws.Config) ([]string, error) {
//	var (
//		DBInstances []string
//		client = rds.NewFromConfig(cfg)
//		outputFunc = func() {}
//	)
//}
