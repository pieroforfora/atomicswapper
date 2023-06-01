tmux new -d  -n"bitcoind" "~/bitcoin-22.0/bin/bitcoind -regtest"
tmux neww -a -n"kaspad" "/bin/bash ~/Devel/startkaspad.sh"
tmux neww -a -n"kaspawallet" "cd ~/Devel;. startwalletdaemon;bash -i"
tmux neww -a -n"bctswap" "cd ~/Devel/atomicswapper/cmd/btcatomicswap/;go run . -net regtest -btc-node 127.0.0.1:18443 -btc-rpc-user pieroforfora -btc-rpc-pass 1234 daemon;bash -i"
tmux neww -a -n"KASswap" "cd ~/Devel/atomicswapper/cmd/kaspaatomicswap/;go run . -net devnet -wallet-password pieroforfora daemon;bash -i"
sleep 10
tmux neww -a -n"SwapperMaker" "cd ~/Devel/atomicswapper/cmd/swappermaker/;go run .;bash -i"
sleep 10
tmux neww -a -n"SwapperTaker" "cd ~/Devel/atomicswapper/cmd/swappertaker/;go run .;bash -i"
tmux neww -a -n"bash"
tmux a
