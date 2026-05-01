#!/usr/bin/env bash

process="goblog"
echo ${process} $1

start(){
      pid=`pgrep ${process}`
      if [ "${pid}"x = ""x ];then
          echo "start new process..."
          nohup ./${process} -config conf/prod.yaml &
      else
          for i in ${pid}
          do
              echo "stopping process [ $i ] gracefully..."
              kill -15 $i
          done
          sleep 3
      fi
      nohup ./${process} -config conf/prod.yaml &
      sleep 1
      pid=`pgrep ${process}`
      echo "new process id: ${pid}"
}

stop(){
    pid=`pgrep ${process}`
    echo ${pid}
    for i in ${pid}
    do
        echo "stopping process [ $i ] gracefully..."
        kill -15 $i
    done
}

status(){
    ps aux | grep -w ${process} | grep -v 'grep'
}


case "$1" in
    start)
	start $1;;
    stop)
	stop ;;
    status)
	status ;;
    *)
	echo "Usage: $0 {start|stop|status}"

esac
