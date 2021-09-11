package daemons

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	zmq3 "github.com/pebbe/zmq3"
	//"log"
	//"math/rand"
	//"strconv"
	//"time"
)

type Client struct {
	client_type int
	client_name string
	params      ObjParams
	Opnum       int
	connection  map[string]*zmq3.Socket // the key is the server name
	poller      *zmq3.Poller
	Phase       string
	seed        int
	incarnation int

	// probability
	rand              *rand.Rand
	writerfailureprob float32
	readerfailureprob float32

	ObjectPickDistribution string

	objectSelector ObjectSelector
}

func (c *Client) setObjectDistribution() {
	c.ObjectPickDistribution = appParams.ObjectPickDistribution
}

func (c *Client) setFailureModel() {
	c.writerfailureprob = appParams.WriterFailureProbability
	c.readerfailureprob = appParams.ReaderFailureProbability
	c.rand = rand.New(rand.NewSource(100))
}

func (c *Client) initObjectSelector() {
	fmt.Println(len(processParams.ObjCatalogue), appParams.ObjectPickDistribution)
	objectSelector, err := createObjectPicker(int64(len(processParams.ObjCatalogue)), appParams.ObjectPickDistribution)
	c.objectSelector = objectSelector
	if err != nil {
		panic(err)
	}
}

func (c *Client) setSeed(seed int) {
	c.seed = seed
}

func (c *Client) getCurrentNameTag() string {
	return c.client_name + "-" + strconv.Itoa(c.incarnation)
}

func (c *Client) getSeed() int {
	return c.seed
}

func (c *Client) hasFailed() bool {

	if c.client_type == READER {
		if c.rand.Float32() < c.readerfailureprob {
			return true
		}
	}
	if c.client_type == WRITER {
		if c.rand.Float32() < c.writerfailureprob {
			return true
		}
	}

	return false
}

func (c *Client) emulateNewClient() {
	c.incarnation += 1
}

func (c *Client) deleteSocket(server string) {
	c.connection[server].Close()
}

func (c *Client) isMessageUseful(Opnum int, Objname string, Phase string) bool {

	if (c.Opnum == Opnum) && (c.params.Objname == Objname) && (c.Phase == Phase) {
		return true
	} else {
		return false
	}
}

func (c *Client) createSocketConnections(serversToConnect []string) {
	for _, server := range serversToConnect {
		c.connection[server] = createDealerSocket([]string{server})
	}
}

func (c *Client) setMaxBackLogPerConnection(serversToConnect []string) {
	for _, server := range serversToConnect {
		c.connection[server].SetBacklog(MAX_BACKLOG)
	}
}

func (c *Client) definePoller(serversToConnect []string) {
	for _, socket := range c.connection {
		c.poller.Add(socket, zmq3.POLLIN)
	}
}

func (r *Client) setParams(params ObjParams, Opnum int, client_type int, client_name string) {
	r.params = params
	r.Opnum = Opnum
	//r.connection = createDealerSocket(r.params.Servers_names)
}

func (r *Client) setPhase(Phase string) {

	r.Phase = Phase
}

func (c *Client) setType(client_type int) {
	c.client_type = client_type
}

func (c *Client) setClientID(client_name string) {
	c.client_name = client_name
}

// Used during ABD write or read. During write Query servers for Tag, during read Query servers for Tag and data
func (c *Client) QueryServers() {
	// create the message. Only header, no payload when querying server
	var message Message

	message.Objname = c.params.Objname
	message.Opnum = c.Opnum
	message.Phase = c.Phase
	message.Objparams = c.params
	message.TagValue_var = TagValue{Tag_var: Tag{Client_id: "", Version_num: 0}, Value: make([]byte, 0)}
	message.Sender = c.client_name

	message_to_send := CreateGobFromMessage(message)

	for _, server := range c.params.Servers_names {
		c.connection[server].SendBytes(message_to_send.Bytes(), NON_BLOCKING)
		//serverMessageCountUp(server)
	}

}

func (c *Client) WriteToServers(TagValue_var TagValue) {

	var message Message

	message.Objname = c.params.Objname
	message.Opnum = c.Opnum
	message.Phase = c.Phase
	message.TagValue_var = TagValue_var
	message.Objparams = c.params
	message.Sender = c.client_name

	message_to_send := CreateGobFromMessage(message)

	for _, server := range c.params.Servers_names {
		c.connection[server].SendBytes(message_to_send.Bytes(), NON_BLOCKING)
		//serverMessageCountUp(server)

	}

}

