package main

import (
	"bufio"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	daemons "../daemons"
)

// Input Types Used to populate Logs

type OperationParams struct {
	srcOpParams daemons.OperationParams
	squareTime  float64
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

type ObjLogsInput map[int]OperationParams    // int is the opnum of the object. map of all operations on the object by a single client
type ClientLogsInput map[string]ObjLogsInput // string is object name. map of all objects that the reader operated on

var readerLogsAllInput, writerLogsAllInput map[string]ClientLogsInput // string is the client name

//Output Types Used to Hold Log Analysis
type AverageClientStat [][]int64 // Two-D Array, clients X Objects, any parameter measurable as int64
var TotalSquareReadTime, TotalSquareWriteTime, TotalReadTime, TotalWriteTime, TotalReadPhases, TotalWritePhases, TotalEncodingTime, TotalDecodingTime, numReads, numWrites AverageClientStat
var TotalReadTime_tmp, TotalWriteTime_tmp, TotalReadNumPhases_tmp AverageClientStat
var numReads_tmp, numWrites_tmp AverageClientStat

var AvWriterInBandMB, AvWriterOutBandMB, AvReaderInBandMB, AvReaderOutBandMB, AvServerInBandMB, AvServerOutBandMB int64

var allReadTimes, allWriteTimes [][]float32

var ObjFailureMap map[string]int

// Variables for handling server failures

const LOGSROOT = ""
const OUTPUTFILE = ""
const RESULTSFOLDER = ""

func hash(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

func get_servers_for_object(objectname string, num_servers int, num_servers_per_object int) []string {
	seed := int(hash([]byte(objectname)))
	rand.Seed(int64(seed))

	all_servers := make([]string, num_servers)
	for ii := 0; ii < num_servers; ii++ {
		all_servers[ii] = "server-" + strconv.Itoa(ii+1)
	}

	sort.Sort(Lexicon(all_servers))

	server_list := make([]string, num_servers_per_object)
	index := rand.Intn(10000)
	for i := 0; i < num_servers_per_object; i++ {
		server_list[i] = all_servers[(index+i)%num_servers]
	}
	//fmt.Println("server list ", seed, objectname, server_list, all_servers)
	return server_list
}

func main() {

	deployyml := flag.String("deployyml", "../deploy_config.yml", "the path of the Deployment Config YML file")
	appyml := flag.String("appyml", "../appl_config.yml", "the path of the Application Config YML file")
	numServerFailures := flag.Int("numServerFailures", 0, "number of server failures: max 2")
	logroot := flag.String("logroot", LOGSROOT, "the path of logs to be analysed")
	outputFile := flag.String("outputFile", OUTPUTFILE, "the name of the output file. The file will be created under the folder"+RESULTSFOLDER)
	outputfolder := flag.String("outputfolder", RESULTSFOLDER, "the path of the output FOLDER")

	flag.Parse()

	DP := daemons.GetDeploymentParams()
	AP := daemons.GetApplicationParams()

	var readConfig interface{}

	readConfig = daemons.ReadConfigFromYML(DP, *deployyml)
	DP = readConfig.(*daemons.DeploymentParameters)

	readConfig = daemons.ReadConfigFromYML(AP, *appyml)
	AP = readConfig.(*daemons.ApplicationParams)

	numReaders := DP.Num_readers
	numWriters := DP.Num_writers
	numServers := DP.Num_servers
	numObjects := AP.NumObjects
	numServersperObj := AP.Num_servers_per_object

	allReadTimes = make([][]float32, *numServerFailures+1)
	allWriteTimes = make([][]float32, *numServerFailures+1)

	nFailureObj := make([][]int, *numServerFailures+1)
	ObjFailureMap = make(map[string]int)
	for i := 0; i <= *numServerFailures; i++ {
		nFailureObj[i] = make([]int, 0)
		allReadTimes[i] = make([]float32, 0)
		allWriteTimes[i] = make([]float32, 0)
	}

	if *numServerFailures == 0 { // There is no failure in this case
		for i := 0; i < numObjects; i++ {
			nFailureObj[0] = append(nFailureObj[0], i)
			name := fmt.Sprintf("Object-%d", i+1)
			ObjFailureMap[name] = 0
		}

	} else {
		APall := daemons.GetApplicationParams()

		for i := 0; i < APall.NumObjects; i++ {
			name := fmt.Sprintf("Object-%d", i+1)
			APall.MasterObjCatalogue[name] = &daemons.ObjParams{
				Numservers:    APall.Num_servers_per_object,
				Objname:       name,
				Algorithm:     APall.DefaultAlgorithm,
				File_size:     APall.Default_file_size,
				Servers_names: get_servers_for_object(name, numServers, numServersperObj)}
			//	fmt.Println("Servers for object", (i + 1), ":", APall.MasterObjCatalogue[name].Servers_names)
		}

		//fmt.Println(APall.MasterObjCatalogue["Object-1"].Servers_names, numServers, numServersperObj)

		//  From the master catelogue isolate the objects which are not stored in the last two servers (no failures)
		// or have a presence in one of the last two servers (one failure) or have presence in both of the last two servers (two failures)

		// Map object name to object ID
		objID := make(map[string]int)
		for ii := 0; ii < numObjects; ii++ {
			name := fmt.Sprintf("Object-%d", ii+1)
			objID[name] = ii
		}

		for objName, catalogue := range APall.MasterObjCatalogue { // fix this, get the objID based on object name
			numFailures := 0

			for _, server := range catalogue.Servers_names {
				for i := 0; i < *numServerFailures; i++ {
					if server == "server-"+strconv.Itoa(i+1) {
						numFailures++
					}

				}
			}
			nFailureObj[numFailures] = append(nFailureObj[numFailures], objID[objName])

			ObjFailureMap[objName] = numFailures
			//	fmt.Println(catalogue.Servers_names, numFailures)
		}
	}

	fmt.Println("Server Failure Information: The ith element of the array is the number of objects experiencing i server failures", nFailureObj)

	// make the slices to store the inputs
	readerLogsAllInput = make(map[string]ClientLogsInput, numReaders)
	writerLogsAllInput = make(map[string]ClientLogsInput, numWriters)

	// Make the slice to store the outputs
	TotalReadTime.MakeSlice(numReaders, numObjects)
	TotalSquareReadTime.MakeSlice(numReaders, numObjects)
	TotalWriteTime.MakeSlice(numWriters, numObjects)
	TotalSquareWriteTime.MakeSlice(numWriters, numObjects)
	TotalReadPhases.MakeSlice(numReaders, numObjects)
	TotalWritePhases.MakeSlice(numWriters, numObjects)
	numReads.MakeSlice(numReaders, numObjects)
	numWrites.MakeSlice(numWriters, numObjects)
	TotalEncodingTime.MakeSlice(numWriters, numObjects)
	TotalDecodingTime.MakeSlice(numReaders, numObjects)

	// Get Bandwidth Stats

	if *numServerFailures == 0 {
		TotalWriterInBandwidth, TotalWriterOutBandwidth := getBandwidthStats(*logroot, "writer-", numWriters)
		TotalReaderInBandwidth, TotalReaderOutBandwidth := getBandwidthStats(*logroot, "reader-", numReaders)
		TotalServerInBandwidth, TotalServerOutBandwidth := getBandwidthStats(*logroot, "server-", numServers)

		AvWriterInBandMB = TotalWriterInBandwidth / int64(numWriters*1024)
		AvWriterOutBandMB = TotalWriterOutBandwidth / int64(numWriters*1024)
		AvReaderInBandMB = TotalReaderInBandwidth / int64(numReaders*1024)
		AvReaderOutBandMB = TotalReaderOutBandwidth / int64(numReaders*1024)
		AvServerInBandMB = TotalServerInBandwidth / int64(numServers*1024)
		AvServerOutBandMB = TotalServerOutBandwidth / int64(numServers*1024)
	}

	// Loop over readers and populate the fields
	populateClientLogs(*logroot, "reader-", numReaders, numObjects, readerLogsAllInput)

	// Loop over writers and populate the fields
	populateClientLogs(*logroot, "writer-", numWriters, numObjects, writerLogsAllInput)

	//getReaderStats
	getReadStats("reader-", numReaders, numObjects, readerLogsAllInput, isACodingAlgorithm(AP.DefaultAlgorithm))

	getWriteStats("writer-", numWriters, numObjects, writerLogsAllInput, isACodingAlgorithm(AP.DefaultAlgorithm))

	// Compute OverAll Average of Statistics. Use this function with caution
	// The below are averages over all objects, irrespective of the number of failures experienced
	averageReadTime, totalNumReads := computeAverageClientStat(TotalReadTime, numReads, numReaders, numObjects)
	averageSqaureReadTime, totalNumReads := computeAverageClientStat(TotalSquareReadTime, numReads, numReaders, numObjects)
	averageReadnumPhases, _ := computeAverageClientStat(TotalReadPhases, numReads, numReaders, numObjects)
	averageDecodingTime, _ := computeAverageClientStat(TotalDecodingTime, numReads, numReaders, numObjects)
	averageWriteTime, totalNumWrites := computeAverageClientStat(TotalWriteTime, numWrites, numWriters, numObjects)
	averageSquareWriteTime, totalNumWrites := computeAverageClientStat(TotalSquareWriteTime, numWrites, numWriters, numObjects)
	averageWritenumPhases, _ := computeAverageClientStat(TotalWritePhases, numWrites, numWriters, numObjects)
	averageEncodingTime, _ := computeAverageClientStat(TotalEncodingTime, numWrites, numWriters, numObjects)

	// Calculate Standard Deviation for read and write times
	varWriteTime := averageSquareWriteTime - averageWriteTime*averageWriteTime
	stdWriteTime := math.Sqrt(float64(varWriteTime))

	varReadTime := averageSqaureReadTime - averageReadTime*averageReadTime
	stdReadTime := math.Sqrt(float64(varReadTime))

	// Print the results to the output file
	var resultspath string = ""
	resultspath = *outputfolder + "/" + *outputFile
	writeTimesout := *outputfolder + "/" + *outputFile + "_writeTimes"
	readTimesout := *outputfolder + "/" + *outputFile + "_readTimes"

	var ff, ff_read, ff_write *os.File
	if resultspath == "" {
		ff, err1 := os.OpenFile(resultspath, os.O_CREATE, 0666)
		defer ff.Close()

		if err1 != nil {
			panic(err1)
		}
	} else {
		ff = os.Stdout
	}

	ff_write, err1 := os.OpenFile(writeTimesout, os.O_CREATE|os.O_WRONLY, 0666)
	defer ff_write.Close()

	if err1 != nil {
		panic(err1)
	}

	ff_read, err2 := os.OpenFile(readTimesout, os.O_CREATE|os.O_WRONLY, 0666)
	defer ff_read.Close()

	if err2 != nil {
		panic(err1)
	}

	// Print All times

	for numFail, readTimesperFailure := range allReadTimes {
		for _, readTime := range readTimesperFailure {
			fmt.Fprintln(ff_read, numFail, readTime)
		}
	}

	for numFail, writeTimesperFailure := range allWriteTimes {
		for _, writeTime := range writeTimesperFailure {
			fmt.Fprintln(ff_write, numFail, writeTime)
		}
	}

	fmt.Fprintln(ff, "Total Number of Writes :", totalNumWrites)
	fmt.Fprintln(ff, "Average Write Time Per Operation in MS:", averageWriteTime)
	fmt.Fprintln(ff, "StdDev Write Time Per Operation in MS:", stdWriteTime)
	fmt.Fprintln(ff, "Average Write Number of Phases per Operation :", averageWritenumPhases)
	if isACodingAlgorithm(AP.DefaultAlgorithm) {
		fmt.Fprintln(ff, "Average Encoding Time Per Operation :", averageEncodingTime)
	}
	fmt.Fprintln(ff, "===================================")
	fmt.Fprintln(ff, "Total Number of Reads :", totalNumReads)
	fmt.Fprintln(ff, "Average Read Time Per Operation in MS:", averageReadTime)
	fmt.Fprintln(ff, "StdDev Read Time Per Operation in MS:", stdReadTime)
	fmt.Fprintln(ff, "Average Read Number of Phases per Operation :", averageReadnumPhases)
	if isACodingAlgorithm(AP.DefaultAlgorithm) {
		fmt.Fprintln(ff, "Average Decoding Time Per Operation :", averageDecodingTime)
	}
	fmt.Fprintln(ff, "===================================")

	// separate the results based on number of server errors

	for i := 0; i <= *numServerFailures; i++ {
		x := len(nFailureObj[i])
		if x > 0 {

			TotalReadTime_tmp.MakeSlice(numReaders, x)
			TotalReadNumPhases_tmp.MakeSlice(numReaders, x)
			TotalWriteTime_tmp.MakeSlice(numWriters, x)
			numReads_tmp.MakeSlice(numReaders, x)
			numWrites_tmp.MakeSlice(numWriters, x)

			if len(TotalReadTime_tmp[0]) != x {
				fmt.Println(i, x, len(TotalReadTime_tmp[0]))
				panic("unexpected length for array")
			}

			for ii := 0; ii < numReaders; ii++ {
				for jj := 0; jj < x; jj++ {
					TotalReadTime_tmp[ii][jj] = TotalReadTime[ii][nFailureObj[i][jj]]
					TotalReadNumPhases_tmp[ii][jj] = TotalReadPhases[ii][nFailureObj[i][jj]]
					numReads_tmp[ii][jj] = numReads[ii][nFailureObj[i][jj]]
				}
			}

			for ii := 0; ii < numWriters; ii++ {
				for jj := 0; jj < x; jj++ {
					TotalWriteTime_tmp[ii][jj] = TotalWriteTime[ii][nFailureObj[i][jj]]
					numWrites_tmp[ii][jj] = numWrites[ii][nFailureObj[i][jj]]
				}
			}

			averageReadTime_tmp, totalNumReads_tmp := computeAverageClientStat(TotalReadTime_tmp, numReads_tmp, numReaders, x)
			averageWriteTime_tmp, totalNumWrites_tmp := computeAverageClientStat(TotalWriteTime_tmp, numWrites_tmp, numWriters, x)
			averageReadnumPhases_tmp, _ := computeAverageClientStat(TotalReadNumPhases_tmp, numReads_tmp, numReaders, x)

			fmt.Fprintln(ff, "Number of ", i, " failure objects: ", x)
			fmt.Fprintln(ff, "Total Number of Writes  :", totalNumWrites_tmp)
			fmt.Fprintln(ff, "Average Write Time Per Operation   in MS:", averageWriteTime_tmp)
			fmt.Fprintln(ff, "Total Number of Reads :", totalNumReads_tmp)
			fmt.Fprintln(ff, "Average Read Time Per Operation in MS:", averageReadTime_tmp)
			fmt.Fprintln(ff, "Average Read Number of Phases per Operation :", averageReadnumPhases_tmp)
			fmt.Fprintln(ff, "===================================")
		}
	}

	if *numServerFailures == 0 {
		fmt.Fprintln(ff, "Average Incoming Bandwidth (MB) per Writer:", AvWriterInBandMB)
		fmt.Fprintln(ff, "Average Outgoing  Bandwidth (MB) per Writer:", AvWriterOutBandMB)
		fmt.Fprintln(ff, "Average Incoming Bandwidth (MB) per Reader:", AvReaderInBandMB)
		fmt.Fprintln(ff, "Average Outgoing  Bandwidth (MB) per Reader:", AvReaderOutBandMB)
		fmt.Fprintln(ff, "Average Incoming Bandwidth (MB) per Server:", AvServerInBandMB)
		fmt.Fprintln(ff, "Average Outgoing  Bandwidth (MB) per Server:", AvServerOutBandMB)

	}

}

func computeAverageClientStat(TotalStat AverageClientStat, numOperations AverageClientStat, xnum int, ynum int) (float32, int64) {

	var totalStat, totalCount float32 = 0, 0
	for i := 0; i < xnum; i++ {
		for j := 0; j < ynum; j++ {
			totalStat = totalStat + float32(TotalStat[i][j])
			totalCount = totalCount + float32(numOperations[i][j])
		}
	}

	if totalCount == 0 {
		return 0, 0
	}

	avgStat := totalStat / totalCount
	//    fmt.Println("stddev %f\n", avgStat, avgSqStat, varStat, stdDev)

	return avgStat, int64(totalCount)
}

func getWriteStats(clientType string, numClients int, numObjects int, clientLogsInput map[string]ClientLogsInput, isCodingAlgo bool) {

	for i := 0; i < numClients; i++ {

		clientName := clientType + strconv.Itoa(i+1)

		for j := 0; j < numObjects; j++ {
			ObjName := "Object-" + strconv.Itoa(j+1)

			if _, OK := clientLogsInput[clientName][ObjName]; !OK {
				numWrites[i][j] = 0
				TotalWriteTime[i][j] = 0
				TotalWritePhases[i][j] = 0
				TotalEncodingTime[i][j] = 0
				TotalSquareWriteTime[i][j] = 0
				continue
			}

			var totalTime, totalCodingTime, totalNumPhases, totalsqTime int64 = 0, 0, 0, 0

			for _, tempOp := range clientLogsInput[clientName][ObjName] {
				totalTime = totalTime + int64(tempOp.srcOpParams.TotalTime)/1e6 //milliseconds
				totalsqTime = totalsqTime + int64(tempOp.squareTime/1e12)       //milliseconds
				totalNumPhases = totalNumPhases + int64(tempOp.srcOpParams.NumPhases)

				numFailForObj := ObjFailureMap[ObjName]
				allWriteTimes[numFailForObj] = append(allWriteTimes[numFailForObj], float32(tempOp.srcOpParams.TotalTime)/1e6)

				if isCodingAlgo {
					totalCodingTime = totalCodingTime + int64(tempOp.srcOpParams.EncodingTime)/1e6
				}

			}

			numWrites[i][j] = int64(len(clientLogsInput[clientName][ObjName]))
			TotalWriteTime[i][j] = totalTime
			TotalWritePhases[i][j] = totalNumPhases
			TotalEncodingTime[i][j] = totalCodingTime
			TotalSquareWriteTime[i][j] = totalsqTime

		}

	}
}

func getReadStats(clientType string, numClients int, numObjects int, clientLogsInput map[string]ClientLogsInput, isCodingAlgo bool) {

	for i := 0; i < numClients; i++ {

		clientName := clientType + strconv.Itoa(i+1)

		for j := 0; j < numObjects; j++ {
			ObjName := "Object-" + strconv.Itoa(j+1)

			if _, OK := clientLogsInput[clientName][ObjName]; !OK {
				numReads[i][j] = 0
				TotalReadTime[i][j] = 0
				TotalReadPhases[i][j] = 0
				TotalDecodingTime[i][j] = 0
				TotalSquareReadTime[i][j] = 0
				continue
			}

			var totalTime, totalCodingTime, totalNumPhases, totalsqTime int64 = 0, 0, 0, 0

			for _, tempOp := range clientLogsInput[clientName][ObjName] {
				totalTime = totalTime + int64(tempOp.srcOpParams.TotalTime)/1e6 //milliseconds
				totalsqTime = totalsqTime + int64(tempOp.squareTime/1e12)       //milliseconds
				totalNumPhases = totalNumPhases + int64(tempOp.srcOpParams.NumPhases)

				numFailForObj := ObjFailureMap[ObjName]
				allReadTimes[numFailForObj] = append(allReadTimes[numFailForObj], float32(tempOp.srcOpParams.TotalTime)/1e6)

				if isCodingAlgo {
					totalCodingTime = totalCodingTime + int64(tempOp.srcOpParams.DecodingTime)/1e6
				}

			}

			numReads[i][j] = int64(len(clientLogsInput[clientName][ObjName]))
			TotalReadTime[i][j] = totalTime
			TotalReadPhases[i][j] = totalNumPhases
			TotalDecodingTime[i][j] = totalCodingTime
			TotalSquareReadTime[i][j] = totalsqTime

		}

	}
}

func getBandwidthStats(logroot string, processType string, numClients int) (TotalInBand int64, TotalOutBand int64) {

	TotalInBand = 0
	TotalOutBand = 0
	var bandin, bandout int

	for i := 0; i < numClients; i++ {

		processname := processType + strconv.Itoa(i+1)

		logpath := logroot + "/" + processname + "/logs/systemlog.txt"
		var file, err = os.OpenFile(logpath, os.O_RDONLY, 0666)

		if err != nil {
			fmt.Errorf("could not read file %s", logpath)
		}

		scanner := bufio.NewScanner(file)
		first := true

		for scanner.Scan() {
			if first {
				first = false
				continue
			}

			line := scanner.Text()
			fields := strings.Split(line, " ")
			bandin, _ = strconv.Atoi(fields[4])
			bandout, _ = strconv.Atoi(fields[5])
			TotalInBand = TotalInBand + int64(bandin)
			TotalOutBand = TotalOutBand + int64(bandout)
		}
		file.Close()

	}

	return

}

func populateClientLogs(logroot string, clientType string, numClients int, numObjects int, clientLogsInput map[string]ClientLogsInput) {

	for i := 0; i < numClients; i++ {

		clientName := clientType + strconv.Itoa(i+1)

		clientLogsInput[clientName] = make(map[string]ObjLogsInput, numObjects)

		logpath := logroot + "/" + clientName + "/logs/experimentlog.txt"
		var file, err = os.OpenFile(logpath, os.O_RDONLY, 0666)

		if err != nil {
			fmt.Errorf("could not read file %s", logpath)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()
			sDec, _ := b64.StdEncoding.DecodeString(line)

			opParams := daemons.OperationParams{}
			//opParams = daemons.OperationParams{}
			err := json.Unmarshal(sDec, &opParams)
			if err != nil {
				panic("Error Unmarshalling Json message")
			}
			if _, OK := clientLogsInput[clientName][opParams.ObjParamsvar.Objname]; !OK {
				clientLogsInput[clientName][opParams.ObjParamsvar.Objname] = make(map[int]OperationParams)
			}
			clientLogsInput[clientName][opParams.ObjParamsvar.Objname][opParams.Opnum] = OperationParams{srcOpParams: opParams,
				squareTime: float64(opParams.TotalTime) * float64(opParams.TotalTime)}

			//	fmt.Println(opParams)

		}
		file.Close()

	}

}

func (statVar *AverageClientStat) MakeSlice(xnum int, ynum int) {
	*statVar = make(AverageClientStat, xnum)

	for i := range *statVar {
		(*statVar)[i] = make([]int64, ynum)
	}

}

func isACodingAlgorithm(algoName string) bool {
	switch algoName {
	case "SODAW", "SODAW_FAST", "SODA_WITH_LIST":
		return true
	default:
		return false
	}
}
