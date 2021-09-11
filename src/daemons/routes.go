package daemons

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	DELIM string = "_"
)

func create_server_list_string() string {
	var serverList string
	i := 0

	for _, e := range deployParams.Servers {
		if i == 0 {
			serverList = serverList + e[0] + DELIM + e[1] + DELIM + e[2]
		} else {
			serverList = serverList + DELIM + e[0] + DELIM + e[1] + DELIM + e[2]
		}
		i = i + 1
	}

	return serverList
}

// sends a http request to an ipaddress with a route
func send_command_to_process(name string, attrib string, param string) (string, error) {
	var url string
	if len(param) > 0 {
		url = "http://" + get_ip_and_applport(name) + "/" + attrib + "/" + param
	} else {
		url = "http://" + get_ip_and_applport(name) + "/" + attrib
	}

	client := http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	//resp, err := http.Get(url)
	if err != nil {
		AppLog.Panicln("Error in communicating with remote process")
		return "", err
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		AppLog.Panicln("Error response while communicating with remote process")
		return "", nil
	}

	return string(contents), nil
}

// sends a http request to an ipaddress with a route
func send_command_to_process_fasttimeout(name string, attrib string, param string) (string, error) {
	var url string
	if len(param) > 0 {
		url = "http://" + get_ip_and_applport(name) + "/" + attrib + "/" + param
	} else {
		url = "http://" + get_ip_and_applport(name) + "/" + attrib
	}

	_, _ = http.Get(url)

	return " ", nil
}

// send command to all processes
func send_command_to_processes(processes []string, attrib string, mesg string) {
	for _, process := range processes {
		send_command_to_process(process, attrib, mesg)
	}
}

func exponential_wait(lambda float64) (interval int64) {

	unif := rand.Float64()
	ln := math.Log(unif)
	_dur := -1000 * (ln) / lambda

	dur := int64(_dur)
	return dur
}

//GetSafetyLog:  get the safety log
func GetAppLog(w http.ResponseWriter, r *http.Request) {
	fmt.Println("INFO\tGetApplicationLog")
	applogfile := GetLogFilepath(APPLOGFILE)
	//fmt.Fprintf(w, safetylogfile)
	AppLogFileHandler.Sync()
	GetLog(w, r, applogfile)
}

// GetOpStats returns the number of lines in the safetylogs (suitable for reader and writer longs)
/*func GetOpStats(w http.ResponseWriter, r *http.Request) {
	filename := GetLogFilepath(appParams.FolderToStore, SAFETYLOGFILE)
	count, err := NumberOfLinesInFile(filename)
	if err != nil {
		fmt.Fprintf(w, "0")
	}
	fmt.Fprintf(w, strconv.Itoa(count))
}*/

// GetExperimentLog : get the log file the stats log
func GetExperimentLog(w http.ResponseWriter, r *http.Request) {
	explogfile := GetLogFilepath(EXPLOGFILE)
	fmt.Println(w, explogfile)
	ExpLogFileHandler.Sync()
	GetLog(w, r, explogfile)
}

// GetStatsLog : get the log file the stats log
func GetSystemLog(w http.ResponseWriter, r *http.Request) {
	AppLog.Println("INFO\tGetSystemLog")
	syslogfile := GetLogFilepath(SYSTEMLOGFILE)
	SystemLogFileHandler.Sync()
	GetLog(w, r, syslogfile)
}

func GetLog(w http.ResponseWriter, r *http.Request, logfilepath string) {
	// Open the file and dump it into the request as a byte array.
	buf, err := ioutil.ReadFile(logfilepath)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		log.Println(err)
		fmt.Fprintf(w, "failed to read the logfile %s\n", logfilepath)
		fmt.Fprintf(w, "remote config %v\n", processParams.remoteConfiguration)
		return
	}
	fmt.Fprintf(w, string(buf))
}

// FlushProcess Log will send HTTP GET request to the designated ip_address to reap the logs
func FlushLog(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tFlushLog")

	err := os.Truncate("logs/logs.txt", 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, "Log flushed")
}
func StopProcess(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tStopProcess")

	KillProcess(w, r) // Change this to Pause Later

}

func StartProcess(w http.ResponseWriter, r *http.Request) {
	fmt.Println("INFO\tStartProcess calling")

	//	active_chan <- true
	log.Println("INFO\tStartProcess succes")

	fmt.Fprintf(w, "Started")
}

func KillProcess(w http.ResponseWriter, r *http.Request) {
	log.Fatal("INFO\tKillProcess called... Shutting down.")
}

