package main
import (
  "time"
  //"net"
  "fmt"
  //"flag"
  "encoding/json"
  "strings"
  "net/http"
  ///"io/ioutil"
  "errors"
  "strconv"
  //"os"
  //"database/sql"
  "github.com/samonzeweb/godb"
  "github.com/samonzeweb/godb/adapters/sqlite"
  "crypto/rand"
  "encoding/hex"
  "github.com/pieroforfora/atomicswapper/interfaces"
  "github.com/pieroforfora/atomicswapper/lib/tagconfig"
  "math"
  "os"
  "context"

)
type Config struct{
  ListenSwapperMaker  string   `env:"LISTEN_MAKER"  cli:"listen-maker"  yaml:"listen_maker" default:":9220"`
  ListenSwapperTaker  string   `env:"LISTEN_TAKER"  cli:"listen-taker"  yaml:"listen_taker" default:"127.0.0.1:9221"`
  DatabaseFile        string   `env:"DATABASE_FILE" cli:"database-file" yaml:"database_file"  default:"swap.db"`
}
var cfg Config

const SWAP_STATUS_REQUESTED =     "0"
const SWAP_STATUS_INITIATED =     "200"
const SWAP_STATUS_PARTICIPATED =  "300"
const SWAP_STATUS_REDEEMED =      "400"
const SWAP_STATUS_REFUNDED =      "500"

const SWAP_ROLE_SWAPPER = "swp"
const SWAP_ROLE_USER =    "usr"
var db *godb.DB
/*
var book =  []byte(strings.ToUpper(`{
    "BTC":{
      "KAS":{
        "BUY":  [[0.0000004,100,200],[0.00000051,201,300],[0.00000080]],
        "SELL": [[0.0000005,100,200],[0.00000053,201,300],[0.00000090]]
      },
      "MONA":{
        "BUY":  [[5,100,200],[5.1,201,300],[8]]
      }
    },
    "LTC":{
      "KAS":{
        "BUY":  [[0.0005,100,200],[0.00051,201,300],[0.00080]],
        "SELL": [[0.0005,100,200],[0.00051,201,300],[0.00080]]
      }
    }
}`))
*/
var limitVolumesPerTrade = []byte(`{
  BTC:  [0.00001000,10],
  KAS:  [1,1000000],
  MONA: [0,0],
  LTC:  [0.00001]
}`)
type orderBook map[string]map[string]map[string][][]float64
var markets orderBook
/*
var limitVolumePerTime = []byte(`{
  "KAS":[3600],
  "MONA:[600],
  "LTC:[10000]
}`)
*/
var walletTotalBalance = []byte(`{
  BTC:       10,
  KAS:  1000000,
  MONA:    1000,
  LTC:        0
}`)
var networksJson = []byte(`{
  "BTC":{ 
    "url":"localhost:8080"
  },
  "KAS":{ 
    "url":"localhost:8081"
  }
}`)


type NetworkStatus struct {
  Url           string  `json:"url"`
  Mediantime    int64
  StatusCode    int
  StatusMessage string
}

