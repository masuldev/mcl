package internal

type (
	RdsTarget struct {
		Name     string
		Endpoint string
	}
)

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
