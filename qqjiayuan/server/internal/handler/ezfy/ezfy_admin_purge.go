package ezfy

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ============ 删号：清空该玩家在**二战库**的全部数据（2026-10-15）============
//
// 用户规则（原话）：「玩家信息管理 → 操作里面的删除 是物理操作，把该玩家的所有数据都删除掉」。
//
// 为什么单独抽一个文件 + 声明式清单：
//
//	二战库有 50+ 张「玩家数据表」，散在一个函数里**极易漏表** —— 老实现只清了 16 张，
//	`ezfy_account`（登录账号）/ `ezfy_officer`（军官）/ `ezfy_equipment`（装备）/
//	`ezfy_exchange`（交易行）/ `ezfy_mail`（邮件）/ `ezfy_friend`（好友）/
//	`ezfy_pvp_*`（风云榜）/ `ezfy_war`（宣战）/ `ezfy_occupy`（占领）/ `ezfy_corps*`（军团）
//	全留着 → 删完号在后台还能看到残留（用户反馈的「不是全删」）。
//	现在改成**声明式清单**，并配 `ezfy_admin_purge_test.go` 的**漏表护栏**：
//	从 `internal/model/ezfy.go` 抽出全部 `TableName()` 表名，逐个要求「在清单里 或 在白名单里」，
//	以后新增二战表忘了加删号清单 → 测试直接失败。
//
// ⚠️ 只动**二战库**（h.DB）。家园库（qq_jiayuan）不动：
//
//	① 家园账号（users）是玩家在家园的资产，不属于二战删号范围；
//	② 家园库里那批老 `ezfy_*` 表是「二战拆库」的回滚退路，按设计保留。
//
// ⚠️ 全部走 **raw DELETE**（不用 GORM 的 Delete(model)）：
//
//	`ezfy_officer` 带 `deleted_at` 是**软删除**表，走 GORM 只会打标记、行还在库里
//	→ 与「物理删除、数据全没」的语义不符。
//
// ⚠️ 整个过程放在**一个事务**里：任一步失败整体回滚，不会留下删一半的脏号。

// ezfyPurgeRule 一条「删号要清空的表」规则：Table = 表名，Cols = 命中**任一**列即删（OR）。
type ezfyPurgeRule struct {
	Table string
	Cols  []string
}

