package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const READ = 0
const WRITE = 1

type Tag struct {
	name string
	id   int64
}

type Operations struct {
	client_name     string
	objectname      string
	opnum           int64
	startime_ms     int64
	endtime_ms      int64
	operation_type  int
	tag_writer_name string
	tag_int         int64
	dataSafe        string
}

var operations map[string][]Operations // key is object name
var _operations []Operations
var indices map[string][]int // key is object name

var temp_operation Operations

type SortIndices []int

var maxOpnums map[string]map[string]int64 // first key is object name, second key is process name

func (s SortIndices) Len() int {
	return len(s)
}
func (s SortIndices) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortIndices) Less(i, j int) bool {
	return _operations[s[i]].endtime_ms < _operations[s[j]].endtime_ms
}

type SortIndicesTag []int

var writeTagSortedIndices []int

var readIndices []int

func (s SortIndicesTag) Len() int {
	return len(s)
}
func (s SortIndicesTag) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortIndicesTag) Less(i, j int) bool {
	tagi := Tag{_operations[s[i]].tag_writer_name, _operations[s[i]].tag_int}
	tagj := Tag{_operations[s[j]].tag_writer_name, _operations[s[j]].tag_int}

	if tagi.id == tagj.id && tagi.name == tagj.name {
		return false
	}

	return isGreaterTag(tagj, tagi)
}

func main() {

	logpath := flag.String("safety_log_path", "../logs/safetylog.txt", "the path of logs to be used for safety checks")
	numWrites := flag.Int64("numwrites", 1000, "the number of writes per writer")
	numReads := flag.Int64("numreads", 1000, "the number of reads per writer")
	numWriters := flag.Int("numwriters", 3, "the number of writers")
	numReaders := flag.Int("numreaders", 3, "the number of readers")
	flag.Parse()

	// Extract Data from the log file
	ReadLogFile(*logpath)

	//Check P1
	fmt.Println("Starting Safety Check P1")
	P1result := safetyCheckP1()
	fmt.Println("P1 Safety Check Pass is ", P1result)

	//Check P2

	fmt.Println("")
	fmt.Println("Starting Safety Check P2")
	P2result := safetyCheckP2()
	fmt.Println("P2 Safety Check Pass is ", P2result)

	//Check P3

	fmt.Println("")
	fmt.Println("Starting Safety Check P3")
	P3result := safetyCheckP3()
	fmt.Println("P3 Safety Check Pass is ", P3result)

	//Check Liveness
	fmt.Println("")
	fmt.Println("Starting Liveness Check")
	livenessResult := livenessCheck(*numReads, *numWrites, *numReaders, *numWriters)
	fmt.Println("Liveness Check is ", livenessResult)

}

// check if first tag is greater than the second tag
func isGreaterTag(tag_in, tag_local Tag) bool {

	if tag_in.id > tag_local.id {
		return true
	} else if tag_in.id < tag_local.id {
		return false
	}

	if tag_in.name > tag_local.name { // Dictionary comparison of strings
		return true
	}

	return false

}

//
func safetyCheckP1() bool {
	for objname := range operations {
		if _safetyCheckP1(objname) == false {
			return false
		}
	}
	return true
}