var networks map[string]NetworkStatus
func checkStatus(n string) bool{
  if networks[n].StatusCode != 200{
    fmt.Printf("Atomic swap for %v is not available\n%v\n",n,networks[n].Url)
    return false
  }
  return true
}
func createServerMaker() error{
  mux := http.NewServeMux()
  mux.HandleFunc("/is-online", isOnlineEndpoint)
  mux.HandleFunc("/initiate", initiateEndpoint)
  mux.HandleFunc("/participate", participateEndpoint)
  mux.HandleFunc("/check", checkEndpoint)
  server := http.Server{
    Addr: cfg.ListenSwapperMaker,
    Handler: mux,
  }
  return server.ListenAndServe()
}
func createServerTaker() error {
  mux := http.NewServeMux()
  mux.HandleFunc("/swap", swapEndpoint)
  mux.HandleFunc("/confirmSwap",confirmSwapEndpoint)

  server := http.Server{
    Addr: cfg.ListenSwapperTaker,
    Handler: mux,
  }
  return server.ListenAndServe()
}
func updateMarkets(){
  for{
    book, err := os.ReadFile("book.json")
    if err != nil {
      fmt.Println("Errors while reading book.json file:", err)
      continue
    }
    err = json.Unmarshal(book,&markets)
    if err != nil{
      fmt.Println("Error while parsing book.json:",err)
      continue
    }
    time.Sleep(5 * time.Second)
  }
}
func updateNetworks(){
  for{
    //var networks  map[string]NetworkStatus
    err:= json.Unmarshal(networksJson,&networks)
    panicIfErr(err)
    for name, net := range networks{
      status := NetworkStatus{Url:net.Url}
      isOnline,err := interfaces.IsOnline(net.Url)
      if err!= nil{
        status.StatusCode = 0
        status.StatusMessage = "no reply from server"
        continue
      }else{
        if isOnline.IsOnline == "true"{
          status.StatusCode = 200
          status.StatusMessage = "ok"
        }else{
          status.StatusCode = 201
          status.StatusMessage = "Offline"
        }
      }
      networks[name]=status
    }
    for idx,network := range networks{
      if network.StatusCode!=200{
        fmt.Println("Error atomic swap offline for network:", idx,network.StatusCode)
      }
    }
    time.Sleep(5 * time.Second)
  }

}

func checkSwapInitiate(){
  for{
    swaps := make([]Swap, 0, 0)
    err := db.Select(&swaps).Where("status_code = ?",SWAP_STATUS_INITIATED).Do()
    panicIfErr(err)
    for _, swap := range swaps {
  audit,err := auditContract(networks[swap.CurrencyToSwapper].Url,swap.ContractInit,swap.TxInit,swap.MaxSecretLen,swap.MinLockTimeInit,swap.AddressSwapper)
      if err != nil {
        fmt.Println("Error Auditing Contract:",err)
        continue
      }
      if audit.IsSpendable == "true" {
        fmt.Println(swap.Id)
        initiateLockTime,err := strconv.ParseInt(audit.LockTime,10,64)
        panicIfErr(err)
        partLockTime := strconv.FormatInt(int64(time.Now().Unix()+(initiateLockTime-time.Now().Unix())/2),10)
        part, err := interfaces.Participate(networks[swap.CurrencyToUser].Url,interfaces.BuildContractInput{
          Them:         swap.AddressUser,
          Amount:       swap.AmountToUser,
          SecretHash:   &audit.SecretHash,
          LockTime:     &partLockTime,
        })
        if err != nil {
          fmt.Println("Error Participating:",err)
          continue
        }
        pushTx,err := interfaces.PushTx(networks[swap.CurrencyToUser].Url,interfaces.PushTxInput{Tx:part.Tx})
        if err != nil {
          fmt.Println(err)
          continue
        }
        fmt.Println("participated")
        fmt.Println("TX:", part.Tx)
        fmt.Println("TXId:", pushTx.TxId)
        fmt.Println("Contract:", part.Contract)
        fmt.Println("LastBlock:", part.LastBlock)
        swap.TxPart = part.Tx
        swap.TxIdPart = part.TxId
        swap.ContractPart = part.Contract
        swap.AddressContractPart = part.ContractAddress
        swap.LastBlock = part.LastBlock
        swap.StatusCode = SWAP_STATUS_PARTICIPATED
        err = db.Update(&swap).Do()
        panicIfErr(err)
      }
    }

    time.Sleep(5 * time.Second)
  }
}