// StartReaders will send a start process message to all readers
func StartReaders(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StartProcess", "readers")
}

// StartWriters will send a start process message to all writers
func StartWriters(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StartProcess", "writers")
}

// StartServers will send a start process message to all servers
func StartServers(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StartProcess", "servers")
}

// StopReader will send a stop process message to all readers
func StopReaders(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StopProcess", "readers")
}

// StopWriters will send a stop process message to all writers
func StopWriters(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StopProcess", "writers")
}

// StopServers will send a stop process message to all servers
func StopServers(w http.ResponseWriter, r *http.Request) {
	clusterCommand("StopProcess", "servers")
}

//set the number of read operations
func DoReadOperations(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	num_reads_str := vars["num_reads"]
	num_reads64, _ := strconv.ParseInt(num_reads_str, 10, 64)
	num_reads := int(num_reads64)

	if processParams.processType == CONTROLLER {
		clusterCommand("DoReads/"+num_reads_str, "readers")
	} else {
		fmt.Println("INFO\tSetting Read Operations")

		go func() {
			for i := 0; i < num_reads; i++ {
				active_chan_reader <- true
			}
		}()
	}
}

// set time limit
func KillServers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	num_servers_str := vars["numServersToKill"]
	num_servers, _ := strconv.ParseInt(num_servers_str, 10, 64)

	if processParams.processType == CONTROLLER {

		for i := 1; i <= int(num_servers); i++ {
			procName := "server-" + strconv.Itoa(i)
			AppLog.Println("killing Server", i)
			send_command_to_process_fasttimeout(procName, "KillSelf", "")
			AppLog.Println("killed Server", i)
		}
	}
}

// set time limit
func SetTimeLimit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	num_seconds_str := vars["setTimeLimit"]
	num_seconds, _ := strconv.ParseInt(num_seconds_str, 10, 64)

	if processParams.processType == CONTROLLER {
		clusterCommand("SetTimeLimit/"+num_seconds_str, "readers")
		clusterCommand("SetTimeLimit/"+num_seconds_str, "writers")
	} else {
		fmt.Println("INFO\tSetting Time Limit")
		timelimit = time.Now().UnixNano() + num_seconds*1e9
		AppLog.Println("Setting lime limit ", timelimit)
	}
}

//set the number of write operations
func DoWriteOperations(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	num_writes_str := vars["num_writes"]
	num_writes64, _ := strconv.ParseInt(num_writes_str, 10, 64)
	num_writes := int(num_writes64)

	AppLog.Println("got a request to write ", num_writes)
	if processParams.processType == CONTROLLER {
		errorMessage := clusterCommand("DoWrites/"+num_writes_str, "writers")
		if errorMessage != "" {
			fmt.Fprintf(w, "ERROR:"+errorMessage)
		}
	} else {
		AppLog.Println("INFO\tSetting Write Operations")

		go func() {
			fmt.Println("adding token  ", len(active_chan_writer), cap(active_chan_writer))

			for i := 0; i < num_writes; i++ {
				active_chan_writer <- true
			}
		}()
	}

}

func clusterCommand(url, daemons string) string {
	//log.Println("INFO\t" + url)

	var errorMessage string
	var processes map[string][]string
	switch {
	case daemons == "readers":
		processes = deployParams.Readers

	case daemons == "writers":
		processes = deployParams.Writers
		AppLog.Println(processes)
	case daemons == "servers":
		processes = deployParams.Servers
	default:
		log.Panicln("Unacceptable daemons provided: %s\n", daemons)
	}

	for procName := range processes {
		_, err := send_command_to_process(procName, url, "")
		if err != nil {
			errorMessage = errorMessage + procName + ":" + err.Error() + "\n"
		}
	}

	return errorMessage
}

func GetDeploymentParamsString() string {
	DP := GetDeploymentParams()

	data, err := json.Marshal(DP)
	if err != nil {
		panic("Cannot marshall deployment parameters")
	}
	encdata := b64.StdEncoding.EncodeToString(data)
	return encdata
}

func SetApplicationParams(w http.ResponseWriter, r *http.Request) {
	fmt.Println("INFO\tSetApplicationParams")

	vars := mux.Vars(r)
	appParamStr := vars["app_param"]
	sDec, _ := b64.StdEncoding.DecodeString(appParamStr)
	err := json.Unmarshal(sDec, &appParams)
	if err != nil {
		log.Panicln("error converting to JSON while Setting application parameters")

	}

	if processParams.processType == CONTROLLER {
		for reader := range deployParams.Readers {
			send_command_to_process(reader, "SetApplicationParams", appParamStr)
		}
		for writer := range deployParams.Writers {
			send_command_to_process(writer, "SetApplicationParams", appParamStr)
		}
		for server := range deployParams.Servers {
			send_command_to_process(server, "SetApplicationParams", appParamStr)
		}
	}
	//log.Println(" Done the application params setup")
	chan_app_params <- true // first
	chan_app_params <- true // twice

}

