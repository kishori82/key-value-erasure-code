package daemons

import (
	"fmt"
	"time"
	//"io/ioutil"
	"math"
	//"math/rand"
	//"net/http"

	//"strconv"
	//"strings"
	//"time"

	"log"
	"regexp"
)

func printHeader(title string) {
	length := len(title)
	numSpaces := 22
	leftHalf := numSpaces + int(math.Ceil(float64(length)/2))
	rightHalf := numSpaces - int(math.Ceil(float64(length)/2))
	fmt.Println("***********************************************")
	fmt.Println("*                                             *")
	fmt.Print("*")
	fmt.Printf("%*s", int(leftHalf), title)
	fmt.Printf("%*s", (int(rightHalf) + 1), " ")
	fmt.Println("*")
	fmt.Println("*                                             *")
	fmt.Println("***********************************************")
}

func usage() {
	fmt.Println("Usage : process --process-type [0(reader), 1(writer), 2(server), 3(controller)]")
}

func Main_daemon(deploy_config *string, appl_config *string, process_name *string, make_config *bool, configure_remotely *bool) {
	//args := os.Args

	//fmt.Printf("%T\n", deploy_config)
	//fmt.Println(appl_config, process_name)

	var serverPatt = regexp.MustCompile(`^server-\d+$`)
	var readerPatt = regexp.MustCompile(`^reader-\d+$`)
	var writerPatt = regexp.MustCompile(`^writer-\d+$`)
	if !(*make_config || serverPatt.MatchString(*process_name) || readerPatt.MatchString(*process_name) ||
		writerPatt.MatchString(*process_name) || *process_name == "controller") {
		log.Fatal("Either Process name is not provided or not a supported process name format, Exiting Code")
	}

	DP := GetDeploymentParams()
	AP := GetApplicationParams()

	// First time running use make_config= true in flag and get the structure of the YML file with default values
	if *make_config {

		DP.LoadDefaultDeploymentParams()
		WriteConfigToYML(DP, "deploy_config.yml")

		AP.LoadDefaultApplicationParams()
		WriteConfigToYML(AP, "appl_config.yml")

		return
	}

	// Set up log handlers
	// We have to do this here itself, since some logs are written even before the processes start. (during system initialization)
	var PP *ProcessParams = nil
	PP = GetProcessParams()
	PP.remoteConfiguration = *configure_remotely
	PP.name = *process_name
	initLogHandlers()

	// Change the YML files as desired and then pass their path to use them for the run. These YML files only has
	// basic parameters.
	// if configured locally
	var err error = nil
	if *configure_remotely == false {
		var readConfig interface{}
		readConfig = ReadConfigFromYML(DP, *deploy_config)
		DP = readConfig.(*DeploymentParameters)

		readConfig = ReadConfigFromYML(AP, *appl_config)
		AP = readConfig.(*ApplicationParams)

		// Calculate the derived parameters for both application and deployment
		DP.InitializeDeploymentParams()
		AP.InitializeApplicationParams()

		// Write the entire config to a new file
		WriteConfigToYML(DP, "deploy_config_all.yml")
		WriteConfigToYML(AP, "appl_config_all.yml")

		// Initialize process parameters (the whole code is run by a single process)
		PP = GetProcessParams()
		err = PP.InitializeProcessParams(*process_name, false)

	} else {
		// Initialize process parameters (the whole code is run by a single process)
		PP = GetProcessParams()
		err = PP.InitializeProcessParamsForRemote(*process_name)
	}

	if err != nil {
		log.Fatal(err)
	}

	timeout = time.Duration(TIMEOUT * time.Second)

	switch PP.GetProcessType() {
	case READER:
		{
			Reader_process()
		}
	case WRITER:
		{
			Writer_process()
		}
	case SERVER:
		{
			Server_process()
		}
	case CONTROLLER:
		{
			Controller_process()
		}
	}

}
