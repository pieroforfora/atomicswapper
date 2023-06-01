package main

import (
  "bufio"
  "crypto/rand"
  "crypto/sha256"
  "encoding/hex"
  //"flag"
  "fmt"
  "os"
  "strconv"
  "strings"
  "time"
  "context"
  "log"
  "bytes"
  "github.com/kaspanet/go-secp256k1"
  "path/filepath"
  //"io/ioutil"
  "net/http"
  //"encoding/json"

  "github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/client"
  "github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
  "github.com/kaspanet/kaspad/cmd/kaspawallet/keys"
  "github.com/kaspanet/kaspad/domain/consensus/utils/constants"
  "github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
  "github.com/kaspanet/kaspad/domain/dagconfig"
  "github.com/kaspanet/kaspad/util"

  "github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
  "github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/bip32"
  "github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
  "github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
  "github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
  "github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
  "github.com/pkg/errors"
  "github.com/tyler-smith/go-bip39"
  UTXO "github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
  "golang.org/x/crypto/blake2b"
  "github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
  "github.com/kaspanet/kaspad/app/appmessage"
  "github.com/pieroforfora/atomicswapper/interfaces"
  "github.com/pieroforfora/atomicswapper/lib/tagconfig"


)

const verify = true


const txVersion = 2

var chainParams = &dagconfig.MainnetParams
type Config struct {
  KaspaWalletUrl  string `env:"KAS_SWAP_WALLET"                cli:"kaspa-wallet"     yaml:"kaspa_wallet"       default:"localhost:8082" help:"host:port of kaspawallet  RPC server"`
  KaspadUrl       string  `env:"KAS_SWAP_NODE_URL"             cli:"kaspad"           yaml:"kaspad"             default:"" help:"host:port of kaspad RPC server"`
  KaspaWalletPass string  `env:"KAS_SWAP_WALLET_PASS"          cli:"wallet-password"      yaml:"wallet_password"    default:""`
  Network         string  `env:"KAS_SWAP_NETWORK"              cli:"net"              yaml:"network"            default:"regtest"`
  LtInit          int64   `env:"KAS_SWAP_LOCKTIME_INITIATE"    cli:"lt-init"    yaml:"ltime_init"      default:"48" help:"locktime initiate method"`
  LtPart          int64   `env:"KAS_SWAP_LOCKTIME_PARTICIPATE" cli:"lt-part"    yaml:"ltime_part"      default:"24" help:"locktime participate method"`
  SecretSize      int64   `env:"KAS_SWAP_SECRET_SIZE"          cli:"secret-size"      yaml:"secret_size"        default:"32"`
  Listen          string  `env:"KAS_SWAP_LISTEN"               cli:"listen"           yaml:"listen"             default:"localhost:8080"`
  GapLimit        string  `env:"KAS_SWAP_GAP_LIMIT"            cli:"gap"              yaml:"gap"                default:"20"`
  Verbose         bool    `env:"KAS_SWAP_VERBOSE"              cli:"verbose"          yaml:"verbose"            default:"false"`
  FeePerInput     uint64  `env:"KAS_SWAP_FEE_PER_INPUT"        cli:"fee-per-input"    yaml:"fee_per_input"      default:"3000"`
}
var cfg= Config{}

var defaultAppDir = util.AppDir("kaspawallet", false)

func defaultKeysFile(netParams *dagconfig.Params) string {
  return filepath.Join(defaultAppDir, netParams.Name, "keys.json")
}
var daemonPassword string

  //kaspad --devnet --utxoindex --archival --nodnsseed  --listen 127.0.0.1:16111 --externalip=127.0.0.1 --allow-submit-block-when-not-synced  --loglevel=trace

  // There are two directions that the atomic swap can be performed, as the
  // initiator can be on either chain.  This tool only deals with creating the
  // Bitcoin transactions for these swaps.  A second tool should be used for the
  // transaction on the other chain.  Any chain can be used so long as it supports
  // OP_SHA256 and OP_CHECKLOCKTIMEVERIFY.
  //
  // Example scenerios using bitcoin as the second chain:
  //
  // Scenerio 1:
  //   cp1 initiates (kas)
  //   cp2 participates with cp1 H(S) (btc)
  //   cp1 redeems btc revealing S
  //     - must verify H(S) in contract is hash of known secret
  //   cp2 redeems kas with S
  //
  // Scenerio 2:
  //   cp1 initiates (btc)
  //   cp2 participates with cp1 H(S) (kas)
  //   cp1 redeems kas revealing S
  //     - must verify H(S) in contract is hash of known secret
  //   cp2 redeems btc with S
/*
func init() {
  flagset.Usage = func() {
    fmt.Println("Usage: kaspaatomicswap [flags] cmd [cmd args]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  initiate <participant address> <amount>")
    fmt.Println("  participate <initiator address> <amount> <secret hash>")
    fmt.Println("  redeem <contract> <contract transaction> <secret>")
    fmt.Println("  refund <contract> <contract transaction>")
    fmt.Println("  extractsecret <redemption transaction> <secret hash>")
    fmt.Println("  auditcontract <contract> <contract transaction>")
    fmt.Println("  auditcontractonline <contract> <contract transaction>")
    fmt.Println("  daemon")
    fmt.Println("  pushtx <tx>")
    fmt.Println()
    fmt.Println("Flags:")
    flagset.PrintDefaults()
  }
}
*/
type command interface {
  runCommand([]string, pb.KaspawalletdClient, context.Context, *keys.File) error
}

// offline commands don't require wallet RPC.
type offlineCommand interface {
  command
  runOfflineCommand() error
}

type daemonOutput interface{
   interfaces.BuildContractOutput | interfaces.SpendContractOutput | interfaces.AuditContractOutput | interfaces.ExtractSecretOutput | interfaces.PushTxOutput | interfaces.AtomicSwapParamsOutput | interfaces.WalletBalanceOutput
}

type daemonCommand interface {
    runDaemonCommand([]string, pb.KaspawalletdClient, context.Context, *keys.File)(any,error)
}

// contractArgs specifies the common parameters used to create the initiator's
// and participant's contract.
type contractArgsCmd struct {
  them       *util.AddressPublicKey
  amount     uint64
  locktime   uint64
  secretHash []byte
  secret     []byte
  secretSize int64
}
type walletBalanceCmd struct {}
type atomicSwapParamsCmd struct {}


type spendArgsCmd struct {
  contract    []byte
  contractTx  *externalapi.DomainTransaction
  secret      []byte

}

type extractSecretCmd struct {
  redemptionTx *externalapi.DomainTransaction
  secretHash   []byte
}

type auditContractCmd struct {
  contract   []byte
  contractTx *externalapi.DomainTransaction
}
type checkRedeemCmd interfaces.CheckRedeemInput
func main() {
  err, showUsage := run()
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
  }
  /*if showUsage {
    flagset.Usage()
  }*/
  if err != nil || showUsage {
    os.Exit(1)
  }
}
func checkCmdArgLength(args []string, required int) (nArgs int) {
  if len(args) < required {
    return 0
  }
  for i, arg := range args[:required] {
    if len(arg) != 1 && strings.HasPrefix(arg, "-") {
      return i
    }
  }
  return required
}

