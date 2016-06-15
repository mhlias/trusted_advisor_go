package awsadvisor

import (
    "fmt"
    "time"
    "bytes"
    "io"
    "encoding/csv"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/ec2"
    "github.com/aws/aws-sdk-go/service/iam"
    "github.com/aws/aws-sdk-go/service/cloudtrail"
)


type AwsAuth struct {

  Users *iam.ListUsersOutput
  UsersDetails *iam.GetAccountAuthorizationDetailsOutput
  CredReport []*UserReport
  last_err error

}

type UserReport struct {
  UserName string
  PassLastRotation time.Time
  Key1LastRotation time.Time
  Key2LastRotation time.Time
}



func (c Config) Sg_is_excluded(sg_id, proto string, port int64) bool {
    
    for _, sg := range c.Conf.Security_groups { 
      if( sg.Id == sg_id && sg.Proto == proto && sg.Port == port){
        return true
      }
    }

  return false
}

func (ec2 *EC2) Is_attached_to_instance(sg_id string) string {

  instances := ""

  for idx, _ := range ec2.Instances.Reservations {
    for _, inst := range ec2.Instances.Reservations[idx].Instances {
      for _, sg := range inst.SecurityGroups {
        if *sg.GroupId == sg_id {
          instances += fmt.Sprintf("%s,", *inst.InstanceId)
        }
      }
    }
  }

  return instances
}

func (ec2 *EC2) Is_attached_to_elb(sg_id string) string {

  elbs := ""

  for _, lb := range ec2.Elbs.LoadBalancerDescriptions {
    for _, sg := range lb.SecurityGroups {
      if *sg == sg_id {
        elbs += fmt.Sprintf("%s,", *lb.LoadBalancerName)
      }
    }
  }

  return elbs
}


func (c Config) Instance_is_excluded(role string) bool {
 
    for _, inst := range c.Conf.Public_instances { 
      if( inst.Role == role){
        return true
      }
    }

  return false
}


func (c Config) User_is_allowed(user string) bool {
 
    for _, u := range c.Conf.Iam.Users { 
      if( u.Username == user){
        return true
      }
    }

  return false
}


func (c Config) Ami_is_approved(ami string) bool {
 
    for _, a := range c.Conf.Approved_amis { 
      if( a.Ami_id == ami){
        return true
      }
    }

  return false
}


func (i *AwsAuth) CheckPasswordPolicy(meta interface{}) string {


  issues := ""

  resp, err := meta.(*AWSClient).iamconn.GetAccountPasswordPolicy(nil)

  if err!=nil {
    panic(err)
  }

  if *resp.PasswordPolicy.MinimumPasswordLength < 14 {
    issues += "Minimum Password length less than 14 chars,"
  }

  if resp.PasswordPolicy.PasswordReusePrevention == nil || *resp.PasswordPolicy.PasswordReusePrevention < 3 {
    issues += "Last 3 Passwords can be reused,"
  }

  if *resp.PasswordPolicy.RequireUppercaseCharacters || *resp.PasswordPolicy.RequireNumbers || *resp.PasswordPolicy.RequireSymbols  {
    issues += "Password Policy doesn't require Uppercase Letters, Numbers and Symbols,"
  }  

  if resp.PasswordPolicy.MaxPasswordAge == nil || *resp.PasswordPolicy.MaxPasswordAge < 90 {
    issues += "Passwords don't expire after at least 90 days,"
  }


  return issues

}


func (i *AwsAuth) GetUsers(meta interface{}) {

  params := &iam.ListUsersInput{}

  i.Users, i.last_err = meta.(*AWSClient).iamconn.ListUsers(params)

  if i.last_err != nil {
    panic(i.last_err)
  }

}


func (i *AwsAuth) GetUsersDetails(meta interface{}) {

  i.UsersDetails, i.last_err = meta.(*AWSClient).iamconn.GetAccountAuthorizationDetails(nil)

  if i.last_err != nil {
    panic(i.last_err)
  }

}


