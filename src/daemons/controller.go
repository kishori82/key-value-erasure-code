//package main
package daemons

func Controller_process() {
	//initUpLogHandlers()
	AppLog.Println("Starting Controller")
	setupComplete = false
	//InitializeParameters()
	//LogParameters()

	go func() {
		waitUntilParamsIsSet()
		setupComplete = true
	}()

	HTTP_Server(processParams.appl_port)

	// Write Part of the Experiment Application Code

	// test code
	// start 50 reads for every reader process

	// start 10 writes for every writer process
}
