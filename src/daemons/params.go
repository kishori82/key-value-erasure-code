package daemons

import (

	//	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"

	"gopkg.in/yaml.v2"
)

func GetDeploymentParams() *DeploymentParameters {
	return &deployParams
}

func GetApplicationParams() *ApplicationParams {
	return &appParams
}

func GetProcessParams() *ProcessParams {
	return &processParams
}

// Service Parameters
type DeploymentParameters struct {
	Cluster     string
	Num_servers int
	Num_readers int
	Num_writers int

	Name_to_processtype map[string]int

	Controller []string //array of IP, port

	Max_num_obj  int
	Max_obj_size int
	Min_obj_size int

	Servers map[string][]string //array of IP, algo port, app port
	Readers map[string][]string
	Writers map[string][]string

	WhoAmI string
}

func (dep *DeploymentParameters) DisplayParams() {
	fmt.Println("Name to process  type ")
	for key, val := range dep.Name_to_processtype {
		fmt.Println("\t", key, val)
	}
	fmt.Println("Servers : ")
	for key, val := range dep.Servers {
		fmt.Println("\t", key, val)
	}
	fmt.Println("Writers : ")
	for key, val := range dep.Writers {
		fmt.Println("\t", key, val)
	}
	fmt.Println("Readers : ")
	for key, val := range dep.Readers {
		fmt.Println("\t", key, val)
	}
}

type ProcessParams struct {
	processType   int
	myipaddr      string
	name          string
	ObjCatalogue  map[string]*ObjParams // only contains information about objects that the process handles
	active        bool
	NumOperations int

	algo_port string // each process can have its own port to communicate with other processes in the algorithm
	appl_port string //this port is used by the application to talk to the service

	remoteConfiguration bool
}

func (pP *ProcessParams) GetProcessType() int {

	return pP.processType
}

func (pP *ProcessParams) DisplayParams() {
	fmt.Println("process type : ", pP.processType)
	fmt.Println("ip address : ", pP.myipaddr)
	fmt.Println("name : ", pP.name)
	fmt.Println("Object catalogue :")
	for key, _ := range pP.ObjCatalogue {
		fmt.Println("\t", key)
		for _, name := range pP.ObjCatalogue[key].Servers_names {
			fmt.Println("\t\t", name)
		}

	}
}

func (pP *ProcessParams) InitializeProcessParamsForRemote(process_name string) error {
	var isServer, isReader, isWriter = false, false, false

	if process_name[0] == 'r' {
		isReader = true
	}
	if process_name[0] == 'w' {
		isWriter = true
	}
	if process_name[0] == 's' {
		isServer = true
	}
	//	AppLog.Println("process name", process_name, isReader, isWriter, isServer, !(isServer || isReader || isWriter || process_name == "controller"))

	if !(isServer || isReader || isWriter || process_name == "controller") {
		return errors.New("Process name is out of range!")
	}
	pP.name = process_name
	pP.active = true

	if isServer {
		pP.processType = SERVER
	}
	if isReader {
		pP.processType = READER
	}
	if isWriter {
		pP.processType = WRITER
	}
	if process_name == "controller" {
		//AppLog.Println("Dockerx  setting type to control")
		pP.processType = CONTROLLER
	}
	pP.algo_port = ALGO_PORT
	pP.appl_port = HTTP_PORT
	pP.remoteConfiguration = true
	return nil
}

func (pP *ProcessParams) InitializeProcessParams(process_name string, remoteConfig bool) error {
	_, isServer := deployParams.Servers[process_name]
	_, isReader := deployParams.Readers[process_name]
	_, isWriter := deployParams.Writers[process_name]

	if !(isServer || isReader || isWriter || process_name == "controller") {
		return errors.New("Process name is out of range!")
	}

	pP.name = process_name
	pP.active = true
	pP.ObjCatalogue = appParams.MasterObjCatalogue // Fix this later based on a policy that decides which processes handle which files

	if isServer {
		pP.processType = SERVER
		pP.myipaddr = deployParams.Servers[process_name][0]
		pP.algo_port = deployParams.Servers[process_name][1]
		pP.appl_port = deployParams.Servers[process_name][2]
	}
	if isReader {
		pP.processType = READER
		pP.myipaddr = deployParams.Readers[process_name][0]
		pP.algo_port = deployParams.Readers[process_name][1]
		pP.appl_port = deployParams.Readers[process_name][2]
	}
	if isWriter {
		pP.processType = WRITER
		pP.myipaddr = deployParams.Writers[process_name][0]
		pP.algo_port = deployParams.Writers[process_name][1]
		pP.appl_port = deployParams.Writers[process_name][2]
	}
	if process_name == "controller" {
		pP.processType = CONTROLLER
		pP.myipaddr = deployParams.Controller[0]
		pP.appl_port = deployParams.Controller[1]
	}
	pP.remoteConfiguration = remoteConfig // whether this process is configured remotely or locally
	return nil
}

