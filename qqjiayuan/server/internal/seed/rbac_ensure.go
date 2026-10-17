package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// EnsureRBACPerms ★★ 2026-10-12 线上事故修复：skip 模式下新菜单权限点不入库。
//
// 事故经过：管理端新增「免战管理」菜单（权限码 module:ezfyPeace）后，超管能看到菜单，
// 点进去却报「你没有权限进入！」。原因是线上 config.yaml 有 `seed.skip: true`
// （多机共享库场景），main.go 因此**整体跳过** seed.Run → seedRBAC 不跑 →
// permissions 表里永远没有 module:ezfyPeace；而超管的菜单是全量展示的
// （前端 /admin/my-perms 返回 super=true），于是「看得见、进不去」。
//
// 所以 skip 分支必须像 EnsureEzfyIndexes 那样把权限点幂等补齐（挂启动链，每次重启都补）。
//
// 补种规则：
//  1. 建齐 modulePerms（清单唯一实现在 seed.go）里缺失的权限点；
//  2. 给「补种前就拥有全部权限」的角色（super_admin / admin 这类全量角色）追加新权限；
//     自定义角色（如只勾了部分权限的「二战」）**不动** —— 由管理员在角色管理里自行勾选，
//     避免悄悄放大权限；
//  3. 内置全量角色（super_admin / admin）**每次启动**都对齐全部权限（只增不删）——
//     存量超管若被角色管理改坏（Replace 少了权限），表现是「能登录进管理端、
//     菜单全亮（前端按角色 code 判 super）、点哪都 403 没数据」，这里每次重启自愈；
//  4. 顺带刷新改过名的权限点（FirstOrCreate 命中不更新）。
//
// 幂等：只 INSERT 缺失行，绝不删除或覆盖已有授权。
func EnsureRBACPerms(db *gorm.DB) {
	// 补种前的权限总数；用它判定「谁是全量角色」（拥有全部权限）
	var before int64
	db.Model(&model.Permission{}).Count(&before)
	type fullRole struct {
		RoleID uint
		C      int64
	}
	var full []fullRole
	if before > 0 {
		db.Table("role_permissions").
			Select("role_id, COUNT(DISTINCT permission_id) AS c").
			Group("role_id").Having("COUNT(DISTINCT permission_id) >= ?", before).
			Scan(&full)
	}

	// 1) 幂等补权限点
	added := 0
	for _, p := range modulePerms {
		var cnt int64
		db.Model(&model.Permission{}).Where("code = ?", p.Code).Count(&cnt)
		if cnt > 0 {
			continue
		}
		if err := db.Create(&p).Error; err == nil {
			added++
			log.Printf("[启动链] RBAC 补权限点 %s（%s / %s）", p.Code, p.Remark, p.Name)
		}
	}
	// 权限名改过的，把老行也刷一遍（与 seedRBAC 保持一致）
	db.Model(&model.Permission{}).Where("code = ?", "module:ezfyBuildLimit").
		Update("name", "风云系统配置")

	// 2) 内置全量角色（super_admin / admin）**每次启动都对齐全部权限**（只增不删）。
	// ★★ 2026-10-17 线上症状「管理端能登录进去，但里面全是没数据」：超管角色是按
	//   roles.code=super_admin 判定全量展示菜单的（MyPerms 只看角色 code），
	//   而页面接口按 role_permissions 逐条校验 —— 一旦超管角色的权限被改少
	//   （角色管理保存是 Replace 全量替换 / 历史补种缺口），就是
	//   「登录能过 → 菜单全亮 → 点哪都 403 → 没数据」。上面第 1 步只在 added>0 时补，
	//   修不了这种「存量被改坏」，所以这里对内置全量角色无条件对齐（幂等 INSERT IGNORE）。
	//   自定义角色（如只勾了部分权限的「二战」「版主」）仍然不动。
	var builtin []struct {
		ID uint
	}
	db.Table("roles").Select("id").Where("code IN ?", []string{"super_admin", "admin"}).Scan(&builtin)
	if len(builtin) > 0 {
		var perms []model.Permission
		db.Find(&perms)
		missingTotal := 0
		for _, r := range builtin {
			var have int64
			db.Table("role_permissions").Where("role_id = ?", r.ID).Count(&have)
			missing := int64(len(perms)) - have
			if missing <= 0 {
				continue
			}
			missingTotal += int(missing)
			for _, p := range perms {
				db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id) VALUES (?, ?)", r.ID, p.ID)
			}
			log.Printf("[启动链] RBAC 内置全量角色 role_id=%d 缺 %d 个权限点，已补齐", r.ID, missing)
		}
		if missingTotal == 0 {
			log.Printf("[启动链] RBAC 内置全量角色(super_admin/admin)权限完整，无需修复")
		}
	}

	// 3) 「补种前就拥有全部权限」的其它全量角色补上新权限（只增不删）
	if added > 0 && len(full) > 0 {
		var perms []model.Permission
		db.Find(&perms)
		for _, r := range full {
			for _, p := range perms {
				db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id) VALUES (?, ?)", r.RoleID, p.ID)
			}
		}
		log.Printf("[启动链] RBAC 已为 %d 个全量角色补上新权限点", len(full))
	}
}
