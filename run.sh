#!/bin/sh
tmux new-session 'registry' \; \
split-window 'nsmgr' \; \
split-window 'icmp-server' \; \
split-window 'NSM_LISTEN_ON_URL=/run/networkservicemesh/icmp.client.sock icmp-client'