//per object safety check
func _safetyCheckP1(objectname string) bool {

	//point to the right objects
	_operations = operations[objectname]

	// Sort (increasing order) the completion time array, get the sorted indices also

	indicesP1 := make([]int, len(indices[objectname]))
	copy(indicesP1, indices[objectname])
	sort.Sort(SortIndices(indicesP1)) // type cast indicesP1 to type SottIndices and then pass it to sort.Sort

	for i := 0; i < len(indicesP1)-1; i++ {
		if _operations[indicesP1[i]].endtime_ms > _operations[indicesP1[i+1]].endtime_ms {
			panic("Sort Failed")
		}
	}
	// Compute the running max tag array of the above permuted tag array
	runningMaxTagArray := make([]Tag, len(indicesP1))
	maxTag := Tag{"", -1}
	var tagIn Tag
	for i := 0; i < len(indicesP1); i++ {

		//fmt.Println(i, indicesP1[i], _operations[indicesP1[i]].tag_writer_name, _operations[indicesP1[i]].tag_int)
		tagIn = Tag{_operations[indicesP1[i]].tag_writer_name, _operations[indicesP1[i]].tag_int}

		if isGreaterTag(tagIn, maxTag) {
			maxTag = tagIn
		}
		runningMaxTagArray[i] = maxTag

	}

	// Loop over all operations (indenfied by indices). For each operation pi identify the position (say index) in the sorted compeltion time array
	// based on the start time of the operation. Check if tag(pi) >= max tag array [index]

	for i := 0; i < len(indicesP1); i++ {
		tagIn = Tag{_operations[i].tag_writer_name, _operations[i].tag_int}
		StartTime := _operations[i].startime_ms

		indexOut := BinarySearchP1(indicesP1, StartTime)

		if indexOut > -1 && isGreaterTag(runningMaxTagArray[indexOut], tagIn) {
			fmt.Println("Operation corresponding to Original index ", i+1, " is not safe")
			fmt.Println("Start time of this operation is ", StartTime)
			fmt.Println("Tag of this operation is ", tagIn)
			fmt.Println("Max Tag of any operation that finished before the operation started is ", runningMaxTagArray[indexOut])
			fmt.Println("End Time of the operation corresponding to max tag violation is ", _operations[indicesP1[indexOut]].endtime_ms)
			return false
		}
	}

	return true
}

//
func safetyCheckP2() bool {
	for objname, _ := range operations {

		if _safetyCheckP2(objname) == false {
			return false
		}
	}
	return true
}

// Check if the tags of all write operations are distinct
func _safetyCheckP2(objectname string) bool {
	// point to the right objectname
	_operations = operations[objectname]

	indicesP2 := make([]int, len(indices[objectname]))
	copy(indicesP2, indices[objectname])
	sort.Sort(SortIndicesTag(indicesP2)) // type cast indicesP2 to type SottIndices and then pass it to sort.Sort

	writeTagSortedIndices = make([]int, 0) // to be used in P3 check
	readIndices = make([]int, 0)

	previousTag := Tag{"", -1}

	for i := 0; i < len(indicesP2); i++ {

		if _operations[indicesP2[i]].operation_type == READ {
			readIndices = append(readIndices, indicesP2[i]) // to be used in P3 check
			continue
		}

		tagi := Tag{_operations[indicesP2[i]].tag_writer_name, _operations[indicesP2[i]].tag_int}

		writeTagSortedIndices = append(writeTagSortedIndices, indicesP2[i]) // to be used in P3 check

		if reflect.DeepEqual(tagi, previousTag) {
			return false
		}

		previousTag = tagi
	}

	for i := 0; i < len(writeTagSortedIndices)-1; i++ {
		tag_i := Tag{_operations[writeTagSortedIndices[i]].tag_writer_name, _operations[writeTagSortedIndices[i]].tag_int}
		tag_ip1 := Tag{_operations[writeTagSortedIndices[i+1]].tag_writer_name, _operations[writeTagSortedIndices[i+1]].tag_int}
		if isGreaterTag(tag_i, tag_ip1) {
			panic("Sort Failed")
		}
	}

	return true
}

func safetyCheckP3() bool {
	for objname, _ := range operations {
		if _safetyCheckP3(objname) == false {
			return false
		}
	}
	return true
}

