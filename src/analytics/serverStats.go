package main

import (
	"bufio"
	"flag"
	"os"
	"strconv"
	"strings"
)

const LOGSROOT = "/tmp/ercost/"
const OUTPUTFILE = "RESULTS"

func main() {
	logroot := flag.String("logroot", LOGSROOT, "the path of logs to be analysed")
	outputFile := flag.String("results", OUTPUTFILE, "the name of the output file.")
	serverName := flag.String("server", "", "name of the server")
	interval := flag.Int64("interval", 10, "interval in seconds to sample")
	flag.Parse()

	//getReaderStats

	readSystemLogs(*logroot, *outputFile, *serverName, *interval)

	// Print the results to the output file
	resultspath := *outputFile
	var ff, err1 = os.OpenFile(resultspath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	defer ff.Close()

	if err1 != nil {
		panic(err1)
	}

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func readSystemLogs(logroot string, outfile string, clientName string, interval int64) {
	logpath := logroot + "/" + clientName + "/logs/systemlog.txt"

	var file, err = os.OpenFile(logpath, os.O_RDONLY, 0666)

	if err != nil {
		panic(err)
	}

	fout, err := os.Create(outfile)
	check(err)
	fout.WriteString("Sec\tCPU\tMemSys\tMemUser\tInKBytes\tOutKBytes\tInPackets\tOutPackets\n")

	scanner := bufio.NewScanner(file)
	var timeSpot int64 = 0
	var initTimeSpot int64 = 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, " ")

		if len(fields) < 8 {
			continue
		}

		timePoint, err := strconv.ParseInt(fields[0], 10, 64)
		timePointSec := timePoint / 1e6

		if timeSpot == 0 {
			timeSpot = timePointSec
			initTimeSpot = timePointSec
		}

		if err != nil {
			continue
		}
		if timeSpot < timePointSec {
			fields[0] = strconv.Itoa(int(timePointSec - initTimeSpot))
			fout.WriteString(strings.Join(fields, "\t") + "\n")
			timeSpot += interval
		}
	}
	file.Close()

}