func run() (err error, showUsage bool) {
  ctx := context.Background()
  args,err := tagconfig.Parse(&cfg,ctx)
  fmt.Println(cfg.KaspaWalletPass)
  if len(args) == 0 {
    return nil, true
  }
  /*
  cmdArgs := 0
  switch args[0] {
  case "initiate":
    cmdArgs = 2
  case "participate":
    cmdArgs = 3
  case "redeem":
    cmdArgs = 3
  case "refund":
    cmdArgs = 2
  case "extractsecret":
    cmdArgs = 2
  case "auditcontract":
    cmdArgs = 2
  case "auditcontractonline":
    cmdArgs = 2
  case "daemon":
    cmdArgs = 0
  case "pushtx":
    cmdArgs = 1
  default:
    return fmt.Errorf("unknown command %v", args[0]), true
  }
  nArgs := checkCmdArgLength(args[1:], cmdArgs)
  flagset.Parse(args[1+nArgs:])
  if nArgs < cmdArgs {
    return fmt.Errorf("%s: too few arguments", args[0]), true
  }
  if flagset.NArg() != 0 {
    return fmt.Errorf("unexpected argument: %s", flagset.Arg(0)), true
  }
  */

  switch cfg.Network {
    case "testnet":
      chainParams = &dagconfig.TestnetParams
    case "devnet":
      chainParams = &dagconfig.DevnetParams
    default:
      chainParams = &dagconfig.MainnetParams
  }

  if cfg.KaspadUrl==""{
    tmp := "localhost:"+string(chainParams.RPCPort)
    cfg.KaspadUrl = tmp
  }



  var cmd command
  isOnline := false
  switch args[0] {
  case "initiate":
    cmd,err = parseBuildArgs(interfaces.BuildContractInput{
        Them:   args[1],
        Amount: args[2],
    })
    if err != nil{
      log.Fatal(err)
    }

  case "participate":
    cmd,err = parseBuildArgs(interfaces.BuildContractInput{
        Them:       args[1],
        Amount:     args[2],
        SecretHash: &args[3],
    })
    if err != nil{
      log.Fatal(err)
    }
  case "redeem":
    cmd,err = parseSpendArgs(interfaces.SpendContractInput{
      Contract:   args[1],
      Tx:         args[2],
      Secret:     args[3],
    })
    if err != nil{
      log.Fatal(err)
    }

  case "refund":
    cmd,err = parseSpendArgs(interfaces.SpendContractInput{
      Contract: args[1],
      Tx:       args[2],
    })
    if err != nil{
      log.Fatal(err)
    }

  case "extractsecret":
    cmd,err = parseExtractSecretArgs(interfaces.ExtractSecretInput{
      Tx:         args[1],
      SecretHash: args[2],
    })
    if err != nil{
      log.Fatal(err)
    }

  case "auditcontract":
    cmd,err = parseAuditContractArgs(interfaces.AuditContractInput{
      Contract: args[1],
      Tx:       args[2],
    })
    if err != nil{
      log.Fatal(err)
    }

  case "auditcontractonline":
    cmd,err = parseAuditContractArgs(interfaces.AuditContractInput{
      Contract: args[1],
      Tx:       args[2],
    })
    if err != nil{
      log.Fatal(err)
    }
    isOnline =true

  case "daemon":
    restApiRequestsHandlers()
    daemonPassword = getPassword()
    fmt.Println("Server is up and running...",cfg.Listen)
    log.Fatal(http.ListenAndServe(cfg.Listen, nil))
    return nil,false

  case "pushtx":
    id,err := pushTx(interfaces.PushTxInput{Tx:args[1]})
    if err != nil {
      log.Fatal(err)
    }
    fmt.Println(*id)
    return nil,false
  }
  if err!= nil{
    log.Fatal(err)
  }
  // Offline commands don't need to talk to the wallet.
  if !isOnline {
    if  cmd, ok := cmd.(offlineCommand); ok {
      return cmd.runOfflineCommand(), false
    }
  }
  daemonClient, tearDown, err := client.Connect(cfg.KaspaWalletUrl)
  if err != nil {
    log.Fatal(err)
  }
  defer tearDown()
  keysFile, _ := keys.ReadKeysFile(chainParams, defaultKeysFile(chainParams))

  password := getPassword()
  mnemonics, _ := keysFile.DecryptMnemonics(password)
  err = cmd.runCommand(mnemonics,daemonClient,ctx,keysFile)

  return err, false
}

func getPassword() string{
  if cfg.KaspaWalletPass != ""{
    return cfg.KaspaWalletPass
  }else{
    return keys.GetPassword("Password:")
  }
}
/*
func isFlagPassed(name string) bool {
  out:=false
  flagset.Visit(func(f *flag.Flag) {
      if f.Name == name {
          out=true
      }
  })
  return out
}
*/
func getTxFee(tx *externalapi.DomainTransaction) uint64{
  allInputSompi := uint64(0)
  allOutputSompi := uint64(0)
  for _, i :=range tx.Inputs{
      allInputSompi += uint64(i.UTXOEntry.Amount())

  }
  for _, o :=range tx.Outputs{
      allOutputSompi += uint64(o.Value)

  }
  return allInputSompi- allOutputSompi
}
func printDomainTransaction(tx *externalapi.DomainTransaction) {
  fmt.Printf("Transaction ID: \t%s\n", consensushashing.TransactionID(tx))
  fmt.Println()
  fmt.Println("Inputs:")
  allInputSompi := uint64(0)
  for index, input := range tx.Inputs {
    allInputSompi += uint64(input.UTXOEntry.Amount())
    fmt.Printf("\t%v:%d\tAmount: %.8f Kaspa\n",input.PreviousOutpoint.TransactionID,index,getKaspaXSompi(input.UTXOEntry.Amount()))
  }
  fmt.Println("\nOutputs:")
  allOutputSompi := uint64(0)
  for index,output := range(tx.Outputs){
    scriptPublicKeyType, scriptPublicKeyAddress, err := txscript.ExtractScriptPubKeyAddress(output.ScriptPublicKey, chainParams)
    if err != nil {
      log.Fatal(err)
    }

    addressString := scriptPublicKeyAddress.EncodeAddress()
    if scriptPublicKeyType == txscript.NonStandardTy {
      scriptPublicKeyHex := hex.EncodeToString(output.ScriptPublicKey.Script)
      addressString = fmt.Sprintf("<Non-standard transaction script public key: %s>", scriptPublicKeyHex)
    }
    fmt.Printf("\t%d:%s\tAmount: %.8f Kaspa\n",
      index, addressString, getKaspaXSompi(output.Value))

    allOutputSompi += uint64(output.Value)

  }
  fmt.Println("")
  fmt.Printf("Fee:\t%d Sompi\n", allInputSompi-allOutputSompi)
  fmt.Printf("GAS:\t%d Sompi\n", tx.Gas)
  fmt.Printf("LockTime:\t%d \n", tx.LockTime)
  fmt.Println("")
  fmt.Println("")
}

func printRpcTransaction(rpcTransaction *appmessage.RPCTransaction){
  fmt.Println("Transaction:")
  fmt.Println("\tVersion:",rpcTransaction.Version)
  fmt.Println("\tLockTime:",rpcTransaction.LockTime)
  fmt.Println("\tSubnetworkID:",rpcTransaction.SubnetworkID)
  fmt.Println("\tGas:",rpcTransaction.Gas)
  fmt.Println("\tPayload:",rpcTransaction.Payload)
  fmt.Println("\tInputs:")
  for i := range rpcTransaction.Inputs{
    fmt.Println("\t\tInput:", i)
    fmt.Println("\t\tSignatureScript:",rpcTransaction.Inputs[i].SignatureScript)
    fmt.Println("\t\tSequence:",rpcTransaction.Inputs[i].Sequence)
    fmt.Println("\t\tSigOpCount:",rpcTransaction.Inputs[i].SigOpCount)
  }
  fmt.Println("Outputs:")
  for i := range rpcTransaction.Outputs{
    fmt.Println("\t\tOutput:",i)
    fmt.Println("\t\tAmount:",rpcTransaction.Outputs[i].Amount)
    fmt.Println("\t\tScriptPublicKey:")
    fmt.Println("\t\t\tVersion:",rpcTransaction.Outputs[i].ScriptPublicKey.Version)
    fmt.Println("\t\t\tScript:",rpcTransaction.Outputs[i].ScriptPublicKey.Script)
  }
}

func printContract( description string, script []byte) {
  fmt.Println("")
  fmt.Println(description+":")
  fmt.Println(hex.EncodeToString(script))
  fmt.Println("DASM:")
  fmt.Println(dasmScript(script))
  fmt.Println("Blake2b:")
  fmt.Println(hex.EncodeToString(getBlake2b(script)))
  fmt.Println("")
}