// ezfyPurgeByUid 按**玩家**归属的表（占位值 = 该玩家的 user_id = 家园号）。
//
// 列名就是归属列；跨玩家引用（好友/邮件/私信/风云榜对手）两个方向都清。
var ezfyPurgeByUid = []ezfyPurgeRule{
	// 登录账号 / 档案
	{"ezfy_account", []string{"game_id"}}, // 二战独立登录账号（game_id = 家园号）
	{"ezfy_profile", []string{"user_id"}},
	{"ezfy_city", []string{"user_id"}}, // 城下子表见 ezfyPurgeByCity
	// 出征 / 战斗 / 战报 / 占领 / 赎城 / 宣战
	{"ezfy_order", []string{"user_id"}},
	{"ezfy_report", []string{"user_id"}},
	{"ezfy_battle", []string{"user_id", "def_user_id"}},
	{"ezfy_occupy", []string{"atk_user_id", "def_user_id"}},
	{"ezfy_ransom", []string{"atk_user_id", "def_user_id"}},
	{"ezfy_war", []string{"atk_user_id", "def_user_id"}},
	// 背包 / 道具 / 装备 / 宝箱 / 签到 / 任务 / 通知
	{"ezfy_item", []string{"user_id"}},
	{"ezfy_item_use_logs", []string{"user_id"}},
	{"ezfy_equipment", []string{"user_id", "original_user_id"}},
	{"ezfy_user_chest", []string{"user_id"}},
	{"ezfy_user_chest_pity", []string{"user_id"}},
	// ★ 2026-10-16 补漏：开箱流水（2026-10-16 新增的 ezfy_chest_logs）有 user_id，属玩家数据，
	//   删号必须一并清掉（漏表护栏 TestEzfyPlayerPurgeCoversAllTables 报出来的）。
	{"ezfy_chest_logs", []string{"user_id"}},
	{"ezfy_user_tech", []string{"user_id"}},
	{"ezfy_sign", []string{"user_id"}},
	{"ezfy_treasure_sign", []string{"user_id"}},
	{"ezfy_gift", []string{"user_id"}},
	{"ezfy_love_card", []string{"user_id"}},
	{"ezfy_task", []string{"user_id"}},
	{"ezfy_notice", []string{"user_id"}},
	{"ezfy_recruit", []string{"user_id"}},
	{"ezfy_preset", []string{"user_id"}},
	{"ezfy_map_star", []string{"user_id"}},
	// 埋点 / 流水
	{"ezfy_convene_log", []string{"user_id"}},
	{"ezfy_diamond_logs", []string{"user_id"}},
	// 社交：好友 / 申请 / 访问记录 / 世界频道 / 军团频道 / 邮件 / 家信
	{"ezfy_friend", []string{"user_id", "friend_id"}},
	{"ezfy_friend_apply", []string{"user_id", "target_id"}},
	{"ezfy_visit_log", []string{"target_uid", "visitor_uid"}},
	{"ezfy_chat", []string{"user_id"}},
	{"ezfy_corps_chat", []string{"user_id"}},
	{"ezfy_mail", []string{"sender_id", "receiver_id"}},
	// ★ 2026-10-18 邮件附件（风云榜奖励/系统发放）：user_id 冗余存收件人，按它一把清
	{"ezfy_mail_attach", []string{"user_id"}},
	{"private_messages", []string{"sender_id", "receiver_id"}},
	// 军团：成员关系 / 入团申请 / 军团商城日志 / 本人发起的军团宣战
	// （军团**本体**由 ezfyPurgeCorpsOf 整体解散，见下）
	{"ezfy_corps_member", []string{"user_id"}},
	{"ezfy_corps_apply", []string{"user_id"}},
	{"ezfy_corps_mall_log", []string{"user_id"}},
	{"ezfy_corps_war", []string{"atk_user_id"}},
	// 交易行 / 风云榜
	{"ezfy_exchange", []string{"seller_id", "buyer_id"}},
	{"ezfy_pvp_week", []string{"user_id"}},
	{"ezfy_pvp_foe", []string{"user_id", "foe_id"}},
	{"ezfy_pvp_exclude", []string{"user_id"}},
	// ★ 2026-10-18 风云榜奖励发放记录（(week,user_id) 唯一）
	{"ezfy_pvp_reward", []string{"user_id"}},
	// ★ 2026-10-16 补漏：风云榜「我的战绩」场次明细（2026-10-16 新增的 ezfy_pvp_detail）——
	//   漏表护栏 TestEzfyPlayerPurgeCoversAllTables 报出来的。
	//   ★ 只按 user_id 清（**不清 foe_id**）：这张表是「谁的战绩明细」，按 foe_id 清会把
	//     **别人**的历史记录删掉；对手名在表里已是快照(foe_nick)，留着不影响。
	{"ezfy_pvp_detail", []string{"user_id"}},
	// ★ 2026-10-17 统帅称号：玩家持有的称号 + 每日冠军结算记录（都是玩家数据，删号必须清；
	//   称号**定义**表 ezfy_title 是全局配置，见 ezfyPurgeGlobalTables）
	{"ezfy_title_grant", []string{"user_id"}},
	{"ezfy_title_daily", []string{"user_id"}},
	// ★ 2026-10-17 人机模块：人机档案 / 攻击计划（两个方向）/ 行为日志，删号必须一起清
	{"ezfy_bot", []string{"user_id"}},
	// ★ 2026-10-28 人机多城市：每座城的行动槽（bot_uid 归属，删号必清）
	{"ezfy_bot_city", []string{"bot_uid"}},
	{"ezfy_bot_target", []string{"bot_uid", "target_uid"}},
	{"ezfy_bot_log", []string{"bot_uid"}},
	// ★ 2026-10-30 人机保护名单（按玩家 user_id）：删号必清，否则同一 uid 重建角色仍被保护
	{"ezfy_bot_peace", []string{"user_id"}},
	// ★ 2026-10-25 活动守将「兜底被俘次数」进度（玩家 × 守将的胜场数）——纯玩家数据，删号必清
	{"ezfy_act_capture_try", []string{"user_id"}},
	// ★ 2026-10-28 活动守将「可抓次数」抓取流水（玩家 × 守将，只增不删）——纯玩家数据，删号必清。
	//   不清的话：同一个 uid 重新注册/重建角色后，会继承「已经抓过」的次数而抓不到守将。
	{"ezfy_act_capture_log", []string{"user_id"}},
}

