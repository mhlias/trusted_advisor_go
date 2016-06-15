package awsadvisor

import (
  "fmt"
  "time"
)

const (
  OK   int = 0
  WARN int = 1
  ERR  int = 2
  UNK  int = 3
)


type Output struct {
  Info_msgs []string
  Security_err_msgs []string
  Security_warn_msgs []string
  Utilisation_msgs []string
  Billing_msgs []string
  Billing_values map[string] float32
  Ecode int
}


func (o *Output) End(mode, metric, profile string, account_name string) int {
  switch mode {
  case "cli":
    
    fmt.Println("General Information:")
    for _, msg := range o.Info_msgs {
      fmt.Println(msg)
    }

    fmt.Println("Billing Information:")
    for _, msg := range o.Billing_msgs {
      fmt.Println(msg)
    }

    fmt.Println("Security Issues Detected (Critical):")
    for _, msg := range o.Security_err_msgs {
      fmt.Println(msg)
    }
    fmt.Println("Security Issues Detected (Warning):")
    for _, msg := range o.Security_warn_msgs {
      fmt.Println(msg)
    }
    fmt.Println("Utilization Issues Detected:")
    for _, msg := range o.Utilisation_msgs {
      fmt.Println(msg)
    }

  case "sensu":

    if o.Ecode == ERR {

       fmt.Println("Critical Security Issues Detected: (Non-critical issues supressed)")
      for _, msg := range o.Security_err_msgs {
        fmt.Println(msg)
      }

    } else {

      fmt.Println("General Information:")
      for _, msg := range o.Info_msgs {
        fmt.Println(msg)
      }

      fmt.Println("Billing Information:")
      for _, msg := range o.Billing_msgs {
        fmt.Println(msg)
      }
      
      fmt.Println("Security Issues Detected:")
      for _, msg := range o.Security_warn_msgs {
        fmt.Println(msg)
      }
      fmt.Println("Utilization Issues Detected:")
      for _, msg := range o.Utilisation_msgs {
        fmt.Println(msg)
      }

    }

    return o.Ecode
      
  case "graphite":
    
    fmt.Printf("aws.account.%s.billing.%s %.2f %d", account_name, metric, o.Billing_values[metric], time.Now().Unix())
    
  }

  return 0

}

func (o *Output) SetWarn() {

  if o.Ecode < WARN {
    o.Ecode = WARN
  }

}


func (o *Output) SetError() {

  if o.Ecode < ERR {
    o.Ecode = ERR
  }

}

func (o *Output) SetUnknown() {

  if o.Ecode < UNK {
    o.Ecode = UNK
  }

}