// obj parameters set once by the controller. kind of constants.
type ObjParams struct {
	Numservers    int
	Servers_names []string
	Objname       string
	Algorithm     string
	File_size     int
	CodeParams    ErasureCodeParams
}

func (Obj ObjParams) getNumServers() int {
	return Obj.Numservers
}

type ErasureCodeParams struct {
	CodeN            int
	CodeK            int
	Rate             float32
	Coding_algorithm string
	FileSize         int
}

type ApplicationParams struct {
	NumObjects int

	// The below three are for a specific application where one executes a constant number of evenly spaced read/write operations
	NumReadOperations    int
	NumWriteOperations   int
	Wait_time_btw_op     int // Fix the data type
	Wait_time_btw_reads  int
	Wait_time_btw_writes int

	FolderToStore string
	UseDisk       bool

	DefaultAlgorithm         string
	Num_servers_per_object   int
	Default_erasurecode_rate float32
	Default_file_size        int

	MasterObjCatalogue       map[string]*ObjParams // string is the Objname
	ObjectPickDistribution   string
	WriterFailureProbability float32
	ReaderFailureProbability float32
	LogInterval              int
}

// This structure needs to be placed in the right file.
type Message struct {
	Objname      string
	Opnum        int
	Phase        string
	TagValue_var TagValue // for sodaw value is the coded value
	Objparams    ObjParams
	Sender       string
}

func get_default_servers_for_object(num_servers_default int) []string {

	all_servers := get_keys(deployParams.Servers)
	return all_servers[0:num_servers_default]

}

func get_servers_for_object(objectname string) []string {
	num_servers := len(deployParams.Servers)
	seed := int(hash([]byte(objectname)))
	rand.Seed(int64(seed))

	all_servers := get_keys(deployParams.Servers)
	server_list := make([]string, appParams.Num_servers_per_object)
	index := rand.Intn(10000)
	for i := 0; i < appParams.Num_servers_per_object; i++ {
		server_list[i] = all_servers[(index+i)%num_servers]
	}
	//	fmt.Println("server list ", seed, objectname, server_list)
	return server_list
}

func (dP *DeploymentParameters) LoadDefaultDeploymentParams() {
	dP.Cluster = DEFAULT_CLUSTER
	dP.Num_servers = DEFAULT_NUM_SERVERS
	dP.Num_readers = DEFAULT_NUM_READERS
	dP.Num_writers = DEFAULT_NUM_WRITERS

	dP.Max_num_obj = DEFAULT_MAXNUMOBJ
	dP.Max_obj_size = DEFAULT_MAXOBJSIZEKB // kB
	dP.Min_obj_size = DEFAULT_MINOBJSIZEKB // kB */
}
func (dP *DeploymentParameters) InitializeDeploymentParams() {

	dP.Servers = make(map[string][]string)
	dP.Readers = make(map[string][]string)
	dP.Writers = make(map[string][]string)
	dP.Name_to_processtype = make(map[string]int)

	if dP.Cluster == DEFAULT_CLUSTER {

		for i := 1; i <= dP.Num_servers; i++ {
			dP.Servers[fmt.Sprintf("server-%d", i)] = []string{"127.0.0.1", fmt.Sprintf("%d", 9000+i), fmt.Sprintf("%d", 9500+i)}
			dP.Name_to_processtype[fmt.Sprintf("server-%d", i)] = SERVER
		}

		for i := 1; i <= dP.Num_readers; i++ {
			dP.Readers[fmt.Sprintf("reader-%d", i)] = []string{"127.0.0.1", fmt.Sprintf("%d", 7000+i), fmt.Sprintf("%d", 7500+i)}
			dP.Name_to_processtype[fmt.Sprintf("reader-%d", i)] = READER
		}

		for i := 1; i <= dP.Num_writers; i++ {
			dP.Writers[fmt.Sprintf("writer-%d", i)] = []string{"127.0.0.1", fmt.Sprintf("%d", 8000+i), fmt.Sprintf("%d", 8500+i)}
			dP.Name_to_processtype[fmt.Sprintf("writer-%d", i)] = WRITER
		}

		dP.Controller = []string{"127.0.0.1", "9999"}
	} else {
		AppLog.Panicln("Only Local Cluster Deployment is Currently Supported. In this Deployment, all processes run in the same machine")
	}

}