func getBlake2b(text []byte) ([]byte){
  hash := blake2b.Sum256(text)
  slice := hash[:]
  return slice
}

func dasmScript(script []byte)(string){
  plainScript,err := txscript.DisasmString(0, script)
  if err != nil {
    fmt.Println("impossible to dasm")
    log.Fatal(err)
  }
return plainScript
}


func extendedKeyFromMnemonicAndPath(mnemonic string, path string, params *dagconfig.Params) (*bip32.ExtendedKey, error) {
  seed := bip39.NewSeed(mnemonic, "")
  version, err := versionFromParams(params)
  if err != nil {
    return nil, err
  }
  master, err := bip32.NewMasterWithPath(seed, version, path)
  if err != nil {
    return nil, err
  }
  return master, nil
}

func derivedKeyToSchnorrKeypair(extendedKey *bip32.ExtendedKey) *secp256k1.SchnorrKeyPair{
  privateKey := extendedKey.PrivateKey()
  schnorrKeyPair,_ := privateKey.ToSchnorr()
  return schnorrKeyPair


}

func rawTxInSignature(extendedKey *bip32.ExtendedKey, tx *externalapi.DomainTransaction, idx int, hashType consensushashing.SigHashType,
  sighashReusedValues *consensushashing.SighashReusedValues, ecdsa bool) ([]byte, error) {

  privateKey := extendedKey.PrivateKey()
  if ecdsa {
    return txscript.RawTxInSignatureECDSA(tx, idx, hashType, privateKey, sighashReusedValues)
  }

  schnorrKeyPair, err := privateKey.ToSchnorr()
  if err != nil {
    return nil, err
  }

  return txscript.RawTxInSignature(tx, idx, hashType, schnorrKeyPair, sighashReusedValues)
}

func searchAddressByBlake2b(addresses []string, blake []byte, extendedPublicKeys []string, ecdsa bool) (*util.Address, *string) {
  for i := range addresses {
    path := fmt.Sprintf("m/%d/%d", libkaspawallet.ExternalKeychain, i+1)
    new_address, _ := libkaspawallet.Address(chainParams, extendedPublicKeys, 1, path, ecdsa)
    if hex.EncodeToString(getBlake2b(new_address.ScriptAddress())) == hex.EncodeToString(blake){
      return &new_address, &path
    }
  }
  return nil,nil
}

func getAddressPath(addresses []string, address string, extendedPublicKeys []string, ecdsa bool) *string {
  for i, taddress := range addresses {
    if taddress == address {
      path := fmt.Sprintf("m/%d/%d", libkaspawallet.ExternalKeychain, i+1)
      new_address, _ := libkaspawallet.Address(chainParams, extendedPublicKeys, 1, path, ecdsa)
      if address == new_address.EncodeAddress() {
        return &path
      }
    }
  }
  return nil
}
func getAddresses(daemonClient pb.KaspawalletdClient, ctx context.Context) []string {
  addressesResponse, err := daemonClient.ShowAddresses(ctx, &pb.ShowAddressesRequest{})
  if err != nil {
    log.Fatal(err)
    return []string{}
  }
  return addressesResponse.Address
}

func versionFromParams(params *dagconfig.Params) ([4]byte, error) {
  switch params.Name {
  case dagconfig.MainnetParams.Name:
    return bip32.KaspaMainnetPrivate, nil
  case dagconfig.TestnetParams.Name:
    return bip32.KaspaTestnetPrivate, nil
  case dagconfig.DevnetParams.Name:
    return bip32.KaspaDevnetPrivate, nil
  case dagconfig.SimnetParams.Name:
    return bip32.KaspaSimnetPrivate, nil
  }
  return [4]byte{}, errors.Errorf("unknown network %s", params.Name)
}

func defaultPath(isMultisig bool) string {
  purpose := SingleSignerPurpose
  if isMultisig {
    purpose = MultiSigPurpose
  }

  return fmt.Sprintf("m/%d'/%d'/0'", purpose, CoinType)
}

const (
  SingleSignerPurpose = 44
  // Note: this is not entirely compatible to BIP 45 since
  // BIP 45 doesn't have a coin type in its derivation path.
  MultiSigPurpose = 45
  // TODO: Register the coin type in https://github.com/satoshilabs/slips/blob/master/slip-0044.md
  CoinType = 111111
)

func getAddressPushes(name string, addresses []string ,blake []byte,keysFile *keys.File)(*util.Address, *string){
  var addr *util.Address
  var path *string
  if keysFile != nil{
    addr, path = searchAddressByBlake2b(addresses,blake,keysFile.ExtendedPublicKeys, keysFile.ECDSA)
  }
  if cfg.Verbose {
    fmt.Println("Pushes -", name,"from Contract:")
    if addr!=nil {
       fmt.Println(*addr)
    }
    fmt.Println(hex.EncodeToString(blake))
    fmt.Println("")
  }
  if addr != nil {
    return addr, path
  } else {
    return nil,nil
  }


}

func parsePushes(contractr []byte,addresses []string, keysFile *keys.File)(*parsedPushes,error){
  pushes, err := txscript.ExtractAtomicSwapDataPushes(0, contractr)
  if err != nil {
    return nil,errors.New(fmt.Sprintf("Impssible to extract Atomic Swap: %v",err))
  }
  if pushes == nil {
    return nil,errors.New("contract is not an atomic swap script recognized by this tool")
  }
  var recipientAddr *util.Address
  var recipient_path *string
  var refundAddr *util.Address
  var refund_path *string
  if keysFile != nil && addresses != nil{
    recipientAddr, recipient_path = getAddressPushes("Recipient", addresses, pushes.RecipientBlake2b[:], keysFile)
    refundAddr, refund_path = getAddressPushes("Refund", addresses, pushes.RefundBlake2b[:], keysFile)
  }

  if cfg.Verbose{
    fmt.Println("Pushes - Secret hash from Contract:")
    fmt.Println(hex.EncodeToString(pushes.SecretHash[:]))
    fmt.Println("")

    fmt.Println("Pushes - Secret size from Contract:")
    fmt.Println(pushes.SecretSize)
    fmt.Println("")

    fmt.Println("Pushes - LockTime from Contract:")
    fmt.Println(pushes.LockTime, "-",time.Unix(int64(pushes.LockTime/1000), 0))
    fmt.Println("")
  }
  return &parsedPushes{
    recipientAddr:    recipientAddr,
    recipient_path:   recipient_path,
    recipientBlake2b: pushes.RecipientBlake2b[:],
    refundAddr:       refundAddr,
    refund_path:      refund_path,
    refundBlake2b: pushes.RefundBlake2b[:],
    secretHash:    pushes.SecretHash[:],
    secretSize:       pushes.SecretSize,
    lockTime:         pushes.LockTime,
    },nil
}
// sendRawTransaction calls the signRawTransaction JSON-RPC method.  It is
// implemented manually as client support is currently outdated from the
// btcd/rpcclient package.
func sendRawTransaction(tx externalapi.DomainTransaction) (*string, error) {
  rpcTransaction := appmessage.DomainTransactionToRPCTransaction(&tx)
  kaspadClient, err := rpcclient.NewRPCClient(cfg.KaspadUrl)

  if err != nil {
    fmt.Println("impossible to connect to kaspad ",cfg.KaspadUrl,err)
    return nil,err
  }
  txID,err :=sendTransaction(kaspadClient, rpcTransaction)
  if err != nil {
    fmt.Println("impossible to send transaction",err)
    return nil,err
  }
  fmt.Println("Transactions were sent successfully!")
  fmt.Println("Transaction ID(s): ")
  fmt.Printf("\t%s\n", txID)
  return &txID,nil

}

func sendTransaction(client *rpcclient.RPCClient, rpcTransaction *appmessage.RPCTransaction) (string, error) {
  submitTransactionResponse, err := client.SubmitTransaction(rpcTransaction, true)
  if err != nil {
    return "", errors.Wrapf(err, "error submitting transaction")
  }
  return submitTransactionResponse.TransactionID, nil
}


