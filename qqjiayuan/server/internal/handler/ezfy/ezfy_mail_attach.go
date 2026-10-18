// 二战风云 —— 邮件附件（★ 2026-10-18）
//
// 用户需求（原话）：
//
//	「风云榜 开发一个 发放宝箱功能 给第一名、第二名、第三名发宝箱；给参与者发宝箱；
//	 给第一二三名发道具；给参与者发道具；给第一二三名发钻石；给参与者发钻石；
//	 发放不是直接进玩家背包，相当于系统给玩家发邮件，玩家领取，7 天不领取默认无效…
//	 风云榜发东西会加修饰就是参与风云榜获得的，普通数据管理那块的发放就是系统发放的；
//	 数据管理里面发放道具也是走邮件吧，不然用户都不知道获得了什么」
//
// 所以这里做的是**一套通用的邮件附件能力**，风云榜奖励和系统发放都复用它：
//
//	ezfy_mail         加 has_attach / claimed / expire_at / batch_no 四列
//	ezfy_mail_attach  一件附件一行（道具 / 宝箱 / 钻石）
//
// 关键口径：
//   - 发放 = **写邮件**，绝不直接进背包；玩家在邮箱里点「领取」才真正到账。
//   - 领取是**一次性**的：`claimed` 用条件 UPDATE 抢锁（`WHERE claimed = 0`），
//     连点/多端并发只有一次能成功，不会重复发奖。
//   - 过期后不可领取（`expire_at` ms），但邮件仍在，玩家能看到「已过期」。
//   - 附件三类的到账口径与游戏内既有渠道**完全一致**：
//     道具 → ezfy_item（原子 upsert）；宝箱 → ezfy_user_chest（唯一键累加）；钻石 → ezfy_profile.diamond（原子自增 + 流水）。
package ezfy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 邮件附件类型（ezfy_mail_attach.kind）
const (
	ezfyAttachItem    = 1 // 道具（ezfy_cfg_item）
	ezfyAttachChest   = 2 // 宝箱（ezfy_cfg_chest）
	ezfyAttachDiamond = 3 // 钻石（无配置，CfgID=0）
)

// ezfyMailAttachExpireDays 带附件邮件的默认领取期限（天）。用户规则「7 天不领取默认无效」。
const ezfyMailAttachExpireDays = 7

// ★ 2026-10-18 领取防刷（用户要求「领取的时候注意防抖、防刷领取」）：
//
//	同一玩家 300ms 内只允许点一次 + 每分钟最多 60 次。
//
// 说明：**真正的防重复发奖**靠 `UPDATE ... WHERE claimed = 0` 的条件更新（多端并发也只有一个成功），
// 这里只挡「脚本连点 / 高频刷接口」这类无意义请求，不参与业务判定。
const (
	ezfyMailClaimMinGapMs = 300
	ezfyMailClaimWinMs    = 60 * 1000
	ezfyMailClaimWinMax   = 60
)

type ezfyMailClaimRate struct {
	lastMs   int64
	winStart int64
	winCount int
}

var (
	ezfyMailClaimMu    sync.Mutex
	ezfyMailClaimRates = map[uint]*ezfyMailClaimRate{}
)

// ezfyMailClaimAllow 领取限流（进程内计数；多机部署各机独立，够用）。
func ezfyMailClaimAllow(uid uint, nowMs int64) (bool, string) {
	ezfyMailClaimMu.Lock()
	defer ezfyMailClaimMu.Unlock()
	if len(ezfyMailClaimRates) > 65536 { // 防无限增长（与其它进程内备忘同一策略）
		ezfyMailClaimRates = map[uint]*ezfyMailClaimRate{}
	}
	r := ezfyMailClaimRates[uid]
	if r == nil {
		r = &ezfyMailClaimRate{winStart: nowMs}
		ezfyMailClaimRates[uid] = r
	}
	if nowMs-r.lastMs < ezfyMailClaimMinGapMs {
		return false, "点得太快了，稍等一下再领"
	}
	if nowMs-r.winStart >= ezfyMailClaimWinMs {
		r.winStart = nowMs
		r.winCount = 0
	}
	if r.winCount >= ezfyMailClaimWinMax {
		return false, "领取太频繁，请稍后再试"
	}
	r.lastMs = nowMs
	r.winCount++
	return true, ""
}