func (appl *ApplicationParams) LoadDefaultApplicationParams() {
	appl.NumObjects = DEFAULT_NUM_OBJECTS
	appl.Wait_time_btw_op = DEFAULT_WAITTIME_BTW_OPS_MS        // in milliseconds
	appl.Wait_time_btw_reads = DEFAULT_WAITTIME_BTW_READS_MS   // in milliseconds
	appl.Wait_time_btw_writes = DEFAULT_WAITTIME_BTW_WRITES_MS // in milliseconds
	appl.FolderToStore = DEFAULT_FOLDER_TO_STORE
	appl.UseDisk = DEFAULT_WRITETODISK
	appl.DefaultAlgorithm = DEFAULT_ALGORITHM
	appl.Num_servers_per_object = DEFAULT_NUM_SERVERS_PER_OBJECT
	appl.Default_erasurecode_rate = DEFAULT_ERASURECODE_RATE
	appl.Default_file_size = DEFAULT_FILE_SIZE
	appl.ObjectPickDistribution = DEFAULT_OBJECT_PICK_DIST
	appl.LogInterval = DEFAULT_LOG_INTERVAL_STATS

}

func (appl *ApplicationParams) InitializeApplicationParams() {

	appl.InitalizeMasterObjCatalogue()

}

// fix this
func (appl *ApplicationParams) InitalizeMasterObjCatalogue() {

	appl.MasterObjCatalogue = make(map[string]*ObjParams)
	num_servers_per_object := appl.Num_servers_per_object

	var quorum_size_majority float64
	quorum_size_majority = math.Ceil(float64(num_servers_per_object+1) / 2.0)

	for i := 0; i < appl.NumObjects; i++ {
		name := fmt.Sprintf("Object-%d", i+1)
		appl.MasterObjCatalogue[name] = &ObjParams{
			Numservers:    appl.Num_servers_per_object,
			Objname:       name,
			Algorithm:     appl.DefaultAlgorithm,
			File_size:     appl.Default_file_size,
			Servers_names: get_servers_for_object(name)}

		if num_servers_per_object > deployParams.Num_servers {
			AppLog.Panicln("The number of servers per object must be smaller than the total number of servers!")
		}

		// Set Erasure Code Params for the object
		if isACodingAlgorithm(appl.MasterObjCatalogue[name].Algorithm) {

			//Servers_names: get_default_servers_for_object(num_servers_per_object)}
			// Check if K is at least a majority
			num_servers_check := (appl.MasterObjCatalogue[name]).getNumServers()

			//			erasurecode_K := math.Floor(float64(appl.Default_erasurecode_rate) * float64(num_servers_check))
			erasurecode_K := math.Floor(float64(appl.Default_erasurecode_rate) * float64(num_servers_per_object))
			if erasurecode_K < quorum_size_majority {

				AppLog.Panicln("Erasure Code K must be at least a majority. Adjust the code rate to allow this!")
			}

			appl.MasterObjCatalogue[name].CodeParams = ErasureCodeParams{CodeN: num_servers_check,
				CodeK: int(erasurecode_K), Rate: appl.Default_erasurecode_rate,
				Coding_algorithm: DEFAULT_ERASURECODE_METHOD,
				FileSize:         appl.MasterObjCatalogue[name].File_size}
		}
	}
}

func DoesFileExist(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// OpenConfig is called whenever we need to change the state of the SystemConfig datastructure.
//The structure is serialized onto disk in a yml file so that subsequent usages of the API can record their changes to state.
func ReadConfigFromYML(config interface{}, filename string) interface{} {

	if !DoesFileExist(filename) {
		AppLog.Panicln(filename + " does not exist")
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		log.Fatalf("%v", err)
	}

	return config
}

//The structure is serialized onto disk in a yml file so that subsequent usages of the API can record their changes to state.
func WriteConfigToYML(config interface{}, filename string) {

	if config == nil {
		log.Fatal("config nil pointer")
	}

	// write the structure back to the yml file on disk
	data, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("%v", err)
	}

	//fmt.Println(data)

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