// getRawChangeAddress calls the getrawchangeaddress JSON-RPC method.  It is
// implemented manually as the rpcclient implementation always passes the
// account parameter which was removed in Bitcoin Core 0.15.
func getRawChangeAddress(daemonClient pb.KaspawalletdClient, ctx context.Context) (util.Address,) {
  changeAddrs, _ := daemonClient.NewAddress(ctx, &pb.NewAddressRequest{})
  changeAddr, _ := util.DecodeAddress(changeAddrs.Address, chainParams.Prefix)
  return changeAddr
}

func promptPublishTx(tx externalapi.DomainTransaction, name string, daemonClient pb.KaspawalletdClient, ctx context.Context) error {
  reader := bufio.NewReader(os.Stdin)
  for {
    fmt.Printf("Publish %s transaction? [y/N] ", name)
    answer, err := reader.ReadString('\n')
    if err != nil {
      return err
    }
    answer = strings.TrimSpace(strings.ToLower(answer))

    switch answer {
    case "y", "yes":
    case "n", "no", "":
      return nil
    default:
      fmt.Println("please answer y or n")
      continue
    }
    txID,err:=sendRawTransaction(tx)
    if err != nil {
      return fmt.Errorf("sendrawtransaction: %v", err)
    }
    fmt.Printf("Published %s transaction (%v)\n", name, txID)
    return nil
  }
}

// builtContract houses the details regarding a contract and the contract
// payment transaction, as well as the transaction to perform a refund.
type builtContract struct {
  contract       []byte
  contractP2SH   util.Address
  contractTx     *externalapi.DomainTransaction
  contractTxHash []byte
  contractFee    uint64
}

type builtSpend struct {
  spendTx       *externalapi.DomainTransaction
  spendTxHash   []byte
  spendTxFee    uint64
}
type auditedContract struct {
  contract      []byte
  contractTx    *externalapi.DomainTransaction
  contractP2SH  util.Address
  recipient     *util.Address
  recipient2b   []byte
  amount        uint64
  author        *util.Address
  author2b      []byte
  secretHash    []byte
  secretSize    int64
  lockTime      uint64
  daaScore      uint64
  txId          externalapi.DomainTransactionID
  vspbs         uint64
  isSpendable   bool
  nUtxo         int
  idx           int
}

type parsedPushes struct {
recipientAddr     *util.Address
recipient_path    *string
recipientBlake2b  []byte
refundAddr        *util.Address
refund_path       *string
refundBlake2b     []byte
secretHash        []byte
secretSize        int64
lockTime          uint64
}

func getContractIn(amount uint64, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) ([]*externalapi.DomainTransactionInput,uint64,[]string) {
  kaspadClient, _ := rpcclient.NewRPCClient(cfg.KaspadUrl)
  addressesResponse, _ := daemonClient.ShowAddresses(ctx, &pb.ShowAddressesRequest{})

  getUTXOsByAddressesResponse,_  := kaspadClient.GetUTXOsByAddresses(addressesResponse.Address)


  inputs := []*externalapi.DomainTransactionInput{}
  input_amount := uint64(0)
  done := false
  change := uint64(0)
  var paths []string

  dagInfo, _ := kaspadClient.GetBlockDAGInfo()
  for _, entry := range getUTXOsByAddressesResponse.Entries {
    if !isUTXOSpendable(entry, dagInfo.VirtualDAAScore) {
      continue
    }
    if input_amount < ( amount+ getFee(inputs)) {
      address := entry.Address
      address_path := getAddressPath(addressesResponse.Address,address,keysFile.ExtendedPublicKeys, keysFile.ECDSA)
      paths = append(paths, *address_path)
      txid, _ := externalapi.NewDomainTransactionIDFromString(entry.Outpoint.TransactionID)
      script_pub_key,_ := hex.DecodeString(entry.UTXOEntry.ScriptPublicKey.Script)
      inputs = append(inputs, &externalapi.DomainTransactionInput{PreviousOutpoint: externalapi.DomainOutpoint{
          TransactionID:    *txid,
          Index:            entry.Outpoint.Index,
        },
        SigOpCount:         1,
        UTXOEntry: UTXO.NewUTXOEntry(
         entry.UTXOEntry.Amount,
          &externalapi.ScriptPublicKey{
            Version: uint16(entry.UTXOEntry.ScriptPublicKey.Version),
            Script: script_pub_key,
          },
          entry.UTXOEntry.IsCoinbase,
          entry.UTXOEntry.BlockDAAScore,
        ),
      })

      input_amount += uint64(entry.UTXOEntry.Amount)
      change = input_amount - amount
    }else{
      done = true
      break
    }
  }
  if !done{
    log.Fatal("not enough inputs to spend")
  }
  return inputs,  change, paths
}

func isUTXOSpendable(entry *appmessage.UTXOsByAddressesEntry, virtualSelectedParentBlueScore uint64) bool {
  blockDAAScore := entry.UTXOEntry.BlockDAAScore
  if !entry.UTXOEntry.IsCoinbase {
    const minConfirmations = 10
    return blockDAAScore+minConfirmations < virtualSelectedParentBlueScore
  }
  coinbaseMaturity := chainParams.BlockCoinbaseMaturity
  return blockDAAScore+coinbaseMaturity < virtualSelectedParentBlueScore
}

// builtContract houses the details regarding a contract and the contract
// payment transaction, as well as the transaction to perform a refund.


// buildContract creates a contract for the parameters specified in args, using
// wallet RPC to generate an internal address to redeem the refund and to sign
// the payment to the contract transaction.
func buildContract(daemonClient pb.KaspawalletdClient, ctx context.Context, mnemonics []string, keysFile *keys.File, args *contractArgsCmd) (*builtContract, error) {
  refundAddr := getRawChangeAddress(daemonClient,ctx)
  refundAddrH := getBlake2b(refundAddr.ScriptAddress())
  themAddrH := getBlake2b(args.them.ScriptAddress())
  contract, err := atomicSwapContract(refundAddrH, themAddrH,
    args.locktime, args.secretHash, args.secretSize)
  if err != nil {
    return nil, err
  }
  contractP2SH, err := util.NewAddressScriptHash(contract, chainParams.Prefix)
  if err != nil {
    return nil, err
  }

  contractP2SHPkScript, err := txscript.PayToScriptHashScript(contract)

  if err != nil {
    return nil, err
  }

  inputs,  changeAmount, paths := getContractIn(args.amount, daemonClient,ctx, keysFile)

  changeAddrs, _ := daemonClient.NewAddress(ctx, &pb.NewAddressRequest{})
  changeAddr, _ := util.DecodeAddress(changeAddrs.Address, chainParams.Prefix)
  changeAddressScript, _ := txscript.PayToAddrScript(changeAddr)
  fees := getFee(inputs)
  domainTransaction := &externalapi.DomainTransaction{
    Version: constants.MaxTransactionVersion,
    Inputs:  inputs,
    Outputs: []*externalapi.DomainTransactionOutput{
      {
        Value: uint64(args.amount),
        ScriptPublicKey: &externalapi.ScriptPublicKey{
          Version: constants.MaxScriptPublicKeyVersion,
          Script:  contractP2SHPkScript,
        },
      },
      {
        Value: uint64(changeAmount)-fees,
        ScriptPublicKey: changeAddressScript,
      },
    },
    LockTime:     0,
    SubnetworkID: subnetworks.SubnetworkIDNative,
    Gas:          0,
    Payload:      nil,
  }
  // Sign all inputs in transaction

  for i, input := range domainTransaction.Inputs {
    derivedKey,_ := getKeys(paths[i],mnemonics, keysFile)
    keyPair := derivedKeyToSchnorrKeypair(derivedKey)
    signatureScript, err := txscript.SignatureScript(domainTransaction, i, consensushashing.SigHashAll, keyPair,
      &consensushashing.SighashReusedValues{})
    if err != nil {
      return nil, err
    }
    input.SignatureScript = signatureScript
  }


  //refundTx, refundFee := buildSpend(contract, domainTransaction, nil, mnemonics,daemonClient,ctx,keysFile)
  txHash,err := serialization.SerializeDomainTransaction(domainTransaction)
  if err != nil {
    log.Fatal(err)
  }

  return &builtContract{
    contract: contract,
    contractP2SH: contractP2SH,
    contractTxHash: txHash,
    contractTx: domainTransaction,
    contractFee: fees,
  }, nil
}

