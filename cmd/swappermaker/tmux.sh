tmux new -d  -n"bitcoind" "~/bitcoin-22.0/bin/bitcoind -regtest"
tmux neww -a  -n"kaspad" "/bin/bash ~/Devel/startkaspad.sh"
tmux neww -a -n"kaspawallet" "cd ~/Devel;. startwalletdaemon;bash -i"
tmux neww -a -n"bctswap" "cd ~/Devel/atomicswap/lib/btcatomicswap/;go run . -regtest -s 127.0.0.1:18443 -rpcuser pieroforfora -rpcpass 1234 -listen 8080 daemon;bash -i"
tmux neww -a -n"KASswap" "cd ~/Devel/atomicswap/lib/kaspaatomicswap/;go run . -regtest -rpcpass pieroforfora -listen 8081 daemon;bash -i"
tmux neww -a -n"bash"
tmux a
