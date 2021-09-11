package daemons

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// GetProcessLogs will send HTTP GET request to the designated ip_address to reap the logs
func FlushLogs(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tFlushLogs")

	if processParams.processType == 3 {
		send_command_to_processes(get_keys(deployParams.Readers), "FlushLog", "")
		send_command_to_processes(get_keys(deployParams.Writers), "FlushLog", "")
		send_command_to_processes(get_keys(deployParams.Servers), "FlushLog", "")
	}
	fmt.Fprintf(w, "Logs Flushed")
}

// StartProcess will send a HTTP GET request to the designated ip address to start the process
func StartAProcess(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tSTART PROCESS")

	if processParams.processType == 3 {
		vars := mux.Vars(r)
		name := vars["ip"]
		send_command_to_process(name, "StartProcess", "")
	}
}

// StopProcess will send a HTTP GET request to the designated ip address to start the process
func StopAProcess(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tSTOP PROCESS")

	if processParams.processType == 3 {
		vars := mux.Vars(r)
		ipaddr := vars["ip"]
		send_command_to_process(ipaddr, "StopProcess", "")
	}
}

// KillProcess will send a HTTP GET request to the designated ip address to kill the process
func KillAProcess(w http.ResponseWriter, r *http.Request) {
	log.Println("INFO\tKILL PROCESS")

	if processParams.processType == 3 {
		vars := mux.Vars(r)
		ipaddr := vars["ip"]
		send_command_to_process(ipaddr, "KillProcess", "")
	}
}
