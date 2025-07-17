package cmd

import (
	"context"
	"strings"

	"github.com/masuldev/mcl/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	s3Command = &cobra.Command{
		Use:   "s3",
		Short: "Exec `s3 list` under AWS with interactive CLI",
		Long:  "Exec `s3 list` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				bucket *internal.S3Bucket
				object *internal.S3Object
				err    error
			)
			ctx := context.Background()

			// 버킷 선택
			argBucket := strings.TrimSpace(viper.GetString("s3-bucket"))
			if argBucket != "" {
				buckets, err := internal.FindS3Buckets(ctx, *credential.awsConfig)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				for _, b := range buckets {
					if b.Name == argBucket {
						bucket = b
						break
					}
				}
			}

			if bucket == nil {
				bucket, err = internal.AskS3Bucket(ctx, *credential.awsConfig)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			// 객체 선택 (선택사항)
			argObject := strings.TrimSpace(viper.GetString("s3-object"))
			argPrefix := strings.TrimSpace(viper.GetString("s3-prefix"))

			if argObject != "" {
				objects, err := internal.FindS3Objects(ctx, *credential.awsConfig, bucket.Name, argPrefix)
				if err != nil {
					internal.RealPanic(internal.WrapError(err))
				}
				for _, obj := range objects {
					if obj.Key == argObject {
						object = obj
						break
					}
				}
			} else if viper.GetBool("s3-list-objects") {
				object, err = internal.AskS3Object(ctx, *credential.awsConfig, bucket.Name, argPrefix)
				if err != nil {
					internal.RealPanic(err)
				}
			}

			// 결과 출력
			if object != nil {
				internal.PrintS3Object("s3", credential.awsConfig.Region, bucket.Name, object.Key,
					internal.FormatBytes(object.Size), object.LastModified.Format("2006-01-02 15:04:05"))
			} else {
				internal.PrintS3Bucket("s3", credential.awsConfig.Region, bucket.Name,
					bucket.CreationDate.Format("2006-01-02 15:04:05"))
			}
		},
	}
)

func init() {
	s3Command.Flags().StringP("bucket", "b", "", "s3 bucket name")
	s3Command.Flags().StringP("object", "o", "", "s3 object key")
	s3Command.Flags().String("prefix", "", "s3 object prefix")
	s3Command.Flags().BoolP("list-objects", "l", false, "list objects in bucket")

	viper.BindPFlag("s3-bucket", s3Command.Flags().Lookup("bucket"))
	viper.BindPFlag("s3-object", s3Command.Flags().Lookup("object"))
	viper.BindPFlag("s3-prefix", s3Command.Flags().Lookup("prefix"))
	viper.BindPFlag("s3-list-objects", s3Command.Flags().Lookup("list-objects"))

	rootCmd.AddCommand(s3Command)
}