func checkSwapRedeem(){
  for{
    swaps := make([]Swap, 0, 0)
    err := db.Select(&swaps).Where("status_code = ?",SWAP_STATUS_PARTICIPATED).Do()
    panicIfErr(err)
    for _, swap := range swaps {
      var checkOut *interfaces.CheckRedeemOutput
      if swap.TxRedeemUser == ""{
        checkOut,err = interfaces.SearchRedeem(networks[swap.CurrencyToUser].Url, interfaces.CheckRedeemInput{
          LastBlock:  swap.LastBlock,
          TxId:       swap.TxIdPart,
          SecretHash: swap.SecretHash,
        })
        if err != nil {
          fmt.Println(err)
          continue
        }
        if checkOut == nil{
          continue
        }
        fmt.Println("checkOut:",checkOut)
        swap.TxRedeemUser   = checkOut.Tx
        swap.TxIdRedeemUser = checkOut.TxId
        swap.Secret         = checkOut.Secret
        err = db.Update(&swap).Do()
      }
      redeemOut,err := interfaces.Redeem(networks[swap.CurrencyToSwapper].Url, interfaces.SpendContractInput{
          Secret:   swap.Secret,
          Contract: swap.ContractInit,
          Tx:       swap.TxInit,
      })
      if err != nil {
        fmt.Println(err)
        continue
      }
      fmt.Println("pushTx: ",redeemOut.Tx)
      pushTx,err := interfaces.PushTx(networks[swap.CurrencyToSwapper].Url, interfaces.PushTxInput{Tx:redeemOut.Tx})
      if err!= nil{
        fmt.Println(err)
        continue
      }
      swap.TxRedeemSwapper = redeemOut.Tx
      swap.TxIdRedeemSwapper = pushTx.TxId
      swap.StatusCode = SWAP_STATUS_REDEEMED
      err = db.Update(&swap).Do()
      panicIfErr(err)
      break
    }
    time.Sleep(5 * time.Second)
  }
}

func checkSwapTimedOut(){
  for{
    swaps := make([]Swap, 0, 0)
    err := db.Select(&swaps).Where("status_code = ?",SWAP_STATUS_PARTICIPATED).Do()
    panicIfErr(err)
    for _, swap := range swaps {
      if swap.TxRedeemSwapper ==""{
        if swap.LocktimePart < networks[swap.CurrencyToUser].Mediantime{
          refundOut, err := interfaces.Refund(networks[swap.CurrencyToUser].Url, interfaces.SpendContractInput{
            Tx:         swap.TxPart,
            Contract:   swap.ContractPart,
          })
          if err != nil{
            fmt.Println(err)
            continue
          }
          pushedTx,err := interfaces.PushTx(networks[swap.CurrencyToUser].Url, interfaces.PushTxInput{Tx:refundOut.Tx})
          if err != nil{
            fmt.Println(err)
            continue
          }
          swap.TxRefundSwapper = refundOut.Tx
          swap.TxIdRefundSwapper = pushedTx.TxId
        }
      }
    }
    time.Sleep(5 * time.Second)
  }
}
/*
func askToContinue(txt string) bool{
  fmt.Println(txt, "[y/N]")
  for {
    answer, err := reader.ReadString('\n')
    if err != nil {
      return err
    }
    answer = strings.TrimSpace(strings.ToLower(answer))

    switch answer {
    case "y", "yes":
      return true
    case "n", "no", "":
      return false
    default:
      fmt.Println("please answer y or n")
      continue
    }

  }
}
*/
func runMaker(){
      go updateMarkets()
      go createServerMaker()
}


