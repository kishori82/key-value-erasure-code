package daemons

import (
	"io/ioutil"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Set_random_seed(seed int64) {
	rand.Seed(seed)
}

func Generate_random_data(rand_bytes []byte, size int64) error {

	for i := 0; i < (int)(size); i++ {
		v := rand.Uint32()
		rand_bytes[i] = (byte)(v)
	}

	return nil
}

func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					AppLog.Panicln("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
}

func CpuUsage() float64 {
	idle0, total0 := getCPUSample()
	time.Sleep(3 * time.Second)
	idle1, total1 := getCPUSample()

	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks

	//fmt.Printf("CPU usage is %f%% [busy: %f, total: %f]\n", cpuUsage, totalTicks-idleTicks, totalTicks)
	return cpuUsage
}

// Make the following function general to accept any key-value map
func get_keys(mapvar map[string][]string) []string {
	keys := []string{}

	for k := range mapvar {
		keys = append(keys, k)
	}
	sort.Sort(Lexicon(keys))
	return keys
}

func get_keys_from_catalogue(mapvar map[string]*ObjParams) []string {
	keys := []string{}
	for k := range mapvar {
		keys = append(keys, k)
	}

	sort.Sort(Lexicon(keys))
	return keys
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func SetUpParameters() {

	DP := GetDeploymentParams()
	AP := GetApplicationParams()

	// Calculate the derived parameters for both application and deployment
	DP.InitializeDeploymentParams()
	AP.InitializeApplicationParams()

}

// initializes the application params, object params and deployment params after
// receiving them from the controller
func InitializeParams() {
	AP := GetApplicationParams()
	AP.InitalizeMasterObjCatalogue()

	PP := GetProcessParams()
	err := PP.InitializeProcessParams(PP.name, true)
	if err != nil {
		panic("Failed to initialized process params!")
	}
}

func waitUntilParamsIsSet() {

	chan_app_params = make(chan bool, 10) // Check: why 1000 ?
	chan_depl_params = make(chan bool, 10)
	var count1, count2 int
	for {
		select {
		case _ = <-chan_app_params:
			count1 = 1
		case _ = <-chan_depl_params:
			count2 = 1
		}

		if count1 > 0 && count2 > 0 {
			break
		}
	}
}
