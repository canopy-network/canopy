#!/bin/bash
sudo netstat -np | grep 50002
sudo ss -K state TIME-WAIT src 127.0.0.1:50002
sudo ss state time-wait sport = 50002 -K

#netstat -np | grep 40002
sudo ss -K state TIME-WAIT src 127.0.0.1:40002
sudo ss state time-wait sport = 40002 -K

#netstat -np | grep 60002
sudo ss -K state TIME-WAIT src 127.0.0.1:60002
sudo ss state time-wait sport = 60002 -K
