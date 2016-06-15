## Overview

A simple Go utility that provides an overview for your AWS account in terms of security, efficiency and billing.


### Features

This tool has 4 modes of operation:
  - cli which is the default and outputs everything in the console
  - sensu which provides output compatible with nagios/sensu plugins and can error and warn
  - graphite that provides some of the max estimated cloudwatch billing metrics
  - web which is ugly and experimental/PoC    

## Security
- Warns & lists instances that are in the public subnet except the ones with tag roles that are allowed to be in a public subnet as specified in the config
- Warns & lists security groups that contain rules for ingress traffic to the world except for the triplets [sg-id,protocol,port] that are whitelisted in the config
- Errors out & lists security groups that contain rules for ingress traffic to the world and attached to instaces or ELBs except for the triplets [sg-id,protocol,port] that are whitelisted in the config
- Warns & lists IAM users not approved in the config
- Warns & lists approved users that do not have mfa enabled
- Warns & lists approved users with a password that have not logged in for X amount of days as specified in the config
- Warns & lists approved users and access keys that have not been used for X amount of days as specified in the config
- Warns & lists users that have directly attached IAM policies.
- Warns & lists users that have password or access keys not rotated in the last 90 days
- Warns & lists instances that do not use one of the approved AMIs as specified in the config
- Warns about IAM password policy if not compliant with: Minimum Password length at least 14 chars, Last 3 Passwords cannot be reused, Password Policy requires Uppercase Letters Numbers and Symbols, Passwords expire after at least 90 days
- Sensu mode surfaces critical security messages first and supresses everything else until they are cleared.
- Errors out when CloudTrail logging is not enabled for current region if the configuration switch is set to true.

## Resiliency
- Warns & lists instances that are not part of an autoscaling group and therefore have no self-healing capabilities

## Costs/Billing 
- Provides estimated costs of instances based on the hours they are running and savings compared to the fully upfront paid resernved instances based on expected usage
- Provides estimated costs of EBS volumes based on the months in usage
- Provides max estimated clouwatch billing summary costs for EC2, S3, Route53, AWS Traffic

## Setup


### Setup Requirements

The project comes with 2 yaml files as examples:
  - config.yaml
  
  Which should contain: 
    - the whitelisted security groups which are allowed to be open to the world (whitelisted) in the form of a triplet [sg-id, protocol, destination_port]
    - the Name tag of the instances that should be allowed to be in the public subnet
    - Approved IAM users for the account
    - Allowed days of inactivity for IAM users with a password before the security check produces a warning
    - List of approved AMIs to be used in any instance in the account and not using one produces an error

  - pricing.yaml

  Which should contain the pricing scheme for per AWS Region and instance type for both ondemand and reserved instances (Currently contains pricing for most instance types in region eu-west-1)


On the instance that it will run you will need to have an AWSCLI profile file for the accounts you want to run the tool on.


### Beginning with trusted_advisor_go

## Usage

The tool accepts the following parameters:

```
  -config string
      Location of aws config profiles to use
  -accountname string
      Name of the aws account for graphite id purposes
  -metric string
      Metric to provide when Mode is graphite (default "AmazonEC2")
  -mode string
      Mode to run in [cli, sensu, graphite, web] (default "cli")
  -profile string
      Name of the profile to use
  -region string
      AWS Region to use (default "eu-west-1")
```

To run it on the cli for an account with profile name project-dev:

```
trusted_advisor_go -config /path/to/aws/profile/config -profile project-dev

```

To output for a sensu check the command would be:

```
trusted_advisor_go -config /path/to/aws/profile/config -profile project-dev -mode sensu

```

To output estimated billing metrics for graphite:

```
trusted_advisor_go -config /path/to/aws/profile/config -profile project-dev -mode graphite -metric AmazonEC2

```

The graphite possible metric values are: [AmazonEC2, AmazonS3, AmazonRoute53, AWSDataTransfer] and the namespace in the form:

```
aws.account.project-dev.billing.metric_name

```

All of the above can be performed using an IAM instance role by using the parameter -userole and with the following access profile in IAM:

```
"Action": [
    "ec2:Describe*",
    "elasticloadbalancing:Describe*",
    "autoscaling:Describe*",
    "cloudwatch:GetMetricData",
    "cloudwatch:GetMetricStatistics",
    "cloudwatch:ListMetrics",
    "iam:GenerateCredentialReport",
    "iam:GetAccountSummary",
    "iam:GetCredentialReport",
    "iam:GetUser",
    "iam:GetUserPolicy",
    "iam:ListAccessKeys",
    "iam:ListUserPolicies",
    "iam:GetAccountAuthorizationDetails",
    "iam:ListUsers"
],

```

### Limitations

The Web mode is a very ugly test/PoC which may or may not improve in the future.
This tool is a PoC and it will improve in the future and become more versatile in the ways it can be used.







