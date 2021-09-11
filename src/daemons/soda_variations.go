/* This file implements algorithms that are variations of the IPDPS 2016 SODA algorithm

MIT License
Copyright (c) 2017 Ercost

Authors:

Prakash Narayana Moorthy (prakashnarayanamoorthy@gmail.com)
Kishori Mohan Konwar (kishori82@gmail.com)

*/

package daemons

import (
	"math"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	zmq3 "github.com/pebbe/zmq3"
	//"log"
	//"math/rand"
	//"strconv"
	//"time"
)

// Implements getTag Phase of SODAW write operation. This is same as the one used
// for ABD algorithm. So we simply reuse the function written for ABD
func (w *Client) getTagSodaw() Tag {

	return w.getTag()

}

func (r *Client) getTagDataSodawFast() (TagValue, bool) {

	var maxTag Tag
	var CodeIndex, seed int

	quorum_size_majority := int(math.Ceil((float64(len(r.params.Servers_names)) + 1) / 2.0))
	code_K := math.Floor(float64(r.params.CodeParams.Rate) * float64(r.params.Numservers))
	codeParams := r.params.CodeParams
	rx_state := make(map[string][]byte, 0)
	isOnePhase := true

	// Query All servers for Tag and Coded Value
	r.QueryServers()

	// Receive TagCoded Values from K servers
	tagCodedElements_rx := r.ReceiveTagValuesFromQuorum(int(code_K)) // We are simply resuing the function from ABD
	if len(tagCodedElements_rx) != int(code_K) {
		panic("Unexpected Number of Responses Received during getData Phase of SODAW FAST")
	}

	// Loop over the K responses
	for i, TagCodedElement_rx := range tagCodedElements_rx {

		CodeIndex = TagCodedElement_rx.CodeIndex

		if i == 0 { // First packet, simply save the received packet, and note the received tag
			rx_state[strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value
			maxTag = TagCodedElement_rx.Tag_var
		} else { // Next Received Packet onwards check is max Tag has increased.

			// If max tag has not changed, then there is a chance that we can complete the read in One Phase
			isOnePhase = isOnePhase && reflect.DeepEqual(TagCodedElement_rx.Tag_var, maxTag)

			// Update max tag
			if maxTag.IsSmallerThan(TagCodedElement_rx.Tag_var) {
				maxTag = TagCodedElement_rx.Tag_var
			}

			// If OnePhase possibility is still true, continue logging received coded element
			if isOnePhase {
				rx_state[strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value
			} else { // else wait until we get majority responses, and simply retturn max tag. OnePhase is false here
				if i >= quorum_size_majority-1 {

					return TagValue{Tag_var: maxTag}, false
				}

			}

		}

		// All K received coded elements will have the same seed, since they belong to the same tag.
		// Note that this seed is used only for safety check
		if i == int(code_K)-1 {
			seed = TagCodedElement_rx.Seed
		}

	}

	start_decoding := time.Now()
	decodedValue := getDecodedValue(rx_state, codeParams)
	elapsed_decoding := time.Since(start_decoding)
	opParams.DecodingTime = elapsed_decoding
	return TagValue{Tag_var: maxTag, Value: decodedValue, Seed: seed}, true
}

func (r *Client) QueryServersforReadSodaw(tag Tag) {
	tagvalue := TagValue{Tag_var: tag, Value: make([]byte, 0)}
	r.WriteToServers(tagvalue) // same function as in ABD
}

// Implements getData Phase of Sodaw read operation
func (r *Client) getDataSodaw(tag_req Tag) TagValue {

	var TagValue_return TagValue

	code_K := math.Floor(float64(r.params.CodeParams.Rate) * float64(r.params.Numservers))
	codeParams := r.params.CodeParams
	// Start requesting for Tag, Value pairs from all servers
	r.QueryServersforReadSodaw(tag_req)

	decodeTagFound := false

	// Variable to accumulate relayed coded elements
	rx_state := make(map[Tag]map[string][]byte, 0)
	var CodeIndex int
	var message_rx Message

	//Keep listening to relay messages, and check for decodability
	for decodeTagFound != true {

		sockets, _ := r.poller.Poll(-1)
		for _, socket := range sockets {

			msgBytes, _ := socket.Socket.RecvMessageBytes(0) // keeping this line out of the if loop helps to drain out un-necessary
			// messages from the channel without processing them
			if decodeTagFound != true {
				messageheader := CreateMessageFromGob(msgBytes[1])
				if r.isMessageUseful(messageheader.Opnum, messageheader.Objname, messageheader.Phase) {
					message_rx = CreateMessageFromGob(msgBytes[0])
					TagCodedElement_rx := message_rx.TagValue_var
					CodeIndex = TagCodedElement_rx.CodeIndex

					if TagCodedElement_rx.Tag_var.IsSmallerThan(tag_req) {

						AppLog.Println("Reader got a tag less than requested tag. This is unexpected!, Exiting")
						AppLog.Println("Requested Tag is", tag_req, " Received Tag is", TagCodedElement_rx.Tag_var)
						os.Exit(9)
					}

					if _, tagExists := rx_state[TagCodedElement_rx.Tag_var]; !tagExists {
						rx_state[TagCodedElement_rx.Tag_var] = make(map[string][]byte, 0)
					}

					rx_state[TagCodedElement_rx.Tag_var][strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value

					if len(rx_state[TagCodedElement_rx.Tag_var]) == int(code_K) {

						// We measure the time taken for decodings
						start_decoding := time.Now()
						decodedValue := getDecodedValue(rx_state[TagCodedElement_rx.Tag_var], codeParams)
						elapsed_decoding := time.Since(start_decoding)
						opParams.DecodingTime = elapsed_decoding

						TagValue_return = TagValue{Tag_var: TagCodedElement_rx.Tag_var, Value: decodedValue, Seed: TagCodedElement_rx.Seed}
						decodeTagFound = true

					}

				}
			}

		}

	}

	return TagValue_return
}

func (r *Client) readCompleteSodaw() {

	r.QueryServersforReadSodaw(Tag{})

}

// Implements putData Phase of SODAW read/write operation
func (c *Client) putDataSodaw(TagValue_var TagValue) {

	erasurecode_K := math.Floor(float64(c.params.CodeParams.Rate) * float64(c.params.Numservers))

	// Send Tag and Coded Values to all servers
	c.WriteCodedValuesToServers(TagValue_var)

	//Receive Acks from K servers
	numAcks := c.ReceiveAcksFromQuorum(int(erasurecode_K))
	if numAcks != int(erasurecode_K) {
		panic("Unexpected Number of Acks Received during putData Phase")
	}
}

// Read
func (r *Client) read_SODAW() TagValue {

	// Phase 1, Get Tag
	var TagValue_var TagValue

	r.setPhase(SODAW_GET_TAG)

	tag_req := r.getTagSodaw() // same function as in ABD, based on majority responses

	r.setPhase(SODAW_GET_DATA)
	TagValue_var = r.getDataSodaw(tag_req)

	r.setPhase(SODAW_READ_COMPLETE)
	r.readCompleteSodaw()

	opParams.NumPhases = 2
	return TagValue_var
}

// Read Fast
func (r *Client) read_SODAW_FAST() TagValue {

	// Phase 1, Get Tag
	var TagValue_var TagValue

	r.setPhase(SODAW_FAST_GET_TAG_DATA) // we change the phase name here with respect to SODA. OTher wise phase names are same

	TagValue_var, IsOnePhase := r.getTagDataSodawFast() // same function as in ABD, based on majority responses

	if IsOnePhase {
		opParams.NumPhases = 1
		return TagValue_var
	}

	tag_req := TagValue_var.Tag_var

	r.setPhase(SODAW_GET_DATA)
	TagValue_var = r.getDataSodaw(tag_req)

	r.setPhase(SODAW_READ_COMPLETE)
	r.readCompleteSodaw()

	opParams.NumPhases = 2
	return TagValue_var
}

// Write
func (w *Client) write_SODAW(Value []byte) Tag {

	// Phase 1, Get Tag

	w.setPhase(SODAW_GET_TAG)
	Tag_var := w.getTagSodaw() // same function as in ABD, based on majority responses

	// increment the integer part
	Tag_var.Version_num++
	//insert writer id
	Tag_var.Client_id = w.getCurrentNameTag()

	w.setPhase(SODAW_PUT_DATA)
	w.putDataSodaw(TagValue{Tag_var: Tag_var, Value: Value, Seed: w.getSeed(), CodeIndex: 0}) // Code Index is not used until we write to disk, in which case we take care just before writing to disk

	opParams.NumPhases = 2
	return Tag_var

}

// Write Fast
func (w *Client) write_SODAW_FAST(Value []byte) Tag {

	return w.write_SODAW(Value)

}

// Server Responses~~~~~~~~~~~~~~~~~~~~~~~~~~~~`
//===============================================

func SODAW_responses(message Message, worker *zmq3.Socket, msg []byte, msg_reply [][]byte) (Message, bool) {
	message_reply := message // message is universal format for implementation purposes, does not really confirm to ABD spec.
	message_reply.Sender = processParams.name

	switch message.Phase {

	case SODAW_GET_TAG:
		AppLog.Println(processParams.name + " responding to SODAW_GET_TAG Phase for object " + message.Objname + ". Request received from " + message.Sender)
		message_reply.TagValue_var.Tag_var = getTagRespSodaw(message.Objname) //message is only meta data

	case SODAW_FAST_GET_TAG_DATA:
		AppLog.Println(processParams.name + " responding to SODAW_FAST_GET_TAG_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)
		message_reply.TagValue_var = getTagDataRespSodawFast(message.Objname) //message is only meta data

	case SODAW_GET_DATA:

		AppLog.Println(processParams.name + " responding to SODAW_GET_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)
		var (
			sendResponse bool
		)

		message_reply.TagValue_var, sendResponse = getDataRespSodaw(message, msg) //the second argument is the zmq indentity

		if !sendResponse {
			return message_reply, false
		}

	case SODAW_READ_COMPLETE:

		AppLog.Println(processParams.name + " responding to SODAW_READ_COMPLETE Phase for object " + message.Objname + ". Request received from " + message.Sender)

		readCompleteRespSodaw(message, msg)
		return message, false

	case SODAW_PUT_DATA:

		AppLog.Println(processParams.name + " responding to SODAW_PUT_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)

		readersToRelay, opnums := putDataRespSodaw(message)

		AppLog.Println(processParams.name + " received set of readers to relay for " + message.Objname)

		if len(readersToRelay) > 0 {
			for i, reader := range readersToRelay {
				msg_reply[0] = reader

				message_reply.Opnum = opnums[i]
				message_reply.Phase = SODAW_GET_DATA
				bytes_buffer_temp_relay := CreateGobFromMessage(message_reply)
				msg_reply[1] = bytes_buffer_temp_relay.Bytes()
				header_message := Message{Objname: message_reply.Objname, Opnum: message_reply.Opnum,
					Phase: message_reply.Phase, Sender: message_reply.Sender}
				bytes_buffer_header := CreateGobFromMessage(header_message)
				msg_reply[2] = bytes_buffer_header.Bytes()
				worker.SendMessage(msg_reply)
				AppLog.Println("Send to reader", i, "out of ", len(readersToRelay), " readers.", "Opnum is ", opnums[i], "Tag is ", message_reply.TagValue_var.Tag_var)

			}

		}

		AppLog.Println(processParams.name+" completed relay operation for "+message.Objname, " Relayed to ", len(readersToRelay), " reads")

		// Create the message reply to the writer
		message_reply.TagValue_var.Value = make([]byte, 0) // remove data part so that message is only meta data
		message_reply.Opnum = message.Opnum
		message_reply.Phase = message.Phase

	default:

		AppLog.Fatalln("Invalid Phase for SODAW or SODAW FAST algorithm! Exiting Code")
	}

	return message_reply, true
}

// Server Side Functions
//==============================

func getDataRespSodaw(message Message, zmq_identity []byte) (TagValue, bool) {

	obj_name := message.Objname
	tag_req := message.TagValue_var.Tag_var
	tag_local := server_state_variables[obj_name].tagCodedElement_var.Tag_var

	// Check if read is already outdated (and hence exists)

	isReadOutDated := doesReadExist(obj_name, message.Sender, message.Opnum) // if it exists, then by desgin, it is outdated

	if isReadOutDated {
		unregisterRead(message, zmq_identity)
		return TagValue{}, false
	}

	// If not add the read to the set of outstanding reads
	registerNewRead(message, zmq_identity, false)

	// Serve the reader with the local tag, coded element if the local tag is greater than or equal to the request tag
	// if local tag is smaller than the request tag, then we return saying that the local tag is smaller, so nothing will be sent back to the reader
	if tag_local.IsSmallerThan(tag_req) {
		return TagValue{}, false
	}

	if appParams.UseDisk {
		server_state_variables[obj_name].tagCodedElement_var, _ = ReadFromDisk(obj_name, appParams.FolderToStore)

	}

	return server_state_variables[obj_name].tagCodedElement_var, true

}

func readCompleteRespSodaw(message Message, zmq_identity []byte) {

	// Check if read is already outdated (and hence exists)
	isReadPresent := doesReadExist(message.Objname, message.Sender, message.Opnum) // if it exists, then by desgin, it is outdated

	if isReadPresent {
		unregisterRead(message, zmq_identity)
		return
	}

	// If not add the read to the set of outstanding reads but with an outdated Flag
	registerNewRead(message, zmq_identity, true)
}

func putDataRespSodaw(message Message) ([][]byte, []int) {

	readersToServe := make([][]byte, 0)
	opnums := make([]int, 0)

	rxTagCodedElementPair := message.TagValue_var
	rxTag := rxTagCodedElementPair.Tag_var

	objName := message.Objname

	for _, readParams := range server_state_variables[objName].readsInProgress {
		if !isGreaterTag(readParams.tag_request, rxTag) && readParams.outdated == false {
			readersToServe = append(readersToServe, readParams.zmq_identity)
			opnums = append(opnums, readParams.opnum)
		}
	}

	if isGreaterTag(rxTag, server_state_variables[objName].tagCodedElement_var.Tag_var) {
		if appParams.UseDisk {
			WriteToDisk(objName, appParams.FolderToStore, rxTagCodedElementPair)
			runtime.GC()
			server_state_variables[objName].tagCodedElement_var = TagValue{Tag_var: rxTag}

		} else {
			server_state_variables[objName].tagCodedElement_var = rxTagCodedElementPair
		}

	}

	return readersToServe, opnums
}

func registerNewRead(message Message, zmq_identity []byte, isOutdated bool) {

	objName := message.Objname

	newRead := &ReadsInProgress{
		reader_name:  message.Sender,
		tag_request:  message.TagValue_var.Tag_var,
		opnum:        message.Opnum,
		zmq_identity: zmq_identity,
		outdated:     isOutdated}

	read_operation_id := generateOperationID(message.Sender, message.Opnum)

	server_state_variables[objName].readsInProgress[read_operation_id] = newRead
}

func unregisterRead(message Message, zmq_identity []byte) {

	objname := message.Objname
	read_operation_id := generateOperationID(message.Sender, message.Opnum)
	delete(server_state_variables[objname].readsInProgress, read_operation_id)
}

func doesReadExist(objname string, reader string, opnum int) bool {

	read_operation_id := generateOperationID(reader, opnum)
	_, OK := server_state_variables[objname].readsInProgress[read_operation_id]

	if OK {
		return true
	}

	return false
}

func getTagRespSodaw(Objname string) Tag {

	return server_state_variables[Objname].tagCodedElement_var.Tag_var
}

func getTagDataRespSodawFast(Objname string) TagValue {

	return server_state_variables[Objname].tagCodedElement_var
}