// ezfyPurgeByCity 按**城市**归属的表（占位值 = 该玩家的城池 id 列表）。
var ezfyPurgeByCity = []ezfyPurgeRule{
	{"ezfy_city_building", []string{"city_id"}},
	{"ezfy_city_troop", []string{"city_id"}},
	{"ezfy_city_tech", []string{"city_id"}},
	{"ezfy_city_target", []string{"city_id"}},
	{"ezfy_city_effect", []string{"city_id"}},
	{"ezfy_train_queue", []string{"city_id"}},
	{"ezfy_wildland", []string{"city_id"}},
	{"ezfy_wounded", []string{"city_id"}},
	{"ezfy_officer", []string{"city_id"}},   // ★ 软删除表 → 这里 raw DELETE 硬删
	{"ezfy_map_area", []string{"owner_id"}}, // owner_id = 占领城市ID（野地/寇城快照）
	{"ezfy_order", []string{"city_id"}},
	{"ezfy_report", []string{"city_id"}},
	{"ezfy_battle", []string{"city_id"}},
	{"ezfy_occupy", []string{"city_id", "atk_city_id"}},
	{"ezfy_ransom", []string{"city_id"}},
	{"ezfy_equipment", []string{"city_id"}},
	{"ezfy_exchange", []string{"city_id"}},
}

// ezfyPurgeSpecialTables 不在上面两张清单、但**已由专门逻辑处理**的表（漏表护栏用）。
var ezfyPurgeSpecialTables = []string{
	"ezfy_corps",          // 军团长 → ezfyPurgeCorpsOf 整体解散（按 corps_id 清）
	"ezfy_corps_relation", // 军团外交关系：只有 corps_id / target_corps_id，随军团解散一起清
}

// ezfyPurgeGlobalTables 与玩家无关的表（全局配置 / 全局活动 / 地图瓦片），删号不碰（漏表护栏用）。
var ezfyPurgeGlobalTables = []string{
	// 配置表
	"ezfy_cfg_building", "ezfy_cfg_building_level", "ezfy_cfg_troop", "ezfy_cfg_tech",
	"ezfy_cfg_tech_level", "ezfy_cfg_wildland", "ezfy_cfg_item", "ezfy_cfg_task_type",
	"ezfy_cfg_task", "ezfy_cfg_general", "ezfy_cfg_skill", "ezfy_cfg_equipment",
	"ezfy_cfg_equip_set", "ezfy_cfg_chest", "ezfy_cfg_chest_item", "ezfy_cfg_scheme",
	"ezfy_cfg_general_skill", // ★ 2026-10-23 名将组合技能池（全局配置，与玩家无关）
	"ezfy_cfg_resource", "ezfy_cfg_rank", "ezfy_cfg_limit", "ezfy_word_filter",
	"ezfy_cfg_battle_formula", // ★ 2026-10-23 战斗公式套（全服配置，与玩家无关）
	// 全局玩法 / 地图
	"ezfy_act_wild",     // 活动野地（全服共享的配置）
	"ezfy_act_level",    // 活动目标等级配置（全服共享的配置）
	"ezfy_activity",     // 节日活动
	"ezfy_corps_mall",   // 军团商城**配置**（玩家购买流水是 ezfy_corps_mall_log，已清）
	"ezfy_exchange_tpl", // 交易行挂单模板
	"ezfy_map_tile",     // 地图瓦片（全服共享地形）
	"ezfy_title",        // 统帅称号**定义**（管理端维护的全局配置，与玩家无关）
	"ezfy_bot_lock",     // 人机 tick 租约锁（单行，与玩家无关）
	"settings",          // 二战系统设置（全服一份）
}

