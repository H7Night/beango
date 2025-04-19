package model

type TransactionRecord struct {
	TransactionTime   string // 时间
	TransactionCat    string // 交易分类
	Counterparty      string // 交易对方
	Commodity         string // 商品说明
	TransactionType   string // 收支
	Amount            string // 金额
	PaymentMethod     string // 收/付款方式
	TransactionStatus string // 交易状态
	Notes             string // 备注
	UUID              string // 交易订单号
	Discount          bool
}
