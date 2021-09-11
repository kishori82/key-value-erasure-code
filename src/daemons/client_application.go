package daemons

import (
	b64 "encoding/base64"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"time"

	"github.com/pebbe/zmq3"
)

func InitializeOpnumTable() {
	OpnumTable = make(map[string]int)
	for key, _ := range processParams.ObjCatalogue {

		OpnumTable[key] = 0
	}
}

func Reader_process() {

	//firstread := true
	setupComplete = false
	active_chan_reader = make(chan bool, MAX_NUM_PENDING_READS)
	fmt.Println("channel  ", len(active_chan_reader), cap(active_chan_reader))

	// Initialize AppLog, SystemLog, DebugLog Log File Handlers
	//	initUpLogHandlers()

	AppLog.Println("INFO\tStarting Reader " + processParams.name)

	go HTTP_Server(processParams.appl_port)

	if processParams.remoteConfiguration {
		waitUntilParamsIsSet()
		InitializeParams()
		RemovePreviousLogFolder(appParams.FolderToStore)
		setupComplete = true
	}

	if ENABLE_SYSTEM_LOGS {
		//go DoEvery(1000*time.Millisecond, LogStats)
		go LogStats()
	}

	InitializeOpnumTable() // Check: Do we need this?

	// Make a writer client and set one time parameters, including connection details
	r := new(Client) //writer client
	r.client_name = processParams.name
	r.client_type = READER
	r.connection = make(map[string]*zmq3.Socket)
	// Create separate delear socket connection to every server
	serversToConnect := get_keys(deployParams.Servers)
	r.createSocketConnections(serversToConnect)
	//set maximum backlog per connection
	r.setMaxBackLogPerConnection(serversToConnect)
	// Create Poller
	r.poller = zmq3.NewPoller()
	r.definePoller(serversToConnect)

	r.setFailureModel()

	r.initObjectSelector()
	timelimit = time.Now().UnixNano() * 2

	initializeMiscellaneousVariables()

	for {
		if r.hasFailed() {
			r.emulateNewClient()
			r.createSocketConnections(serversToConnect)
			//set maximum backlog per connection
			r.setMaxBackLogPerConnection(serversToConnect)
			// Create Poller
			r.poller = zmq3.NewPoller()
			r.definePoller(serversToConnect)
		}

		if time.Now().UnixNano() > timelimit {
			for len(active_chan_reader) > 0 {
				<-active_chan_reader
			}
		}

		active := <-active_chan_reader //start

		if active == true && deployParams.Num_servers > 0 {

			/*if firstread {
				wait_before_firstread()
				firstread = false

			} else {
				waitForSometimeBetweenReads()
			} */

			waitForSometimeBetweenReads()
			r.clearIncomingChannels()
			doARead(r)

		}
	}

}

func Writer_process() {

	setupComplete = false

	active_chan_writer = make(chan bool, MAX_NUM_PENDING_WRITES)
	fmt.Println("channel  ", len(active_chan_writer), cap(active_chan_writer))
	// Initialize AppLog, SystemLog, DebugLog Log File Handlers
	//initUpLogHandlers()
	//
	//log.Println("INFO\tStarting Writer " + processParams.name)

	AppLog.Println("listening to ", processParams.appl_port)
	go HTTP_Server(processParams.appl_port)

	if processParams.remoteConfiguration {
		waitUntilParamsIsSet()

		InitializeParams()

		RemovePreviousLogFolder(appParams.FolderToStore)
	}
	//initUpLogHandlers()

	if ENABLE_SYSTEM_LOGS {
		//go DoEvery(1000*time.Millisecond, LogStats)
		go LogStats()
	}

	InitializeOpnumTable()

	// Make a writer client and set one time parameters, including connection details
	w := new(Client) //writer client
	w.client_name = processParams.name
	w.client_type = WRITER
	w.connection = make(map[string]*zmq3.Socket)
	// Create separate delear socket connection to every server
	serversToConnect := get_keys(deployParams.Servers)

	w.createSocketConnections(serversToConnect)
	//set maximum backlog per connection
	w.setMaxBackLogPerConnection(serversToConnect)

	w.poller = zmq3.NewPoller()
	w.definePoller(serversToConnect)

	w.setFailureModel()

	w.initObjectSelector()

	timelimit = time.Now().UnixNano() * 2

	initializeMiscellaneousVariables()

	for {
		if w.hasFailed() {
			w.emulateNewClient()
			w.createSocketConnections(serversToConnect)
			//set maximum backlog per connection
			w.setMaxBackLogPerConnection(serversToConnect)
			w.poller = zmq3.NewPoller()
			w.definePoller(serversToConnect)
		}

		if time.Now().UnixNano() > timelimit {
			for len(active_chan_writer) > 0 {
				<-active_chan_writer
			}
		}

		active := <-active_chan_writer //start

		if active == true && deployParams.Num_servers > 0 {

			waitForSometimeBetweenWrites()
			w.clearIncomingChannels()
			doAWrite(w)

		}
	}

}