func (c *Client) WriteCodedValuesToServers(TagValue_var TagValue) {

	var message Message

	message.Objname = c.params.Objname
	message.Opnum = c.Opnum
	message.Phase = c.Phase
	message.Objparams = c.params
	message.Sender = c.client_name

	// Create the Array of Coded Elements, We measure the time taken for encoding
	start_encoding := time.Now()
	codedElementsArray := generateCodedElements(TagValue_var.Value, c.params.CodeParams)
	elapsed_encoding := time.Since(start_encoding)
	opParams.EncodingTime = elapsed_encoding

	for i, server := range c.params.Servers_names {
		//  Check why dont we add the opnum ?
		message.TagValue_var = TagValue{Tag_var: TagValue_var.Tag_var, Value: codedElementsArray[i], Seed: TagValue_var.Seed, CodeIndex: i, Opnum: TagValue_var.Opnum}
		message_to_send := CreateGobFromMessage(message)

		c.connection[server].SendBytes(message_to_send.Bytes(), NON_BLOCKING) // currently ZMQ sends to all servers in a round robin fashion,

	}

}

func (w *Client) ReceiveTagsFromQuorum(quorum_size int) []Tag {
	tags_rx := make([]Tag, quorum_size)

	messages_rx := w.ReceiveMessagesFromQuorum(quorum_size)

	// receive the tags
	for i := 0; i < quorum_size; i++ {
		tags_rx[i] = messages_rx[i].TagValue_var.Tag_var
	}

	return tags_rx
}

// Function returns number of ACKS received
func (w *Client) ReceiveAcksFromQuorum(quorum_size int) int {
	messages_rx := w.ReceiveMessagesFromQuorum(quorum_size)
	return len(messages_rx)
}

func (w *Client) ReceiveTagValuesFromQuorum(quorum_size int) []TagValue {
	tagvalue_rx := make([]TagValue, quorum_size)
	messages_rx := w.ReceiveMessagesFromQuorum(quorum_size)

	// receive the tags
	for i := 0; i < quorum_size; i++ {
		tagvalue_rx[i] = messages_rx[i].TagValue_var
	}

	return tagvalue_rx
}

func (w *Client) ReceiveMessagesFromQuorum(quorum_size int) []Message {

	messages_rx := make([]Message, quorum_size)
	num_responses := 0

	//  Process messages from every connected socket
	for num_responses < quorum_size {
		sockets, _ := w.poller.Poll(-1) // blocking
		for _, socket := range sockets {
			msgBytes, _ := socket.Socket.RecvMessageBytes(0)
			// messages from the channel without processing them
			//message1 := CreateMessageFromGob(msgBytes)
			//serverMessageCountDown(message1.Sender)
			//AppLog.Printf("SENDER %v %v", message1.Sender, server_response[message1.Sender])
			//AppLog.Printf("SENDER I hate to be here")
			if num_responses < quorum_size {
				messageheader := CreateMessageFromGob(msgBytes[1])
				if w.isMessageUseful(messageheader.Opnum, messageheader.Objname, messageheader.Phase) {
					messages_rx[num_responses] = CreateMessageFromGob(msgBytes[0])
					num_responses++
				}
			}

		}
	}

	return messages_rx
}

func (c *Client) clearIncomingChannels() {

	sockets, _ := c.poller.Poll(1) // 1 nano second time out
	for len(sockets) > 0 {
		for _, socket := range sockets {
			_, _ = socket.Socket.RecvMessageBytes(0)
		}
		sockets, _ = c.poller.Poll(1)
	}
}

func (r *Client) read() TagValue {

	var TagValue_var TagValue

	switch r.params.Algorithm {
	case ABD:
		TagValue_var = r.read_ABD()

	case ABD_FAST:
		TagValue_var = r.read_ABD_FAST()

	case SODAW:

		TagValue_var = r.read_SODAW()

	case SODAW_FAST:

		TagValue_var = r.read_SODAW_FAST()

	case PRAKIS:

		TagValue_var = r.read_PRAKIS()

	default:
		AppLog.Panicln(errors.New("Unknown Algorithm"))
	}
	return TagValue_var
}

func (w *Client) write(Value []byte) Tag {

	var Tag_var Tag
	switch w.params.Algorithm {
	case ABD:

		Tag_var = w.write_ABD(Value)

	case ABD_FAST:

		Tag_var = w.write_ABD_FAST(Value)

	case SODAW:

		Tag_var = w.write_SODAW(Value)

	case SODAW_FAST:

		Tag_var = w.write_SODAW_FAST(Value)

	case PRAKIS:

		Tag_var = w.write_PRAKIS(Value)

	default:
		AppLog.Panicln(errors.New("Unknown Algorithm"))
	}

	return Tag_var

}