// ezfyPurgeCorpsOf 解散该玩家**创建**的军团：清成员/入团申请/军团聊天/军团商城日志/
// 军团关系/军团战，最后删军团本体。其他成员随之恢复「无军团」。
func (h *EzfyAdmin) ezfyPurgeCorpsOf(tx *gorm.DB, uid uint) error {
	var corpsIds []uint
	if err := tx.Model(&model.EzfyCorps{}).Where("leader_user_id = ?", uid).Pluck("id", &corpsIds).Error; err != nil {
		return fmt.Errorf("ezfy_corps: %w", err)
	}
	if len(corpsIds) == 0 {
		return nil
	}
	for _, t := range []string{"ezfy_corps_member", "ezfy_corps_apply", "ezfy_corps_chat", "ezfy_corps_mall_log"} {
		if err := tx.Exec("DELETE FROM "+t+" WHERE corps_id IN ?", corpsIds).Error; err != nil {
			return fmt.Errorf("%s: %w", t, err)
		}
	}
	if err := tx.Exec("DELETE FROM ezfy_corps_relation WHERE corps_id IN ? OR target_corps_id IN ?",
		corpsIds, corpsIds).Error; err != nil {
		return fmt.Errorf("ezfy_corps_relation: %w", err)
	}
	if err := tx.Exec("DELETE FROM ezfy_corps_war WHERE atk_corps_id IN ? OR def_corps_id IN ?",
		corpsIds, corpsIds).Error; err != nil {
		return fmt.Errorf("ezfy_corps_war: %w", err)
	}
	return tx.Exec("DELETE FROM ezfy_corps WHERE id IN ?", corpsIds).Error
}

// ezfyPurgePlayerData 把该玩家在二战库的全部数据**硬删**（单事务，任一步失败整体回滚）。
//
// uid = 玩家的 user_id（= 家园号）；cityIds = 该玩家名下城池 id（调用方先查好）。
// 返回受影响行数合计（管理端提示用）。
func (h *EzfyAdmin) ezfyPurgePlayerData(uid uint, cityIds []int64) (int64, error) {
	var total int64
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// ① 军团先解散（军团本体 + 连带行）
		if err := h.ezfyPurgeCorpsOf(tx, uid); err != nil {
			return err
		}
		// ② 按玩家
		for _, r := range ezfyPurgeByUid {
			res := tx.Exec("DELETE FROM "+r.Table+" WHERE "+ezfyPurgeEqCond(r.Cols),
				ezfyPurgeArgs(uid, len(r.Cols))...)
			if res.Error != nil {
				return fmt.Errorf("%s: %w", r.Table, res.Error)
			}
			total += res.RowsAffected
		}
		// ③ 按城市（没有城就跳过，避免 `IN (NULL)` 之类的空条件）
		if len(cityIds) > 0 {
			for _, r := range ezfyPurgeByCity {
				res := tx.Exec("DELETE FROM "+r.Table+" WHERE "+ezfyPurgeInCond(r.Cols),
					ezfyPurgeArgs(cityIds, len(r.Cols))...)
				if res.Error != nil {
					return fmt.Errorf("%s: %w", r.Table, res.Error)
				}
				total += res.RowsAffected
			}
		}
		return nil
	})
	return total, err
}

// ezfyPurgeEqCond `a = ? OR b = ?`（列数 1 时退化成单条件）。
func ezfyPurgeEqCond(cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, c+" = ?")
	}
	return strings.Join(parts, " OR ")
}

// ezfyPurgeInCond `a IN ? OR b IN ?`（列数 1 时退化成单条件）。
func ezfyPurgeInCond(cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, c+" IN ?")
	}
	return strings.Join(parts, " OR ")
}

// ezfyPurgeArgs 把同一个值重复 n 次（与 ezfyPurgeEqCond / ezfyPurgeInCond 的占位符一一对应）。
func ezfyPurgeArgs(v interface{}, n int) []interface{} {
	args := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		args = append(args, v)
	}
	return args
}
