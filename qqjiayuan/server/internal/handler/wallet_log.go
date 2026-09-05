package handler

import (
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// addWalletLog 记录钱包收支流水（钱包页明细）
// currency: coins=G币 / yuanbao=元宝 / jinzuan=金钻 / youquan=友友券 / flower=鲜花
func addWalletLog(db *gorm.DB, uid uint, kind, title, currency string, delta int) {
	if delta == 0 {
		return
	}
	db.Create(&model.WalletLog{UserID: uid, Kind: kind, Title: title, Currency: currency, Delta: delta})
}
