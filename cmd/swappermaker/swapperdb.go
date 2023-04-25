package main

import(
)
type Swap struct {
  Id                      string    `db:"id,key"`
  Date                    int64     `db:"date"`
  StatusCode              string    `db:"status_code"`
  CurrencyToUser          string    `db:"currency_user"`
  CurrencyToSwapper       string    `db:"currency_swapper"`
  AmountToSwapper         string    `db:"amount_swapper"`
  AmountToUser            string    `db:"amount_user"`
  AddressSwapper          string    `db:"address_swapper"`
  AddressUser             string    `db:"address_user"`
  AddressContractInit     string    `db:"address_init"`
  AddressContractPart     string    `db:"address_part"`
  MaxSecretLen            int       `db:"max_secret_len"`
  SecretHash              string    `db:"secret_hash"`
  LastBlock               string    `db:"last_block"`
  MinLockTimeInit         int64     `db:"min_locktime"`
  LocktimeInit            int64     `db:"locktime_init"`
  LocktimePart            int64     `db:"locktime_part"`
  SecretLen               int       `db:"secret_len"`
  Secret                  string    `db:"secret"`
  ContractInit            string    `db:"contract_init"`
  ContractPart            string    `db:"contract_part"`
  TxIdInit                string    `db:"tx_id_init"`
  TxIdPart                string    `db:"tx_id_part"`
  TxIdRedeemUser          string    `db:"tx_id_redeem_user"`
  TxIdRedeemSwapper       string    `db:"tx_id_redeem_swapper"`
  TxIdRefundUser          string    `db:"tx_id_refund_user"`
  TxIdRefundSwapper       string    `db:"tx_id_refund_swapper"`
  TxInit                  string    `db:"tx_init"`
  TxPart                  string    `db:"tx_part"`
  TxRedeemUser            string    `db:"tx_redeem_user"`
  TxRedeemSwapper         string    `db:"tx_redeem_swapper"`
  TxRefundUser            string    `db:"tx_refund_user"`
  TxRefundSwapper         string    `db:"tx_refund_swapper"`
}

func (*Swap) TableName()string{
  return "tbl_swap"
}

func (*Swap) CreateTableSQL() string{
  return `CREATE TABLE tbl_swap(
    id                    CHAR(65) PRIMARY KEY,
    date                  INTEGER,
    status_code           INTEGER,
    currency_user         CHAR(5),
    currency_swapper      CHAR(5),
    amount_swapper        TEXT,
    amount_user           TEXT,
    address_swapper       TEXT,
    address_user          TEXT,
    address_init          TEXT,
    address_part          TEXT,
    max_secret_len        INTEGER,
    min_locktime          INTEGER,
    locktime_init         INTEGER,
    locktime_part         INTEGER,
    secret_len            INTEGER,
    secret                TEXT,
    secret_hash           TEXT,
    last_block            TEXT,
    contract_init         TEXT,
    contract_part         TEXT,
    tx_id_init            TEXT,
    tx_id_part            TEXT,
    tx_id_redeem_user     TEXT,
    tx_id_redeem_swapper  TEXT,
    tx_id_refund_user     TEXT,
    tx_id_refund_swapper  TEXT,
    tx_init               TEXT,
    tx_part               TEXT,
    tx_redeem_user        TEXT,
    tx_redeem_swapper     TEXT,
    tx_refund_user        TEXT,
    tx_refund_swapper     TEXT
  );
  `
}
/*
func GetSwapSQL(id string)string{
  return `SELECT
      id,
      date,
      status_code,
      currency_user,
      currency_swapper,
      amount_swapper,
      amount_user,
      address_swapper,
      address_user,
      address_contract,
      max_secret_len,
      min_locktime,
      contract,
      tx_initiate,
      tx_participate,
      tx_redeem_user,
      tx_redeem_swapper,
      tx_refund_user,
      tx_refund_swapper
    FROM
      tbl_swap
    WHERE id="`+id+`":"`
}

func (swap *Swap)Get(id string, db sql.DB) (error){
  row := db.QueryRow(GetSwapSQL(id))
  *swap = Swap{}
  err:= row.Scan(&swap.Id,
      &swap.Date,
      &swap.StatusCode,
      &swap.CurrencyToUser,
      &swap.CurrencyToSwapper,
      &swap.AmountToSwapper,
      &swap.AmountToUser,
      &swap.AddressSwapper,
      &swap.AddressUser,
      &swap.AddressContract,
      &swap.MaxSecretLen,
      &swap.MinLockTimeInitiate,
      &swap.Contract,
      &swap.TxInitiate,
      &swap.TxParticipate,
      &swap.TxRedeemUser,
      &swap.TxRedeemSwapper,
      &swap.TxRefundUser,
      &swap.TxRefundSwapper)
  if err!=nil{  panic(err)}
  return err

}
func dbCreate(){

}

type Network struct {
  Name          string `db:"name,key"`
  Url           string `db:"url"`
}

func (*Network) TableName() string{
  return "tbl_network"
}

func (*Network) CreateTableSQL() string{
  return `CREATE TABLE tbl_network (
      name  TEXT NOT NULL,
      url   TEXT NOT NULL
\    );
    CREATE INDEX idx_network_name
      ON tbl_network(name);
  `
}

type Market struct {
  Name    string  `db:"market"`
  Price     string  `db:"price"`
  VolumeMin string  `db:"volume_min"`
  VolumeMax string  `db:"volume_max"`
}

func (*Market) TableName()string{
  return "tbl_market"
}

func (*Market) CreateTableSQL() string{
  return `CREATE TABLE tbl_market (
      name        TEXT,
      price       TEXT,
      volume_min  TEXT,
      volume_max  TEXT
    );
    CREATE INDEX idx_market_name
      ON tbl_market(name);
  `
}
*/
