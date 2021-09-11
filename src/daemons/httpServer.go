package daemons

import (
	"net/http"

	"github.com/gorilla/mux"
)

func HTTP_Server(port string) {
	// Setup HTTP functionality
	router := mux.NewRouter()
	//AppLog.Println("Running http server")
	AppLog.Println("Running http server")

	// Routes to Controller (invoekd by you and me)

	router.HandleFunc("/DoReads/{num_reads:[0-9]+}", DoReadOperations)
	router.HandleFunc("/DoWrites/{num_writes:[0-9]+}", DoWriteOperations)
	router.HandleFunc("/SetTimeLimit/{setTimeLimit:[0-9]+}", SetTimeLimit)
	router.HandleFunc("/KillServers/{numServersToKill:[0-9]+}", KillServers)

	router.HandleFunc("/GetAppLog", GetAppLog)
	router.HandleFunc("/GetSystemLog", GetSystemLog)
	router.HandleFunc("/GetExperimentLog", GetExperimentLog)
	router.HandleFunc("/FlushLogs", FlushLogs)

	router.HandleFunc("/StopReaders", StopReaders)
	router.HandleFunc("/StopWriters", StopWriters)
	router.HandleFunc("/StopServers", StopServers)

	router.HandleFunc("/StartReaders", StartReaders)
	router.HandleFunc("/StartWriters", StartWriters)
	router.HandleFunc("/StartServers", StartServers)

	router.HandleFunc("/GetReaders", GetReaders)
	router.HandleFunc("/GetWriters", GetWriters)
	router.HandleFunc("/GetServers", GetServers)

	// Routes to Clients and Servers (invoked by the controller)

	router.HandleFunc("/StopProcess", StopProcess)
	router.HandleFunc("/StartProcess", StartProcess)
	router.HandleFunc("/KillSelf", KillSelf)

	// Routes used by both users and controller

	router.HandleFunc("/SetReaders/{ip:[0-9._]+}", SetReaders)
	router.HandleFunc("/SetWriters/{ip:[0-9._]+}", SetWriters)
	router.HandleFunc("/SetServers/{ip:[0-9._]+}", SetServers)

	router.HandleFunc("/SetAlgorithm/{id}", SetAlgorithm)

	router.HandleFunc("/SetRunId/{id}", SetRunId)
	router.HandleFunc("/SetWriteTo/{param}", SetWriteTo)

	router.HandleFunc("/GetName", GetName)
	router.HandleFunc("/SetName/{param}", SetName)

	router.HandleFunc("/SetApplicationParams/{app_param}", SetApplicationParams)
	router.HandleFunc("/SetDeploymentParams/{depl_param}", SetDeploymentParams)
	router.HandleFunc("/isSetup", isSetup)
	router.HandleFunc("/GetOpNum", GetOpNum)
	router.HandleFunc("/GetNumRemOps", GetNumRemOps)

	router.HandleFunc("/StopAProcess/{ip}", StopAProcess)
	router.HandleFunc("/StartAProcess/{ip}", StartAProcess)
	router.HandleFunc("/KillAProcess/{ip}", KillAProcess)

	router.HandleFunc("/SetSeed/{seed:[0-9]+}", SetSeed)
	router.HandleFunc("/GetSeed", GetSeed)
	router.HandleFunc("/GetParams", GetParams)

	router.HandleFunc("/SetReadRateDistribution/{param:[a-zA-Z0-9._]+}", SetReadRateDistribution)
	router.HandleFunc("/SetWriteRateDistribution/{param:[a-zA-Z0-9._]+}", SetWriteRateDistribution)

	router.HandleFunc("/SetFileSize/{size:[0-9.]+}", SetFileSize)
	router.HandleFunc("/GetFileSize", GetFileSize)
	http.ListenAndServe(":"+port, router) //Fix port
}
