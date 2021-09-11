#!/bin/bash


rm -rf /tmp/ercost/*

numwriters=`grep "num_writers" deploy_config.yml | cut -d: -f2`
numreaders=`grep "num_readers" deploy_config.yml | cut -d: -f2`
numservers=`grep "num_servers" deploy_config.yml | cut -d: -f2`



# servers
for ((x=1; x<=$numservers;x++))
do
   echo  "starting  server-$x" 
   #./process -process_name=server-$x > logs/server-${x}.log &
   ./process -process_name=server-$x &
done

   echo  "starting  controller" 
   #./process -process_name="controller" > logs/controller.log &
   ./process -process_name="controller" &