func executeSwap(args []string)(error){

  argsSwapIn := interfaces.InitiateSwapIn{
    Market:   args[2],
    Address:  args[3],
    Amount:   args[4],
  }
  swapperUrl := args[1]

  response,err := interfaces.Post(swapperUrl, &argsSwapIn)
  if err != nil{return err}
  fmt.Println(response)
  var initiateSwap interfaces.InitiateSwapOut
  interfaces.ParseBodyResponse(response,&initiateSwap)
  fmt.Println("Initiate Swap")
  fmt.Println(initiateSwap)
  fmt.Println("------------")
  maxlen,err := strconv.Atoi(initiateSwap.MaxSecretLen)
  panicIfErr(err)
  minlocktime,err := strconv.ParseInt(initiateSwap.MinLockTimeInitiate,10,64)
  panicIfErr(err)
  swap := Swap {
    Id:                   initiateSwap.SwapId,
    Date:                 time.Now().Unix(),
    StatusCode:           "0",
    MyRole:               "usr",
    CurrencyToUser:       initiateSwap.CurrencyToUser,
    CurrencyToSwapper:    initiateSwap.CurrencyToSwapper,
    AmountToSwapper:      initiateSwap.AmountToSwapper,
    AmountToUser:         initiateSwap.AmountToUser,
    AddressSwapper:       initiateSwap.SwapperAddress,
    AddressUser:          initiateSwap.UserAddress,
    MaxSecretLen:         maxlen,
    MinLockTimeInit:      minlocktime,
    SwapperUrl:           swapperUrl,
  }
  err = db.Insert(&swap).Do()
  panicIfErr(err)
  confirmSwapRequested(swap.Id)
  return nil
}

func confirmSwapRequested(id string)(error){
  swap := Swap {}
  err := db.Select(&swap).Where("id= ?",id).Do()
  if err != nil {
      return err
  }
  minlocktime := strconv.FormatInt(swap.MinLockTimeInit,64)
  maxlen := strconv.Itoa(swap.MaxSecretLen)
  if swap.StatusCode == SWAP_STATUS_REQUESTED{
    err = db.Insert(&swap).Do()
    panicIfErr(err)
    params := interfaces.BuildContractInput{
      Them:       swap.AddressSwapper,
      Amount:     swap.AmountToSwapper,
      LockTime:   &minlocktime,
      SecretLen:  &maxlen,
    }
    builtSwap, err := interfaces.Initiate(networks[swap.CurrencyToSwapper].Url,params)
    if err != nil{return err}
    fmt.Println(builtSwap)
    fmt.Printf("Secret:%v\nSecretHash:%v\nContract:%v\nTx:%v\nTxFee:%v",
      builtSwap.Secret,
      builtSwap.SecretHash,
      builtSwap.Contract,
      builtSwap.Tx,
      builtSwap.TxFee,
    )
    swap.Secret=builtSwap.Secret
    swap.SecretHash = builtSwap.SecretHash
    swap.ContractInit = builtSwap.Contract
    swap.TxInit = builtSwap.Tx
    db.Update(&swap).Do()
  }
  return nil
}

func confirmPushInitiateTransaction(id string)(error){
  swap := Swap{}
  db.Select(&swap).Where("id = ?", id)
  _,err := interfaces.PushTx(networks[swap.CurrencyToSwapper].Url,interfaces.PushTxInput{Tx:swap.TxInit})
  if err != nil {return err}
  done := interfaces.ParticipateIn{
    SwapId:       swap.Id,
    ContractTx:   swap.TxInit,
    Contract:     swap.ContractInit,
    SecretHash:   swap.SecretHash,
  }
  _,err = interfaces.Post(swap.SwapperUrl + "/participate",&done)
  if err != nil {return err}
  fmt.Println("waiting for swapper to participate")
  return nil
}
func waitParticipateAndRedeem()(error){
  for{
    swaps := make([]Swap,0,0)
    db.Select(&swaps).Where("status_code = ?", SWAP_STATUS_INITIATED)
    for _,swap := range swaps {
      response, err := interfaces.Post(swap.SwapperUrl + "/check", &interfaces.SwapStatusIn{
        SwapId: swap.Id,
      })
      if err != nil { return err }
      var swapStatus interfaces.SwapStatusOut
      interfaces.ParseBodyResponse(response,&swapStatus)
      if swapStatus.StatusCode== SWAP_STATUS_PARTICIPATED {
        //response,err = interfaces.Post("http://localhost:8080/auditcontract",&interfaces.PushTxInput{Tx:builtSwap.Tx})
        audit,err := auditContract(networks[swap.CurrencyToUser].Url,swap.ContractPart,swap.TxPart,swap.MaxSecretLen,swap.MinLockTimeInit,swap.AddressUser)
        if err!= nil {fmt.Println(err)}
        if audit.IsSpendable == "true" {
          fmt.Println("Participate is spendable")
          redeemOut,err := interfaces.Redeem(networks[swap.CurrencyToUser].Url,interfaces.SpendContractInput{
            Secret:   swap.Secret,
            Contract: swapStatus.ContractPart,
            Tx:       swapStatus.TxPart,
          })
          if err != nil {
            fmt.Println("RedeemError:",err)
            continue
          }
          fmt.Println(response,redeemOut)
          fmt.Println("pushTx: ",redeemOut.Tx)
          fmt.Println("pushTx: ",redeemOut.TxId)
          _,err = interfaces.PushTx(networks[swap.CurrencyToUser].Url,interfaces.PushTxInput{Tx:redeemOut.Tx})
          fmt.Println("response:",response)
          fmt.Println("error:",err)
          swap.TxRedeemUser = redeemOut.Tx
          swap.TxIdRedeemUser = redeemOut.TxId
          db.Update(&swap).Do()
        

          break
        }else{
          fmt.Println("participate is not spendable")
        }
      }else{
        fmt.Println("status code is not 300: ", swapStatus.StatusCode )
      }
    }
    time.Sleep(60* time.Second)
  }
}


