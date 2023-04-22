package interfaces

type AtomicSwapParamsOutput struct {
  ReciptAddress             string  `json:"ReciptAddress"`
  MaxSecretLen              string  `json:"MaxSecretLen"`
  MinLockTimeInitiate       string  `json:"MinLockTimeInitiate"`
  MinLockTimeParticipate    string  `json:"MinLockTimePartecipate"`
}

type WalletBalanceOutput struct {
  Available         string             `json:"Available"`
  Pending           string             `json:"Pending"`
  AddressBalances   []AddressBalance  `json:"Addresses"`
}

type AddressBalance struct {
  Address   string `json:"Address"`
  Available string `json:"Available"`
  Pending   string `json:"Pending"`
}

type BuildContractOutput struct {
  //if participate this will be an empty string
  Secret              string  `json:"Secret,omitempty"`
  SecretHash          string  `json:"SecretHash"`
  Contract            string  `json:"Contract"`
  ContractAddress     string  `json:"ContractAddress"`
  TxID                string  `json:"ContractTransactionID"`
  Tx                  string  `json:"ContractTransaction"`
  TxFee               string  `json:"TransactionFee"`
  LastBlock           string  `json:"LastBlock"`
}
type BuildContractInput struct {
  Them        string  `json:"RecipientAddress"`
  Amount      string  `json:"Amount"`
  //if nil or empty string I'll initiate a conctract
  //participate otherwise
  SecretHash  *string `json:"SecretHash,omitempty"`
  LockTime    *string `json:"LockTime,omitempty"`
  SecretLen   *string `json:"SecretLen,omitempty"`
}
type SpendContractOutput struct {
  Tx    string `json:"SpendTransaction"`
  TxID  string `json:"SpendTransactionID"`
  TxFee string `json:"TransactionFee"`
}
type SpendContractInput struct {
  //if nil or empty string I'll start refund
  //redeem otherwise
  Secret    string `json:"Secret"`
  Contract  string  `json:"Contract"`
  Tx        string  `json:"ContractTransaction"`
}
type AuditContractInput struct {
  Contract  string `json:"Contract"`
  Tx        string `json:"Tx"`
}
type  AuditContractOutput struct {
  ContractAddress   string `json:"ContractAddress"`
  //if I don't konw the address this is empty string
  RecipientAddress  string `json:"RecipientAddress,omitempty"`
  //Recipient2b       string `json:"RecipientBlake2b"`
  Amount            string `json:"ContractAmount"`
  //if I don't konw the address this is empty string
  RefundAddress     string `json:"RefundAddress,omitempty"`
  //Refund2b          string `json:"RefundBlake2b"`
  SecretHash        string `json:"SecretHash"`
  SecretLen         string `json:"SecretLen"`
  LockTime          string `json:"LockTime"`
  TxId              string `json:"TxId"`
  //Utxo blockDAAScore
  //DaaScore          string `json:"DaaScore"`
  //virtualSelectedParentBlueScore
  //VSPBS             string `json:"VSPBS"`
  //minConfirmations :=10
  //blockDAAScore+minConfirmations < virtualSelectedParentBlueScore 
  IsSpendable       string `json:"IsSpendable"`
}
type ExtractSecretInput struct {
  Tx          string `json:"Transaction"`
  SecretHash  string `json:"SecretHash"`
}
type ExtractSecretOutput struct {
  Secret  string `json:"Secret"`
}
type IsOnlineOutput struct {
  IsOnline string `json:"IsOnline"`
}
type PushTxInput struct {
  Tx    string `json:"Tx"`
}
type PushTxOutput struct {
  TxId    string `json:"TxId"`
}
type ErrOutput struct {
  Err string `json:"Err"`
}
type CheckRedeemInput struct {
 /* Address     string  `json:"Address"`
  SecretHash  string  `json:"SecretHash"`*/
  LastBlock string
  TxId  string
  SecretHash string
}
type CheckRedeemOutput struct {
 Secret string `json:"Secret"`
}
