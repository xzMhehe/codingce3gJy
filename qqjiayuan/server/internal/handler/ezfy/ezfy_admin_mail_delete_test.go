package ezfy

import (
	"strings"
	"testing"
)

// ============ 管理端「系统邮件」删除护栏 ============
//
// 用户需求（2026-10-25 原话）：
//
//	「系统邮件 可以删除，有时候可能发错了、或者发错道具了要能一起删除」
//
// 这个功能有两条**不能被后续改动破坏**的硬约束，这里各加一条断言：
//
//  1. **只删系统邮件**：删除语句必须带 `sender_id = 0`。
//     少了它，管理端就能把玩家之间互发的邮件物理删掉（不可恢复）。
//  2. **邮件和附件一起删**：`ezfy_mail_attach` 与 `ezfy_mail` 必须在同一个删除流程里清掉。
//     只删邮件会留下附件孤儿行（`MailClaim` 按 mail_id 查，孤儿行不会再被任何路径读到，
//     但删号清理 / 对账会把它当真实数据）。
func TestAdminMailDeleteOnlySystemAndWithAttach(t *testing.T) {
	src := rawFile(t, "ezfy_admin_mail.go")
	body := funcBody(src, "func (a *EzfyAdmin) AdminEzfyMailSystemDelete")
	if body == "" {
		t.Fatal("找不到 AdminEzfyMailSystemDelete：系统邮件删除接口被改名/删除了？")
	}
	// 1) 删 ezfy_mail 的那一行必须自己带 sender_id = 0（不能依赖外层 where 变量，避免以后被重构掉）
	mailDel := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "Delete(&model.EzfyMail{})") {
			mailDel = line
			break
		}
	}
	if mailDel == "" {
		t.Fatal("AdminEzfyMailSystemDelete 里没有删除 ezfy_mail 的语句")
	}
	if !strings.Contains(mailDel, "sender_id = 0") {
		t.Fatalf("删除 ezfy_mail 的语句没带 sender_id = 0（会把玩家互发的邮件也删掉）：%s", strings.TrimSpace(mailDel))
	}
	// 2) 附件必须一起删
	if !strings.Contains(body, "Delete(&model.EzfyMailAttach{})") {
		t.Fatal("AdminEzfyMailSystemDelete 没有删除 ezfy_mail_attach：会留下附件孤儿行")
	}
	// 3) 事务包裹（附件与邮件不能一半成功一半失败）
	if !strings.Contains(body, ".Begin()") || !strings.Contains(body, ".Commit()") {
		t.Fatal("AdminEzfyMailSystemDelete 没用事务：可能出现「邮件没了、附件还在」")
	}
}

// TestAdminMailDeleteRouteRegistered 路由必须挂在系统邮件同一权限点下。
func TestAdminMailDeleteRouteRegistered(t *testing.T) {
	src := rawFile(t, "../../router/router.go")
	line := ""
	for _, l := range strings.Split(src, "\n") {
		if strings.Contains(l, `"/ezfy-mails/system/delete"`) {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatal("router.go 未注册 POST /admin/ezfy-mails/system/delete")
	}
	if !strings.Contains(line, "module:ezfyWords") {
		t.Fatalf("删除接口的权限点应与列表/发送一致（module:ezfyWords）：%s", strings.TrimSpace(line))
	}
}
