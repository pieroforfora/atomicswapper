package main
import(
//  "net/http"
  "github.com/pieroforfora/atomicswapper/interfaces"
  //"io/ioutil"
  //"bytes"
  //"encoding/json"
  "fmt"
  "time"
)
func main(){

  args := interfaces.InitiateSwapIn{
    Market:   "KAS-BTC",
    Amount:   "1000",
    Address:  "mpraiqEbjyQoS2mjzvCuPu1PYq66fRu9yD",
  }
  response,err := interfaces.Post("http://localhost:7080/initiate",&args)
  panicIfErr(err)
  fmt.Println(response)

  var initiateSwap interfaces.InitiateSwapOut
  interfaces.ParseBodyResponse(response,&initiateSwap)
  fmt.Println("Initiate Swap")
  fmt.Println(initiateSwap)
  fmt.Println("------------")
  params := interfaces.BuildContractInput{
    Them:       initiateSwap.SwapperAddress,
    Amount:     initiateSwap.AmountToSwapper,
    LockTime:   &initiateSwap.MinLockTimeInitiate,
    SecretLen:  &initiateSwap.MaxSecretLen,
  }
  builtSwap, err := interfaces.Initiate("localhost:8081",params)
  panicIfErr(err)
  fmt.Println(builtSwap)
  fmt.Printf("Secret:%v\nSecretHash:%v\nContract:%v\nTx:%v\nTxFee:%v",
    builtSwap.Secret,
    builtSwap.SecretHash,
    builtSwap.Contract,
    builtSwap.Tx,
    builtSwap.TxFee,
  )

  response,err = interfaces.Post("http://localhost:8081/pushtx",&interfaces.PushTxInput{Tx:builtSwap.Tx})
  fmt.Println(response)
  fmt.Println("SECRET:",builtSwap.Secret)
  done := interfaces.ParticipateIn{
    SwapId:       initiateSwap.SwapId,
    ContractTx:   builtSwap.Tx,
    Contract:     builtSwap.Contract,
    SecretHash:   builtSwap.SecretHash,
  }
  response, err = interfaces.Post("http://localhost:7080/participate",&done)
  fmt.Println("waiting for swapper to participate")



/*
  initiateSwap := interfaces.InitiateSwapOut{
    SwapId:     "524eae8a6bc43e9c1f6267819d47e647426d6b55828a36f3aa5051faae5e56",
  }
  builtSwap := interfaces.BuildContractOutput{
    Secret:     "306334fb6d01bdb163fa73897a448355809deb84b19716710e8d3136f6576208",
    SecretHash: "f484013c6663a5fcfc2207a988be8d1c045773f4f1f2c5804f6eca1b502bf4e8",
  }
*/
  for{
    response, err := interfaces.Post("http://localhost:7080/check",&interfaces.SwapStatusIn{
      SwapId: initiateSwap.SwapId,
    })
    fmt.Println(err)
    var swapStatus interfaces.SwapStatusOut
    interfaces.ParseBodyResponse(response,&swapStatus)
    if swapStatus.StatusCode== "300" {
      //response,err = interfaces.Post("http://localhost:8080/auditcontract",&interfaces.PushTxInput{Tx:builtSwap.Tx})
      audit,err:=auditContract("localhost:8080",swapStatus.ContractPart,swapStatus.TxPart)
      if err!= nil {fmt.Println(err)}
      if audit.IsSpendable == "true" {
        fmt.Println("Participate is spendable")
        response,err = interfaces.Post("http://localhost:8080/redeem",&interfaces.SpendContractInput{
          Secret:   builtSwap.Secret,
          Contract: swapStatus.ContractPart,
          Tx:       swapStatus.TxPart,
        })
        if err != nil {
          fmt.Println("RedeemError:",err)
          continue
        }
        var redeemOut interfaces.SpendContractOutput
        err= interfaces.ParseBodyResponse(response,&redeemOut)
        if err != nil {
          fmt.Println("ErrorParsing redeem:",err)
          continue
        }
        fmt.Println(response,redeemOut)
        fmt.Println("pushTx: ",redeemOut.Tx)
        fmt.Println("pushTx: ",redeemOut.TxId)
        response,err = interfaces.Post("http://localhost:8080/pushtx",&interfaces.PushTxInput{Tx:redeemOut.Tx})
        fmt.Println("response:",response)
        fmt.Println("error:",err)
        break
      }else{
        fmt.Println("participate is not spendable")
      }
    }else{
      fmt.Println("status code is not 300: ", swapStatus.StatusCode )
    }
    time.Sleep(60 * time.Second)
  }
}
func panicIfErr(err error){
  if err != nil{
    panic(err)
  }
}

func auditContract(url string, contract string,tx string)(*interfaces.AuditContractOutput, error){

  response, err := interfaces.Post("http://"+url+"/auditcontract",&interfaces.AuditContractInput{
    Contract: contract,
    Tx:       tx,
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
  return &audit,nil
}

