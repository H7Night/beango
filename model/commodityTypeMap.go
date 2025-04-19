package model

// commodity 分类关键词映射
var CommodityTypeMap = map[string]string{
	"还款":     "支出",
	"信用卡还款":  "支出",
	"花呗自动还款": "支出",
	"买入":     "支出",
	"收益发放":   "收入",
	"转入":     "跳过",
	"已存入零钱":  "收入",
	"对方已收钱":  "支出",
}
