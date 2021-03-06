//go:build !generate
// +build !generate

package keyvaluetags

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	awsbase "github.com/hashicorp/aws-sdk-go-base"
)

// Custom S3 tag service update functions using the same format as generated code.

// S3BucketListTags lists S3 bucket tags.
// The identifier is the bucket name.
func S3BucketListTags(conn *s3.S3, identifier string) (KeyValueTags, error) {
	input := &s3.GetBucketTaggingInput{
		Bucket: aws.String(identifier),
	}

	output, err := conn.GetBucketTagging(input)

	// S3 API Reference (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketTagging.html)
	// lists the special error as NoSuchTagSetError, however the existing logic used NoSuchTagSet
	// and the AWS Go SDK has neither as a constant.
	if awsbase.IsAWSErr(err, "NoSuchTagSet", "") {
		return New(nil), nil
	}

	if err != nil {
		return New(nil), err
	}

	return S3KeyValueTags(output.TagSet), nil
}

// S3BucketUpdateTags updates S3 bucket tags.
// The identifier is the bucket name.
func S3BucketUpdateTags(conn *s3.S3, identifier string, oldTagsMap interface{}, newTagsMap interface{}) error {
	oldTags := New(oldTagsMap)
	newTags := New(newTagsMap)

	// We need to also consider any existing system tags.
	allTags, err := S3BucketListTags(conn, identifier)

	if err != nil {
		return fmt.Errorf("error listing resource tags (%s): %w", identifier, err)
	}

	sysTags := allTags.Removed(allTags.IgnoreAws())

	if len(newTags)+len(sysTags) > 0 {
		input := &s3.PutBucketTaggingInput{
			Bucket: aws.String(identifier),
			Tagging: &s3.Tagging{
				TagSet: newTags.Merge(sysTags).S3Tags(),
			},
		}

		_, err := conn.PutBucketTagging(input)

		if err != nil {
			return fmt.Errorf("error setting resource tags (%s): %w", identifier, err)
		}
	} else if len(oldTags) > 0 && len(sysTags) == 0 {
		input := &s3.DeleteBucketTaggingInput{
			Bucket: aws.String(identifier),
		}

		_, err := conn.DeleteBucketTagging(input)

		if err != nil {
			return fmt.Errorf("error deleting resource tags (%s): %w", identifier, err)
		}
	}

	return nil
}

// S3ObjectListTags lists S3 object tags.
func S3ObjectListTags(conn *s3.S3, bucket, key string) (KeyValueTags, error) {
	input := &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	output, err := conn.GetObjectTagging(input)

	if err != nil {
		return New(nil), err
	}

	return S3KeyValueTags(output.TagSet), nil
}

// S3ObjectUpdateTags updates S3 object tags.
func S3ObjectUpdateTags(conn *s3.S3, bucket, key string, oldTagsMap interface{}, newTagsMap interface{}) error {
	oldTags := New(oldTagsMap)
	newTags := New(newTagsMap)

	if len(newTags) > 0 {
		input := &s3.PutObjectTaggingInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Tagging: &s3.Tagging{
				TagSet: newTags.IgnoreAws().S3Tags(),
			},
		}

		_, err := conn.PutObjectTagging(input)

		if err != nil {
			return fmt.Errorf("error setting resource tags (%s/%s): %w", bucket, key, err)
		}
	} else if len(oldTags) > 0 {
		input := &s3.DeleteObjectTaggingInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		_, err := conn.DeleteObjectTagging(input)

		if err != nil {
			return fmt.Errorf("error deleting resource tags (%s/%s): %w", bucket, key, err)
		}
	}

	return nil
}