// check if every read tag is one of the write tags or the default tag. Also check if readsafe is always true.
// This means that value read is same as the value written for the corresponding write
func _safetyCheckP3(objectname string) bool {

	// point to the right objectname
	_operations = operations[objectname]

	indicesP2 := make([]int, len(indices[objectname]))
	copy(indicesP2, indices[objectname])
	sort.Sort(SortIndicesTag(indicesP2)) // type cast indicesP2 to type SottIndices and then pass it to sort.Sort

	writeTagSortedIndices = make([]int, 0) // to be used in P3 check
	readIndices = make([]int, 0)

	previousTag := Tag{"", -1}

	for i := 0; i < len(indicesP2); i++ {

		if _operations[indicesP2[i]].operation_type == READ {
			readIndices = append(readIndices, indicesP2[i]) // to be used in P3 check
			continue
		}

		tagi := Tag{_operations[indicesP2[i]].tag_writer_name, _operations[indicesP2[i]].tag_int}

		writeTagSortedIndices = append(writeTagSortedIndices, indicesP2[i]) // to be used in P3 check

		if reflect.DeepEqual(tagi, previousTag) {
			return false
		}

		previousTag = tagi
	}

	// now the actual P3 check begins

	defaultTag := Tag{"writer-1", 0}

	for i := 0; i < len(writeTagSortedIndices)-2; i++ {
		tag_i := Tag{_operations[writeTagSortedIndices[i]].tag_writer_name, _operations[writeTagSortedIndices[i]].tag_int}
		tag_ip1 := Tag{_operations[writeTagSortedIndices[i+1]].tag_writer_name, _operations[writeTagSortedIndices[i+1]].tag_int}
		if isGreaterTag(tag_i, tag_ip1) {
			panic("Sort Failed")
		}
	}

	for i := 0; i < len(readIndices); i++ {

		tagRead := Tag{_operations[readIndices[i]].tag_writer_name, _operations[readIndices[i]].tag_int}
		if !reflect.DeepEqual(tagRead, defaultTag) && !BinarySearchP3(writeTagSortedIndices, tagRead) {

			fmt.Println("Found a read whose tag was never in the system :", tagRead)
			return false
		}

		if _operations[readIndices[i]].dataSafe == "false" {

			fmt.Println("Found a read whose decoded data was not same as the encoded data")
			return false
		}

	}
	return true
}

// Returns true if the input tag is in the array
func BinarySearchP3(inputArray []int, tagin Tag) bool {

	startIndex := 0
	endIndex := len(inputArray) - 1
	var median int

	for startIndex <= endIndex {

		median = (startIndex + endIndex) / 2
		medianTag := Tag{_operations[inputArray[median]].tag_writer_name, _operations[inputArray[median]].tag_int}

		if isGreaterTag(tagin, medianTag) {
			startIndex = median + 1
		} else if isGreaterTag(medianTag, tagin) {
			endIndex = median - 1
		} else {
			return true
		}

	}

	return false

}

// Returns the index of the last element in the inputArray (the first input) whose corresponding operation's completion time is less than or equal to value (the second input)
func BinarySearchP1(inputArray []int, value int64) int {

	if value < _operations[inputArray[0]].endtime_ms {
		return -1
	} // This takes care of the fact that the search value is smaller than all the inputs

	startIndex := 0
	endIndex := len(inputArray) - 1
	var median int

	for startIndex <= endIndex {

		if median == (startIndex+endIndex)/2 {
			return median
		}

		median = (startIndex + endIndex) / 2

		if _operations[inputArray[median]].endtime_ms == value {
			startIndex = median
		} else if _operations[inputArray[median]].endtime_ms < value {
			startIndex = median + 1
		} else {
			endIndex = median - 1
		}

	}
	if startIndex == 0 {
		return 0 // This line is executed only if the search value is the first input
	}
	return startIndex - 1

}

