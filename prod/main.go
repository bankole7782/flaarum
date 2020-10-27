// prod provides the commands which helps in making a flaarum server production ready.
package main

import (
  "github.com/bankole7782/flaarum/flaarum_shared"
  "io/ioutil"
  "fmt"
  "os"
  "github.com/gookit/color"
  "encoding/json"
  "github.com/tidwall/pretty"
  "strings"
)


func main() {
  if len(os.Args) < 2 {
    color.Red.Println("expected a command. Open help to view commands.")
    os.Exit(1)
  }

  switch os.Args[1] {
  case "--help", "help", "h":
    fmt.Println(`Flaarum's prod makes a flaarum instance production ready.

Supported Commands:

    r     Read the current key string used 

    c     Creates / Updates and prints a new key string

    mpr   Make production ready. It also creates and prints a key string. It expects a google cloud bucket
          as its only argument.

    masr  Make autoscaling ready. This is for the control instance. It expects in the following order flaarum-data-instance-name
          timezone machine-type-morning machine-type-evening.

          Example: sudo flaarum.prod masr flaarum-2sb WAT e2-highcpu-5 e2-highcpu-2
      `)

  case "r":
    keyPath := flaarum_shared.GetKeyStrPath()
    raw, err := ioutil.ReadFile(keyPath)
    if err != nil {
      color.Red.Printf("Error reading key string path.\nError:%s\n", err)
      os.Exit(1)
    }
    fmt.Println(string(raw))

  case "c":
    keyPath := flaarum_shared.GetKeyStrPath()
    randomString := flaarum_shared.UntestedRandomString(50)

    err := ioutil.WriteFile(keyPath, []byte(randomString), 0777)
    if err != nil {
      color.Red.Printf("Error creating key string path.\nError:%s\n", err)
      os.Exit(1)
    }
    fmt.Print(randomString)

  case "mpr":
    if len(os.Args) != 3 {
      color.Red.Println("Expecting the backup_bucket as the only argument")
      os.Exit(1)
    }
    keyPath := flaarum_shared.GetKeyStrPath()
    randomString := flaarum_shared.UntestedRandomString(50)

    err := ioutil.WriteFile(keyPath, []byte(randomString), 0777)
    if err != nil {
      color.Red.Printf("Error creating key string path.\nError:%s\n", err)
      os.Exit(1)
    }
    fmt.Print(randomString)

    confPath, err := flaarum_shared.GetConfigPath()
    if err != nil {
      panic(err)
    }

    conf := map[string]string {
      "debug": "false",
      "in_production": "true",
      "port": "22318",
      "backup_bucket": os.Args[2],
    }

    jsonBytes, err := json.Marshal(conf)
    if err != nil {
      panic(err)
    }

    prettyJson := pretty.Pretty(jsonBytes)

    err = ioutil.WriteFile(confPath, prettyJson, 0777)
    if err != nil {
      panic(err)
    }

  case "masr":
    if len(os.Args) != 6 {
      color.Red.Println("Expecting 5 arguments. Check the help for documentation")
      os.Exit(1)
    }

    conf := map[string]string {
      "instance": os.Args[2],
      "timezone": os.Args[3],
      "machine-type-morning": os.Args[4],
      "machine-type-evening": os.Args[5],
    }

    jsonBytes, err := json.Marshal(conf)
    if err != nil {
      panic(err)
    }

    prettyJson := pretty.Pretty(jsonBytes)

    confPath, err := flaarum_shared.GetConfigPath()
    if err != nil {
      panic(err)
    }

    confPath = strings.Replace(confPath, "flaarum.json", "flaarumctl.json", 1)

    err = ioutil.WriteFile(confPath, prettyJson, 0777)
    if err != nil {
      panic(err)
    }

  default:
    color.Red.Println("Unexpected command. Run the Flaarum's prod with --help to find out the supported commands.")
    os.Exit(1)
  }


}
