package daemons

import (
	"fmt"
	"log"
	"os"
	"time"

	gostatgrab "../gostatgrab"
)

var (
	AppLog               *log.Logger
	SystemLog            *log.Logger
	DebugLog             *log.Logger
	ExpLog               *log.Logger
	AppLogFileHandler    *os.File
	SystemLogFileHandler *os.File
	DebugLogFileHandler  *os.File
	ExpLogFileHandler    *os.File
)

// SetupLog creates a log file as processname/filename
func SetupLog(filename string) (*log.Logger, *os.File) {
	var logpath string
	logpath = MakeAndGetLogFilepath(filename)
	var file, err1 = os.OpenFile(logpath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)

	if err1 != nil {
		panic(err1)
	} else {
		fmt.Println("error free logpath  set up ", filename, logpath)
	}
	return log.New(file, "", 0), file
}

func MakeAndGetLogFilepath(filename string) string {
	folder := DEFAULT_FOLDER_TO_STORE
	var LogFile string = ""
	if processParams.remoteConfiguration {
		err := os.MkdirAll("/tmp/logs/", 0777)
		if err != nil {
			AppLog.Panicln("ERROR : Cannot create process folder  : " + "/tmp/logs/")
		}
		LogFile = "/tmp/logs/" + filename
	} else {
		fmt.Println("I am here")
		err := os.MkdirAll(folder+"/"+processParams.name+"/logs/", 0777)
		if err != nil {
			AppLog.Panicln("ERROR : Cannot create process folder  : " + folder + "/" + processParams.name + "/logs/")
		}
		LogFile = folder + "/" + processParams.name + "/logs/" + filename
	}
	return LogFile
}

func GetLogFilepath(filename string) string {
	folder := DEFAULT_FOLDER_TO_STORE
	var LogFile string = ""
	if processParams.remoteConfiguration {
		LogFile = "/tmp/logs/" + filename
	} else {
		LogFile = folder + "/" + processParams.name + "/logs/" + filename
	}

	return LogFile
}

//LogStats : this function logs statistics system
func LogStats() {
	var networkInBytes uint64 = 0
	var networkOutBytes uint64 = 0
	var networkInPackets uint64 = 0
	var networkOutPackets uint64 = 0

	var pnetworkInBytes uint64 = 0
	var pnetworkOutBytes uint64 = 0
	var pnetworkInPackets uint64 = 0
	var pnetworkOutPackets uint64 = 0

	var ready bool = false
	var interval time.Duration = time.Duration(appParams.LogInterval)

	for {
		memstats, err := gostatgrab.GetMemStats()
		cpupercent, err1 := gostatgrab.GetCpuPercents()
		networkstats, err2 := gostatgrab.GetNetworkIoStats()
		if err != nil || err1 != nil || err2 != nil {
			continue
		}

		networkInBytes = 0
		networkOutBytes = 0
		networkInPackets = 0
		networkOutPackets = 0
		for _, networkstat := range networkstats {
			networkInBytes += networkstat.ReadBytes
			networkOutBytes += networkstat.WriteBytes
			networkInPackets += networkstat.ReadPackets
			networkOutPackets += networkstat.WritePackets
		}
		instant := time.Now()
		/*
		   fmt.Printf("Time:                  %v\n", int64(instant.UnixNano()/1e9) - pinstant)
		   fmt.Printf("CPUPercent:            %.2f\n", cpupercent.User)
		   fmt.Printf("MemTotal (MB):         %d\n", memstats.Total/(1024*1024))
		   fmt.Printf("MemUsed (MB):          %d\n", memstats.Used/(1024*1024))
		   fmt.Printf("NetInBytes (KB):       %d\n", (networkInBytes - pnetworkInBytes)/1024)
		   fmt.Printf("NetOoutbytes (KB):     %d\n", (networkOutBytes -pnetworkOutBytes)/1024)
		   fmt.Printf("NetInPackets:          %d\n", networkInPackets - pnetworkInPackets)
		   fmt.Printf("NetOutPackets:         %d\n", networkOutPackets - pnetworkOutPackets)
		   fmt.Printf("\n")
		*/
		if ready {
			SystemLog.Printf("%v %.2f %d %d %d %d %d %d\n",
				(int64(instant.UnixNano() / 1e3)),
				cpupercent.User,
				memstats.Total/(1024*1024),
				memstats.Used/(1024*1024),
				((networkInBytes - pnetworkInBytes) / 1024),
				((networkOutBytes - pnetworkOutBytes) / 1024),
				(networkInPackets - pnetworkInPackets),
				(networkOutPackets - pnetworkOutPackets))
		}
		ready = true
		time.Sleep(interval * time.Second)

		pnetworkInBytes = networkInBytes
		pnetworkOutBytes = networkOutBytes

		pnetworkInPackets = networkInPackets
		pnetworkOutPackets = networkOutPackets
	}

}

//DoEvery function repeats itself every d time Duration
func DoEvery(d time.Duration, f func()) {
	for _ = range time.Tick(d) {
		f()
	}
}

func initLogHandlers() {
	AppLog, AppLogFileHandler = SetupLog(APPLOGFILE)
	SystemLog, SystemLogFileHandler = SetupLog(SYSTEMLOGFILE)
	DebugLog, DebugLogFileHandler = SetupLog(DEBUGLOGFILE)
	ExpLog, ExpLogFileHandler = SetupLog(EXPLOGFILE)
}

/*
func initUpLogHandlers_new(processName string) {

	AppLog = SetupLog_new(APPLOGFILE, processName)
	SystemLog = SetupLog_new(SYSTEMLOGFILE, processName)
	DebugLog = SetupLog_new(DEBUGLOGFILE, processName)
	ExpLog = SetupLog_new(EXPLOGFILE, processName)
}
*/
