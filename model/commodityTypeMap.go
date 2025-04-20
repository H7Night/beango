package model

// CommodityTypeMap 分类关键词映射
var CommodityTypeMap = map[string]string{
	"还款":     "支出",
	"信用卡还款":  "支出",
	"花呗自动还款": "支出",
	"买入":     "支出",
	"对方已收钱":  "支出",
	"收益发放":   "收入",
	"已存入零钱":  "收入",
	"自动转入":   "skip",
	"转账收款到":  "skip",
}
