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
	"github.com/gookit/color"
  "os"
  "io/ioutil"
  "strings"
)


func waitForOperation(project, zone string, service *compute.Service, op *compute.Operation) error {
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


func resizeToMorningMachineType() {
	ctx := context.Background()
  client, err := google.DefaultClient(ctx, compute.ComputeScope)
  if err != nil {
  	panic(err)
  }

	computeService, err := compute.New(client)
	if err != nil {
		panic(err)
	}

	op, err := computeService.Instances.Stop(confObject["project"], confObject["zone"], confObject["instance"]).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}


	op, err = computeService.Instances.SetMachineType(confObject["project"], confObject["zone"], confObject["instance"], 
		&compute.InstancesSetMachineTypeRequest{
			MachineType: "/zones/" + confObject["zone"] + "/machineTypes/" + confObject["machine-type-morning"],		
	}).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}

	op, err = computeService.Instances.Start(confObject["project"], confObject["zone"], confObject["instance"]).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully resized to morning machine-type")
}

func resizeToEveningMachineType() {
	ctx := context.Background()
  client, err := google.DefaultClient(ctx, compute.ComputeScope)
  if err != nil {
  	panic(err)
  }

	computeService, err := compute.New(client)
	if err != nil {
		panic(err)
	}

	op, err := computeService.Instances.Stop(confObject["project"], confObject["zone"], confObject["instance"]).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}


	op, err = computeService.Instances.SetMachineType(confObject["project"], confObject["zone"], confObject["instance"], 
		&compute.InstancesSetMachineTypeRequest{
			MachineType: "/zones/" + confObject["zone"] + "/machineTypes/" + confObject["machine-type-evening"],		
	}).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}

	op, err = computeService.Instances.Start(confObject["project"], confObject["zone"], confObject["instance"]).Context(ctx).Do()
	if err != nil {
		panic(err)
	}
	err = waitForOperation(confObject["project"], confObject["zone"], computeService, op)
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully resized to evening machine-type")
}


var confObject map[string]string

func main() {
	inProd, err := flaarum_shared.GetSetting("in_production")
	if err != nil {
		color.Red.Println("unexpected error. Have you installed  and launched flaarum?")
		os.Exit(1)	
	}
	if inProd != "true" && inProd != "t" {
		color.Red.Println("No need to create backups when not in production mode.")
		os.Exit(1)
	}

  confPath, err := flaarum_shared.GetCtlConfigPath()
  if err != nil {
    panic(err)
  }

	raw, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(err)
	}

	o := make(map[string]string)
	err = json.Unmarshal(raw, &o)
	if err != nil {
		panic(err)
	}

	confObject = o

  loc, err := time.LoadLocation(o["timezone"])
  if err != nil {
  	panic(err)
  }

	scheduler := gocron.NewScheduler(loc)
	scheduler.Every(1).Day().At("08:00").Do(resizeToMorningMachineType)
	scheduler.Every(1).Day().At("22:00").Do(resizeToEveningMachineType)

	scheduler.StartBlocking()
}