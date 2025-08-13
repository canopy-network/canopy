#!/bin/bash
echo "net.ipv4.tcp_fin_timeout=30" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_tw_reuse=1" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_tw_recycle=1" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_max_tw_buckets=200000" | sudo tee -a /etc/sysctl.conf
