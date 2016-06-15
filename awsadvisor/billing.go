package awsadvisor

import (
    "time"
    "io/ioutil"
    "path/filepath"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/cloudwatch"

    "gopkg.in/yaml.v2"

)



type EC2Costs struct {
  Ondemand map[string] map[string] map[string] struct {
    Region string
    Price_per_hour float32
    Period int16
  }
  Reserved map[string] map[string] map[string] []*struct {
    Region string
    Price_per_hour float32
    Period int16
  }
  EBS map[string] map[string] struct {
    Region string
    Price_per_gbmonth float32
    Price_per_iopmonth float32
  }
}


type Estimates struct {
  Current_cost float32
  Reserved_cost map[int16] Reservation
}

type Reservation struct{
  Term_cost float32
  Cost_saving_after int16
  Diff_with_10h float32
}


type Billing struct {

  Metric, Namespace, DimensionName, DimensionVal string 
  Starting int
  Period int64
  Max_bill map[string] float32

}


func (b *Billing) GetMetrics(meta interface{}, service_name string) {

  metric_params := &cloudwatch.GetMetricStatisticsInput{
  EndTime:    aws.Time(time.Now().AddDate(0, 0, 21)),
  MetricName: aws.String(b.Metric),
  Namespace:  aws.String(b.Namespace),
  Period:     aws.Int64(b.Period),
  StartTime:  aws.Time(time.Now().AddDate(0, 0, b.Starting)),
  Statistics: []*string{
  aws.String("Maximum"),
  },
  Dimensions: []*cloudwatch.Dimension{
    { // Required
      Name:  aws.String(b.DimensionName), // Required
      Value: aws.String(service_name),
    },
    {
      Name: aws.String("Currency"),
      Value: aws.String("USD"),
    },
  },
  }

  metrics_resp, metrics_err := meta.(*AWSClient).billingconn.GetMetricStatistics(metric_params)

  if metrics_err != nil {
    panic(metrics_err)
  }

  
  for _, cost := range metrics_resp.Datapoints {    
    b.Max_bill[service_name] = float32(*cost.Maximum)
  }

}

func (ec2costs *EC2Costs) ParsePrices(prices_filename string) {
  
  pricesFile, _ := filepath.Abs(prices_filename)
  yamlPrices, file_err := ioutil.ReadFile(pricesFile)

  if (file_err != nil) {
    panic(file_err)
  }

  yaml_err := yaml.Unmarshal(yamlPrices, &ec2costs)

  if (yaml_err != nil) {
    panic(yaml_err)
  }

}

func (ec2cost *EC2Costs) GetEstimate(hours_run int64, instance_size, region, platform string) *Estimates {

  est := new(Estimates)

  est.Current_cost = ec2cost.Ondemand[region][platform][instance_size].Price_per_hour * float32(hours_run)

  est.Reserved_cost = make(map[int16]Reservation)

  for _, r := range ec2cost.Reserved[region][platform][instance_size] {
    tmp := est.Reserved_cost[r.Period]
    tmp.Term_cost = float32(r.Period*24) * r.Price_per_hour
    tmp.Cost_saving_after = int16(float32(tmp.Term_cost - est.Current_cost)/ec2cost.Ondemand[region][platform][instance_size].Price_per_hour)
    tmp.Diff_with_10h =  ec2cost.Ondemand[region][platform][instance_size].Price_per_hour*float32(10*int16(float32(r.Period)*0.714)) - tmp.Term_cost
    est.Reserved_cost[r.Period] = tmp
  }

  return est

}


func (ec2cost *EC2Costs) GetEBSCost(region, voltype string, size, months_run, iops int64) float32 {


  sum := float32(size*months_run) * ec2cost.EBS[region][voltype].Price_per_gbmonth + float32(iops * months_run) * ec2cost.EBS[region][voltype].Price_per_iopmonth

  return sum

}




