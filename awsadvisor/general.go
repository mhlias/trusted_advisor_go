package awsadvisor

import (
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/ec2"
    "github.com/aws/aws-sdk-go/service/elb"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
    "github.com/aws/aws-sdk-go/service/autoscaling"
    
)


type EC2 struct {

  Instances *ec2.DescribeInstancesOutput
  Elbs *elb.DescribeLoadBalancersOutput
  Autoscaled *autoscaling.DescribeAutoScalingInstancesOutput
  Security_groups *ec2.DescribeSecurityGroupsOutput
  Volumes *ec2.DescribeVolumesOutput
  last_err error

}


type Cwatch struct {

  Metric, Namespace, DimensionName, DimensionVal string 
  Starting int
  Period int64

}



func (e *EC2) GetInstances(meta interface{}) {

  e.Instances, e.last_err = meta.(*AWSClient).ec2conn.DescribeInstances(nil)
  if e.last_err != nil {
    panic(e.last_err)
  }

  e.Autoscaled, e.last_err = meta.(*AWSClient).asgconn.DescribeAutoScalingInstances(nil)
  if e.last_err != nil {
    panic(e.last_err)
  }


}

func (e *EC2) GetElbs(meta interface{}) {

  e.Elbs, e.last_err = meta.(*AWSClient).elbconn.DescribeLoadBalancers(nil)
  if e.last_err != nil {
    panic(e.last_err)
  }

}

func (e *EC2) GetVolumes(meta interface{}) {

  params := &ec2.DescribeVolumesInput{}
  
  e.Volumes, e.last_err = meta.(*AWSClient).ec2conn.DescribeVolumes(params)

  if e.last_err != nil {
    panic(e.last_err)
  }

}

func (c *Cwatch) GetMetrics(meta interface{}) (*cloudwatch.GetMetricStatisticsOutput, error) {

  metric_params := &cloudwatch.GetMetricStatisticsInput{
  EndTime:    aws.Time(time.Now()),
  MetricName: aws.String(c.Metric),
  Namespace:  aws.String(c.Namespace),
  Period:     aws.Int64(c.Period),
  StartTime:  aws.Time(time.Now().AddDate(0, 0, c.Starting)),
  Statistics: []*string{
  aws.String("Average"),
  },
  Dimensions: []*cloudwatch.Dimension{
    {
      Name:  aws.String(c.DimensionName),
      Value: aws.String(c.DimensionVal),
    },
  },
  Unit: aws.String("Count"),
  }

  metrics_resp, metrics_err := meta.(*AWSClient).cloudwatchconn.GetMetricStatistics(metric_params)

  return metrics_resp, metrics_err


}


func (e *EC2) IsAutoscaled(id string) bool {

  for _,inst := range e.Autoscaled.AutoScalingInstances {

    if *inst.InstanceId == id {
      return true
    }

  }

  return false


}







