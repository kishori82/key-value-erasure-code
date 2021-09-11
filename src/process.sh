#!/bin/bash
process_name=$1
export GODEBUG='cgocheck=0'
/ercost/core/src/process -process_name=${process_name} -config_remotely
