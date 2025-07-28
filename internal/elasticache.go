package internal

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
)

type (
	ElastiCacheTarget struct {
		Name     string
		Endpoint string
		Id       string
		Status   string
		Engine   string
		Port     int32
	}
)

func FindElastiCacheCluster(ctx context.Context, cfg aws.Config) (map[string]*ElastiCacheTarget, error) {
	client := elasticache.NewFromConfig(cfg)
	table := make(map[string]*ElastiCacheTarget)

	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{
		MaxRecords: aws.Int32(maxOutputResults),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range output.CacheClusters {
			name := aws.ToString(cluster.CacheClusterId)
			clusterId := aws.ToString(cluster.CacheClusterId)

			var endpoint string
			var port int32

			// ConfigurationEndpoint가 nil이 아닌 경우에만 접근
			if cluster.ConfigurationEndpoint != nil {
				endpoint = aws.ToString(cluster.ConfigurationEndpoint.Address)
				port = aws.ToInt32(cluster.ConfigurationEndpoint.Port)
			}

			table[clusterId] = &ElastiCacheTarget{
				Name:     name,
				Endpoint: endpoint,
				Id:       clusterId,
				Status:   aws.ToString(cluster.CacheClusterStatus),
				Engine:   aws.ToString(cluster.Engine),
				Port:     port,
			}
		}
	}

	return table, nil
}

// 페이징을 지원하는 ElastiCache 클러스터 조회
func FindElastiCacheClusterWithPaging(ctx context.Context, cfg aws.Config, page int) (map[string]*ElastiCacheTarget, error) {
	client := elasticache.NewFromConfig(cfg)
	table := make(map[string]*ElastiCacheTarget)

	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{
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

		for _, cluster := range output.CacheClusters {
			name := aws.ToString(cluster.CacheClusterId)
			clusterId := aws.ToString(cluster.CacheClusterId)

			var endpoint string
			var port int32

			// ConfigurationEndpoint가 nil이 아닌 경우에만 접근
			if cluster.ConfigurationEndpoint != nil {
				endpoint = aws.ToString(cluster.ConfigurationEndpoint.Address)
				port = aws.ToInt32(cluster.ConfigurationEndpoint.Port)
			}

			table[clusterId] = &ElastiCacheTarget{
				Name:     name,
				Endpoint: endpoint,
				Id:       clusterId,
				Status:   aws.ToString(cluster.CacheClusterStatus),
				Engine:   aws.ToString(cluster.Engine),
				Port:     port,
			}
		}
	}

	return table, nil
}

func AskElastiCacheTarget(ctx context.Context, cfg aws.Config) (*ElastiCacheTarget, error) {
	table, err := FindElastiCacheCluster(ctx, cfg)
	if err != nil {
		return nil, err
	}

	displayMap := make(map[string]*ElastiCacheTarget, len(table))
	options := make([]string, 0, len(table))
	for _, target := range table {
		option := fmt.Sprintf("%s (%s) - %s", target.Name, target.Id, target.Engine)
		options = append(options, option)
		displayMap[option] = target
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("ElastiCache 클러스터를 찾을 수 없습니다")
	}

	prompt := &survey.Select{
		Message: "ElastiCache 클러스터를 선택하세요:",
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

func PrintElastiCache(cmd, region, name, id, endpoint, status, engine string, port int32) {
	LogAwsServiceDetail(cmd, region, name, id, fmt.Sprintf("%s:%d", endpoint, port), status, engine)
}