func doARead(r *Client) {

	objectname := retrieveObjectnameForCurrentOperation(r.objectSelector)

	Opnum := incrementOpnum(objectname)

	objparams := *processParams.ObjCatalogue[objectname]

	AppLog.Println("READ START:", processParams.name, "Params:", objparams, " ObjectOpnum:", Opnum)

	opParams = OperationParams{} // This structure is used to collect any logs that ultimately need to be collected, We do not print any logs while the operation is in progresss

	r.params = objparams
	r.Opnum = Opnum

	global_start = time.Now()
	start := time.Now()

	TagValue_var := r.read()

	elapsed := time.Since(start)
	end := time.Now()
	global_read_elapsed = time.Since(global_start)

	AppLog.Println("READ END:", objectname, processParams.name, " ObjectOpnum:", Opnum, "Time taken:", elapsed, "Size Read", len(TagValue_var.Value))
	globalOpNum++
	if CHECK_SAFETY {
		datawritten := make([]byte, objparams.File_size) // check the first 1024 bytes
		rand.Seed(int64(TagValue_var.Seed))
		AppLog.Println(TagValue_var.Seed)
		rand.Read(datawritten)
		dataSafe := reflect.DeepEqual(datawritten, TagValue_var.Value)
		AppLog.Println("FOR_SAFETY_CHECK: ", "READ", processParams.name, objectname, Opnum, int64(start.UnixNano()/1e3), int64(end.UnixNano()/1e3), TagValue_var.Tag_var.Client_id, TagValue_var.Tag_var.Version_num, dataSafe)
	}

	writeMonitoredStats()

	if ENBALE_EXP_LOGS {
		getOperationParams(r)
		opParams.TotalTime = global_read_elapsed
		data := CreateLogLineForOperation(opParams)
		encdata := b64.StdEncoding.EncodeToString(data)
		ExpLog.Println(encdata)
	}

}

var globalOpNum int = 0

