#!/bin/bash


deploy_config=$1
appl_config=$2

numwriters=`grep "num_writers" ${deploy_config} | cut -d: -f2`
numreaders=`grep "num_readers" ${deploy_config} | cut -d: -f2`
numservers=`grep "num_servers" ${deploy_config} | cut -d: -f2`

# writers
echo  "     writers" 
for ((x=1; x<=$numwriters; x++))
do
   #echo  "starting  writer-$x" 
   #./process -process_name=writer-$x > logs/writer-${x}.log &
   ./process -process_name=writer-$x -deploy_config ${deploy_config}  -appl_config ${appl_config} &
done


# readers
echo  "     readers" 
for ((x=1; x<=$numreaders;x++))
do
#   echo  "starting  reader-$x" 
   #./process -process_name=reader-$x > logs/reader-${x}.log  &
   ./process -process_name=reader-$x -deploy_config ${deploy_config}  -appl_config ${appl_config} &
done

# servers
echo  "     servers" 
for ((x=1; x<=$numservers;x++))
do
#   echo  "starting  server-$x" 
   #./process -process_name=server-$x > logs/server-${x}.log &
   ./process -process_name=server-$x -deploy_config ${deploy_config}  -appl_config ${appl_config} &
done

echo  "     controller" 
   #./process -process_name="controller" > logs/controller.log &
./process -process_name="controller" -deploy_config ${deploy_config}  -appl_config ${appl_config} &