func getKeys(path string, mnemonics []string, keysFile *keys.File)(*bip32.ExtendedKey, []byte){
  extendedKey, _ := extendedKeyFromMnemonicAndPath(mnemonics[0], defaultPath(false), chainParams)
  derivedKey, err := extendedKey.DeriveFromPath(path)
  if err != nil { log.Fatal(err)}
  return derivedKey, getSerializedPublicKey(derivedKey, keysFile)
}

func getSerializedPublicKey(derivedKey *bip32.ExtendedKey, keysFile *keys.File)([]byte){
  publicKey,_ := derivedKey.PublicKey()
  if keysFile.ECDSA {
    serializedECDSAPublicKey, err := publicKey.Serialize()
    if err != nil {
      log.Fatal("impossible to serialize public key")
    }
  return serializedECDSAPublicKey[:]
  } else {
    publicKey.ToSchnorr()
    schnorrPublicKey, err := publicKey.ToSchnorr()
    if err != nil {
      log.Fatal("impossible to get schnorr public key")
    }
    serializedSchnorrPublicKey, err := schnorrPublicKey.Serialize()
    if err != nil {
      log.Fatal("impossible to serialize schnorr public key")
    }
    return serializedSchnorrPublicKey[:]

  }
}

func getFee(inputs []*externalapi.DomainTransactionInput) uint64{
  return uint64(cfg.FeePerInput)*uint64(len(inputs)+1)
}

func getContractOut(contractr []byte, tx *externalapi.DomainTransaction) *int {
  contractHash, _ := txscript.PayToScriptHashScript(contractr)
  for idx, outputs := range tx.Outputs {
    if hex.EncodeToString(contractHash) == hex.EncodeToString(outputs.ScriptPublicKey.Script){
      return &idx
    }
  }
  return nil
}

func spendContract(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File, args *spendArgsCmd)(*builtSpend,error) {
  contract_idx := getContractOut(args.contract,args.contractTx)
  txid := consensushashing.TransactionID(args.contractTx)

  addressesResponse,  err := daemonClient.ShowAddresses(ctx, &pb.ShowAddressesRequest{})
  if err != nil{
    log.Fatal(err)
  }
  addresses := addressesResponse.Address
  parsed,_:= parsePushes(args.contract, addresses,keysFile)
  isRedeem := (args.secret != nil)
  if (parsed.refundAddr == nil || parsed.refund_path == nil) && !isRedeem {
    log.Fatal("refundAddress is unknown I'm not able to sign refund transaction")
  } else {
    if (parsed.recipientAddr == nil || parsed.recipient_path == nil) && isRedeem{
      log.Fatal("redeemAddress is unknown I'm not able to sign redeem transaction")
    }
  }
  var recipientAddr *util.Address
  var recipient_path *string
  if isRedeem{
    parsed.lockTime=uint64(0)
    recipientAddr = parsed.recipientAddr
    recipient_path = parsed.recipient_path
  }else{
    recipientAddr= parsed.refundAddr
    recipient_path = parsed.refund_path
  }
  derivedKey,serializedPublicKey := getKeys(*recipient_path,mnemonics,keysFile)
  inputs := []*externalapi.DomainTransactionInput{{
    PreviousOutpoint: externalapi.DomainOutpoint{
      TransactionID: *txid,
      Index:         0,
    },
    SigOpCount: 1,
    //  Sequence: math.MaxUint64-1,
    UTXOEntry: UTXO.NewUTXOEntry(args.contractTx.Outputs[*contract_idx].Value,args.contractTx.Outputs[*contract_idx].ScriptPublicKey,false,0),
  }}

  spend_fees := getFee(inputs)

  script_pubkey, _ := txscript.PayToAddrScript(*recipientAddr)

  outputs := []*externalapi.DomainTransactionOutput{{
    Value:           (args.contractTx.Outputs[0].Value - spend_fees) ,
    ScriptPublicKey: script_pubkey,
  }}

  domainTransaction := &externalapi.DomainTransaction{
    Version:  constants.MaxTransactionVersion,
    Outputs:  outputs,
    Inputs:   inputs,
    LockTime: parsed.lockTime,
    Gas:      0,
    Payload:  []byte{},
  }
  sighashReusedValues := &consensushashing.SighashReusedValues{}
  signature,_ := rawTxInSignature(derivedKey, domainTransaction, 0, consensushashing.SigHashAll, sighashReusedValues, keysFile.ECDSA)

  var sigScript []byte
  if isRedeem{
    sigScript, _ = redeemP2SHContract(args.contract,  signature, serializedPublicKey, args.secret)
  } else {
    sigScript, _ = refundP2SHContract(args.contract,  signature, serializedPublicKey)
  }
  domainTransaction.Inputs[0].SignatureScript =   sigScript

  txHash,err := serialization.SerializeDomainTransaction(domainTransaction)
  return &builtSpend{
    spendTx:      domainTransaction,
    spendTxHash:  txHash,
    spendTxFee:   spend_fees,
  },nil
}

func sha256Hash(x []byte) []byte {
  h := sha256.Sum256(x)
  return h[:]
}
func getSecret(size *int64)([]byte,[]byte, error){
  if size == nil {
    size = &cfg.SecretSize
  }
  //var secret [int(secretSize)]byte
  var secret =make([]byte,*size)
  _, err := rand.Read(secret[:])
  if err != nil {
    return nil,nil,err
  }
  secretHash := sha256Hash(secret[:])
  return secret[:],secretHash,nil
}
func importAddress(address string){
  
}
func (cmd *contractArgsCmd) runCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) error {
  b, err := buildContract(daemonClient, ctx, mnemonics, keysFile,cmd)
  if err != nil {
    return err
  }
  printCommand(cmd.secret,cmd.secretHash,b)
  return promptPublishTx(*b.contractTx, "contract",daemonClient,ctx)
}

func (cmd *contractArgsCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) (any,error){
  data, err := buildContract(daemonClient, ctx, mnemonics, keysFile,cmd)
  if err != nil {
    return nil,err
  }
  secret := hex.EncodeToString(cmd.secret)
  fmt.Println("SECRET: ", secret)
  return any(interfaces.BuildContractOutput{
    Secret:     secret,
    SecretHash: hex.EncodeToString(cmd.secretHash),
    TxFee:      strconv.FormatUint(data.contractFee,10),
    Contract:   hex.EncodeToString(data.contract),
    Tx:         hex.EncodeToString(data.contractTxHash),
  }),nil
}

func printTransaction(tx *externalapi.DomainTransaction, name string){
  if cfg.Verbose{
    printDomainTransaction(tx)
  }
  fmt.Printf("%v transaction (%v):\n", name, consensushashing.TransactionID(tx))
  txHash,err  := serialization.SerializeDomainTransaction(tx)
  if err != nil{
    log.Fatal("Impossible To deserialize %v Transaction:",name, tx)
  }
  fmt.Printf("%x\n\n", txHash)

}

func printCommand(secret []byte, secretHash []byte, b *builtContract){
  if secret != nil{
    fmt.Printf("Secret:      %x\n", secret)
  }
  fmt.Printf("Secret hash: %x\n\n", secretHash)
  fmt.Printf("Contract (%v):\n", b.contractP2SH)
  fmt.Printf("%x\n\n", b.contract)
  printTransaction(b.contractTx, "ContractTx")
  //printTransaction(b.refundTx,"RefundTx")

}

func (cmd *spendArgsCmd) runCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) error {
  out,_ := spendContract(mnemonics,daemonClient,ctx,keysFile,cmd)
  printTransaction(out.spendTx,"RedeemTx")
  return promptPublishTx(*out.spendTx, "redeem",daemonClient,ctx)
}

