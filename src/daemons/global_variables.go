package daemons

import (
	"time"
)

//Client Application Params
var appParams ApplicationParams
var processParams ProcessParams
var deployParams DeploymentParameters
var dataForWrite []byte //used only for write operation
var DataForAllWrites [][]byte
var OpnumTable map[string]int
var setupComplete bool
var opParams OperationParams
var timelimit int64 = 0
var timeout time.Duration
var global_read_elapsed time.Duration
var global_start time.Time

type QuitChan struct {
	quit chan int
}

// SODAW parameters

//we define the reader and writer types only and not as variables, since these variables are stack variables (no scope after the opreation is complete)

//SODAW server state variables

var server_state_variables map[string]*SingleObjServerState

type ReadsInProgress struct {
	reader_name  string
	opnum        int
	tag_request  Tag
	zmq_identity []byte
	outdated     bool
}

type WritesInProgress struct {
	tagValue    TagValue
	isFinalized bool
}

type SingleObjServerState struct {
	tagCodedElement_var TagValue
	readsInProgress     map[string]*ReadsInProgress
	latestOpnumWriters  map[string]int                       // map of writers, used only in Prakis
	writesInProgress    map[string]map[int]*WritesInProgress // map of writers, map of opnums, used only in Prakis
}

// golabl channels

var active_chan_reader chan bool
var active_chan_writer chan bool
var reset_chan chan bool
var chan_app_params, chan_depl_params chan bool

/*func init() {
	active_chan_writer = make(chan bool, MAX_NUM_PENDING_WRITES)
}*/

type OperationParams struct {
	NumPhases    int
	EncodingTime time.Duration //useful only if the algorithm uses erasrure codes
	DecodingTime time.Duration //useful only if the algorithm uses erasrure codes
	StartTime    time.Duration
	EndTime      time.Duration
	TotalTime    time.Duration
	ClientName   string
	Opnum        int
	ObjParamsvar ObjParams
}
