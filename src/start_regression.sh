#!/bin/bash



numreads=100
numwrites=100


# set the number of writes if supplied
if [ ! -z $2 ];  then
   numwrites=$2
fi

# set the number of reads if supplied
if [ ! -z $3 ];  then
   numreads=$3
fi



check_operations() {  #num_readers/writers  deployment_config
  v=0 
  numclients=`grep "$2" $1 | cut -d: -f2`
  for c in `seq $numclients`; do
    port=$(( $c + $3 ))
    N=`curl -m 1 -s http://localhost:${port}/GetOpNum`
    if [ $v -lt $N ] ; then 
      v=$N
    fi
  done    
  echo $v
}


waituntil_operations() {    #numprewrites/numreads "num_writers/num_readers" "port 8500/7500" ${dep_file}
   num=`check_operations $4 $2 $3 #num_writers 8500`
   echo $num, $1
   while [ $num -lt $1 ] ; do 
       num=`check_operations $4 $2 $3 #num_writers 8500`
       sleep 1
       echo -ne "$2 :  $num/$1 \033[0K\r"
   done
   echo 
}


algorithm_regtest() {   #1st arg: deployment_config file  ; #2nd: application_config file
    dep_file=$1
    app_file=$2



    numprewrites=$(( `grep "numobjects" ${app_file} | cut -d: -f2` ))
    numwriters=$(( `grep "num_writers" ${dep_file} | cut -d: -f2` ))
    numservers=$(( `grep "num_servers" ${dep_file} | cut -d: -f2` ))
    numreaders=$(( `grep "num_readers" ${dep_file} | cut -d: -f2` ))
    
    
    for i in `seq ${numreaders}`; do 
       #echo rm -rf /tmp/reader-$i
       rm -rf /tmp/reader-$i
    done
    
    for i in `seq ${numwriters}`; do 
       #echo rm -rf /tmp/writer-$i
       rm -rf /tmp/writer-$i
    done
    
    for i in `seq ${numservers}`; do 
       #echo rm -rf /tmp/server-$i
       rm -rf /tmp/server-$i
    done
    
    
    echo "Starting the servers : ${numservers}, readers : ${numreaders}  and writers : ${numwriters}"
    regtestdata/reg_start_process.sh ${dep_file} ${app_file}
    sleep 2
    
    #do 100 writes
    echo "Doing pre-writes for ${numprewrites} objects"
    echo 
    curl http://localhost:9999/DoWrites/${numprewrites}

    waituntil_operations ${numprewrites} "num_writers" "8500" ${dep_file}

    echo "Doing ${numreads} Reads"
    curl http://localhost:9999/DoReads/${numreads}
    echo "Doing ${numwrites} Writes"
    curl http://localhost:9999/DoWrites/${numwrites}

    numtotwrites=$(( $numwrites + $numprewrites ))

    waituntil_operations ${numtotwrites} "num_writers" "8500"  ${dep_file}
    waituntil_operations ${numreads} "num_readers" "7500"  ${dep_file}

    rm -rf /tmp/verify
    mkdir /tmp/verify

    cat  /tmp/*er-*/logs/applog.txt  | grep FOR_SAFETY_CHECK | sed -e 's/FOR_SAFETY_CHECK:\s*//g'   >  /tmp/verify/safety.log
    
    echo analytics/safetyAnalyzer -safety_log_path /tmp/verify/safety.log -numreaders ${numreaders} -numwriters ${numwriters} -numreads ${numreads} -numwrites ${numtotwrites}
    analytics/safetyAnalyzer -safety_log_path /tmp/verify/safety.log -numreaders ${numreaders} -numwriters ${numwriters} -numreads ${numreads} -numwrites ${numtotwrites}

    #stop servers and clients
    echo "Stopping the servers"
    regtestdata/reg_kill_process.sh $1 $2
}


# public
instruct ()
{
	echo "Usage:  ./start_regression  abd/abd_fast/sodaw/sodaw_fast default with run all"
}

test_all() {
   echo 
   echo "1. ABD TESTING"
   dep_file=regtestdata/deploy_config_abd.yml
   app_file=regtestdata/appl_config_abd.yml
   time algorithm_regtest ${dep_file}  ${app_file}
   
   echo 
   echo "2. ABD FAST TESTING"
   dep_file=regtestdata/deploy_config_abd_fast.yml
   app_file=regtestdata/appl_config_abd_fast.yml
   time algorithm_regtest ${dep_file}  ${app_file}
   
   echo 
   echo "3. SODAW TESTING"
   dep_file=regtestdata/deploy_config_sodaw.yml
   app_file=regtestdata/appl_config_sodaw.yml
   time algorithm_regtest ${dep_file}  ${app_file}
   
   echo "4. SODAW FAST TESTING"
   dep_file=regtestdata/deploy_config_sodaw_fast.yml
   app_file=regtestdata/appl_config_sodaw_fast.yml
   time algorithm_regtest ${dep_file}  ${app_file}
}

# "main"


# pick the choice of tests
case "$1" in
	abd)
	       echo "1. ABD TESTING"
               dep_file=regtestdata/deploy_config_abd.yml
               app_file=regtestdata/appl_config_abd.yml
               time algorithm_regtest ${dep_file}  ${app_file}
		;;
	abd_fast)
	       echo "1. ABD_FAST TESTING"
               dep_file=regtestdata/deploy_config_abd_fast.yml
               app_file=regtestdata/appl_config_abd_fast.yml
               time algorithm_regtest ${dep_file}  ${app_file}
		;;
	sodaw)
	       echo "1. SODAW TESTING"
               dep_file=regtestdata/deploy_config_sodaw.yml
               app_file=regtestdata/appl_config_sodaw.yml
               time algorithm_regtest ${dep_file}  ${app_file}
		;;
	sodaw_fast)
	       echo "1. SODAW_FAST TESTING"
               dep_file=regtestdata/deploy_config_sodaw_fast.yml
               app_file=regtestdata/appl_config_sodaw_fast.yml
               time algorithm_regtest ${dep_file}  ${app_file}
		;;
	all)   # run test for all algorithms
		test_all
		;;
	help|*)
		instruct
		;;
esac