func livenessCheck(numReads int64, numWrites int64, numReaders int, numWriters int) bool {

	var _numWrites int64
	var _numReads int64

	for i := 1; i <= numWriters; i++ {

		_numWrites = 0

		for objectname := range operations {

			_numWrites = _numWrites + maxOpnums[objectname]["writer-"+strconv.Itoa(i)]
		}

		if _numWrites != numWrites {
			fmt.Println("Writer-", i, " is not live")
			return false
		}

	}

	for i := 1; i <= numReaders; i++ {

		_numReads = 0

		for objectname := range operations {

			_numReads = _numReads + maxOpnums[objectname]["reader-"+strconv.Itoa(i)]
		}

		if _numReads != numReads {
			fmt.Println("Reader-", i, " is not live")
			fmt.Println("EXpected Reads:", numReads, " Observerd Reads", _numReads)
			return false
		}

	}

	var _numWritesTOT, _numReadsTOT int64 = 0, 0

	for objectname := range operations {

		for _, operation := range operations[objectname] {

			if operation.operation_type == WRITE {
				_numWritesTOT++
			}

			if operation.operation_type == READ {
				_numReadsTOT++
			}

		}
	}

	if numWrites*int64(numWriters) != _numWritesTOT || numReads*int64(numReaders) != _numReadsTOT {
		fmt.Println("Expected writes : ", numWrites*int64(numWriters), " Actual writes : ", _numWritesTOT)
		fmt.Println("Expected reads : ", numReads*int64(numReaders), " Actual reads : ", _numReadsTOT)
		return false
	}

	return true
}

func ReadLogFile(logpath string) {

	var file, err = os.OpenFile(logpath, os.O_RDONLY, 0666)

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)

	lineCounts := make(map[string]int)
	operations = make(map[string][]Operations)
	indices = make(map[string][]int)

	for scanner.Scan() {
		line := strings.Split(scanner.Text(), " ")
		objectname := line[2]

		if _, OK := lineCounts[objectname]; !OK {
			lineCounts[objectname] = 0
		}
		lineCounts[objectname]++
	}
	file.Close()

	maxOpnums = make(map[string]map[string]int64)
	for objname, _ := range lineCounts {
		//fmt.Println(objname, lineCounts[objname])
		operations[objname] = make([]Operations, lineCounts[objname])
		indices[objname] = make([]int, lineCounts[objname])
		maxOpnums[objname] = make(map[string]int64)
	}

	//set the linecounts to 0
	for objectname, _ := range lineCounts {
		lineCounts[objectname] = 0
	}

	file, err = os.OpenFile(logpath, os.O_RDONLY, 0666)
	defer file.Close()
	scanner1 := bufio.NewScanner(file)

	var read_or_write string
	for scanner1.Scan() {
		line := strings.Split(scanner1.Text(), " ")

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading log file:", err)
		}

		i := 0
		read_or_write = line[i]
		i += 1
		temp_operation.client_name = line[i]
		i += 1
		temp_operation.objectname = line[i]
		i += 1
		temp_operation.opnum, _ = strconv.ParseInt(line[i], 10, 64)
		i += 1
		temp_operation.startime_ms, _ = strconv.ParseInt(line[i], 10, 64)
		i += 1
		temp_operation.endtime_ms, _ = strconv.ParseInt(line[i], 10, 64)
		i += 1
		temp_operation.tag_writer_name = line[i]
		i += 1
		temp_operation.tag_int, _ = strconv.ParseInt(line[i], 10, 64)
		i += 1

		if read_or_write == "READ" {
			temp_operation.operation_type = READ
			temp_operation.dataSafe = line[i]
		} else {
			temp_operation.operation_type = WRITE
		}

		operations[temp_operation.objectname][lineCounts[temp_operation.objectname]] = temp_operation
		indices[temp_operation.objectname][lineCounts[temp_operation.objectname]] = lineCounts[temp_operation.objectname]

		if _, OK := maxOpnums[temp_operation.client_name]; !OK {
			maxOpnums[temp_operation.objectname][temp_operation.client_name] = temp_operation.opnum
		}

		if temp_operation.opnum > maxOpnums[temp_operation.objectname][temp_operation.client_name] {
			maxOpnums[temp_operation.objectname][temp_operation.client_name] = temp_operation.opnum
		}
		lineCounts[temp_operation.objectname]++
	} // end of the forloop
}
