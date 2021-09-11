/* This file implements algorithms based on erasure codes that use Lists for temporary storage at the servers

MIT License
Copyright (c) 2017 Ercost

Authors:

Prakash Narayana Moorthy (prakashnarayanamoorthy@gmail.com)
Kishori Mohan Konwar (kishori82@gmail.com)

*/

package daemons

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"time"

	zmq3 "github.com/pebbe/zmq3"
)

// Write Functions

//Phase 1 Write Function
func (w *Client) putDataPrakis(value []byte) Tag {

	// Send Coded Values to all servers. Since the function accepts tagvalue as input, create empty tag
	tagvalue := TagValue{Value: value, Opnum: w.Opnum, Tag_var: Tag{Client_id: w.client_name}, Seed: w.seed}
	w.WriteCodedValuesToServers(tagvalue)

	//Receive tags from K servers

	erasurecode_K := math.Floor(float64(w.params.CodeParams.Rate) * float64(w.params.Numservers))
	tags_rx := w.ReceiveTagsFromQuorum(int(erasurecode_K))
	if len(tags_rx) != int(erasurecode_K) {
		panic("Unexpected Number of Tags Received during putData Phase of PRAKIS algorithm")
	}

	// Compute Max Tag. We exepct every received tag to have client id = this writer's id. So we only check the integer part
	var maxz = -1

	for _, tag := range tags_rx {
		if tag.Client_id != w.client_name {
			panic("Unexpected client Id obtained during putData Phase of PRAKIS algorithm")
		}

		if tag.Version_num > maxz {
			maxz = tag.Version_num
		}
	}

	return Tag{Client_id: w.client_name, Version_num: maxz}

}

//Phase 2 Write Function
func (w *Client) putTagPrakis(tag Tag) {

	// Send the tag to all servers. Use the multi purpose function WriteToServers by setting value as empty
	tagvalue := TagValue{Tag_var: tag, Value: make([]byte, 0), Opnum: w.Opnum}
	w.WriteToServers(tagvalue)

	//Receive Acks from K servers
	erasurecode_K := math.Floor(float64(w.params.CodeParams.Rate) * float64(w.params.Numservers))
	numAcks := w.ReceiveAcksFromQuorum(int(erasurecode_K))
	if numAcks != int(erasurecode_K) {
		panic("Unexpected Number of Acks Received during putTag Phase of Prakis")
	}

}

//=================================================================================================

// Read Functions

