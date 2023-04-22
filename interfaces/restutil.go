package interfaces

import(
  "errors"
  "io/ioutil"
  "net/http"
  "encoding/json"
  "fmt"
  "bytes"
)

func ParseBody(r *http.Request,args any)(error){

  reqBody, err := ioutil.ReadAll(r.Body)
  if err != nil {
    return errors.New(fmt.Sprintf("failed to read body: %v", err))
  }
  json.Unmarshal(reqBody, &args)
  return nil
}

func ParseBodyResponse(r *http.Response,args any)(error){

  reqBody, err := ioutil.ReadAll(r.Body)
  if err != nil {
    return errors.New(fmt.Sprintf("failed to read body: %v", err))
  }
  json.Unmarshal(reqBody, &args)
  return nil
}

func WriteResult(w http.ResponseWriter, err error, result any){
    w.Header().Set("Content-Type", "application/json")
  if err != nil {
    fmt.Println("/////////////////////////////////error",err)
    json.NewEncoder(w).Encode(ErrOutput{Err:fmt.Sprintf("%v",err)})
  } else {
    json.NewEncoder(w).Encode(result)
  }
}

func Post[T *CheckRedeemInput| *InitiateSwapIn | *BuildContractInput | *ParticipateIn | *SwapStatusIn | *AuditContractInput | *PushTxInput | *SpendContractInput](url string,args T)(*http.Response, error){
  fmt.Println("URL:",url)
  argsByte,err := json.Marshal(args)
  if err != nil{return nil,err}
  request, err := http.NewRequest("POST", url, bytes.NewBuffer(argsByte))
  if err != nil{return nil,err}
  request.Header.Set("Content-Type", "application/json;")
  client := &http.Client{}
  response, err := client.Do(request)
  if err != nil{fmt.Println(err);return nil,err}
  //body, _ := ioutil.ReadAll(response.Body)
  //fmt.Println("response Body:", string(body))
  return response,nil
}