func (cmd *spendArgsCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File)(any, error) {
  data,err := spendContract(mnemonics,daemonClient,ctx,keysFile,cmd)
  if err != nil {
    return nil,err
  }
  return any(interfaces.SpendContractOutput{
        Tx:     hex.EncodeToString(data.spendTxHash),
        TxFee:  strconv.FormatUint(data.spendTxFee,10),
  }),nil
}

func (cmd *extractSecretCmd) runCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) error {
  return cmd.runOfflineCommand()
}

func (cmd *extractSecretCmd) runOfflineCommand() error {
  // Loop over all pushed data from all inputs, searching for one that hashes
  // to the expected hash.  By searching through all data pushes, we avoid any
  // issues that could be caused by the initiator redeeming the participant's
  // contract with some "nonstandard" or unrecognized transaction or script
  // type.
  for _, in := range cmd.redemptionTx.Inputs {
    pushes, err := txscript.PushedData(in.SignatureScript)
    if err != nil {
      return err
    }
    for _, push := range pushes {
      if bytes.Equal(sha256Hash(push), cmd.secretHash) {
        fmt.Printf("Secret: %x\n", push)
        return nil
      }
    }
  }
  return errors.New("transaction does not contain the secret")
}
func (cmd *extractSecretCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) error {
  return cmd.runOfflineCommand()
}

func (cmd *auditContractCmd) runCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) error {
  audited,err := auditContract(daemonClient, ctx,keysFile, *cmd)
  if err != nil {
    panic(err)
  }
  printAuditResult(*audited)
  return nil

}

func (cmd *auditContractCmd) runOfflineCommand() error {
  audited,err := auditContract(nil, nil, nil, *cmd)
  if err != nil {
    panic(err)
  }
  printAuditResult(*audited)
  return nil
}

func (cmd *auditContractCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) (any,error) {
  data,err := auditContract(daemonClient, ctx,keysFile, *cmd)
  if err != nil{
    fmt.Println("EEEEEEEEEEEEE",err)
    return nil,err
  }
  var rec *string
  if data.recipient != nil{
    r := (*data.recipient).String()
    rec = &r
  }else{
    rec = nil
  }
  var aut *string
  if data.author != nil{
    r := (*data.author).String()
    aut = &r
  }else{
    aut = nil
  }
  return any(interfaces.AuditContractOutput{
    ContractAddress:  fmt.Sprintf("%v",data.contractP2SH),
    RecipientAddress: *rec,
    //Recipient2b:      fmt.Sprintf("%x",data.recipient2b),
    Amount:           strconv.FormatUint(data.amount,10),
    RefundAddress:    *aut,
    //Refund2b:         fmt.Sprintf("%x",data.author2b),
    SecretHash:       fmt.Sprintf("%x",data.secretHash),
    LockTime:         fmt.Sprintf("%d",data.lockTime),
    TxId:             fmt.Sprintf("%x",data.txId),
    SecretLen:        strconv.FormatInt(data.secretSize,10),
    //TxFee:            strconv.FormatUint(data.txFee,10),
    //DaaScore:         fmt.Sprintf("%d",data.daaScore),
    //VSPBS:            fmt.Sprintf("%d",data.vspbs),
    IsSpendable:      strconv.FormatBool(data.isSpendable),
  }),err
}
func (cmd walletBalanceCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) (any,error) {
  balanceResponse,err := daemonClient.GetBalance(ctx, &pb.GetBalanceRequest{})
  balanceAddresses := []interfaces.AddressBalance{{}}
  for _,balance := range balanceResponse.AddressBalances{
    balanceAddresses = append(balanceAddresses, interfaces.AddressBalance{
      Address:    balance.Address,
      Available:  strconv.FormatUint(balance.Available,10),
      Pending:    strconv.FormatUint(balance.Pending,10),
    })
  }
  return any(interfaces.WalletBalanceOutput{
    Available:  strconv.FormatUint(balanceResponse.Available,10),
    Pending:    strconv.FormatUint(balanceResponse.Pending,10),
    AddressBalances: balanceAddresses,
  }),err


}
func (cmd atomicSwapParamsCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) (any,error) {
  changeAddrs:= getRawChangeAddress(daemonClient,ctx)
  return any(interfaces.AtomicSwapParamsOutput{
    ReciptAddress:          changeAddrs.String(),
    MaxSecretLen:           strconv.FormatInt(cfg.SecretSize,10),
    MinLockTimeInitiate:    strconv.FormatInt(cfg.LtInit,10),
    MinLockTimeParticipate: strconv.FormatInt(cfg.LtPart,10),
  }),nil
}

func (cmd checkRedeemCmd) runDaemonCommand(mnemonics []string, daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File) (any,error) {
  kaspadClient, _ := rpcclient.NewRPCClient(cfg.KaspadUrl)
  getBlocksR,_  := kaspadClient.GetBlocks(cmd.LastBlock,true,true)
  secretHash,err := hex.DecodeString(cmd.SecretHash)
  if err != nil {return nil,err}
  for _,block := range getBlocksR.Blocks{
    for _, transaction := range block.Transactions{
      for _, input := range transaction.Inputs{
        outpoint:= input.PreviousOutpoint
        if outpoint.TransactionID == cmd.TxId{
          redeemTx,err := appmessage.RPCTransactionToDomainTransaction(transaction)
          if err != nil {return nil,err}
          secret,_ := extractSecret(extractSecretCmd{redemptionTx:redeemTx,secretHash:secretHash})
          //if err != nil {return nil,err}
          return any(interfaces.CheckRedeemOutput{Secret:hex.EncodeToString(secret)}),nil
        }
      }
    }
  }
  return nil, errors.New("not found")
}
func getKaspaXSompi(value uint64) float64{
  return float64(value)/float64(constants.SompiPerKaspa)
}

func extractSecretFromSignature(signature []byte, secretHash []byte)([]byte, error){
  pushes, err := txscript.PushedData(signature)
  if err != nil {
    return nil,err
  }
  for _, push := range pushes {
    if bytes.Equal(sha256Hash(push), secretHash) {
      return push,nil
    }
  }
  return nil,nil
}
func extractSecret(args extractSecretCmd) ([]byte, error) {
  for _, in := range args.redemptionTx.Inputs {
    push,_ :=extractSecretFromSignature(in.SignatureScript,args.secretHash)
    if push!= nil{
      return push,nil
    }
  }
  return nil,errors.New("transaction does not contain the secret")

}
func auditContract(daemonClient pb.KaspawalletdClient, ctx context.Context, keysFile *keys.File,cmd auditContractCmd)(*auditedContract, error){
  var addresses []string
  if daemonClient != nil && ctx != nil {
    addresses = getAddresses(daemonClient,ctx)
  }
  contractP2SH, err := util.NewAddressScriptHash(cmd.contract, chainParams.Prefix)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("Impossible to determine contract hash: %v",err))
  }
  parsed,err := parsePushes(cmd.contract,addresses,keysFile)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("Impossible to  parse pushes: %v",err))
  }
  idx := getContractOut(cmd.contract,cmd.contractTx)
  if idx == nil{
    return nil, errors.New(fmt.Sprintf("Transaction does no contain contract"))
  }
  amount := cmd.contractTx.Outputs[*idx].Value
  txId := consensushashing.TransactionID(cmd.contractTx)
  var isSpendable bool
  var daaScore uint64
  var vspbs uint64
  var nUtxo int
  kaspadClient, _ := rpcclient.NewRPCClient(cfg.KaspadUrl)
  if kaspadClient != nil {
    getUTXOsByAddressesResponse,_  := kaspadClient.GetUTXOsByAddresses([]string{contractP2SH.EncodeAddress()})
    dagInfo, _ := kaspadClient.GetBlockDAGInfo()
    vspbs = dagInfo.VirtualDAAScore
    nUtxo = len(getUTXOsByAddressesResponse.Entries)
    for _, entry := range getUTXOsByAddressesResponse.Entries {
      if entry.Outpoint.TransactionID == txId.String(){
        isSpendable = isUTXOSpendable(entry, vspbs)
        daaScore = entry.UTXOEntry.BlockDAAScore
      }
    }
  }
  fmt.Println("locktime",parsed.lockTime)

  return &auditedContract{
  contract:     cmd.contract,
  contractTx:   cmd.contractTx,
  contractP2SH: contractP2SH,
  recipient:    parsed.recipientAddr,
  recipient2b:  parsed.recipientBlake2b,
  amount:       amount,
  author:       parsed.refundAddr,
  author2b:     parsed.refundBlake2b,
  secretHash:   parsed.secretHash,
  secretSize:   parsed.secretSize,
  lockTime:     parsed.lockTime,
  daaScore:     daaScore,
  txId:         *txId,
  idx:          *idx,
  nUtxo:        nUtxo,
  vspbs:        vspbs,
  isSpendable:  isSpendable,
  },nil
}