func isSetup(w http.ResponseWriter, r *http.Request) {
	//log.Println("INFO\tisSetup")

	if setupComplete {
		fmt.Fprintf(w, "%s", YES)
	} else {
		fmt.Fprintf(w, "%s", NO)
	}
}

func GetOpNum(w http.ResponseWriter, r *http.Request) {
	//	log.Println("INFO\tetOpNum")
	fmt.Fprintf(w, "%d", globalOpNum)
}

func GetNumRemOps(w http.ResponseWriter, r *http.Request) {
	//	log.Println("INFO\tetOpNum")
	if processParams.processType == READER {
		fmt.Fprintf(w, "%d", len(active_chan_reader))
	}

	if processParams.processType == WRITER {
		fmt.Fprintf(w, "%d", len(active_chan_writer))
	}

}

func SetDeploymentParams(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tSetDeploymentParams")

	vars := mux.Vars(r)
	deplParamStr := vars["depl_param"]
	sDec, _ := b64.StdEncoding.DecodeString(deplParamStr)

	err := json.Unmarshal(sDec, &deployParams)
	if err != nil {
		log.Panicln("error converting to JSON while Setting deployment parameters")
	}

	processParams.name = deployParams.WhoAmI

	//log.Println("Dockerx  whoami and type  ", deployParams.WhoAmI, processParams.processType)

	//ioutil.WriteFile("/tmp/controller.1.txt", []byte(deployParams.WhoAmI), 0777)

	if processParams.processType == CONTROLLER {
		for reader := range deployParams.Readers {
			deployParams.WhoAmI = reader
			//	AppLog.Println("Dockerx  reader destination ", reader)
			deplParamStr = GetDeploymentParamsString()
			send_command_to_process(reader, "SetDeploymentParams", deplParamStr)
		}

		for writer := range deployParams.Writers {
			deployParams.WhoAmI = writer
			deplParamStr = GetDeploymentParamsString()
			send_command_to_process(writer, "SetDeploymentParams", deplParamStr)
		}

		for server := range deployParams.Servers {
			deployParams.WhoAmI = server
			deplParamStr = GetDeploymentParamsString()
			send_command_to_process(server, "SetDeploymentParams", deplParamStr)
		}
	}

	chan_depl_params <- true
}

// set  name
func SetName(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tSetName")

	vars := mux.Vars(r)
	name := vars["param"]
	ips := strings.Split(name, DELIM)

	if len(ips) != 1 {
		log.Panicln("Expected 1 parameter in SetName, found ", len(ips))
	}
	processParams.name = name
}

// get name
func GetName(w http.ResponseWriter, r *http.Request) {
	//log.Println("INFO\tGetName")

	fmt.Fprintf(w, "%s", processParams.name)
}

func getDaemons(url, daemons string) string {
	//log.Println("INFO\t" + url)

	ipstr := ""
	switch {
	case daemons == "readers":
		for ip := range deployParams.Readers {
			ipstr += " " + ip
		}
	case daemons == "writers":
		for ip := range deployParams.Writers {
			ipstr += " " + ip
		}
	case daemons == "servers":
		for ip := range deployParams.Servers {
			ipstr += " " + ip
		}
	default:
		log.Panicln("Unacceptable daemon provided")
	}
	return ipstr
}

// returns the list of readers
func GetReaders(w http.ResponseWriter, r *http.Request) {
	ipstr := getDaemons("GetReaders", "readers")
	fmt.Fprintf(w, "%s", ipstr)
}

// returns the set of servers
func GetServers(w http.ResponseWriter, r *http.Request) {
	ipstr := getDaemons("GetServers", "servers")
	fmt.Fprintf(w, "%s", ipstr)
}

// returns the list of writers
func GetWriters(w http.ResponseWriter, r *http.Request) {
	ipstr := getDaemons("GetWriters", "writers")
	fmt.Fprintf(w, "%s", ipstr)
}

// KillSelf will end this server
func KillSelf(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tController going down...")

	fmt.Fprintf(w, "Controller going down...")
	defer log.Fatal("Controller Exiting")
}
