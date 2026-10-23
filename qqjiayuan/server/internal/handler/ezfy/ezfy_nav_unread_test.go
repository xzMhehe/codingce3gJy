package ezfy

// ★ 2026-10-23 用户规则「聊天、邮箱有未读的也要有军情那样的感叹号」：
//   - 顶部导航「聊天」角标 = 未读私聊（private_messages.receiver_id=我 AND is_read=0）
//   - 顶部导航「邮箱」角标 = 未读邮件（ezfy_mail.receiver_id=我 AND is_read=0 AND recv_del=0）
//
// 本测试锁死「接口 → 路由 → 前端角标」这条链路，防止后续重构把提示改没。

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
			t.Fatalf("顶部导航聊天/邮箱角标缺失（缺 %s）", want)
		}
	}

	chat := readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyChat.vue")
	if !strings.Contains(chat, "去好友页查看") {
		t.Fatal("聊天页缺少「未读私聊 → 好友页」引导入口")
	}
}
