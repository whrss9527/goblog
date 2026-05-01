#!/bin/bash

# 需要监控的服务名称
SERVICE_NAME="goblog"
# 启动服务的命令或脚本路径
START_COMMAND="sh startup.sh start"

# 日志文件
LOG_FILE="./blog_monitor.log"

check_and_restart() {
    # 检查进程是否在运行
    if ! pgrep -x "$SERVICE_NAME" > /dev/null; then
        echo "$(date) - $SERVICE_NAME not running, restarting..." | tee -a "$LOG_FILE"
        # 执行启动命令
        $START_COMMAND &
    fi
}

# 作为守护进程运行
while true; do
    check_and_restart
    sleep 30  # 每 30 秒检查一次
done