// ezfyMailAttachKindName 附件类型的展示名（管理端/玩家端文案用）。
func ezfyMailAttachKindName(kind int) string {
	switch kind {
	case ezfyAttachItem:
		return "道具"
	case ezfyAttachChest:
		return "宝箱"
	case ezfyAttachDiamond:
		return "钻石"
	}
	return "物品"
}

// ezfyMailDraft 待发送的一封系统邮件（发件人固定为系统，正文由调用方拼好）。
type ezfyMailDraft struct {
	ReceiverID uint
	Subject    string
	Content    string
	Attaches   []model.EzfyMailAttach // Kind/CfgID/Num/Name 由调用方填，MailID/UserID 这里补
}

// ezfyMailSendBatch 批量发送**带附件的系统邮件**。
//
// 性能：全员发放动辄几百上千人，绝不能一人一次 Create（跨 WAN 的 RDS 会超时）。
// 这里先用 CreateInBatches 写邮件，再用 `batch_no` 回查邮件 id（不依赖自增连号，
// 并发发放也不会串行），最后把附件一次性批量插入。
//
// 返回批次号（同一批的所有邮件共用），失败返回 error（调用方负责事务/回滚提示）。
func ezfyMailSendBatch(db *gorm.DB, drafts []ezfyMailDraft, expireDays int) (string, error) {
	if len(drafts) == 0 {
		return "", nil
	}
	batchNo := ezfyNewBatchNo()
	now := time.Now()
	var expireAt int64
	if expireDays > 0 {
		expireAt = now.AddDate(0, 0, expireDays).UnixMilli()
	}
	mails := make([]model.EzfyMail, 0, len(drafts))
	for _, d := range drafts {
		if d.ReceiverID == 0 {
			continue
		}
		mails = append(mails, model.EzfyMail{
			SenderId:   0, // 系统邮件
			ReceiverId: d.ReceiverID,
			SenderName: "系统",
			Subject:    d.Subject,
			Content:    d.Content,
			HasAttach:  boolToInt(len(d.Attaches) > 0),
			ExpireAt:   expireAt,
			BatchNo:    batchNo,
			CreatedAt:  now,
		})
	}
	if len(mails) == 0 {
		return "", nil
	}
	if err := db.CreateInBatches(&mails, 200).Error; err != nil {
		return "", err
	}
	// 按批次号回查真实邮件 id（收件人 → 邮件 id）
	var saved []struct {
		ID         uint
		ReceiverId uint
	}
	db.Model(&model.EzfyMail{}).
		Select("id, receiver_id").
		Where("batch_no = ?", batchNo).Scan(&saved)
	mailOf := make(map[uint]uint, len(saved))
	for _, s := range saved {
		mailOf[s.ReceiverId] = s.ID
	}
	// 附件批量入库
	rows := make([]model.EzfyMailAttach, 0, len(drafts)*2)
	for _, d := range drafts {
		mid := mailOf[d.ReceiverID]
		if mid == 0 {
			continue
		}
		for _, a := range d.Attaches {
			if a.Num <= 0 {
				continue
			}
			a.ID = 0
			a.MailID = mid
			a.UserID = d.ReceiverID
			rows = append(rows, a)
		}
	}
	if len(rows) > 0 {
		if err := db.CreateInBatches(&rows, 500).Error; err != nil {
			return batchNo, err
		}
	}
	return batchNo, nil
}

// ezfyNewBatchNo 生成发放批次号（时间前缀 + 8 字节随机，便于人工对账）。
func ezfyNewBatchNo() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("B%d", time.Now().UnixNano())
	}
	return "B" + time.Now().Format("20060102150405") + "-" + hex.EncodeToString(b)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ezfyMailAttachMap 批量取多封邮件的附件（一次 SQL，避免列表 N+1）。
func ezfyMailAttachMap(db *gorm.DB, mailIDs []uint) map[uint][]model.EzfyMailAttach {
	out := map[uint][]model.EzfyMailAttach{}
	if len(mailIDs) == 0 {
		return out
	}
	var rows []model.EzfyMailAttach
	db.Where("mail_id IN ?", mailIDs).Order("id ASC").Find(&rows)
	for _, r := range rows {
		out[r.MailID] = append(out[r.MailID], r)
	}
	return out
}

// ezfyMailAttachView 附件 → 前端 JSON（带类型名，前端直接展示）。
func ezfyMailAttachView(a model.EzfyMailAttach) gin.H {
	return gin.H{
		"kind": a.Kind, "kind_name": ezfyMailAttachKindName(a.Kind),
		"cfg_id": a.CfgID, "num": a.Num, "name": a.Name,
	}
}