func  printAuditResult(audited auditedContract) error{
  if audited.secretSize != cfg.SecretSize {
    return fmt.Errorf("contract specifies strange secret size %v", audited.secretSize)
  }


  fmt.Printf("Contract address:           %v\n", audited.contractP2SH)
  fmt.Printf("Contract value:             %.8f\n", getKaspaXSompi(audited.contractTx.Outputs[audited.idx].Value))
  fmt.Printf("Recipient blake2b:          %x\n", audited.recipient2b)
  if audited.recipient != nil {
    fmt.Printf("Recipient address:        %v\n", audited.recipient)
  }
  fmt.Printf("Author's refund blake2b:    %x\n", audited.author2b)
  if audited.author != nil {
    fmt.Printf("Autor's refund address:   %v\n", audited.author)
  }
  fmt.Println("")
  fmt.Printf("Secret hash(len:%d):        %x\n", audited.secretSize, audited.secretHash)

  if audited.lockTime>= uint64(constants.LockTimeThreshold) {
    t := time.Unix(int64(audited.lockTime/1000), 0)
    fmt.Printf("Locktime: %v\n", t.UTC())
    reachedAt := time.Until(t).Truncate(time.Second)
    if reachedAt > 0 {
      fmt.Printf("Locktime reached in %v\n", reachedAt)
    } else {
      fmt.Printf("Contract refund time lock has expired\n")
    }
  } else {
    fmt.Printf("Locktime: block %v\n", audited.lockTime)
  }

  return nil
}



// atomicSwapContract returns an output script that may be redeemed by one of
// two signature scripts:
//
//   <their sig> <their pubkey> <initiator secret> 1
//
//   <my sig> <my pubkey> 0
//
// The first signature script is the normal redemption path done by the other
// party and requires the initiator's secret.  The second signature script is
// the refund path performed by us, but the refund can only be performed after
// locktime.
func atomicSwapContract(pkhMe, pkhThem []byte, locktime uint64, secretHash []byte, secretSize int64) ([]byte, error) {
  fmt.Println("LOCKTIME:",locktime)
  b := txscript.NewScriptBuilder()
  b.AddOp(txscript.OpIf) // Normal redeem path
  {
    // Require initiator's secret to be a known length that the redeeming
    // party can audit.  This is used to prevent fraud attacks between two
    // currencies that have different maximum data sizes.
    b.AddOp(txscript.OpSize)
    b.AddInt64(secretSize)
    b.AddOp(txscript.OpEqualVerify)

    // Require initiator's secret to be known to redeem the output.
    b.AddOp(txscript.OpSHA256)
    b.AddData(secretHash)
    b.AddOp(txscript.OpEqualVerify)

    // Verify their signature is being used to redeem the output.  This    
    // would normally end with OP_EQUALVERIFY OP_CHECKSIG but this has been
    // moved outside of the branch to save a couple bytes.
    b.AddOp(txscript.OpDup)
    b.AddOp(txscript.OpBlake2b)
    b.AddData(pkhThem)
  }
  b.AddOp(txscript.OpElse) // Refund path
  {
    // Verify locktime and drop it off the stack (which is not done by
    // CLTV).
    b.AddLockTimeNumber(locktime)
    b.AddOp(txscript.OpCheckLockTimeVerify)

    //removed as SomeOne235 commit in txscripts extractAtomicSwapDataPushes
    //    b.AddOp(txscript.OpDrop)

    // Verify our signature is being used to redeem the output.  This would
    // normally end with OP_EQUALVERIFY OP_CHECKSIG but this has been moved
    // outside of the branch to save a couple bytes.
    b.AddOp(txscript.OpDup)
    b.AddOp(txscript.OpBlake2b)
    b.AddData(pkhMe)
  }
  b.AddOp(txscript.OpEndIf)

  // Complete the signature check.
  b.AddOp(txscript.OpEqualVerify)
  b.AddOp(txscript.OpCheckSig)
  return b.Script()
}

func redeemP2SHContract(contract, sig, pubkey, secret []byte) ([]byte, error) {
  b := txscript.NewScriptBuilder()

  b.AddData(sig)
  b.AddData(pubkey)
  b.AddData(secret)
  b.AddInt64(1)
  b.AddData(contract)
  return b.Script()
}

func refundP2SHContract(contract, sig, pubkey []byte) ([]byte, error) {
  b := txscript.NewScriptBuilder()
  b.AddData(sig)
  b.AddData(pubkey)
  b.AddInt64(0)
  b.AddData(contract)
  return b.Script()
}

func domainTransactionFromStringHash(sTx string) (*externalapi.DomainTransaction,error){
    fmt.Println("debug domaintransactionstring:",sTx)
    txBytes, err := hex.DecodeString(sTx)
    if err != nil {
      return nil, errors.New(fmt.Sprintf("failed to decode contract transaction: %v\n%v", err,sTx))
    }
    return serialization.DeserializeDomainTransaction(txBytes)

}

func addressFromString(address string) (*util.AddressPublicKey,error){
  cp2Addr, err := util.DecodeAddress(address, chainParams.Prefix)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to decode participant address: %v\naddress: %v", err,address))
  }
  if !cp2Addr.IsForPrefix(chainParams.Prefix) {
    return nil,errors.New(fmt.Sprintf("participant address is not "+
      "intended for use on %v", chainParams.Name))
  }
  cp2AddrP2PKH, ok := cp2Addr.(*util.AddressPublicKey)
  if !ok {
    return nil, errors.New(fmt.Sprintf("participant address is not P2PKH"))
  }
  return cp2AddrP2PKH, nil
}

func amountFromString(samount string)(*uint64, error){
  amountF64, err := strconv.ParseFloat(samount, 64)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to decode amount: %v", err))
  }
  amount := uint64(amountF64)* uint64(constants.SompiPerKaspa)
  return &amount,nil
}

func contractFromString(scontract string)([]byte,error){
  contract, err := hex.DecodeString(scontract)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to decode contract: %v", err))
  }
  return contract,nil
}

func secretFromString(ssecret *string)([]byte,error){
  if ssecret == nil || *ssecret == ""{
    return nil,nil
  }
  secret, err := hex.DecodeString(*ssecret)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to decode secret: %v", err))
  }
  return secret,nil
}

func secretHashFromString(ssecretHash *string)([]byte,error){
  secretHash, err := hex.DecodeString(*ssecretHash)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to decode secretHash: %v", err))
  }
  if len(secretHash) != sha256.Size {
    return nil, errors.New("secret hash has wrong size")
  }
  return secretHash,nil
}

