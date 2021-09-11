#!/bin/bash


deploy_config=$1

numwriters=`grep "num_writers" ${deploy_config} | cut -d: -f2`
numreaders=`grep "num_readers" ${deploy_config} | cut -d: -f2`
numservers=`grep "num_servers" ${deploy_config} | cut -d: -f2`

# writers
echo  "  writers" 
for ((x=1; x<=$numwriters; x++))
do
#   echo  "killing  writer-$x" 
   ps -eaf | grep writer-$x  | awk '{print $2}' | xargs kill 
done


# readers
echo  "  readers" 
for ((x=1; x<=$numreaders;x++))
do
#   echo  "killing  reader-$x" 
   ps -eaf | grep reader-$x  | awk '{print $2}' | xargs kill 
done

# servers
echo  "  servers" 
for ((x=1; x<=$numservers;x++))
do
#   echo  "killing  server-$x" 
   ps -eaf | grep server-$x  | awk '{print $2}' | xargs kill 
done

#echo  "killing  controller" 
echo  " controller" 
ps -eaf | grep "controller"  | awk '{print $2}' | xargs kill 
