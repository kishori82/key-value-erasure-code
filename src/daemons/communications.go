package daemons

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	zmq3 "github.com/pebbe/zmq3"
	//"math/rand"
	"strconv"
	//"time"
)

var server_response map[string]int

func initializeMiscellaneousVariables() {
	server_response = make(map[string]int)
	serversToConnect := get_keys(deployParams.Servers)
	for _, servername := range serversToConnect {
		server_response[servername] = 0
	}
}
func serverMessageCountUp(servername string) {
	server_response[servername] = server_response[servername] + 1
}
func serverMessageCountDown(servername string) {
	server_response[servername] = server_response[servername] - 1
}

func writeMonitoredStats() {
	serversToConnect := get_keys(deployParams.Servers)
	line := "SERVER RESPONSES"
	for _, servername := range serversToConnect {
		line = line + " " + strconv.Itoa(server_response[servername])
	}
	DebugLog.Printf("%s", line)
}

func get_ip_and_applport(process_name string) string {
	return get_ip_and_port(process_name, HTTP_PORT_IDX)
}

func get_ip_and_port(process_name string, PORT_IDX int) string {

	_, ok := deployParams.Name_to_processtype[process_name]
	if !ok {
		if PORT_IDX == HTTP_PORT_IDX {
			return "*:8080"
		}
		if PORT_IDX == ALGO_PORT_IDX {
			return "*:8081"
		}
	}

	process_type := deployParams.Name_to_processtype[process_name]
	switch process_type {
	case SERVER:
		return deployParams.Servers[process_name][0] + ":" + deployParams.Servers[process_name][PORT_IDX]
	case READER:
		return deployParams.Readers[process_name][0] + ":" + deployParams.Readers[process_name][PORT_IDX]
	case WRITER:
		return deployParams.Writers[process_name][0] + ":" + deployParams.Writers[process_name][PORT_IDX]
	default:
		log.Fatal("unknown process type")
		return ""
	}

}

func get_ip(process_name string) string {

	_, ok := deployParams.Name_to_processtype[process_name]
	if !ok {
		log.Fatal("Cannot find out the ip for ", processParams)
		return ""
	}

	process_type := deployParams.Name_to_processtype[process_name]
	switch process_type {
	case SERVER:
		return deployParams.Servers[process_name][0]
	case READER:
		return deployParams.Readers[process_name][0]
	case WRITER:
		return deployParams.Writers[process_name][0]
	default:
		log.Fatal("unknown process type")
		return ""
	}
}

func get_ip_and_algoport(process_name string) string {
	return get_ip_and_port(process_name, ALGO_PORT_IDX)
}

func createDealerSocket(servers []string) *zmq3.Socket {

	// create a dealer
	dealer, _ := zmq3.NewSocket(zmq3.DEALER)
	var IP string
	for _, server := range servers {
		if processParams.remoteConfiguration {
			IP = "tcp://" + get_ip(server) + ":" + ALGO_PORT
		} else {
			IP = "tcp://" + get_ip_and_algoport(server)
		}
		dealer.Connect(IP)
	}
	return dealer
}

func createRouterSocket() *zmq3.Socket {

	// Create a router socket
	router, err := zmq3.NewSocket(zmq3.ROUTER)

	if err != nil {
		log.Fatal("Cannot create router socket")
	}

	return router
}

func CreateJsonFromMessage(message Message) []byte {

	//var message_to_send bytes.Buffer // to send out with zmq
	// Create an encoder and send a Value.
	data, err := json.Marshal(message)

	//	err := message_to_respond_enc.Encode(message)
	if err != nil {
		AppLog.Panicln("Error gobfying message")
	}

	return data
}

func CreateMessageFromJson(messageBytes []byte) Message {

	var m Message

	err := json.Unmarshal(messageBytes, &m)
	if err != nil {
		AppLog.Panicln("Error gobfying message")
	}
	return m
}

func CreateGobFromMessage(message Message) bytes.Buffer {

	var message_to_send bytes.Buffer // to send out with zmq
	// Create an encoder and send a Value.
	enc := gob.NewEncoder(&message_to_send)
	err := enc.Encode(message)

	//	err := message_to_respond_enc.Encode(message)
	if err != nil {
		fmt.Println("Error gobfying message")
	}
	return message_to_send
}

func CreateMessageFromGob(messageBytes []byte) Message {

	var buffer bytes.Buffer
	var m Message

	buffer.Write(messageBytes)
	dec := gob.NewDecoder(&buffer)
	dec.Decode(&m)

	return m
}
