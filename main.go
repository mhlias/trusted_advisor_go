package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/mhlias/trusted_advisor_go/awsadvisor"
)



// Output structure
var output = &awsadvisor.Output{}

type Page struct {
	Header  string
	Footer  string
	Column1 template.HTML
	Column2 template.HTML
	Column3 template.HTML
	Column4 template.HTML
}

func main() {

	awsconfigPtr := flag.String("config", "", "Location of aws config profiles to use")
	profilePtr := flag.String("profile", "", "Name of the profile to use")
	regionPtr := flag.String("region", "eu-west-1", "AWS Region to use")
	modePtr := flag.String("mode", "cli", "Mode to run in [cli, sensu, graphite, web]")
	metricPtr := flag.String("metric", "AmazonEC2", "Metric to provide when Mode is graphite")
	userolePtr := flag.Bool("userole", false, "Use instance role instead of aws config file")
	accountnamePtr := flag.String("accountname", "", "Name of the aws account for graphite id purposes")
	cfgfilePtr := flag.String("cfgfile", "config.yaml", "Location of custom config yaml file")
	pricelistPtr := flag.String("pricelist", "pricing.yaml", "Location of AWS Pricing list yaml file")

	c1 := make(chan int)
  c2 := make(chan int)

	flag.Parse()

	if (len(*awsconfigPtr) <= 0 || len(*profilePtr) <= 0) && !*userolePtr {
		fmt.Println("Please provide the following required parameters:")
		flag.PrintDefaults()
		return
	}

	accountname := "aws_account"

	if *modePtr == "graphite" {

		if len(*accountnamePtr) > 0 {
			accountname = *accountnamePtr
		} else if len(*profilePtr) > 0 {
			accountname = *profilePtr
		} else {
			errors.New("You need to specify at least a profile name or accountname when in graphite mode!")
			return
		}

	}

	cfg := &awsadvisor.Config{Awsconf: *awsconfigPtr, Profile: *profilePtr, Region: *regionPtr, Cfgfile: *cfgfilePtr}

	client := cfg.Connect()

	// Parse Amazon price list
	ec2costs := new(awsadvisor.EC2Costs)

	ec2costs.ParsePrices(*pricelistPtr)

	ec2 := new(awsadvisor.EC2)
	c := new(awsadvisor.Cwatch)

	output.Ecode = 0

	c.Period = 3600 * 24 * 7
	c.Starting = -7

	// CloudWatch for Billing
	billing := new(awsadvisor.Billing)

	if *modePtr == "sensu" || *modePtr == "cli" || *modePtr == "web" {

		go func() {

			for {

				output = &awsadvisor.Output{}

				go func() {

					ec2.GetInstances(client)

          c2 <- 1

					// pull out instance IDs:
					output.Info_msgs = append(output.Info_msgs, fmt.Sprintf("> Number of instances: %d", len(ec2.Instances.Reservations)))

					for idx, _ := range ec2.Instances.Reservations {
						for _, inst := range ec2.Instances.Reservations[idx].Instances {
							output.Info_msgs = append(output.Info_msgs, fmt.Sprintf("    - Instance ID: %s", *inst.InstanceId))
							/*c.Metric = "CPUCreditBalance"
							c.Namespace = "AWS/EC2"
							c.DimensionName = "InstanceId"
							c.DimensionVal = *inst.InstanceId
							metrics, metrics_err := c.GetMetrics(client)

							if metrics_err != nil {
								fmt.Println(metrics_err.Error())
								return
							}*/

							if !ec2.IsAutoscaled(*inst.InstanceId) {
								output.Info_msgs = append(output.Info_msgs, fmt.Sprintf("\tWARNING - Instance is not part of an Autoscaling group, no self healing available!!!"))
								output.SetWarn()
							}

              if !cfg.Ami_is_approved(*inst.ImageId) {
                output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("AMI: %s used on instance: %s is not in the approved list!\n", *inst.ImageId, *inst.InstanceId))
                output.SetError()
              }

							seconds_since_started := inst.LaunchTime.Unix()
							seconds_now := time.Now().Unix()

							hours_in_use := (seconds_now - seconds_since_started) / 3600

              tmp_platform := "linux"

              if inst.Platform != nil {
                tmp_platform = "windows"
              }
 

							instance_costs := ec2costs.GetEstimate(hours_in_use, *inst.InstanceType, cfg.Region, tmp_platform)

							output.Info_msgs = append(output.Info_msgs, fmt.Sprintf("\tInstance cost since it was launched %d hours ago: $%.2f\n\tYou would be making a saving with 1 year reserved instance with a cost of $%.2f if you use the instance %d hours more in a year and\n\tcompared to a 10h Ondemand instance (Ondemand - Reserved) the cost diff is: $ %0.2f\n\tYou would be making a saving with 3 year upfront reserved instance with a cost of $%.2f if you use the instance %d hours more in 3 years and\n\tcompared to a 10h Ondemand instance (Ondemand - Reserved) the cost diff is: $%0.2f\n", hours_in_use, instance_costs.Current_cost, instance_costs.Reserved_cost[365].Term_cost, instance_costs.Reserved_cost[365].Cost_saving_after, instance_costs.Reserved_cost[365].Diff_with_10h, instance_costs.Reserved_cost[1095].Term_cost, instance_costs.Reserved_cost[1095].Cost_saving_after, instance_costs.Reserved_cost[1095].Diff_with_10h))
							/*for _, datapoint := range metrics.Datapoints {
								if *datapoint.Average < 20 {
									output.Utilisation_msgs = append(output.Utilisation_msgs, fmt.Sprintf("Warning Instance %s cpu overutilized!!! Optimize or increase the instance type", *inst.InstanceId))
									output.SetWarn()
								} else if *datapoint.Average > 100 {
									output.Utilisation_msgs = append(output.Utilisation_msgs, fmt.Sprintf("Warning Instance %s cpu underutilized!!! Utilize better or reduce the instance type", *inst.InstanceId))
									output.SetWarn()
								}
							}*/

							name := awsadvisor.GetTag(inst.Tags, "Name")
							if inst.PublicIpAddress != nil {
								if len(*inst.PublicIpAddress) > 0 && !cfg.Instance_is_excluded(name) {
									output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("Instance: %s is in the Public Subnet!!!", *inst.InstanceId))
									output.SetError()
								}
							}
						}
					}

					c1 <- 1

				}()

				go func() {

					ec2.GetVolumes(client)

					for _, vl := range ec2.Volumes.Volumes {

						if *vl.State == "available" {
							seconds_since_started := vl.CreateTime.Unix()
							seconds_now := time.Now().Unix()

							months_in_use := int64(math.Ceil(float64((seconds_now - seconds_since_started) / (3600 * 24 * 30))))

							var iops int64

							iops = 0

							if vl.Iops != nil && *vl.VolumeType == "io1" {
								iops = *vl.Iops
							}

							vol_costs := ec2costs.GetEBSCost(cfg.Region, *vl.VolumeType, *vl.Size, months_in_use, iops)

							output.Utilisation_msgs = append(output.Utilisation_msgs, fmt.Sprintf("Warning EBS Volume %s not used!!! Cost since creation: $%.2f Use or destroy the volume.", *vl.VolumeId, vol_costs))
							output.SetWarn()
						}

					}

					c1 <- 2

				}()

				go func() {

					ec2.GetSgs(client)
          ec2.GetElbs(client)


          _ = <- c2

					for _, sg := range ec2.Security_groups.SecurityGroups {
						for _, rule := range sg.IpPermissions {
							var port int64
							if rule.ToPort != nil {
								port = *rule.ToPort
							} else {
								port = 0
							}
							if !cfg.Sg_is_excluded(*sg.GroupId, *rule.IpProtocol, port) {
								for _, iprng := range rule.IpRanges {
									if *iprng.CidrIp == "0.0.0.0/0" {
                    
                    attached_insts := ec2.Is_attached_to_instance(*sg.GroupId)
                    attached_elbs := ec2.Is_attached_to_elb(*sg.GroupId)
                  
                    if  len(attached_insts) > 0 {
                      output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("Open to the WORLD rule in SG: %s in VPC: %s and attached to instances: %s\n", *sg.GroupId, *sg.VpcId, attached_insts))
                      output.SetError()
                    } else {
										  output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("Open to the WORLD rule in SG: %s in VPC: %s but not attached to any instances.\n", *sg.GroupId, *sg.VpcId))
										  output.SetWarn()
                    }

                    if len(attached_elbs) > 0 {
                      output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("Open to the WORLD rule in SG: %s in VPC: %s and attached to ELBs: %s\n", *sg.GroupId, *sg.VpcId, attached_elbs))
                      output.SetError()
                    }
                    
									}
								}
							}
						}
					}


          if cfg.Conf.Cloudtrail.Check_region {
            if !ec2.Cloudtrail_is_enabled(client) {
              output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("CloudTrail is not enabled in region: %s\n", *regionPtr))
              output.SetError()
            }
          }

					c1 <- 3

				}()

				go func() {

					aws_iam := new(awsadvisor.AwsAuth)

					aws_iam.GetUsers(client)

					for _, v := range aws_iam.Users.Users {

						if !cfg.User_is_allowed(*v.UserName) {
							output.Security_err_msgs = append(output.Security_err_msgs, fmt.Sprintf("IAM user %s is not in the approved list of users!!!\n", *v.UserName))
							output.SetError()

						} else {

              has_pass := aws_iam.User_has_password(client, *v.UserName)
              user_keys := aws_iam.User_has_expired_key(client, *v.UserName, cfg.Conf.Iam.General.Expire_key_last_use)

              if has_pass {

  							if !aws_iam.User_has_mfa(client, *v.UserName) {
  								output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM user %s has no MFA device enabled for his account!!!\n", *v.UserName))
  								output.SetWarn()
  							}

  							if v.PasswordLastUsed != nil {
  								if time.Now().Unix()-int64(cfg.Conf.Iam.General.Expire_pass_last_login*24*3600) > v.PasswordLastUsed.Unix() {
  									output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM user %s has not logged in with a password for over %d days, maybe the account needs to be deprecated\n", *v.UserName, cfg.Conf.Iam.General.Expire_pass_last_login))
  									output.SetWarn()
  								}
  							}

              }

              if len(user_keys) > 0 {
                output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM user %s has not used these keys: %s for over %d days, maybe the account needs to be deprecated\n", *v.UserName, user_keys, cfg.Conf.Iam.General.Expire_key_last_use))
                output.SetWarn()
              }

						}

					}

          aws_iam.GetUsersDetails(client)


          for _, v := range aws_iam.UsersDetails.UserDetailList {
            if len(v.AttachedManagedPolicies)>0 || len(v.UserPolicyList)>0 {
              output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM user %s has directly attached IAM policies!!!\n", *v.UserName))
              output.SetWarn()
            }
          }

          aws_iam.GetCredentialsReport(client)

          for _, v := range aws_iam.CredReport {
            if time.Now().Unix()-int64(90*24*3600) > v.PassLastRotation.Unix() || time.Now().Unix()-int64(90*24*3600) > v.Key1LastRotation.Unix() || time.Now().Unix()-int64(90*24*3600) > v.Key2LastRotation.Unix() {
              output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM user %s has password or access keys that were not rotated for over 90 days!!!\n", v.UserName))
              output.SetWarn()
            }
          }

          policy_results := aws_iam.CheckPasswordPolicy(client)

          if len(policy_results) >0 {
            output.Security_warn_msgs = append(output.Security_warn_msgs, fmt.Sprintf("IAM Password Policy is not compliant with the following issues: %s\n", policy_results))
            output.SetWarn()
          }

					c1 <- 4

				}()


				go func() {

					billing.Metric = "EstimatedCharges"
					billing.Namespace = "AWS/Billing"
					billing.DimensionName = "ServiceName"
					billing.DimensionVal = "AmazonEC2"
					billing.Starting = -1
					billing.Period = 3600 * 24

					billing.Max_bill = make(map[string]float32)

					billing.GetMetrics(client, "AmazonEC2")
					billing.GetMetrics(client, "AmazonS3")
					billing.GetMetrics(client, "AmazonRoute53")
					billing.GetMetrics(client, "AWSDataTransfer")

					for k, v := range billing.Max_bill {
						output.Billing_msgs = append(output.Billing_msgs, fmt.Sprintf("The estimated maximum cost for %s up to now is: $%.2f\n", k, v))
					}

					output.Billing_values = billing.Max_bill

					c1 <- 5

				}()

				if *modePtr == "web" {
					time.Sleep(10 * time.Second)
				} else {
					break
				}
			}

			c1 <- 6

		}()

	} else if *modePtr == "graphite" {

		billing.Metric = "EstimatedCharges"
		billing.Namespace = "AWS/Billing"
		billing.DimensionName = "ServiceName"
		billing.DimensionVal = "AmazonEC2"
		billing.Starting = -1
		billing.Period = 3600 * 24

		billing.Max_bill = make(map[string]float32)

		billing.GetMetrics(client, *metricPtr)

		for k, v := range billing.Max_bill {
			output.Billing_msgs = append(output.Billing_msgs, fmt.Sprintf("The estimated maximum cost for %s up to now is: $%.2f\n", k, v))
		}

		output.Billing_values = billing.Max_bill

	}

	if *modePtr == "web" {

		http.HandleFunc("/", handler)
		http.HandleFunc("/getdata", handlerRefresh)
		http.ListenAndServe(":8080", nil)

	} else if *modePtr == "graphite" {

		os.Exit(output.End(*modePtr, *metricPtr, *profilePtr, accountname))

	} else {
		done := 0
		for {
			done += <-c1
			if done == 21 {
				os.Exit(output.End(*modePtr, *metricPtr, *profilePtr, accountname))
			}

		}
	}

}

