package daemons

import (
	"errors"
	"log"
	"strconv"
	"time"

	zmq3 "github.com/pebbe/zmq3"
)

//status
func server_logger() {
	var cpu_use float64

	cpu_use = CpuUsage()
	for true {
		if processParams.active == true {
			log.Printf("INFO\t%.2f\n", cpu_use)
		}
		time.Sleep(2 * 1000 * time.Millisecond)
		cpu_use = CpuUsage()
	}

}

func Server_process() {

	//log.Println("Starting server\n")

	go HTTP_Server(processParams.appl_port)

	setupComplete = false
	if processParams.remoteConfiguration {
		waitUntilParamsIsSet()
		InitializeParams()
		RemovePreviousLogFolder(appParams.FolderToStore)
		setupComplete = true
	}
	//initUpLogHandlers()

	if ENABLE_SYSTEM_LOGS {
		//go DoEvery(1000*time.Millisecond, LogStats)
		go LogStats()
	}
	server_daemon()
}

// initializeStateVariables initializes the state variable for the objects
func initializeStateVariables() {
	server_state_variables = make(map[string]*SingleObjServerState)
	for _, Objname := range get_keys_from_catalogue(processParams.ObjCatalogue) {
		server_state_variables[Objname] = &SingleObjServerState{
			tagCodedElement_var: TagValue{Tag_var: Tag{Client_id: "writer-1", Version_num: 0}},
			readsInProgress:     make(map[string]*ReadsInProgress),
			latestOpnumWriters:  make(map[string]int),
			writesInProgress:    make(map[string]map[int]*WritesInProgress)}

	}
}

func server_daemon() {

	// Set the Default State Variables
	initializeStateVariables()
	// Set the ZMQ sockets

	var unique_tag string
	frontend, _ := zmq3.NewSocket(zmq3.ROUTER)
	defer frontend.Close()

	if processParams.remoteConfiguration {
		frontend.Bind("tcp://*:" + ALGO_PORT)
		unique_tag = ALGO_PORT
	} else {
		frontend.Bind("tcp://" + get_ip_and_algoport(processParams.name))
		unique_tag = processParams.name
	}

	//  Backend socket talks to workers over inproc
	backend, _ := zmq3.NewSocket(zmq3.DEALER)
	defer backend.Close()
	backend.Bind("inproc://backend-" + unique_tag)

	AppLog.Println("frontend router", "tcp://"+get_ip_and_algoport(processParams.name))
	go server_worker()

	//  Connect backend to frontend via a proxy

	AppLog.Println("Server worker started")
	err := zmq3.Proxy(frontend, backend, nil) // a blocking line
	AppLog.Fatalln("Proxy interrupted:", err)
	AppLog.Println("Exiting")
}

func server_worker() {
	var respond bool
	worker, _ := zmq3.NewSocket(zmq3.DEALER)
	defer worker.Close()

	var unique_tag string
	if processParams.remoteConfiguration {
		unique_tag = ALGO_PORT
	} else {
		unique_tag = processParams.name
	}

	worker.Connect("inproc://backend-" + unique_tag)
	msg_reply := make([][]byte, 3)

	for i := 0; i < len(msg_reply); i++ {
		msg_reply[i] = make([]byte, 0) // the frist frame  specifies the identity of the sender, the second specifies the content
	}
	var message_reply Message
	for {
		//  The DEALER socket gives us the reply envelope and message
		msg, err := worker.RecvMessageBytes(0)
		if err != nil {
			AppLog.Println("Somthing Wrong", err)
		}

		message := CreateMessageFromGob(msg[1])

		if !isObjInSever(message.Objname) {
			AppLog.Println("Received message is", message)
			AppLog.Fatalln("Server does not store the object, Incorrect parameters")
		}

		switch message.Objparams.Algorithm {
		case ABD, ABD_FAST:

			message_reply = ABD_responses(message)

		case SODAW, SODAW_FAST:

			message_reply, respond = SODAW_responses(message, worker, msg[0], msg_reply)
			if respond == false {
				continue
			}

		case PRAKIS:

			message_reply, respond = PRAKIS_responses(message, worker, msg[0])
			if respond == false {
				continue
			}

		default:
			AppLog.Println("Algorithm ", message.Objparams.Algorithm, "is ", errors.New("Unknown Algorithm"))
		}

		//  Send Reply
		msg_reply[0] = msg[0]
		bytes_buffer_temp := CreateGobFromMessage(message_reply)
		msg_reply[1] = bytes_buffer_temp.Bytes()
		header_message := Message{Objname: message_reply.Objname, Opnum: message_reply.Opnum,
			Phase: message_reply.Phase, Sender: message_reply.Sender}
		bytes_buffer_header := CreateGobFromMessage(header_message)
		msg_reply[2] = bytes_buffer_header.Bytes()
		worker.SendMessage(msg_reply)

	}
}

// Server Utility functions
func isObjInSever(Objname string) bool {
	_, isPresent := processParams.ObjCatalogue[Objname]
	return isPresent

}

func generateOperationID(sender string, opnum int) string {

	return sender + "_" + strconv.Itoa(opnum)
}
