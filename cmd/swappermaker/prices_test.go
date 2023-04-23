package main
import (
  "testing"
  "fmt"
  "encoding/json"
)
var book_test =  []byte(`{
    "BTC":{
      "KAS":{
        "BUY":  [[0.00000080]],
        "SELL": [[0.0000008]]
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
}`)

func TestGetPriceSimple(t *testing.T){
  var markets orderBook
  json.Unmarshal(book_test,&markets)
  volume     :=  float64(1000)
  exp        :=  float64(0.0008)
  price      :=  getPrice("KAS","BTC",volume,markets)
  amount := float64(float64(price)*float64(volume))
  diff   := float64(amount-exp)


  fmt.Println("Volume:", volume)
  fmt.Println("Price:", price)
  fmt.Println("Expected amount:",exp)
  fmt.Println("Calculated amount:", amount)
  fmt.Println("diff:",diff)
  fmt.Println("")

  if roundFloat(amount,8) != roundFloat(exp,8) {
    t.Fatalf("Error amount should be 0.0008: %v %v %v",price, volume, amount)
  }
}

func TestGetPriceSell(t *testing.T){
  var markets orderBook
  json.Unmarshal(book_test,&markets)
  volume     :=  float64(0.0008)
  exp        :=  float64(1000)
  price      :=  getPrice("BTC","KAS",volume,markets)
  amount := float64(float64(price)*float64(volume))
  diff   := float64(amount-exp)


  fmt.Println("Volume:", volume)
  fmt.Println("Price:", price)
  fmt.Println("Expected amount:",exp)
  fmt.Println("Calculated amount:", amount)
  fmt.Println("diff:",diff)
  fmt.Println("")

  if roundFloat(amount,8) != roundFloat(exp,8) {
    t.Fatalf("Error amount should be 0.0008: %v %v %v",price, volume, amount)
  }
}
func TestGetPriceRverse(t *testing.T){
  var markets orderBook
  json.Unmarshal(book_test,&markets)
  volume     :=  float64(-0.0008)
  exp        :=  float64(-1000)
  price      :=  getPrice("BTC","KAS",volume,markets)
  amount := float64(float64(price)*float64(volume))
  diff   := float64(amount-exp)


  fmt.Println("Volume:", volume)
  fmt.Println("Price:", price)
  fmt.Println("Expected amount:",exp)
  fmt.Println("Calculated amount:", amount)
  fmt.Println("diff:",diff)
  fmt.Println("")

  if roundFloat(amount,8) != roundFloat(exp,8) {
    t.Fatalf("Error amount should be 0.0008: %v %v %v",price, volume, amount)
  }
}
func TestGetPriceRverseSell(t *testing.T){
  var markets orderBook
  json.Unmarshal(book_test,&markets)
  volume     :=  float64(-1000)
  exp        :=  float64(-0.0008)
  price      :=  getPrice("KAS","BTC",volume,markets)
  amount := float64(float64(price)*float64(volume))
  diff   := float64(amount-exp)


  fmt.Println("Volume:", volume)
  fmt.Println("Price:", price)
  fmt.Println("Expected amount:",exp)
  fmt.Println("Calculated amount:", amount)
  fmt.Println("diff:",diff)
  fmt.Println("")

  if roundFloat(amount,8) != roundFloat(exp,8) {
    t.Fatalf("Error amount should be 0.0008: %v %v %v",price, volume, amount)
  }
}