func main(){
  ctx := context.Background()
  cmdArgs,err := tagconfig.Parse(&cfg,ctx)
  panicIfErr(err)

  fmt.Printf("opening database file:%v\n",cfg.DatabaseFile)
  ddb, err := godb.Open(sqlite.Adapter, cfg.DatabaseFile)
  panicIfErr(err)
  // OPTIONAL: Set logger to show SQL execution logs
  //ddb.SetLogger(log.New(os.Stderr, "", 0))
  db = ddb

  go updateNetworks()
  go checkSwapInitiate()
  go checkSwapRedeem()
  go checkSwapTimedOut()
  if len(cmdArgs) > 0{
    switch cmdArgs[0]{
      case "maker":
        runMaker()
      case "take":
        executeSwap(cmdArgs)
      case "taker":
        createServerTaker()
      default:
        runMaker()
    }
  }

  //restApiHandlers()
  done := make(chan struct{})
  <-done
}

func getPrice(currencyFrom, currencyTo string, volume float64, markets orderBook)float64{
    if volume <0{
      t:=currencyFrom
      currencyFrom=currencyTo
      currencyTo=t
    }
    price := getPriceF1(currencyFrom, currencyTo, volume,true,markets)
    if price == 0 {
      price = getPriceF1(currencyTo,currencyFrom,volume,false,markets)
      if price == 0{
        price = getPriceF2(currencyTo,currencyFrom,volume,markets)
      }
    }
    return roundFloat(price,8)
}

func getPriceByVolume(prices  [][]float64, volume float64) float64{
  realvolume := volume
  for _,p := range prices{
    reverse :=false
    if volume <0{
      realvolume =volume*(-1)/p[0]
      reverse = true
    }
    if len(p) ==1{
      if reverse{ return 1/p[0]}
      return p[0]
    }
    if len(p) == 2 && realvolume > p[1]{
      if reverse{ return 1/p[0]}
      return p[0]
    }
    if len(p)== 3 && realvolume > p[1] && realvolume< p[2]{
      if reverse{ return 1/p[0]}
      return p[0]
    }
  }
  return 0
}

func getPriceF1(currencyFrom, currencyTo string, volume float64, isBuy bool, markets orderBook)float64{
  if prices,ok := markets[currencyFrom][currencyTo]; ok {
    if isBuy{
        price := getPriceByVolume(prices["SELL"],volume)
        if price > 0{
          return 1/price
        }
        return 0
    }else{
      return getPriceByVolume(prices["BUY"],volume)
    }
  }
  return 0
}

