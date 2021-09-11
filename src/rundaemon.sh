#!/bin/bash

export LD_LIBRARY_PATH=/usr/local/lib:/home/docker/COLAS/src/abd:/home/docker/COLAS/src/soda:/home/docker/COLAS/src/codes:/home/docker/COLAS/src/sodaw:/home/docker/COLAS/src/casgc

args=("$@")
ptype=${args[0]}
filesize=${args[1]}

if [ -z "$filesize" ]; then
	/home/docker/COLAS/src/process --process-type ${ptype}
else
	/home/docker/COLAS/src/process --process-type ${ptype} --filesize ${filesize}
fi