//Phase 1 Read Function
//This function is very much like getTagDataSodawFast. The only difference is that here we log the recieved opnum as well, possibly
//to be used in the next phase.
func (r *Client) getTagDataPrakis() (TagValue, bool) {
	var maxTag Tag
	var CodeIndex, seed, opnum_req int

	code_K := math.Floor(float64(r.params.CodeParams.Rate) * float64(r.params.Numservers))
	codeParams := r.params.CodeParams
	rx_state := make(map[string][]byte, 0)
	isOnePhase := true

	// Query All servers for Tag and Coded Value
	r.QueryServers()

	// Receive TagCoded Values from K servers
	tagCodedElements_rx := r.ReceiveTagValuesFromQuorum(int(code_K)) // We are simply resuing the function from ABD
	if len(tagCodedElements_rx) != int(code_K) {
		panic("Unexpected Number of Responses Received during getData Phase of Prakis")
	}

	// Loop over the K responses
	codeindex_temp := make([]int, 0)
	writer_id_temp := make([]string, 0)
	for i, TagCodedElement_rx := range tagCodedElements_rx {

		CodeIndex = TagCodedElement_rx.CodeIndex
		codeindex_temp = append(codeindex_temp, CodeIndex)
		writer_id_temp = append(writer_id_temp, TagCodedElement_rx.Tag_var.Client_id)

		if i == 0 { // First packet, simply save the received packet, and note the received tag
			rx_state[strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value
			maxTag = TagCodedElement_rx.Tag_var
			opnum_req = TagCodedElement_rx.Opnum // this line is not there in SodawFast

		} else { // Next Received Packet onwards check is max Tag has increased.

			// If max tag has not changed, then there is a chance that we can complete the read in One Phase
			isOnePhase = isOnePhase && reflect.DeepEqual(TagCodedElement_rx.Tag_var, maxTag)

			// Update max tag
			if maxTag.IsSmallerThan(TagCodedElement_rx.Tag_var) {
				maxTag = TagCodedElement_rx.Tag_var
				opnum_req = TagCodedElement_rx.Opnum
			}

			// If OnePhase possibility is still true, continue logging received coded element
			if isOnePhase {
				rx_state[strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value
			}

		}
	}

	if isOnePhase {

		fmt.Println("Code Index Array:", codeindex_temp)
		fmt.Println("Writer Anme Array:", writer_id_temp)
		fmt.Println("Object name", r.params.Objname, "Opnum:", r.Opnum)
		start_decoding := time.Now()
		decodedValue := getDecodedValue(rx_state, codeParams)
		elapsed_decoding := time.Since(start_decoding)
		opParams.DecodingTime = elapsed_decoding
		seed = tagCodedElements_rx[0].Seed                                      // All K tuples will have the same seed, use any
		return TagValue{Tag_var: maxTag, Value: decodedValue, Seed: seed}, true //opnum is not needed in this case

	}

	return TagValue{Tag_var: maxTag, Opnum: opnum_req}, false

}

//Phase 2 Read Auxilliary Function
func (r *Client) QueryServersforRelayPrakis(tag Tag, opnum int) {
	tagvalue := TagValue{Tag_var: tag, Opnum: opnum, Value: make([]byte, 0)}
	r.WriteToServers(tagvalue) // same function as in ABD
}

//Phase 2 Read Auxilliary Function
func (r *Client) commitTagPrakis(tag Tag, opnum int) {
	tagvalue := TagValue{Tag_var: tag, Opnum: opnum, Value: make([]byte, 0), ToCommit: true}
	r.WriteToServers(tagvalue) // same function as in ABD
}

//Phase 2 Read Function
// Like Phase 1, this function is also very similar to the one used on SodawFast, but we write it in full for easy understanding
func (r *Client) relayDataPrakis(tagReq Tag, opnumReq int) TagValue {
	var tagvalueRead TagValue

	code_K := math.Floor(float64(r.params.CodeParams.Rate) * float64(r.params.Numservers))
	codeParams := r.params.CodeParams
	// Start requesting for Tag, Value pairs from all servers
	r.QueryServersforRelayPrakis(tagReq, opnumReq)

	decodeTagFound := false

	// Variable to accumulate relayed coded elements
	rx_state := make(map[Tag]map[string][]byte)
	var CodeIndex int
	var message_rx Message
	tagsRequested := make(map[Tag]bool)

	//Keep listening to relay messages, and check for decodability
	for decodeTagFound != true {

		sockets, _ := r.poller.Poll(-1)
		for _, socket := range sockets {

			msgBytes, _ := socket.Socket.RecvMessageBytes(0) // keeping this line out of the if loop helps to drain out un-necessary
			// messages from the channel without processing them
			if decodeTagFound != true {
				messageheader := CreateMessageFromGob(msgBytes[1])
				DebugLog.Println("Read Opnum", r.Opnum)
				DebugLog.Println("Receieved Relay message for request tag", tagReq)
				if r.isMessageUseful(messageheader.Opnum, messageheader.Objname, messageheader.Phase) {
					message_rx = CreateMessageFromGob(msgBytes[0])
					DebugLog.Println("Message is Useful")
					TagCodedElement_rx := message_rx.TagValue_var
					CodeIndex = TagCodedElement_rx.CodeIndex

					if TagCodedElement_rx.Tag_var.IsSmallerThan(tagReq) {

						AppLog.Println("Reader got a tag less than requested tag. This is unexpected!, Exiting")
						AppLog.Println("Requested Tag is", tagReq, " Received Tag is", TagCodedElement_rx.Tag_var)
						os.Exit(9)
					}

					if _, tagExists := rx_state[TagCodedElement_rx.Tag_var]; !tagExists {
						rx_state[TagCodedElement_rx.Tag_var] = make(map[string][]byte, 0)
					}

					rx_state[TagCodedElement_rx.Tag_var][strconv.Itoa(CodeIndex)] = TagCodedElement_rx.Value

					DebugLog.Println("Receieved ", len(rx_state[TagCodedElement_rx.Tag_var]), " codewords so far for", TagCodedElement_rx.Tag_var)
					if len(rx_state[TagCodedElement_rx.Tag_var]) == int(code_K) {

						// We measure the time taken for decodings
						start_decoding := time.Now()
						decodedValue := getDecodedValue(rx_state[TagCodedElement_rx.Tag_var], codeParams)
						elapsed_decoding := time.Since(start_decoding)
						opParams.DecodingTime = elapsed_decoding

						tagvalueRead = TagValue{Tag_var: TagCodedElement_rx.Tag_var, Value: decodedValue, Seed: TagCodedElement_rx.Seed}
						decodeTagFound = true
						DebugLog.Println("Read Opnum", r.Opnum, "Read Complete")

					} else if tagReq.IsSmallerThan(TagCodedElement_rx.Tag_var) {
						if _, OK := tagsRequested[TagCodedElement_rx.Tag_var]; !OK { // a tag higher than requested tag received,
							// send commit-tag request to all servers for this tag, but do this in the background. This is done to take care
							// of writer failures
							tagsRequested[TagCodedElement_rx.Tag_var] = true
							r.commitTagPrakis(TagCodedElement_rx.Tag_var, TagCodedElement_rx.Opnum) // Check if a thread is needed?
							DebugLog.Println("Artificial Read Commit for Tag", TagCodedElement_rx.Tag_var)

						}
					}

				} else {
					DebugLog.Println("Message is Useless")
				}
			}

		}

	}

	return tagvalueRead
}

//Phase 2 Read Function to ask servers to stop relaying. No ACK expected
func (r *Client) readCompletePrakis() {

	r.QueryServersforRelayPrakis(Tag{}, 0)

}

//===========================================================================================

// Write  PRAKIS : highlevel
func (w *Client) write_PRAKIS(value []byte) Tag {

	// Phase 1, PUT DATA

	w.setPhase(PRAKIS_PUT_DATA)
	tag := w.putDataPrakis(value) // tag corresponds to the tag of this write operation

	// Phase 2, PUT TAG

	w.setPhase(PRAKIS_PUT_TAG)
	w.putTagPrakis(tag)

	opParams.NumPhases = 2
	return tag

}

// Read PRAKIS : highlevel
func (r *Client) read_PRAKIS() TagValue {

	var TagValue_var TagValue

	// Phase 1, Get Tag and Coded Data from K servers. If all tags are same, decode as well.
	r.setPhase(PRAKIS_GET_TAG_DATA)
	TagValue_var, IsOnePhase := r.getTagDataPrakis()

	if IsOnePhase {
		opParams.NumPhases = 1
		return TagValue_var
	}

	tag_req := TagValue_var.Tag_var
	opnum_req := TagValue_var.Opnum

	// Phase 2. Not all tags are same in phase 1, so we ask servers to relay coded elements, until decoding is finished.
	r.setPhase(PRAKIS_RELAY_DATA)
	TagValue_var = r.relayDataPrakis(tag_req, opnum_req)

	// Send read complete to every server before returning. We do not wait for any ACKs here. This helps the servers to stop relaying
	r.setPhase(PRAKIS_READ_COMPLETE)
	r.readCompletePrakis()

	opParams.NumPhases = 2
	return TagValue_var
}

//=========================================================================================
// Server Prakis : highlevel

func PRAKIS_responses(message Message, worker *zmq3.Socket, senderZMQID []byte) (Message, bool) {
	message_reply := message // message is universal format for implementation purposes, does not really confirm to ABD spec.
	message_reply.Sender = processParams.name
	var shouldRespond bool

	switch message.Phase {

	case PRAKIS_PUT_DATA:

		AppLog.Println(processParams.name + " responding to PRAKIS_PUT_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)
		tagReply, shouldCommit := putDataRespPrakis(message) // do we need to pass whole message ? Check:
		if shouldCommit {
			AppLog.Println(processParams.name + " tag already finalized, committing locally " + message.Objname)
			readersToRelay, readOpnums, tagValueToRelay := commitTagRespPrakis(message.Objname, message.TagValue_var.Tag_var, message.TagValue_var.Opnum)

			if len(readersToRelay) > 0 {
				AppLog.Println(processParams.name + " received set of readers to relay for " + message.Objname)
				message_reply.TagValue_var = tagValueToRelay
				relayMessage(readersToRelay, readOpnums, worker, message_reply) // relay message
				AppLog.Println(processParams.name+" completed relay operation for "+message.Objname, " Relayed to ", len(readersToRelay), " reads")
			} else {
				AppLog.Println(processParams.name + " Not relaying to any reader " + message.Objname)
			}

			// Nothing to respond to the writer, since the tag is already finalized
			message_reply = Message{}
			shouldRespond = false
		} else {

			// prepare reply message to writer
			message_reply.TagValue_var = TagValue{Tag_var: tagReply} // Insert the tag into message reply, other fields of TagValue are not required
			shouldRespond = true

		}

	case PRAKIS_GET_TAG_DATA:
		AppLog.Println(processParams.name+" responding to PRAKIS_GET_TAG_DATA Phase for object "+message.Objname+". Request received from "+message.Sender, "Opnum:", message.Opnum)
		message_reply.TagValue_var = getTagDataRespPrakis(message.Objname)
		shouldRespond = true

	case PRAKIS_RELAY_DATA:

		if message.TagValue_var.ToCommit {

			AppLog.Println(processParams.name + "responding to PRAKIS_READ_COMMIT_TAG Phase for object " + message.Objname + ". Request received from " + message.Sender)
			readersToRelay, readOpnums, tagValueToRelay := commitTagRespPrakis(message.Objname, message.TagValue_var.Tag_var, message.TagValue_var.Opnum) // commit tag

			if len(readersToRelay) > 0 {
				AppLog.Println(processParams.name + " received set of readers to relay for " + message.Objname)
				message_reply.TagValue_var = tagValueToRelay
				relayMessage(readersToRelay, readOpnums, worker, message_reply) // relay message
				AppLog.Println(processParams.name+" completed relay operation for "+message.Objname, " Relayed to ", len(readersToRelay), " reads")
			} else {
				AppLog.Println(processParams.name + " Not relaying to any reader " + message.Objname)
			}

			// Nothing to respond to the reader that send the commit tag request
			message_reply = Message{}
			shouldRespond = false
		} else {

			AppLog.Println(processParams.name+" responding to PRAKIS_RELAY_DATA Phase for object "+message.Objname+". Request received from "+message.Sender, "Opnum:", message.Opnum)

			// Register the read. There might or might not be a response back to the reader
			var tagValueReply TagValue
			tagValueReply, shouldRespond = registerReadPrakis(message, senderZMQID) //the second argument is the zmq indentity
			AppLog.Println(processParams.name+" Registered reader with ZMQ ID ", senderZMQID)

			// We pass the whole message since a lot of the fields are need to register the read operation

			// The requested tag can be committed, if present in the list
			readersToRelay, readOpnums, tagValueToRelay := commitTagRespPrakis(message.Objname, message.TagValue_var.Tag_var, message.TagValue_var.Opnum)
			if len(readersToRelay) > 0 {
				AppLog.Println(processParams.name + " received set of readers to relay for " + message.Objname)
				message_reply.TagValue_var = tagValueToRelay
				relayMessage(readersToRelay, readOpnums, worker, message_reply) // relay message
				AppLog.Println(processParams.name+" completed relay operation for "+message.Objname, " Relayed to ", len(readersToRelay), " reads")
			} else {
				AppLog.Println(processParams.name + " Not relaying to any reader " + message.Objname)
			}

			if shouldRespond {
				message_reply.TagValue_var = tagValueReply
			}
		}

	case PRAKIS_READ_COMPLETE:

		AppLog.Println(processParams.name+" responding to PRAKIS_READ_COMPLETE Phase for object "+message.Objname+". Request received from "+message.Sender, "Opnum:", message.Opnum)

		readCompleteRespPrakis(message.Objname, message.Sender, message.Opnum)

		// Nothing to respond to the reader
		message_reply = Message{}
		shouldRespond = false

	case PRAKIS_PUT_TAG:

		AppLog.Println(processParams.name + " responding to PRAKIS_PUT_TAG Phase for object " + message.Objname + ". Request received from " + message.Sender)

		readersToRelay, readOpnums, tagValueToRelay := commitTagRespPrakis(message.Objname, message.TagValue_var.Tag_var, message.TagValue_var.Opnum)

		if len(readersToRelay) > 0 {
			AppLog.Println(processParams.name + " received set of readers to relay for " + message.Objname)
			message_reply.TagValue_var = tagValueToRelay
			relayMessage(readersToRelay, readOpnums, worker, message_reply) // relay message
			AppLog.Println(processParams.name+" completed relay operation for "+message.Objname, " Relayed to ", len(readersToRelay), " reads")
		} else {
			AppLog.Println(processParams.name + " Not relaying to any reader " + message.Objname)
		}

		// Create the message reply to the writer
		message_reply.TagValue_var = TagValue{} // remove data part so that message is only meta data. We are only sending an ACK to the writer
		shouldRespond = true

	default:

		AppLog.Fatalln("Invalid Phase for PRAKIS algorithm! Exiting Code")
	}

	return message_reply, shouldRespond
}

// Server Functions

// Commit a tag. This function takes care of garbagge collection as well. Return the tag-value pair corresponding to the tag, and the set
// of outstanding readers to which this pair must be relayed
func commitTagRespPrakis(objname string, tag Tag, opnum int) (readersToRelay [][]byte, readOpnums []int, tagValueToRelay TagValue) {

	if isWriteInProgress(objname, tag, opnum) { // i.e., the corresponding coded value in list, so this can be finalized

		//current write information from list
		tagValuefromList := server_state_variables[objname].writesInProgress[tag.Client_id][opnum].tagValue

		// Store it in tagValueToRelay and also Change the tag to the incoming tag
		tagValueToRelay = tagValuefromList
		tagValueToRelay.Tag_var = tag

		// Update final tuple if incoming tag is higher than local finlaized
		// This is local write. Disk write is not implemented yet for Prakis
		if server_state_variables[objname].tagCodedElement_var.Tag_var.IsSmallerThan(tag) {
			server_state_variables[objname].tagCodedElement_var = tagValueToRelay
		}

		// Get the list of oustanding reads corresponding to this tag. Also, get the corresponding readOpnums
		readersToRelay = make([][]byte, 0)
		readOpnums = make([]int, 0)

		for _, readParams := range server_state_variables[objname].readsInProgress {
			if !isGreaterTag(readParams.tag_request, tag) {
				readersToRelay = append(readersToRelay, readParams.zmq_identity)
				readOpnums = append(readOpnums, readParams.opnum)
			}
		}

		// Perform garbagge collection
		delete(server_state_variables[objname].writesInProgress[tag.Client_id], opnum)

	} else if opnum > server_state_variables[objname].latestOpnumWriters[tag.Client_id] {
		// Add meta information to writesInProgress about the yet unseen write operation

		newWrite := &WritesInProgress{tagValue: TagValue{Tag_var: tag, Opnum: opnum}, isFinalized: true}

		if _, OK := server_state_variables[objname].writesInProgress[tag.Client_id]; !OK {
			server_state_variables[objname].writesInProgress[tag.Client_id] = make(map[int]*WritesInProgress)
		}

		server_state_variables[objname].writesInProgress[tag.Client_id][opnum] = newWrite
	}

	return
}

// Check whether the list contains an entry corresponding to the tag with label FIN
func isWriteFinalized(objname string, tag Tag, opnum int) bool {

	writerEntriesInList, OK := server_state_variables[objname].writesInProgress[tag.Client_id]
	if !OK {
		return false
	}

	writerEntryForGivenOpnum, OKK := writerEntriesInList[opnum]
	if !OKK {
		return false
	}

	if !writerEntryForGivenOpnum.isFinalized {
		return false
	}

	return true
}

// Check whether the list contains an entry corresponding to the tag with label Pre
func isWriteInProgress(objname string, tag Tag, opnum int) bool {

	writerEntriesInList, OK := server_state_variables[objname].writesInProgress[tag.Client_id]
	if !OK {
		return false
	}

	writerEntryForGivenOpnum, OKK := writerEntriesInList[opnum]
	if !OKK {
		return false
	}

	if writerEntryForGivenOpnum.isFinalized {
		return false
	}

	// found an entry in the list for the writer, and for the given opnum, that is not finalized (should contain the coded element also)

	if len(writerEntryForGivenOpnum.tagValue.Value) == 0 {
		AppLog.Panicln("Coded element missing for a Pre entry in List. Exiting Code")
	}

	return true
}

// Add the incoming tagvalue pair to list, if the tag is not yet final, else notify to finalise the tag (respond via shouldCommit)
func putDataRespPrakis(message Message) (tagOut Tag, shouldCommit bool) {

	objname := message.Objname
	tagIn := message.TagValue_var.Tag_var
	opnum := message.TagValue_var.Opnum
	if isWriteFinalized(objname, tagIn, opnum) {
		server_state_variables[objname].writesInProgress[tagIn.Client_id][opnum].isFinalized = false
		tagFinal := server_state_variables[objname].writesInProgress[tagIn.Client_id][opnum].tagValue.Tag_var
		server_state_variables[objname].writesInProgress[tagIn.Client_id][opnum].tagValue = message.TagValue_var
		server_state_variables[objname].writesInProgress[tagIn.Client_id][opnum].tagValue.Tag_var = tagFinal
		// if this is a finalized tag, we should use the final tag instead of the tag from the writer. Also, we change the isFinalized
		// state to false, only becasue the commit-tag function called below only works on PRE tags.
		shouldCommit = true
	} else {

		if isWriteInProgress(objname, tagIn, opnum) {
			AppLog.Panic("Write is in Progress even before receving the value from the writer. Something is Wrong!!!")
		}

		// initialize the inner map for writesInProgress, if the writer is encountered for the first time
		if _, OK := server_state_variables[objname].writesInProgress[tagIn.Client_id]; !OK {
			server_state_variables[objname].writesInProgress[tagIn.Client_id] = make(map[int]*WritesInProgress)
		}

		// Get the integer part of the finalized tag, and increment it, and make the new tag for this write
		newz := server_state_variables[objname].tagCodedElement_var.Tag_var.Version_num + 1
		newTag := Tag{Client_id: tagIn.Client_id, Version_num: newz}
		newTagValue := message.TagValue_var
		newTagValue.Tag_var = newTag
		newWrite := &WritesInProgress{tagValue: newTagValue, isFinalized: false}
		server_state_variables[objname].writesInProgress[tagIn.Client_id][opnum] = newWrite

		tagOut = newTag
		shouldCommit = false
	}
	return
}

// Respond with the local tuple to the reader
func getTagDataRespPrakis(objname string) TagValue {

	return server_state_variables[objname].tagCodedElement_var
}

// Register the read operation
func registerReadPrakis(message Message, senderZMQID []byte) (tagValueReply TagValue, shouldRespond bool) {

	// Register the new read
	objName := message.Objname
	newRead := &ReadsInProgress{
		reader_name:  message.Sender,
		tag_request:  message.TagValue_var.Tag_var,
		opnum:        message.Opnum,
		zmq_identity: senderZMQID}
	readOperationID := generateOperationID(message.Sender, message.Opnum)
	server_state_variables[objName].readsInProgress[readOperationID] = newRead

	// Prepare the retun message
	if !isGreaterTag(message.TagValue_var.Tag_var, server_state_variables[objName].tagCodedElement_var.Tag_var) {
		tagValueReply = server_state_variables[objName].tagCodedElement_var
		shouldRespond = true
	} else {
		shouldRespond = false
	}
	return
}

func readCompleteRespPrakis(objname string, reader string, opnum int) {

	//Unregister the read
	readOperationID := generateOperationID(reader, opnum)
	delete(server_state_variables[objname].readsInProgress, readOperationID)

	return
}

// Function that relays the message_reply to a set of outstanding readers. It is assumed that the tagValue is already a part of the message_reply input
func relayMessage(readersToRelay [][]byte, readOpnums []int, worker *zmq3.Socket, message_reply Message) {

	// ZMQ reply message (Created from algorithm reply message)
	msg_reply := make([][]byte, 3)

	for i := 0; i < len(msg_reply); i++ {
		msg_reply[i] = make([]byte, 0) // the frist frame  specifies the identity of the sender, the second specifies the content
	}

	message_reply.Phase = PRAKIS_RELAY_DATA

	for i, reader := range readersToRelay {
		msg_reply[0] = reader
		message_reply.Opnum = readOpnums[i]
		bytes_buffer_temp_relay := CreateGobFromMessage(message_reply)
		msg_reply[1] = bytes_buffer_temp_relay.Bytes()
		header_message := Message{Objname: message_reply.Objname, Opnum: message_reply.Opnum,
			Phase: message_reply.Phase, Sender: message_reply.Sender}
		bytes_buffer_header := CreateGobFromMessage(header_message)
		msg_reply[2] = bytes_buffer_header.Bytes()
		worker.SendMessage(msg_reply)
		AppLog.Println("Send to reader", i, "out of ", len(readersToRelay), " readers. ZMQ ID is ", reader, ". Opnum is ", readOpnums[i], "Tag is ", message_reply.TagValue_var.Tag_var)

	}
}
