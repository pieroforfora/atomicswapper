package interfaces
import(
  "errors"
  "fmt"
  "net/http"
)
func PushTx(url string, input PushTxInput)(*PushTxOutput,error){
  response,err := Post("http://"+url+"/pushtx", &input)
  if err!= nil{
    return nil, errors.New(fmt.Sprintf("failed to push transaction: %v",err))
  }
  var out *PushTxOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func Initiate(url string, input BuildContractInput)(*BuildContractOutput, error){
  fmt.Println("initiate",input)
  response, err := Post("http://"+url+"/initiate", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to initiate swap: %v",err))
  }
  var out *BuildContractOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}
func Participate(url string, input BuildContractInput)(*BuildContractOutput, error){
  response, err := Post("http://"+url+"/participate", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to participate: %v",err))
  }
  var out *BuildContractOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func Redeem(url string, input SpendContractInput)(*SpendContractOutput, error){
  response, err := Post("http://"+url+"/redeem", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to redeem: %v",err))
  }
  var out *SpendContractOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func Refund(url string, input SpendContractInput)(*SpendContractOutput, error){
  response, err := Post("http://"+url+"/refund", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to refund: %v",err))
  }
  var out *SpendContractOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func WalletBalance(url string)(*WalletBalanceOutput, error){
  response, err := http.Get("http://"+url+"/walletBalance")
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to get wallet balance: %v",err))
  }
  var out *WalletBalanceOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func IsOnline(url string)(*IsOnlineOutput, error){
  response, err := http.Get("http://"+url+"/is-online")
  if err != nil {
    return nil, errors.New(fmt.Sprintf("IsNotOnline%v",err))
  }
  var out *IsOnlineOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func SearchRedeem(url string, input CheckRedeemInput)(*CheckRedeemOutput, error){
  response, err := Post("http://"+url+"/searchredeem", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("searchRedeem: %v\n %v",err,input))
  }
  var out *CheckRedeemOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func AuditContract(url string, input AuditContractInput)(*AuditContractOutput, error){
  response, err := Post("http://"+url+"/auditcontract", &input)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("failed to push transaction: %v",err))
  }
  var out *AuditContractOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}

func SwapParams(url string, getaddress bool)(*AtomicSwapParamsOutput, error){
  response, err := Post("http://"+url+"/newswap", &getaddress)
  if err != nil {
    return nil, errors.New(fmt.Sprintf("%v\n%v",err,getaddress))
  }
  var out *AtomicSwapParamsOutput
  err = ParseBodyResponse(response,&out)
  return out, nil
}
