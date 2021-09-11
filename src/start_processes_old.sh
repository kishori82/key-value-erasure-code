#!/bin/bash


rm logs/*

writers=`grep "writer-" deploy_config.yml | cut -d: -f1 | sort | uniq`
readers=`grep "reader-" deploy_config.yml | cut -d: -f1 | sort | uniq`
servers=`grep "server-" deploy_config.yml | cut -d: -f1 | sort | uniq`


# writers
for x in ${writers[@]}
do
   echo  "starting  $x" 
   ./process -process_name=$x > logs/${x}.log &
done


# readers
for x in ${readers[@]}
do
   echo  "starting  $x" 
   ./process -process_name=$x > logs/${x}.log  &
done

# servers
for x in ${servers[@]}
do
   echo  "starting  $x" 
   ./process -process_name=$x > logs/${x}.log &
done

   echo  "starting  controller" 
   ./process -process_name="controller" > logs/controller.log &

