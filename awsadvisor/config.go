package awsadvisor


import (
    "fmt"
    "log"
    "io/ioutil"
    "path/filepath"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ec2"
    "github.com/aws/aws-sdk-go/service/elb"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
    "github.com/aws/aws-sdk-go/service/iam"
    "github.com/aws/aws-sdk-go/service/autoscaling"
    "github.com/aws/aws-sdk-go/service/cloudtrail"

    "gopkg.in/yaml.v2"
)



type AWSClient struct {
  cloudwatchconn     *cloudwatch.CloudWatch
  ec2conn            *ec2.EC2
  elbconn            *elb.ELB
  billingconn        *cloudwatch.CloudWatch
  region             string
  iamconn            *iam.IAM
  asgconn            *autoscaling.AutoScaling
  ctrailconn         *cloudtrail.CloudTrail
}


type Config struct {
  Awsconf string
  Profile string
  Region  string
  Cfgfile string
  Conf    yamlConfig
}


type yamlConfig struct {
  Security_groups []*SecurityGroup
  Instances []*Instance
  Public_instances []*PublicInst
  Approved_amis []*Ami
  Iam struct {
    Users [] *User
    Roles [] *Role
    General struct {
      Expire_pass_last_login int16
      Expire_key_last_use int16
    }

  }
  Cloudtrail struct {
    Check_region bool
  }
  
}

type User struct {
  Username string
}

type Role struct {
  Name string
}

type Ami struct {
  Ami_id string
}


type SecurityGroup struct {
  Securitygroup string
  Id string
  Proto string
  Port int64
}

type Instance struct {
  Name string
  Min float32
  Max float32
}

type PublicInst struct {
  Role string
}



func (c *Config) Connect() interface{} {


  c.readConf()

  var client AWSClient
  
  awsConfig := new(aws.Config)
  us1Config := new(aws.Config)

  if len(c.Profile)>0 {
    awsConfig = &aws.Config{
      Credentials: credentials.NewSharedCredentials(c.Awsconf, fmt.Sprintf("profile %s", c.Profile)),
      Region:      aws.String(c.Region),
      MaxRetries:  aws.Int(3),
    }

    us1Config =  &aws.Config{
      Credentials: credentials.NewSharedCredentials(c.Awsconf, fmt.Sprintf("profile %s", c.Profile)),
      Region:      aws.String("us-east-1"),
      MaxRetries:  aws.Int(3),
    }

  } else {
    // use instance role
    awsConfig = &aws.Config{
      Region:      aws.String(c.Region),
    }

    us1Config = &aws.Config{
      Region:      aws.String("us-east-1"),
    }

  }


  sess := session.New(awsConfig)

  us1sess := session.New(us1Config)

  client.cloudwatchconn = cloudwatch.New(sess)

  client.ec2conn = ec2.New(sess)

  client.elbconn = elb.New(sess)

  client.billingconn = cloudwatch.New(us1sess)

  client.iamconn = iam.New(sess)

  client.asgconn = autoscaling.New(sess)

  client.ctrailconn = cloudtrail.New(sess)


  return &client

}

func (c *Config) readConf() {

  configFile, _ := filepath.Abs(c.Cfgfile)
  yamlConf, file_err := ioutil.ReadFile(configFile)

  if (file_err != nil) {
    log.Println("[ERROR] File does not exist: ", file_err)
  }

  yaml_err := yaml.Unmarshal(yamlConf, &c.Conf)

  if (yaml_err != nil) {
    panic(yaml_err)
  }

}


