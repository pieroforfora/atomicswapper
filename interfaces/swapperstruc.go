package interfaces

type InitiateSwapIn struct {
  Market string `json:"Market"`
  Amount string `json:"Amount"`
  Address string `json:"Address"`
}

type InitiateSwapOut struct {
  SwapId              string  `json:"SwapId"`
  SwapperAddress      string  `json:"SwapperAddress"`
  UserAddress         string  `json:"UserAddress"`
  MaxSecretLen        string  `json:"MaxSecretLen"`
  MinLockTimeInitiate string  `json:"MinLockTimeInitiate"`
  SwapDate            string  `json:"SwapDate"`
  AmountToUser        string  `json:"AmountToUser"`
  AmountToSwapper     string  `json:"AmountToSwapper"`
  CurrencyToSwapper   string  `json:"CurrencyToSwapper"`
  CurrencyToUser      string  `json:"CurrencyToUser"`
}

type ParticipateIn struct {
  SwapId          string  `json:"SwapId"`
  ContractTx      string  `json:"ContractTx"`
  Contract        string  `json:"Contract"`
  SecretHash      string  `json:"SecretHash"`
}

type ParticipateOut struct {
  SwapId      string  `json:"SwapId"`
  ContractTx  string  `json:"ContractTx"`
  Contract    string  `json:"Contract"`
}

type SwapStatusIn struct{
  SwapId        string  `json:"SwapId"`
}

type SwapStatusOut struct{
  Id                      string    `json:"id"`
  Date                    int64     `json:"date"`
  StatusCode              string    `json:"status_code"`
  CurrencyToUser          string    `json:"currency_user"`
  CurrencyToSwapper       string    `json:"currency_swapper"`
  AmountToSwapper         string    `json:"amount_swapper"`
  AmountToUser            string    `json:"amount_user"`
  AddressSwapper          string    `json:"address_swapper"`
  AddressUser             string    `json:"address_user"`
  AddressContractInit     string    `json:"address_init"`
  AddressContractPart     string    `json:"address_part"`
  MaxSecretLen            int       `json:"max_secret_len"`
  MinLockTimeInit         int64     `json:"min_locktime"`
  SecretHash              string    `json:"secret_hash"`
  ContractInit            string    `json:"contract_init"`
  ContractPart            string    `json:"contract_part"`
  TxInit                  string    `json:"tx_init"`
  TxPart                  string    `json:"tx_part"`
  TxRedeemUser            string    `json:"tx_redeem_user"`
  TxRedeemSwapper         string    `json:"tx_redeem_swapper"`
  TxRefundUser            string    `json:"tx_refund_user"`
  TxRefundSwapper         string    `json:"tx_refund_swapper"`

}

type IsOnlineOut struct {
  Network       string `json:"Network"`
  StatusCode    string `json:"StatusCode"`
  StatusMessage string `json:"StatusMessage"`
}