func getPriceF2(currencyFrom, currencyTo string, volume float64, markets orderBook)float64{
  for _, baseCurrency := range markets{
    if pricesFrom, ok := baseCurrency[currencyFrom]; ok {
      if pricesTo, ok := baseCurrency[currencyTo]; ok{
        priceFrom := getPriceByVolume(pricesFrom["SELL"], volume)
        if priceFrom > 0{
          priceTo := getPriceByVolume(pricesTo["BUY"], volume)
          if priceTo > 0{
            return priceFrom/priceTo
          }
        }
      }
    }
  }
  return 0
}

func restApiHandlers(){
  http.HandleFunc("/is-online", isOnlineEndpoint)
  http.HandleFunc("/initiate", initiateEndpoint)
  http.HandleFunc("/participate", participateEndpoint)
  http.HandleFunc("/check", checkEndpoint)
  http.HandleFunc("/swap", swapEndpoint)
}

func swapEndpoint(w http.ResponseWriter, r *http.Request){
  var args interfaces.SwapIn
  interfaces.ParseBody(r,&args)
}
func confirmSwapEndpoint(w http.ResponseWriter, r *http.Request){

}
func checkEndpoint(w http.ResponseWriter, r *http.Request){
  var args interfaces.SwapStatusIn
  interfaces.ParseBody(r,&args)
  swap := Swap{}
  err := db.Select(&swap).Where("id = ?",args.SwapId).Do()
  fmt.Println(err)
  fmt.Println(swap.Id)
  interfaces.WriteResult(w,err,interfaces.SwapStatusOut{
    Id:                   swap.Id,
    Date:                 swap.Date,
    StatusCode:           swap.StatusCode,
    MyRole:               swap.MyRole,
    RemoteUrl:            swap.RemoteUrl,
    CurrencyToUser:       swap.CurrencyToUser,
    CurrencyToSwapper:    swap.CurrencyToSwapper,
    AmountToSwapper:      swap.AmountToSwapper,
    AmountToUser:         swap.AmountToUser,
    AddressSwapper:       swap.AddressSwapper,
    AddressUser:          swap.AddressUser,
    AddressContractInit:  swap.AddressContractInit,
    AddressContractPart:  swap.AddressContractPart,
    MaxSecretLen:         swap.MaxSecretLen,
    MinLockTimeInit:      swap.MinLockTimeInit,
    SecretHash:           swap.SecretHash,
    ContractInit:         swap.ContractInit,
    ContractPart:         swap.ContractPart,
    TxInit:               swap.TxInit,
    TxPart:               swap.TxPart,
    TxRedeemUser:         swap.TxRedeemUser,
    TxRedeemSwapper:      swap.TxRedeemSwapper,
    TxRefundUser:         swap.TxRefundUser,
    TxRefundSwapper:      swap.TxRefundSwapper,
  })
}

func isOnlineEndpoint(w http.ResponseWriter, r *http.Request){
  isOnlineOut := make([]interfaces.IsOnlineOut,len(networks))
  i:=0
  for idx,network:=range networks {
    isOnlineOut[i]=interfaces.IsOnlineOut{
      Network:        idx,
      StatusCode:     strconv.Itoa(network.StatusCode),
      StatusMessage:  network.StatusMessage,
    }
    i+=1
  }
  interfaces.WriteResult(w,nil,isOnlineOut)
}
//TODO still missing to check balances
func parseMarket(market string) (string,string,error){
  pair := strings.Split(strings.ToUpper(market), "-")
  if !checkStatus(pair[0]) || !checkStatus(pair[1]) {
    fmt.Printf("atomicswap  is offline %v:%v - %v:%v",pair[0],networks[pair[0]].StatusCode,pair[1],networks[pair[1]].StatusCode)
    return "","", errors.New(fmt.Sprintf("atomic swap is offline %v:%v - %v:%v",pair[0],networks[pair[0]].StatusCode,pair[1],networks[pair[1]].StatusCode))
  }
  return pair[0],pair[1],nil
}
func parseInitiateSwap(args interfaces.InitiateSwapIn)(*string,*string,*float64, *float64, error){
   amountF64, err := strconv.ParseFloat(args.Amount, 64)
  if err != nil {
    fmt.Printf("failed to decode amount: %v", err)
    return nil,nil,nil,nil, errors.New(fmt.Sprintf("falied to decode amount(%v) %v",args.Amount, err))
  }
  currencyToSwapper,currencyToUser,err := parseMarket(args.Market)
  
  priceF64 := getPrice(currencyToSwapper,currencyToUser,amountF64, markets)

  if amountF64 < 0 {
    temp := currencyToUser
    currencyToUser = currencyToSwapper
    currencyToSwapper = temp
    amountF64 = amountF64 * -1
    priceF64 = 1/priceF64
  }

return &currencyToUser, &currencyToSwapper, &priceF64, &amountF64,nil
}

