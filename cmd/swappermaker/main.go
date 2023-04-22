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
  "log"
  "strconv"
  //"os"
//  "database/sql"
  "github.com/samonzeweb/godb"
  "github.com/samonzeweb/godb/adapters/sqlite"
  "crypto/rand"
  "encoding/hex"
  "github.com/pieroforfora/atomicswapper/interfaces"

)
var db *godb.DB
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
    "url":"localhost:8080",
    "StatusCode":0,
    "statusMessage":""
  },
  "KAS":{ 
    "url":"localhost:8081",
    "StatusCode":0,
    "statusMessage":""
  }
}`)


type NetworkStatus struct {
  Url           string  `json:"url"`
  StatusCode    int     `json:"StatusCode"`
  StatusMessage string  `json:"statusMessage"`
}
var networks map[string]NetworkStatus
func checkStatus(n string) bool{
  if networks[n].StatusCode != 200{
    fmt.Printf("Atomic swap for %v is not available\n%v\n",n,networks[n].Url)
    return false
  }
  return true
}

func updateMarkets(){
  for{
    fmt.Println("updateMarkets")
    err := json.Unmarshal(book,&markets)
    if err != nil{
      fmt.Println(err)
    }
    time.Sleep(5 * time.Second)
  }
}
func updateNetworks(){
  for{
    //var networks  map[string]NetworkStatus
    err:= json.Unmarshal(networksJson,&networks)
    panicIfErr(err)
    fmt.Println(networks)
    for name, net := range networks{
      status := NetworkStatus{Url:net.Url}
      isOnlineResponse,err := http.Get("http://"+net.Url+"/is-online")
      if err!= nil{
        fmt.Println(err)
        status.StatusCode=0
        status.StatusMessage="no reply from server"
      }else{
        var isOnline bool
        interfaces.ParseBodyResponse(isOnlineResponse,&isOnline)
        if isOnline==true{
          status.StatusCode=200
          status.StatusMessage="ok"
        }else{
          status.StatusCode=201
          status.StatusMessage="Offline"
        }
      }
      networks[name]=status
    }
    fmt.Println("updateNetworks")
    for idx,network := range networks{
      fmt.Println(idx,network.StatusCode)
    }
    time.Sleep(5 * time.Second)
  }

}

func checkSwapInitiate(){
  for{
    swaps := make([]Swap, 0, 0)
    err := db.Select(&swaps).Where("status_code = ?","200").Do()
    panicIfErr(err)
    for _, swap := range swaps {
      audit, err := auditContract(swap)
      if err != nil {
        panic(err)
      }
      if audit.IsSpendable == "true" {
        fmt.Println(swap.Id)
        initiateLockTime,err := strconv.ParseInt(audit.LockTime,10,64)
        panicIfErr(err)
        partLockTime := strconv.FormatInt(int64(time.Now().Unix()+(initiateLockTime-time.Now().Unix())/2),10)
        response, err := interfaces.Post("http://"+networks[swap.CurrencyToUser].Url+"/participate",&interfaces.BuildContractInput{
          Them:         swap.AddressUser,
          Amount:       swap.AmountToUser,
          SecretHash:   &audit.SecretHash,
          LockTime:     &partLockTime,
        })
        panicIfErr(err)
        var part interfaces.BuildContractOutput
        interfaces.ParseBodyResponse(response,&part)
        response,err = interfaces.Post("http://" + networks[swap.CurrencyToUser].Url + "/pushtx",&interfaces.PushTxInput{Tx:part.Tx})
        panicIfErr(err)
        fmt.Println("participated")
        fmt.Println("TX:", part.Tx)
        fmt.Println("TXId:", part.TxID)
        fmt.Println("Contract:", part.Contract)
        fmt.Println("LastBlock:", part.LastBlock)
        swap.TxPart = part.Tx
        swap.TxIdPart = part.TxID
        swap.ContractPart = part.Contract
        swap.AddressContractPart = part.ContractAddress
        swap.LastBlock = part.LastBlock
        swap.StatusCode = "300"
        db.Update(&swap).Do()
      }
    }

    time.Sleep(5 * time.Second)
  }
}

func checkSwapRedeem(){
  for{
    swaps := make([]Swap, 0, 0)
    err := db.Select(&swaps).Where("status_code = ?","300").Do()
    panicIfErr(err)
    for _, swap := range swaps {
      fmt.Println("SWAP:",swap.Id,len(swaps))
      resp,err := interfaces.Post("http://"+networks[swap.CurrencyToUser].Url+"/check",&interfaces.CheckRedeemInput{
        LastBlock:  swap.LastBlock,
        TxId:       swap.TxIdPart,
        SecretHash: swap.SecretHash,
      })
      fmt.Println(err)
      fmt.Println(resp)
    }
    time.Sleep(5 * time.Second)
  }
}

func main(){
  ddb, err := godb.Open(sqlite.Adapter, "./swap.db")
  panicIfErr(err)
  // OPTIONAL: Set logger to show SQL execution logs
  //ddb.SetLogger(log.New(os.Stderr, "", 0))
  db = ddb

  go updateMarkets()
  go updateNetworks()
  go checkSwapInitiate()
  //TODO
  go checkSwapRedeem()
  //TODO go checkSwapTimedOut()
  restApiHandlers()
  fmt.Println("Server is up and running...")
  log.Fatal(http.ListenAndServe(":7080", nil))
}

func getPrice(currencyFrom, currencyTo string, volume float64, markets orderBook)float64{
  price := getPriceF1(currencyFrom, currencyTo, volume,true,markets)
  if price == 0 {
    price = getPriceF1(currencyTo,currencyFrom,volume,false,markets)
    if price == 0{
      price = getPriceF2(currencyTo,currencyFrom,volume,markets)
    }
  }
  return price
}
func getPriceByVolume(prices  [][]float64, volume float64) float64{
  for _,p := range prices{
    if len(p) ==1{
      return p[0]
    }
    if len(p) == 2 && volume > p[1]{
      return p[0]
    }
    if len(p)== 3 && volume > p[1] && volume< p[2]{
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
      StatusCode:     string(network.StatusCode),
      StatusMessage:  network.StatusMessage,
    }
    i+=1
  }
  interfaces.WriteResult(w,nil,isOnlineOut)
}
//TODO still missing to check balances
func initiateEndpoint(w http.ResponseWriter, r *http.Request){
  var args interfaces.InitiateSwapIn
  interfaces.ParseBody(r,&args)
  amountF64, err := strconv.ParseFloat(args.Amount, 64)
  if err != nil {
    fmt.Printf("failed to decode amount: %v", err)
    return
  }
  pair := strings.Split(strings.ToUpper(args.Market), "-")
  if !checkStatus(pair[0]) || !checkStatus(pair[1]) {
    fmt.Printf("atomicswap  is offline %v:%v - %v:%v",pair[0],networks[pair[0]].StatusCode,pair[1],networks[pair[1]].StatusCode)
    return
  }
  price := getPrice(pair[0],pair[1],amountF64, markets)
  newSwapResponse,err := http.Get("http://"+networks[pair[0]].Url+"/newswap" )
  if err!=nil{
    fmt.Println("errore:",err)
  }

  var atomicSwapParams interfaces.AtomicSwapParamsOutput
  interfaces.ParseBodyResponse(newSwapResponse,&atomicSwapParams)

  swapId :=make([]byte,31)
  _, err = rand.Read(swapId[:])
  panicIfErr(err)
  fmt.Println(hex.EncodeToString(swapId))
  fmt.Println("max_len:",atomicSwapParams.MaxSecretLen)
  maxlen,err := strconv.Atoi(atomicSwapParams.MaxSecretLen)
  panicIfErr(err)
  mintime,err := strconv.ParseInt(atomicSwapParams.MinLockTimeInitiate, 10, 64)
  panicIfErr(err)
  swap := Swap {
    Id:                   hex.EncodeToString(swapId),
    Date:                 time.Now().Unix(),
    StatusCode:           "0",
    CurrencyToUser:       pair[1],
    CurrencyToSwapper:    pair[0],
    AmountToSwapper:      strconv.FormatFloat(amountF64,'f',8,64),
    AmountToUser:         strconv.FormatFloat(amountF64*price,'f',8,64),
    AddressSwapper:       atomicSwapParams.ReciptAddress,
    AddressUser:          args.Address,
    MaxSecretLen:         maxlen,
    MinLockTimeInit:      mintime,
  }
  err = db.Insert(&swap).Do()
  panicIfErr(err)

  //db.Execute(swap.CreateTableSQL())
  fmt.Println(&swap)
  if price>0{
    fmt.Printf("you are selling: %v %.8f\n",pair[0], amountF64)
    fmt.Printf("you are buying: %v %.8f\n",pair[1], amountF64*price)
    fmt.Printf("price(%v): %.8f per %v\n",pair[1], 1*price, pair[0])
    fmt.Println("please initiate a contract transaction on: ", pair[0], " netowork")
    fmt.Printf("use %v as redeem\n", atomicSwapParams.ReciptAddress)
    fmt.Printf("use a %v lenght secret\n",atomicSwapParams.MaxSecretLen)
    fmt.Printf("use appropriate fee to have locktime at least %v hours after fully confirmed transaction\n",atomicSwapParams.MinLockTimeInitiate)
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
func auditContract(swap Swap)(*interfaces.AuditContractOutput, error){

  response, err := interfaces.Post("http://"+networks[swap.CurrencyToSwapper].Url+"/auditcontract",&interfaces.AuditContractInput{
    Contract: swap.ContractInit,
    Tx:       swap.TxInit,
  })
  panicIfErr(err)
  var audit interfaces.AuditContractOutput
  interfaces.ParseBodyResponse(response,&audit)
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
  if swap.MaxSecretLen < auditSecretLen{
    return nil, errors.New("secretLen: " + strconv.Itoa(swap.MaxSecretLen) + " < " + audit.SecretLen)
  }
  if swap.AddressSwapper != audit.RecipientAddress{
    return nil, errors.New("recipientAddress: " + swap.AddressSwapper + " != " + audit.RecipientAddress)

  }
  if swap.MinLockTimeInit < auditLockTime{
    return nil, errors.New("locktime: " + strconv.FormatInt(swap.MinLockTimeInit,64) + " < " + audit.LockTime)
  }
  return &audit,nil
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
  audit,err := auditContract(swap)
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