func handler(w http.ResponseWriter, r *http.Request) {

	t, _ := template.ParseFiles("templates/overview.html")

	var col1, col2, col3, col4 string

	col1 = fmt.Sprintf("General Information:</br>")
	for _, msg := range output.Info_msgs {
		col1 += fmt.Sprintf("%s <br> ", msg)
	}

	col2 = fmt.Sprintf("Billing Information:<br>")
	for _, msg := range output.Billing_msgs {
		col2 += fmt.Sprintf("%s <br> ", msg)
	}

	col3 = fmt.Sprintf("Security Issues Detected (Critical):<br>")
	for _, msg := range output.Security_err_msgs {
		col3 += fmt.Sprintf("%s <br> ", msg)
	}

  col3 += fmt.Sprintf("Security Issues Detected (Warning):<br>")
  for _, msg := range output.Security_warn_msgs {
    col3 += fmt.Sprintf("%s <br> ", msg)
  }

	col4 = fmt.Sprintf("Utilization Issues Detected:<br>")
	for _, msg := range output.Utilisation_msgs {
		col4 += fmt.Sprintf("%s <br> ", msg)
	}

	col1html := template.HTML(col1)
	col2html := template.HTML(col2)
	col3html := template.HTML(col3)
	col4html := template.HTML(col4)

	p := &Page{Header: "AWS account information", Footer: "Copyright ITV 2016", Column2: col1html, Column1: col2html, Column3: col3html, Column4: col4html}
	t.Execute(w, p)

}

func handlerRefresh(w http.ResponseWriter, r *http.Request) {

	var col2 string

	col2 = fmt.Sprintf("Billing Information at %s: <br>", time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"))
	for _, msg := range output.Billing_msgs {
		col2 += fmt.Sprintf("%s <br> ", msg)
	}

	fmt.Fprintf(w, col2)

}