func initiateEndpoint(w http.ResponseWriter, r *http.Request){
  var args interfaces.InitiateSwapIn
  interfaces.ParseBody(r,&args)
  currencyToUser,currencyToSwapper,priceF64,amountF64,err := parseInitiateSwap(args)
  if err!=nil{
    fmt.Println("errore:",err)
    interfaces.WriteResult(w,err,nil)
    return

  }
  fmt.Println(networks[*currencyToSwapper].Url)
  atomicSwapParams,err := interfaces.SwapParams(networks[*currencyToSwapper].Url,true)
  if err!=nil{
    fmt.Println(err)
    interfaces.WriteResult(w,err,nil)
    return
  }

  swapId :=make([]byte,31)
  _, err = rand.Read(swapId[:])
  panicIfErr(err)
  maxlen,err := strconv.Atoi(atomicSwapParams.MaxSecretLen)
  if err!=nil{
    fmt.Println("errore:",err)
    interfaces.WriteResult(w,err,nil)
    return
  }
  mintime,err := strconv.ParseInt(atomicSwapParams.MinLockTimeInitiate, 10, 64)
  if err!=nil{
    fmt.Println("errore:",err)
    interfaces.WriteResult(w,err,nil)
    return
  }
  swap := Swap {
    Id:                   hex.EncodeToString(swapId),
    Date:                 time.Now().Unix(),
    StatusCode:           "0",
    MyRole:               "swp",
    CurrencyToUser:       *currencyToUser,
    CurrencyToSwapper:    *currencyToSwapper,
    AmountToSwapper:      strconv.FormatFloat(*amountF64,'f',8,64),
    AmountToUser:         strconv.FormatFloat(*amountF64**priceF64,'f',8,64),
    AddressSwapper:       atomicSwapParams.ReciptAddress,
    AddressUser:          args.Address,
    MaxSecretLen:         maxlen,
    MinLockTimeInit:      mintime,
  }
  err = db.Insert(&swap).Do()
  panicIfErr(err)

  //db.Execute(swap.CreateTableSQL())
  if *priceF64>0{
    fmt.Printf("New Swap:\nId: %v",swap.Id)
    fmt.Printf("user is selling: %v %.8f\n",*currencyToSwapper, *amountF64)
    fmt.Printf("user is buying: %v %.8f\n",*currencyToUser, *amountF64**priceF64)
    fmt.Printf("price(%v): %.8f per %v\n",*currencyToUser, *priceF64, *currencyToSwapper)
    fmt.Println("user should initiate a contract transaction on: ", *currencyToSwapper, " netowork")
    fmt.Printf("user should use %v as redeem\n", atomicSwapParams.ReciptAddress)
    fmt.Printf("user should use a %v lenght secret\n",atomicSwapParams.MaxSecretLen)
    fmt.Printf("user should use appropriate fee to have locktime at least %v hours after fully confirmed transaction\n",atomicSwapParams.MinLockTimeInitiate)
    interfaces.WriteResult(w,err,interfaces.InitiateSwapOut{
      SwapId:              swap.Id,
      SwapperAddress:      swap.AddressSwapper,
      UserAddress:         swap.AddressUser,
      MaxSecretLen:        strconv.Itoa(swap.MaxSecretLen),
      MinLockTimeInitiate: strconv.FormatInt(swap.MinLockTimeInit,10),
      SwapDate:            strconv.FormatInt(swap.Date,10),
      AmountToUser:        swap.AmountToUser,
      AmountToSwapper:     swap.AmountToSwapper,
      CurrencyToSwapper:   swap.CurrencyToSwapper,
      CurrencyToUser:      swap.CurrencyToUser,
    })
  }else{
    fmt.Println("swap not available")
  }
}

