/* This file implements algorithms based on replication
MIT License
Copyright (c) 2017 Ercost

Authors:

Prakash Narayana Moorthy (prakashnarayanamoorthy@gmail.com)
Kishori Mohan Konwar (kishori82@gmail.com)

*/

package daemons

import (
	"math"
	"reflect"
	//"log"
	//"math/rand"
	//"strconv"
	//"time"
)

// Implements getData Phase of ABD or ABD_FAST read operation
func (r *Client) getData() (TagValue, bool) {

	var maxTagValue TagValue
	maxTagValue.Tag_var = Tag{"", -1}
	var quorum_size float64
	quorum_size = math.Ceil((float64(len(r.params.Servers_names)) + 1) / 2.0)
	// Start requesting for Tag, Value pairs from all servers
	r.QueryServers()
	IsOnePhase := true

	// Receive TagValues from a quorum
	tagvalues_rx := r.ReceiveTagValuesFromQuorum(int(quorum_size))
	if len(tagvalues_rx) != int(quorum_size) {
		panic("Unexpected Number of Tags Received during getTag Phase")
	}

	for i, tagvalue := range tagvalues_rx {

		if i > 0 {
			IsOnePhase = IsOnePhase && reflect.DeepEqual(tagvalue.Tag_var, maxTagValue.Tag_var)
		}

		if maxTagValue.Tag_var.IsSmallerThan(tagvalue.Tag_var) {
			maxTagValue = tagvalue
		}
	}

	return maxTagValue, IsOnePhase

}

// Implements getTag Phase of ABD write operation
func (w *Client) getTag() Tag {

	var maxTag = Tag{" ", -1}
	var quorum_size float64

	quorum_size = math.Ceil((float64(len(w.params.Servers_names)) + 1) / 2.0)

	// Start requesting for Tag, Value pairs from all servers
	w.QueryServers()

	//Receive Tags from a quorum
	AppLog.Println("Going to receive tags")
	tags_rx := w.ReceiveTagsFromQuorum(int(quorum_size))
	AppLog.Println("Received the tags")
	if len(tags_rx) != int(quorum_size) {
		panic("Unexpected Number of Tags Received during getTag Phase")
	}

	for _, tag := range tags_rx {
		if maxTag.IsSmallerThan(tag) {
			maxTag = tag
		}
	}

	return maxTag
}

// Implements putData Phase of ABD read/write operation
func (c *Client) putData(TagValue_var TagValue) {

	var quorum_size float64
	quorum_size = math.Ceil((float64(len(c.params.Servers_names)) + 1) / 2.0)

	// Send TagValue to all servers
	c.WriteToServers(TagValue_var)

	//Receive Acks from a quorum
	numAcks := c.ReceiveAcksFromQuorum(int(quorum_size))
	if numAcks != int(quorum_size) {
		panic("Unexpected Number of Acks Received during putData Phase")
	}

}

func (r *Client) read_ABD() TagValue {

	// Phase 1, Get Tag

	r.setPhase(ABD_GET_DATA)
	TagValue_var, _ := r.getData()

	r.setPhase(ABD_PUT_DATA)
	r.putData(TagValue_var)

	opParams.NumPhases = 2
	return TagValue_var
}

func (r *Client) read_ABD_FAST() TagValue {

	// Phase 1, Get Tag

	r.setPhase(ABD_GET_DATA)
	TagValue_var, IsOnePhase := r.getData()

	if IsOnePhase {
		opParams.NumPhases = 1
		return TagValue_var
	}

	r.setPhase(ABD_PUT_DATA)
	r.putData(TagValue_var)

	opParams.NumPhases = 2
	return TagValue_var
}

func (w *Client) write_ABD(Value []byte) Tag {

	// Phase 1, Get Tag

	w.setPhase(ABD_GET_TAG)
	Tag_var := w.getTag()

	// increment the integer part
	Tag_var.Version_num++
	//insert writer id
	//	Tag_var.Client_id = w.client_name
	Tag_var.Client_id = w.getCurrentNameTag()
	//w.client_name = w.getCurrentNameTag()

	w.setPhase(ABD_PUT_DATA)
	w.putData(TagValue{Tag_var: Tag_var, Value: Value, Seed: w.getSeed(), CodeIndex: 0})

	opParams.NumPhases = 2
	return Tag_var

}

func (w *Client) write_ABD_FAST(Value []byte) Tag {

	return w.write_ABD(Value)
	// Note that we do not bother giving new phase names here, since it is the same algorithm as ABD, and there is no point in distinguishing
}

// Server Responses~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//==============================================================
func getTagResp(Objname string) Tag {

	return server_state_variables[Objname].tagCodedElement_var.Tag_var
}

func getDataResp(Objname string) TagValue {

	return server_state_variables[Objname].tagCodedElement_var
}

func putDataResp(Objname string, TagValue_input TagValue) bool {

	if isGreaterTag(TagValue_input.Tag_var, server_state_variables[Objname].tagCodedElement_var.Tag_var) {
		server_state_variables[Objname].tagCodedElement_var = TagValue_input
		return true
	}

	return false
}

func ABD_responses(message Message) Message {

	message_reply := message // message is universal format for implementation purposes, does not really confirm to ABD spec.
	message_reply.Sender = processParams.name

	switch message.Phase {

	case ABD_GET_TAG:
		AppLog.Println(processParams.name + " responding to ABD_GET_TAG Phase for object " + message.Objname + ". Request received from " + message.Sender)
		message_reply.TagValue_var.Tag_var = getTagResp(message.Objname) //message is only meta data

	case ABD_GET_DATA:

		AppLog.Println(processParams.name + " responding to ABD_GET_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)
		message_reply.TagValue_var = getDataResp(message.Objname) //message contains actual data

	case ABD_PUT_DATA:

		AppLog.Println(processParams.name + " responding to ABD_PUT_DATA Phase for object " + message.Objname + ". Request received from " + message.Sender)
		dataChanged := putDataResp(message.Objname, message.TagValue_var)
		if dataChanged {
			AppLog.Println("Local (tag, value) updated with incoming pair")
		} else {
			AppLog.Println("Incoming (tag, value) pair ignored based on tag comparison")
		}

		message_reply.TagValue_var.Value = make([]byte, 0) // remove data part so that message is only meta data

	default:

		AppLog.Fatalln("Invalid Phase for ABD algorithm! Exiting Code")
	}

	return message_reply
}