// MailClaim POST /games/ezfy/mail/claim {id} —— 领取邮件附件
//
// 流程：行锁读邮件 → 校验（是系统邮件 / 有附件 / 未领 / 未过期）→ **条件 UPDATE 抢锁**
// → 逐件入账 → 提交。抢锁那一步是防重复领取的核心：
// 即使连点两下、或两个端同时点，`WHERE claimed = 0` 只有一个能 RowsAffected=1。
func (h *EzfyHandler) MailClaim(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	// ★ 防抖 / 防刷：连点、脚本高频请求直接挡掉（真正的防重复发奖靠下面的条件 UPDATE）
	if ok, msg := ezfyMailClaimAllow(uid, time.Now().UnixMilli()); !ok {
		resp.FailData(c, 429, msg, nil)
		return
	}
	h.cfgs()
	h.ensureProfile(uid)

	now := time.Now().UnixMilli()
	var got []gin.H
	tx := h.DB.Begin()
	if tx.Error != nil {
		resp.ServerError(c, tx.Error)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			resp.ServerError(c, fmt.Errorf("领取失败"))
		}
	}()

	var m model.EzfyMail
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND receiver_id = ? AND recv_del = 0", req.ID, uid).First(&m).Error; err != nil {
		tx.Rollback()
		resp.NotFound(c, "邮件不存在")
		return
	}
	if m.SenderId != 0 || m.HasAttach == 0 {
		tx.Rollback()
		resp.ParamError(c, "这封邮件没有可领取的附件")
		return
	}
	if m.Claimed == 1 {
		tx.Rollback()
		resp.ParamError(c, "奖励已领取，请勿重复领取")
		return
	}
	if m.ExpireAt > 0 && now >= m.ExpireAt {
		tx.Rollback()
		resp.ParamError(c, "奖励已过期（超过领取期限）")
		return
	}
	// ★ 抢锁：只有把 claimed 从 0 改成 1 的那一次才继续往下发奖
	res := tx.Model(&model.EzfyMail{}).
		Where("id = ? AND claimed = 0", m.ID).Update("claimed", 1)
	if res.Error != nil {
		tx.Rollback()
		resp.ServerError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		resp.ParamError(c, "奖励已领取，请勿重复领取")
		return
	}

	var as []model.EzfyMailAttach
	tx.Where("mail_id = ?", m.ID).Order("id ASC").Find(&as)
	for _, a := range as {
		if a.Num <= 0 {
			continue
		}
		switch a.Kind {
		case ezfyAttachItem:
			// 与 addItem 同口径：原子 upsert（唯一键 uk_user_item），不卡背包单件上限
			if err := tx.Exec(`INSERT INTO ezfy_item (user_id, cfg_id, count) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE count = count + ?`, uid, a.CfgID, a.Num, a.Num).Error; err != nil {
				tx.Rollback()
				resp.ServerError(c, err)
				return
			}
		case ezfyAttachChest:
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "chest_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"count":      gorm.Expr("count + ?", a.Num),
					"updated_at": time.Now(),
				}),
			}).Create(&model.EzfyUserChest{UserID: uid, ChestId: a.CfgID, Count: int(a.Num)}).Error; err != nil {
				tx.Rollback()
				resp.ServerError(c, err)
				return
			}
		case ezfyAttachDiamond:
			if err := tx.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
				Update("diamond", gorm.Expr("diamond + ?", a.Num)).Error; err != nil {
				tx.Rollback()
				resp.ServerError(c, err)
				return
			}
			var bal int64
			tx.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).Pluck("diamond", &bal)
			tx.Create(&model.EzfyDiamondLog{
				UserId: uid, Change: a.Num, Balance: bal,
				Reason: "邮件领取: " + m.Subject,
			})
		}
		got = append(got, ezfyMailAttachView(a))
	}
	if err := tx.Commit().Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	ezfyChestCacheDel(uid)

	// 领取提示（前端弹「获得 …」）
	names := make([]string, 0, len(got))
	for _, g := range got {
		names = append(names, fmt.Sprintf("%s×%v", g["name"], g["num"]))
	}
	resp.OK(c, gin.H{
		"msg":     "领取成功：" + strings.Join(names, "、"),
		"got":     got,
		"claimed": 1,
	})
}