func panicIfErr(err error){
  if err!= nil{
    panic(err)
  }
}
func auditContract(url string,contract string,tx string, secretLen int, locktime int64,address string)(*interfaces.AuditContractOutput, error){

  audit, err := interfaces.AuditContract(url,interfaces.AuditContractInput{
    Contract: contract,
    Tx:       tx,
  })
  panicIfErr(err)
  fmt.Printf("contractAddress:%v\nrecipientAddress:%v\namount:%v\nrefundAddress:%v\nSecretHash:%v\nsecretLen:%v\nisSpendable:%v\nlocktime:%v\nsecretHash:%v\n",
    audit.ContractAddress,
    audit.RecipientAddress,
    audit.Amount,
    audit.RefundAddress,
    audit.SecretHash,
    audit.SecretLen,
    audit.IsSpendable,
    audit.LockTime,
    audit.SecretHash,
  )
  var auditSecretLen int
  var auditLockTime int64
  auditSecretLen, err = strconv.Atoi(audit.SecretLen)
  panicIfErr(err)
  auditLockTime, err = strconv.ParseInt(audit.LockTime,10,64)
  panicIfErr(err)
  if secretLen < auditSecretLen{
    return nil, errors.New(fmt.Sprintf("secretLen: %v < %v", secretLen, audit.SecretLen))
  }
  if address != audit.RecipientAddress {
    return nil, errors.New("Address: " + address + " != " + audit.RecipientAddress)

  }
  if locktime < auditLockTime{
    return nil, errors.New(fmt.Sprintf("locktime: %v < %v" , locktime,  audit.LockTime))
  }
  return audit,nil
}

func participateEndpoint(w http.ResponseWriter, r *http.Request){
  fmt.Println("PARTICIPATE")

  var args interfaces.ParticipateIn
  interfaces.ParseBody(r,&args)
  fmt.Println(args.Contract)
  swap := Swap{}
  err := db.Select(&swap).Where("id = ?",args.SwapId).Do()
  fmt.Println(err)
  fmt.Println(swap.Id)
  if swap.ContractInit != "" && swap.ContractInit != args.Contract{
    fmt.Printf("Contract error for initiated Swap:\n%v\n%v\n", swap.ContractInit,args.Contract)
    return
  }
  if !checkStatus(swap.CurrencyToSwapper){
    fmt.Printf("atomicswap  is offline %v:%v",swap.CurrencyToSwapper,networks[swap.CurrencyToSwapper].StatusCode)
    return
  }
  swap.TxInit = args.ContractTx
  swap.ContractInit = args.Contract
  audit,err := auditContract(networks[swap.CurrencyToSwapper].Url,swap.ContractInit,swap.TxInit,swap.MaxSecretLen,swap.MinLockTimeInit,swap.AddressSwapper)
  if err != nil{
    fmt.Println(err)
    return
  }
  swap.SecretHash = audit.SecretHash
  swap.AddressContractInit = audit.ContractAddress
  swap.TxIdInit = audit.TxId
  swap.StatusCode = "200"
  err = db.Update(&swap).Do()
  panicIfErr(err)
}

func roundFloat(val float64, precision uint) float64 {
    ratio := math.Pow(10, float64(precision))
    return math.Round(val*ratio) / ratio
}

