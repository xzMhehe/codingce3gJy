package ezfy

// ★ 2026-10-23 用户规则「有未读的也要有军情那样的感叹号，不然玩家都不知道来没来」：
//   - 顶部导航「好友」角标 = 未读私聊（private_messages.receiver_id=我 AND is_read=0）
//   - 顶部导航「邮箱」角标 = 未读邮件（ezfy_mail.receiver_id=我 AND is_read=0 AND recv_del=0）
//
// ★★ 2026-10-23 用户二次纠正：「聊天不需要感叹号，是**好友**那块需要」——
//   聊天频道（世界/军团/系统）没有未读概念，私聊入口在「好友 → 会话列表」，
//   所以角标必须挂「好友」，**不能**挂「聊天」。
//
// 本测试锁死「接口 → 路由 → 前端角标」这条链路，防止后续重构把提示改没/挂错位置。

import (
	"strings"
	"testing"
)

func TestNavUnreadBadgesWiring(t *testing.T) {
	router := rawFile(t, "../../router/router.go")
	if !strings.Contains(router, `ezfyG.GET("/unread", ezfyH.NavUnread)`) {
		t.Fatal("缺少顶部导航未读数路由 GET /games/ezfy/unread")
	}

	src := rawFile(t, "ezfy_mail.go")
	for _, want := range []string{
		"func (h *EzfyHandler) NavUnread(",
		`Where("receiver_id = ? AND is_read = 0", uid).Count(&chat)`,
		`Where("receiver_id = ? AND is_read = 0 AND recv_del = 0", uid).Count(&mail)`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("未读数接口口径缺失（缺 %s）", want)
		}
	}

	ui := readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue")
	for _, want := range []string{
		"unreadChat > 0",
		"unreadMail > 0",
		"/games/ezfy/unread",
		"this.loadUnread()",
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("顶部导航未读角标缺失（缺 %s）", want)
		}
	}
	// 未读私聊角标必须挂在「好友」上
	if !strings.Contains(ui, `go('friends')">好友<svg`) {
		t.Fatal("「好友」导航没有未读私聊感叹号（用户：好友那块需要）")
	}
	// 「聊天」不能有角标（用户明确「聊天不需要感叹号」）
	if strings.Contains(ui, `go('chat')">聊天<svg`) {
		t.Fatal("「聊天」导航不该挂感叹号（聊天频道没有未读概念，角标挂「好友」）")
	}
}