func parseBuildArgs(args interfaces.BuildContractInput)(*contractArgsCmd,error){
  cp2AddrP2PKH,err := addressFromString(args.Them)
  if err != nil {
    return nil, err
  }
  amount,err := amountFromString(args.Amount)
  if err != nil {
    return nil, err
  }
  var secret []byte
  var secretHash []byte
  var lockTime uint64
  slen:= int64(0)
  if args.SecretHash == nil || *args.SecretHash == ""{
//////////////
    slen, err = strconv.ParseInt(*args.SecretLen,10,64)
    if err != nil{
      return nil,err
    }
    secret,secretHash,_ = getSecret(&slen)
    if args.LockTime == nil{
      lockTime = uint64(time.Now().Add(time.Duration(cfg.LtInit) * time.Hour).Unix())
    }else{
      lockTime,err = strconv.ParseUint(*args.LockTime,10,64)
      if err != nil{
        return nil,err
      }
    }
  }else{
    secret = nil
    secretHash,err = secretHashFromString(args.SecretHash)
    if err != nil{
      return nil, err
    }
    lockTime = uint64(time.Now().Add(time.Duration(cfg.LtPart) * time.Hour).Unix()*1000)
  }
  return &contractArgsCmd{
    them:       cp2AddrP2PKH,
    amount:     *amount,
    secretHash: secretHash,
    secret:     secret,
    secretSize: slen,
    locktime:   lockTime,
  },nil
}

func parseSpendArgs(args interfaces.SpendContractInput)(*spendArgsCmd, error){
    contract, err := contractFromString(args.Contract)
    if err!=nil{
      return nil, err
    }
    contractTx, err := domainTransactionFromStringHash(args.Tx)
    if err!=nil{
      return nil, err
    }
    secret, err := secretFromString(&args.Secret)
    if err!=nil{
      return nil, err
    }

    return &spendArgsCmd{
      contract:   contract,
      contractTx: contractTx,
      secret:     secret,
    }, nil
}

func parseExtractSecretArgs(args interfaces.ExtractSecretInput) (*extractSecretCmd,error){
  secretHash, err := secretHashFromString(&args.SecretHash)
  if err != nil {
    return nil, err
  }
  redeemTx, err := domainTransactionFromStringHash(args.Tx)
  if err !=nil{
    return nil, err
  }
  return &extractSecretCmd{
    redemptionTx: redeemTx,
    secretHash:   secretHash,
  },nil
}

func parseAuditContractArgs(args interfaces.AuditContractInput) (*auditContractCmd,error){
  contractTx, err := domainTransactionFromStringHash(args.Tx)
  if err !=nil{
    return nil, errors.New(fmt.Sprintf("failed to decode redeemTx: %v\n%v\n", err,args.Tx))
  }
  contract, err := contractFromString(args.Contract)
  return &auditContractCmd{
    contract:   contract,
    contractTx: contractTx,
  },nil
}

// Define a handler for each endpoint
func restApiRequestsHandlers() {
  http.HandleFunc("/is-online", isOnlineEndpoint)
  http.HandleFunc("/initiate", initiateEnpoint)
  http.HandleFunc("/participate", participateEndpoint)
  http.HandleFunc("/redeem", redeemEndpoint)
  http.HandleFunc("/refund", refundEndpoint)
  http.HandleFunc("/auditcontract", auditEndpoint)
  http.HandleFunc("/extractsecret", extractSecretEndpoint)
  http.HandleFunc("/pushtx",pushTxEndpoint)
  http.HandleFunc("/walletbalance",walletBalanceEndpoint)
  http.HandleFunc("/newswap",atomicSwapParamsEndpoint)
  http.HandleFunc("/check",checkRedeemEndpoint)
}

// Check if BTC / KAS network are available
func isOnlineEndpoint(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  out := "false"
  if isOnline() {
    out = "true"
  }
  interfaces.WriteResult(w,nil,interfaces.IsOnlineOutput{
    IsOnline: out,
  })
}

// Check if BTC / KAS network are available

// Dummy function  - To be replaced
//TODO check kaspad & kaspawallet daemons
func isOnline() bool {
  return true
}
func pushTx(input interfaces.PushTxInput) (*string, error){
  tx,err := domainTransactionFromStringHash(input.Tx)
  if err != nil {
    return nil,err
  }
  return sendRawTransaction(*tx)
}
func getNil(s *string) string{
  if s == nil {
    return ""
  }else{
    return *s
  }
}
func pushTxEndpoint(w http.ResponseWriter, r *http.Request) {
  var args interfaces.PushTxInput
  interfaces.ParseBody(r,&args)
  txId,err := pushTx(args)
  interfaces.WriteResult(w,err,interfaces.PushTxOutput{
    TxId: fmt.Sprintf("%v",getNil(txId)),
  })
}
func mainEndPoint(cmd daemonCommand,err error, w http.ResponseWriter, r *http.Request){
  if err!=nil {
    fmt.Println(err)
    interfaces.WriteResult(w,err,nil)
    return
  }
  daemonClient, tearDown, err := client.Connect(cfg.KaspaWalletUrl)
  if err!=nil {
    fmt.Println(err)
    interfaces.WriteResult(w,errors.New("error"),nil)
    return
  }
  defer tearDown()
  ctx, cancel := context.WithTimeout(context.Background(), (10 * time.Minute))
  defer cancel()

  keysFile, err := keys.ReadKeysFile(chainParams, defaultKeysFile(chainParams))
  if err!=nil {
    fmt.Println(err)
    interfaces.WriteResult(w,errors.New("error"),nil)
    return
  }
  mnemonics, err := keysFile.DecryptMnemonics(daemonPassword)
  if err!=nil {
    fmt.Println(err)
    interfaces.WriteResult(w,errors.New("error"),nil)
    return
  }

  out, err := cmd.runDaemonCommand(mnemonics,daemonClient,ctx,keysFile)
  interfaces.WriteResult(w,err,out)
}
// Initiate swap contract
func initiateEnpoint(w http.ResponseWriter, r *http.Request) {
  buildSwapContractEndpoint(w,r)
}
func participateEndpoint(w http.ResponseWriter,r *http.Request){
  buildSwapContractEndpoint(w,r)
}

func redeemEndpoint(w http.ResponseWriter,r *http.Request){
  spendSwapContractEndpoint(w,r)
}

func refundEndpoint(w http.ResponseWriter,r *http.Request){
  spendSwapContractEndpoint(w,r)
}

func auditEndpoint(w http.ResponseWriter,r *http.Request){
  var  args interfaces.AuditContractInput
  interfaces.ParseBody(r,&args)
  input,err := parseAuditContractArgs(args)
  mainEndPoint(input,err,w,r)
}

func extractSecretEndpoint(w http.ResponseWriter,r *http.Request){
  var args interfaces.ExtractSecretInput
  interfaces.ParseBody(r,&args)
  input,err := parseExtractSecretArgs(args)
  if err!=nil {
    fmt.Println(err)
    interfaces.WriteResult(w,err,nil)
    return
  }
  data,err := extractSecret(*input)
  interfaces.WriteResult(w,err,interfaces.ExtractSecretOutput{Secret:fmt.Sprintf("%x",data)})
}

func buildSwapContractEndpoint(w http.ResponseWriter, r *http.Request) {
  var  args interfaces.BuildContractInput
  interfaces.ParseBody(r,&args)
  buildArgs,err := parseBuildArgs(args)
  mainEndPoint(buildArgs,err,w,r)
}

func spendSwapContractEndpoint(w http.ResponseWriter, r *http.Request) {
  var args interfaces.SpendContractInput
  interfaces.ParseBody(r,&args)
  input, err := parseSpendArgs(args)
  mainEndPoint(input,err,w,r)
}
func atomicSwapParamsEndpoint(w http.ResponseWriter, r *http.Request) {
  mainEndPoint(atomicSwapParamsCmd{},nil,w,r)
}
func walletBalanceEndpoint(w http.ResponseWriter, r *http.Request){
  mainEndPoint(walletBalanceCmd{},nil,w,r)
}
func checkRedeemEndpoint(w http.ResponseWriter, r *http.Request) {
  var args interfaces.CheckRedeemInput
  interfaces.ParseBody(r,&args)
  input:=checkRedeemCmd{LastBlock:args.LastBlock,TxId:args.TxId,SecretHash:args.SecretHash}
  mainEndPoint(input,nil,w,r)
}