func doAWrite(w *Client) {

	var objectname string

	if globalOpNum < len(processParams.ObjCatalogue) { // this is for the initial writes
		objectname = retrieveObjectByIndex(globalOpNum)
	} else {
		objectname = retrieveObjectnameForCurrentOperation(w.objectSelector)
	}

	Opnum := incrementOpnum(objectname)

	objparams := *processParams.ObjCatalogue[objectname]

	AppLog.Println("WRITE START:", processParams.name, "Params:", objparams, " ObjectOpnum:", Opnum)

	// Set Client parameters that are dependent on the operation
	w.params = objparams
	w.Opnum = Opnum

	opParams = OperationParams{} // This structure is used to collect any logs that ultimately need to be collected, We do not print any logs while the operation is in progresss

	// generate data for write

	if globalOpNum == 0 { //Generate NUM_PKTS_WRITE Data packets. For each write, we pick one of these packets
		DataForAllWrites = make([][]byte, NUM_PKTS_WRITE)
		for ii := 0; ii < NUM_PKTS_WRITE; ii++ {
			DataForAllWrites[ii] = make([]byte, objparams.File_size)
			rand.Read(DataForAllWrites[ii])
		}
	}

	var seed int
	if CHECK_SAFETY {
		dataForWrite = make([]byte, objparams.File_size)
		seed = rand.Intn(1e9)
		rand.Seed(int64(seed))
		w.setSeed(seed)
		AppLog.Println(seed)
		rand.Read(dataForWrite)
	} else {
		dataForWrite = DataForAllWrites[rand.Intn(NUM_PKTS_WRITE)]
	}

	start := time.Now()

	Tag_var := w.write(dataForWrite)

	elapsed := time.Since(start)
	end := time.Now()

	AppLog.Println("WRITE END:", objectname, processParams.name, " ObjectOpnum:", Opnum, "Time taken:", elapsed)

	if CHECK_SAFETY {
		AppLog.Println("FOR_SAFETY_CHECK: ", "WRITE", processParams.name, objectname, Opnum, int64(start.UnixNano()/1e3), int64(end.UnixNano()/1e3), Tag_var.Client_id, Tag_var.Version_num, seed)
	}
	writeMonitoredStats()

	if ENBALE_EXP_LOGS {
		getOperationParams(w)
		opParams.TotalTime = elapsed
		data := CreateLogLineForOperation(opParams)
		encdata := b64.StdEncoding.EncodeToString(data)
		ExpLog.Println(encdata)
	}

	globalOpNum++
	if globalOpNum >= len(processParams.ObjCatalogue) {
		setupComplete = true
	}

}

func getOperationParams(c *Client) {
	opParams.ClientName = c.client_name
	opParams.Opnum = c.Opnum
	opParams.ObjParamsvar = c.params
}

func waitForSometime() {
	waitTimeNanoSeconds := rand.ExpFloat64() * float64(appParams.Wait_time_btw_op) * 1e6
	time.Sleep(time.Duration(waitTimeNanoSeconds))
}

// Wait for an exponential amounf of time. Average is appParams.Wait_time_btw_reads MilliSeconds
func waitForSometimeBetweenReads() {
	waitTimeNanoSeconds := rand.ExpFloat64() * float64(appParams.Wait_time_btw_reads) * 1e6
	time.Sleep(time.Duration(waitTimeNanoSeconds))
}

// Wait for an exponential amounf of time. Average is appParams.Wait_time_btw_writes MilliSeconds
func waitForSometimeBetweenWrites() {
	waitTimeNanoSeconds := rand.ExpFloat64() * float64(appParams.Wait_time_btw_writes) * 1e6
	time.Sleep(time.Duration(waitTimeNanoSeconds))
}

func wait_before_firstread() {
	waitTimeNanoSeconds := 10 * appParams.Wait_time_btw_op * 1e6
	time.Sleep(time.Duration(waitTimeNanoSeconds))
}

type Lexicon []string

func (s Lexicon) Len() int {
	return len(s)
}
func (s Lexicon) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Lexicon) Less(i, j int) bool {
	return s[i] < s[j]
}

func retrieveObjectByIndex(i int) string {
	var keys []string

	for key, _ := range processParams.ObjCatalogue {
		keys = append(keys, key)
	}

	sort.Sort(Lexicon(keys))

	return keys[i]
}

func retrieveObjectnameForCurrentOperation(randobj ObjectSelector) string {

	var keys []string

	for key, _ := range processParams.ObjCatalogue {
		keys = append(keys, key)
	}

	keytopick := randobj.Rand()

	//keytopick := rand.Intn(len(processParams.ObjCatalogue))

	Objname := keys[keytopick]

	return Objname
}

func incrementOpnum(Objname string) int {

	OpnumTable[Objname]++
	return OpnumTable[Objname]
}
