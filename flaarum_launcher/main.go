// This program launches a server with flaarum installed and setup even with backup
package main

import (
	"strings"
	"os"
	"github.com/gookit/color"
	"fmt"
  "github.com/tidwall/pretty"
  "io/ioutil"
  "encoding/json"
  "os/exec"
  "time"
  "github.com/bankole7782/flaarum/flaarum_shared"
)


func main() {
	if len(os.Args) < 2 {
		color.Red.Println("Expecting a command. Run with help subcommand to view help.")
		os.Exit(1)
	}

	switch os.Args[1] {
  case "--help", "help", "h":
    fmt.Println(`Flaarum's launcher creates and configures a flaarum server on Google Cloud.

Supported Commands:

    init   Creates a config file. Edit to your own requirements. Some of the values can be gotten from
           Google Cloud's documentation. 

    l      Launches a configured instance based on the config created above.

      `)
  case "init":

  	initObject := map[string]string {
  		"project": "",
  		"zone": "",
  		"region": "",
  		"disk-size": "10GB",
  		"machine-type": "f1-micro",
  	}

    jsonBytes, err := json.Marshal(initObject)
    if err != nil {
      panic(err)
    }

    prettyJson := pretty.Pretty(jsonBytes)

    configFileName := "l" + time.Now().Format("20060102T150405") + ".json"

    writePath, err := flaarum_shared.GetFlaarumPath(configFileName)
    if err != nil {
    	panic(err)
    }

    err = ioutil.WriteFile(writePath, prettyJson, 0777)
    if err != nil {
      panic(err)
    }

    fmt.Printf("Edit the file at '%s' before launching.\n", writePath)

  case "l":
  	if len(os.Args) != 3 {
  		color.Red.Println("The l command expects a launch file as the next argument.")
  		os.Exit(1)
  	}

    inputPath, err := flaarum_shared.GetFlaarumPath(os.Args[2])
    if err != nil {
    	panic(err)
    }

  	raw, err := ioutil.ReadFile(inputPath)
  	if err != nil {
  		panic(err)
  	}

  	o := make(map[string]string)
  	err = json.Unmarshal(raw, &o)
  	if err != nil {
  		panic(err)
  	}

		instanceName := fmt.Sprintf("flaarum-%s", strings.ToLower(flaarum_shared.UntestedRandomString(4)))
		diskName := fmt.Sprintf("%s-disk", instanceName)
  	
  	o["instance"] = instanceName
  	o["disk"] = diskName

		cmd0 := exec.Command("gcloud", "services", "enable", "compute.googleapis.com", "--project", o["project"])

		out, err := cmd0.CombinedOutput()
		if err != nil {
			color.Red.Println(out)
      color.Red.Println(err.Error())
		}

		scriptPath := flaarum_shared.G("startup_script.sh")
		cmd := exec.Command("gcloud", "compute", "--project", o["project"], "instances", "create", o["instance"], 
			"--zone", o["zone"], "--machine-type", o["machine-type"], "--image", "ubuntu-minimal-2004-focal-v20200702",
			"--image-project", "ubuntu-os-cloud", "--boot-disk-size", "10GB", 
			"--create-disk", "mode=rw,size=10,type=pd-ssd,name=" + o["disk"],
			"--metadata-from-file", "startup-script=" + scriptPath,
		)

		out, err = cmd.CombinedOutput()
		if err != nil {
      color.Red.Println(out)
			panic(err)
		}

		cmd2 := exec.Command("gcloud", "compute", "resource-policies", "create", "snapshot-schedule", o["instance"] + "-schdl",
	    "--description", "MY WEEKLY SNAPSHOT SCHEDULE", "--max-retention-days", "60", "--start-time", "22:00",
	    "--weekly-schedule", "sunday", "--region", o["region"], "--on-source-disk-delete", "keep-auto-snapshots",
	    "--project", o["project"],
		)

		out, err = cmd2.CombinedOutput()
		if err != nil {
      color.Red.Println(out)
			panic(err)
		}

		cmd3 := exec.Command("gcloud", "compute", "disks", "add-resource-policies", o["disk"], "--resource-policies",
			o["instance"] + "-schdl", "--zone", o["zone"], "--project", o["project"],
		)

		out, err = cmd3.CombinedOutput()
		if err != nil {
      color.Red.Println(out)      
			panic(err)
		}

		cmd4 := exec.Command("gcloud", "services", "enable", "vpcaccess.googleapis.com", "--project", o["project"])

		_, err = cmd4.CombinedOutput()
		if err != nil {
      color.Red.Println(out)      
		}


		cmd5 := exec.Command("gcloud", "compute", "networks", "vpc-access", "connectors", "create",
			o["instance"] + "-vpcc", "--network", "default", "--region", o["region"],
			"--range", "10.8.0.0/28", "--project", o["project"])

		out, err = cmd5.CombinedOutput()
		if err != nil {
      color.Red.Println(out)
			panic(err)
		}

		fmt.Println("Instance Name: " + o["instance"])
		fmt.Println("VPC Connector: " + o["instance"] + "-vpcc. Needed for Appengine and Cloud run.")
		fmt.Println("Please ssh into your instance. Run 'flaarum.prod r' to get your key for your program.")

		outObject := map[string]string {
			"instance": o["instance"], "vpc_connector": o["instance"] + "-vpcc",
		}

    jsonBytes, err := json.Marshal(outObject)
    if err != nil {
      panic(err)
    }

    outFileName := "lr" + os.Args[2][1:]
    outPath, err := flaarum_shared.GetFlaarumPath(outFileName)
    if err != nil {
    	panic(err)
    }

    prettyJson := pretty.Pretty(jsonBytes)

    err = ioutil.WriteFile(outPath, prettyJson, 0777)
    if err != nil {
      panic(err)
    }

    fmt.Printf("Results stored at '%s'.\n", outPath)

  default:
    color.Red.Println("Unexpected command. Run the launcher with --help to find out the supported commands.")
    os.Exit(1)
	}

}