func (i *AwsAuth) GetCredentialsReport(meta interface{}) {

  _, err := meta.(*AWSClient).iamconn.GenerateCredentialReport(nil)

  if err == nil {
    resp2, err2 := meta.(*AWSClient).iamconn.GetCredentialReport(nil)

    if err2 == nil {

     

      reader := csv.NewReader(bytes.NewReader(resp2.Content))
      reader.Comma = ','

      for {
        
        record, err := reader.Read()
        // end-of-file is fitted into err
        if err == io.EOF {
          break
        } else if err != nil {
          fmt.Println("Error:", err)
          return
        }

        var t1,t2,t3 time.Time
        
        if record[5] == "N/A" {
          t1 , _ = time.Parse(time.RFC3339, record[2])
        } else {
          t1, _ = time.Parse(time.RFC3339, record[5])
        }

        if record[9] == "N/A" {
          t2, _ = time.Parse(time.RFC3339, record[2])
        } else {
          t2, _ = time.Parse(time.RFC3339, record[9])
        }

        if record[14] == "N/A" {
          t3, _ = time.Parse(time.RFC3339, record[2])
        } else {
          t3, _ = time.Parse(time.RFC3339, record[14])
        }

        

        i.CredReport = append(i.CredReport, &UserReport{UserName:record[0], PassLastRotation: t1, Key1LastRotation: t2, Key2LastRotation: t3})

      }

    }


  }


}



func (a *EC2) GetSgs(meta interface{}) {

  params := &ec2.DescribeSecurityGroupsInput{}
  a.Security_groups, a.last_err= meta.(*AWSClient).ec2conn.DescribeSecurityGroups(params)

  if a.last_err != nil {
    panic(a.last_err)
    return
  }

}

func (i *AwsAuth) User_has_mfa(meta interface{}, username string) bool {

  params := &iam.ListMFADevicesInput{
    UserName: aws.String(username),
  }
  resp, err := meta.(*AWSClient).iamconn.ListMFADevices(params)

  if err != nil {
    return false
  }

  if len(resp.MFADevices)<1 {
    return false
  }

  return true

}

func (i *AwsAuth) User_has_expired_key(meta interface{}, username string, exp_days int16) string {

  keys_expired := ""

  params := &iam.ListAccessKeysInput{
    UserName: aws.String(username),
  }


  resp, err := meta.(*AWSClient).iamconn.ListAccessKeys(params)

  if err != nil {
    return keys_expired
  }

  if len(resp.AccessKeyMetadata)>0 {
    
    for _, ak := range resp.AccessKeyMetadata {

      if *ak.Status == "Active" {

        params2 := &iam.GetAccessKeyLastUsedInput{
          AccessKeyId: aws.String(*ak.AccessKeyId), // Required
        }
        resp2, _ := meta.(*AWSClient).iamconn.GetAccessKeyLastUsed(params2)

        if resp2.AccessKeyLastUsed.LastUsedDate != nil {
          if time.Now().Unix()-int64(exp_days*24*3600) > resp2.AccessKeyLastUsed.LastUsedDate.Unix() {
            keys_expired += fmt.Sprintf("%s,", *ak.AccessKeyId)
          }
        } else {
          if time.Now().Unix()-int64(exp_days*24*3600) > ak.CreateDate.Unix() {
            keys_expired += fmt.Sprintf("%s,", *ak.AccessKeyId)
          }
        }

      }

    }

  }

  return keys_expired


}

func (e *EC2) Cloudtrail_is_enabled(meta interface{}) bool {


  resp, err := meta.(*AWSClient).ctrailconn.DescribeTrails(nil)

  if err != nil {
    return false
  }

  if len(resp.TrailList)<1 {
    return false
  }

  for _, t := range resp.TrailList {

    params := &cloudtrail.GetTrailStatusInput{
      Name: aws.String(*t.Name), // Required
    }
    resp2, err2 := meta.(*AWSClient).ctrailconn.GetTrailStatus(params)

    if err2 != nil {
      return false
    }

    if *resp2.IsLogging {
      return true
    }

  }

  return false

}



func (i *AwsAuth) User_has_password(meta interface{}, username string) bool {

  params := &iam.GetLoginProfileInput{
    UserName: aws.String(username),
  }
  resp, err := meta.(*AWSClient).iamconn.GetLoginProfile(params)

  if err != nil {
    return false
  }

  if (resp.LoginProfile.CreateDate.Unix() > 0) {
    return true
  }

  return false

}