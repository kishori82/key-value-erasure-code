package daemons

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"

	erasure "../../submodules/erasure"
)

type Tag struct {
	Client_id   string
	Version_num int
}

type TagValue struct {
	Tag_var   Tag
	Value     []byte
	Seed      int
	CodeIndex int
	Opnum     int  // used with PRAKIS
	ToCommit  bool // used with PRAKIS, while doing read commit
}

// Check if Tag2 is larger than or equal to Tag1
func (t *Tag) IsSmallerThan(x Tag) bool {

	if t.Version_num < x.Version_num {
		return true
	} else if t.Version_num > x.Version_num {
		return false
	} else {
		if t.Client_id < x.Client_id {
			return true
		} else {
			return false
		}
	}
}

//Compares tags. Returns true if first tag is greater than the second tag
func isGreaterTag(tag_in, tag_local Tag) bool {

	if tag_in.Version_num > tag_local.Version_num {
		return true
	} else if tag_in.Version_num < tag_local.Version_num {
		return false
	}

	if tag_in.Client_id > tag_local.Client_id { // Dictionary comparison of strings
		return true
	}

	return false

}

func isACodingAlgorithm(algoName string) bool {
	switch algoName {
	case SODAW, SODAW_FAST, PRAKIS:
		return true
	default:
		return false
	}
}

func generateCodedElements(value []byte, codeParams ErasureCodeParams) [][]byte {

	if codeParams.Coding_algorithm != CAUCHY {
		panic("Unsupported Erasure Coding Algorithm")
	}

	params, _ := erasure.ParseEncoderParams(uint8(codeParams.CodeK), uint8(codeParams.CodeN)-uint8(codeParams.CodeK), erasure.Cauchy)
	encoder := erasure.NewEncoder(params)
	codedElementsArray, err := encoder.Encode(value)

	if err != nil {
		panic("Error in Encoding Library")
	}

	encoder.FreeUpEncoderInstance()
	return codedElementsArray
}

func getDecodedValue(codedElementsArray map[string][]byte, codeParams ErasureCodeParams) []byte {

	params, _ := erasure.ParseEncoderParams(uint8(codeParams.CodeK), uint8(codeParams.CodeN)-uint8(codeParams.CodeK), erasure.Cauchy)
	encoder := erasure.NewEncoder(params)

	chunks := make([][]byte, codeParams.CodeN)
	num_valid_chunks := 0
	for i := 0; i < codeParams.CodeN; i++ {

		_, chunkExists := codedElementsArray[strconv.Itoa(i)]
		if chunkExists {
			chunks[i] = codedElementsArray[strconv.Itoa(i)]
			num_valid_chunks++
		} else {
			chunks[i] = nil
		}

	}

	if num_valid_chunks != codeParams.CodeK {
		fmt.Println("My K is ", codeParams.CodeK)
		fmt.Println("Recevied number of blocks:", num_valid_chunks)
		AppLog.Panicln("We need to pass exactly K coded blocks to the decoder")
	}

	decodedValue, err := encoder.Decode(chunks, codeParams.FileSize)

	if err != nil {
		DebugLog.Println(len(chunks))
		DebugLog.Println(err)
		AppLog.Panicln("Decoding Error in ISAL")
	}

	return decodedValue

}

func CreateGobForDisk(tagvalue TagValue) []byte {

	data, err := json.Marshal(tagvalue)

	//	err := message_to_respond_enc.Encode(message)
	if err != nil {
		AppLog.Panicln("Error gobfying message")
	}

	return data
}

func CreateTypeFromDiskGob(data []byte) TagValue {
	var m TagValue

	err := json.Unmarshal(data, &m)
	if err != nil {
		AppLog.Panicln("Error gobfying message")
	}
	runtime.GC()
	return m
}

func CreateLogLineForOperation(operation OperationParams) []byte {
	data, err := json.Marshal(operation)

	if err != nil {
		AppLog.Panicln("Error gobfying message")
	}

	return data
}

/*
func CreateGobForDisk(tagvalue TagValue) bytes.Buffer {

	var message_to_store bytes.Buffer

	// Create an encoder and send a Value.
	enc := gob.NewEncoder(&message_to_store)
	err := enc.Encode(tagvalue)
	if err != nil {
		fmt.Println("Error gobfying message")
	}

	runtime.GC()
	return message_to_store
}

func CreateTypeFromDiskGob(data []byte) TagValue {

	var buffer bytes.Buffer
	var m TagValue

	buffer.Write(data)
	dec := gob.NewDecoder(&buffer)
	dec.Decode(&m)

	return m
}
*/

func hash(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}
func WriteToDisk(objectname string, folder string, tagvalue TagValue) error {
	subdir := processParams.name + "/" + strconv.Itoa(int(hash([]byte(objectname))%100))

	err := os.MkdirAll(folder+"/"+subdir, 0777)
	if err != nil {
		return err
	}

	buffer := CreateGobForDisk(tagvalue)

	filepath := folder + "/" + subdir + "/" + objectname

	/*
		file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		bytesWritten, err := file.Write(buffer)
		if err != nil {
			log.Fatal(err)
		}
		file.Sync()
		file.Close()
	*/

	err = ioutil.WriteFile(filepath, buffer, 0777)

	//	fmt.Println("bytes written ", bytesWritten, len(buffer))

	return err
}

func ReadFromDisk(objectname string, folder string) (TagValue, error) {
	subdir := processParams.name + "/" + strconv.Itoa(int(hash([]byte(objectname))%100))
	filepath := folder + "/" + subdir + "/" + objectname
	content, err := ioutil.ReadFile(filepath)

	tagvalue := CreateTypeFromDiskGob(content)
	return tagvalue, err
}

func RemovePreviousLogFolder(folder string) {

	if !processParams.remoteConfiguration {
		if processParams.name != "" {
			_ = os.RemoveAll(folder + "/" + processParams.name)
		}
	}
}
