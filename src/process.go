package main

import (
	"flag"

	daemons "./daemons"
)

func main() {

	//args := os.Args
	deploy_config := flag.String("deploy_config", "deploy_config.yml", "the deployment config file in YML format")
	appl_config := flag.String("appl_config", "appl_config.yml", "the application config file in YML format")
	process_name := flag.String("process_name", "", "the name of the process")
	make_config := flag.Bool("make_config", false, "Simply make the default yml files and exit.")
	configure_remotely := flag.Bool("config_remotely", false, "Configure remotely via HTTP ports 8080.")

	flag.Parse()

	/*
		dataForWrite := make([]byte, 10000000)
		rand.Read(dataForWrite)
		i := 0
		err := os.MkdirAll("/data/"+"/reader-1", 0777)

		for _ = range time.Tick(1000 * time.Millisecond) {
			err = ioutil.WriteFile("/data/reader-1/"+"data-"+strconv.Itoa(i)+".txt", dataForWrite, 1777)

			if err != nil {
				panic("hello")
			}
			i = i + 1
		}
	*/

	daemons.Main_daemon(deploy_config, appl_config, process_name, make_config, configure_remotely)

}
