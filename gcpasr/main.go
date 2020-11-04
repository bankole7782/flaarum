package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"time"
	"github.com/bankole7782/flaarum/flaarum_shared"
	"encoding/json"
	"context"
	"golang.org/x/oauth2/google"
  compute "google.golang.org/api/compute/v1"
  "io/ioutil"
  "strings"
  "net/http"
	"crypto/tls"
	"github.com/pkg/errors"
  "github.com/bankole7782/zazabul"
  "strconv"
)


var confObject zazabul.Config

var MTs = []string{
	"e2-highcpu-2",
	"e2-highcpu-4",
	"e2-highcpu-8",
	"e2-highcpu-16",
	"e2-highcpu-32",
}

func main() {
  confPath, err := flaarum_shared.GetCtlConfigPath()
  if err != nil {
    panic(err)
  }

	conf, err := zazabul.LoadConfigFile(confPath)
	if err != nil {
		panic(err)
	}

	confObject = conf
	resizeFrequency, err := strconv.ParseUint(conf.Get("resize_frequency"), 10, 64)
	if err != nil {
		panic(err)
	}
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(resizeFrequency).Hours().Do(resizeMachineType)
	scheduler.StartBlocking()
}


func waitForOperationZone(project, zone string, service *compute.Service, op *compute.Operation) error {
	ctx := context.Background()
	for {
		result, err := service.ZoneOperations.Get(project, zone, op.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("Failed retriving operation status: %s", err)
		}

		if result.Status == "DONE" {
			if result.Error != nil {
				var errors []string
				for _, e := range result.Error.Errors {
					errors = append(errors, e.Message)
				}
				return fmt.Errorf("Operation failed with error(s): %s", strings.Join(errors, ", "))
			}
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}


func resizeMachineType() {
	config := &tls.Config { InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}

	httpCl := &http.Client{Transport: tr}

	resp, err := httpCl.Get(fmt.Sprintf("https:%s:%d/get-and-delete-stats", confObject.Get("instance_ip"), flaarum_shared.PORT))
	if err != nil {
		panic(errors.Wrap(err, "http error"))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(errors.Wrap(err, "ioutil error"))
	}

	if resp.StatusCode != 200 {
		panic(errors.New(string(body)))
	}

	respObj := make(map[string]int64)
  err = json.Unmarshal(body, &respObj)
  if err != nil {
    panic(errors.Wrap(err, "json error."))
  }

  nextActionCPU := whatToDo(respObj["cpu_usage"])
  nextActionRAM := whatToDo(respObj["ram_usage"])

  if nextActionCPU == "incr" || nextActionRAM == "incr" {
  	// do increase
  	if confObject.Get("machine_type") == MTs[len(MTs) - 1] {
  		fmt.Println("No resizing. You've gotten to the max server.")
  		return
  	}
  	index := flaarum_shared.FindIn(MTs, confObject.Get("machine_type"))
  	innerResizeMachine(MTs[index + 1])

  	fmt.Println("Successfully resized the flaarum server")
  } else if nextActionCPU == "dcr" || nextActionRAM == "dcr" {
  	// do decrease
		if confObject.Get("machine_type") == MTs[0] {
  		fmt.Println("No resizing. You've gotten to the smallest server.")
  		return
  	}
  	index := flaarum_shared.FindIn(MTs, confObject.Get("machine_type"))
  	innerResizeMachine(MTs[index - 1])

  	fmt.Println("Successfully resized the flaarum server")
  } else {
  	fmt.Println("No need for resize.")
  }
}


func whatToDo(state int64) string {
	if state >= 80 {
		return "incr"
	} else if state <= 20 {
		return "dcr"
	} else {
		return "remain"
	}
}


func innerResizeMachine(mt string) {
	ctx := context.Background()
  client, err := google.DefaultClient(ctx, compute.ComputeScope)
  if err != nil {
  	panic(err)
  }

	computeService, err := compute.New(client)
	if err != nil {
		panic(err)
	}

	op, err := computeService.Instances.Stop(confObject.Get("project"), confObject.Get("zone"), confObject.Get("instance")).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperationZone(confObject.Get("project"), confObject.Get("zone"), computeService, op)
	if err != nil {
		panic(err)
	}

	op, err = computeService.Instances.SetMachineType(confObject.Get("project"), confObject.Get("zone"), confObject.Get("instance"), 
		&compute.InstancesSetMachineTypeRequest{
			MachineType: "/zones/" + confObject.Get("zone") + "/machineTypes/" + mt,		
	}).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperationZone(confObject.Get("project"), confObject.Get("zone"), computeService, op)
	if err != nil {
		panic(err)
	}

	op, err = computeService.Instances.Start(confObject.Get("project"), confObject.Get("zone"), confObject.Get("instance")).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperationZone(confObject.Get("project"), confObject.Get("zone"), computeService, op)
	if err != nil {
		panic(err)
	}

	// save the machine_type in use
	confObject.Update(map[string]string{
		"machine_type": mt,
	})

  confPath, err := flaarum_shared.GetCtlConfigPath()
  if err != nil {
    panic(err)
  }

  err = confObject.Write(confPath)
  if err != nil {
    panic(err)
  }
}
