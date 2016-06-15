package awsadvisor

import(
    "github.com/aws/aws-sdk-go/service/ec2"
)

func GetTag(tags []*ec2.Tag, key string) string {
  for _, tag := range tags {
    if key == *tag.Key {
      return *tag.Value
    }
  }
  return "not_found"
}