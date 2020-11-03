// gcprb repeatedly runs 'inout' and stores the output file to google cloud storage.
package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"time"
	"os/exec"
	"os"
	"context"
	"github.com/gookit/color"
	"cloud.google.com/go/storage"
	"io/ioutil"
	"io"
	"github.com/bankole7782/flaarum/flaarum_shared"
	"github.com/bankole7782/flaarum"
	"path/filepath"
	"strings"
)


func createBackupAndSaveToGCloudStorage() {

	tmp, _ := flaarum_shared.GetConfigPath()
	bkupFilePath := strings.Replace(tmp, "flaarum.json", "last-bkup-dt.txt", 1)
	raw, err := ioutil.ReadFile(bkupFilePath)
	if err != nil {
		contents := flaarum.RightDateTimeFormat(time.Now())
		ioutil.WriteFile(bkupFilePath, []byte(contents), 0777)
	}

	lastBackupTime, _ := time.Parse(flaarum_shared.BROWSER_DATETIME_FORMAT, string(raw))
	daysSinceLastBackup := time.Now().Sub(lastBackupTime).Hours() / 24

	if daysSinceLastBackup >= 14 {

		keyStrPath := flaarum_shared.GetKeyStrPath()
		raw, err := ioutil.ReadFile(keyStrPath)
		if err != nil {
			color.Red.Println(err)
			os.Exit(1)
		}
		keyStr := string(raw)

		cl := flaarum.NewClient(fmt.Sprintf("https://127.0.0.1:%d/", flaarum_shared.PORT), keyStr, "first_proj")

		err = cl.Ping()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		projects, err := cl.ListProjects()
		if err != nil {
			panic(err)
		}

		ctx := context.Background()
	  client, err := storage.NewClient(ctx)
	  if err != nil {
	  	panic(err)
	  }
	  defer client.Close()

	  for _, projName := range projects {
	  	out, err := exec.Command("flaarum.inout", "out", projName).Output()
	  	if err != nil {
	  		panic(err)
	  	}

	  	f, err := os.Open(string(out))
	  	if err != nil {
	  		panic(err)
	  	}
	  	defer f.Close()

	  	// Upload an object with storage.Writer.
		  wc := client.Bucket(bucketName).Object(filepath.Base(string(out))).NewWriter(ctx)
		  if _, err = io.Copy(wc, f); err != nil {
		    panic(err)
		  }
		  if err := wc.Close(); err != nil {
		    panic(err)
		  }

	  }

	  fmt.Println("Successfully ran backup at " + time.Now().String())

	  contents := flaarum.RightDateTimeFormat(time.Now())
		ioutil.WriteFile(bkupFilePath, []byte(contents), 0777)
	}

}

var bucketName string

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

  // test code
	ctx := context.Background()
  client, err := storage.NewClient(ctx)
  if err != nil {
  	panic(err)
  }
  defer client.Close()

  buck, err := flaarum_shared.GetSetting("backup_bucket")
  if err != nil {
  	panic(err)
  }

  configFilePath, err := flaarum_shared.GetConfigPath()
	if err != nil {
		panic(err)
	}

  if buck == "" {
  	color.Red.Println("Create a Google Cloud Storage bucket for backups and add it to the config file")
  	color.Red.Printf("This config file can be found at '%s'.\n", configFilePath)
  	os.Exit(1)
  }

  bucketName = buck

	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(30).Minutes().Do( createBackupAndSaveToGCloudStorage )
	scheduler.StartBlocking()
}
