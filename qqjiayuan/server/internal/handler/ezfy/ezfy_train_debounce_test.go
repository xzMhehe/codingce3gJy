package ezfy

import (
	"strings"
	"testing"
	"time"
)

// ============ 训练页 [开始训练] 防重复提交 守卫（2026-10-25）============
//
// 用户需求（原话）：「开始训练 改成 [开始训练] 并且去除按钮样式，然后前端做好防抖、
// 后端做好多点卡控，加个1s延迟吧」。
//
// 这条改动很容易被后人「优化」掉（比如觉得 1 秒窗口多余、或者把 a 改回 button），
// 所以把三条口径钉死：
//  1. 前端：训练页那个控件必须是文字链接（不得再是 <button>），文案带方括号；
//  2. 前端：doTrainPre 必须先判 trainBusy（防抖），且请求结束后 1 秒才解锁；
//  3. 后端：Train handler 在扣资源/建队列之前做 1 秒请求级去重，
//     trainTroop 在拿到城市锁之后再读 DB 复核一次（跨进程/多机同库兜底）。

func TestTrainPreButtonIsPlainLink(t *testing.T) {
	mod := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyArmy.vue"))
	i := strings.Index(mod, "ezfy.doTrainPre()")
	if i < 0 {
		t.Fatal("EzfyArmy.vue 里找不到 ezfy.doTrainPre() 的点击点（训练页 [开始训练] 被改名/删除了？）")
	}
	lo := i - 500
	if lo < 0 {
		lo = 0
	}
	seg := mod[lo : i+200]
	if strings.Contains(seg, "<button") {
		t.Fatalf("训练页 [开始训练] 必须去掉按钮样式（<button>），改成 <a href=\"javascript:;\">文字链接：\n%s", seg)
	}
	if !strings.Contains(seg, "[{{ ezfy.trainMode === 'defence' ? '开始建造' : '开始训练' }}]") {
		t.Fatalf("训练页控件文案必须带方括号（[开始训练] / [开始建造]）：\n%s", seg)
	}
}

func TestTrainPreFrontendDebounce(t *testing.T) {
	vue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	body := ezfyFuncBody(t, "../../../../web/src/views/ezfy/Ezfy.vue", "doTrainPre () {")
	if !strings.Contains(body, "if (this.trainBusy) return") {
		t.Fatalf("doTrainPre 缺少防抖：连点/请求未返回时必须直接 return（this.trainBusy）\n%s", body)
	}
	if !strings.Contains(body, "this.trainBusy = true") {
		t.Fatalf("doTrainPre 没有把 trainBusy 置为 true，防抖形同虚设\n%s", body)
	}
	// 1 秒冷却：setTimeout(..., 1000) 解锁（与后端 1 秒窗口口径一致）
	if !strings.Contains(body, "1000") || !strings.Contains(body, "this.trainBusy = false") {
		t.Fatalf("doTrainPre 缺少 1 秒冷却解锁（setTimeout → trainBusy = false）\n%s", body)
	}
	if !strings.Contains(vue, "trainBusy: false") {
		t.Fatal("Ezfy.vue 的 data 里缺少 trainBusy 初始值（模板 :class 会读到 undefined）")
	}
}

func TestTrainHandlerThrottle(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) Train(c *gin.Context) {")
	hit := strings.Index(body, "ezfyTrainDupHit(")
	if hit < 0 {
		t.Fatalf("Train handler 缺少 1 秒请求级去重（ezfyTrainDupHit）：连点会下两条征兵命令\n%s", body)
	}
	call := strings.Index(body, "h.trainTroop(")
	if call < 0 {
		t.Fatalf("Train handler 里找不到 h.trainTroop 调用\n%s", body)
	}
	if hit > call {
		t.Fatal("1 秒去重必须放在 h.trainTroop 之前（在扣资源/建队列前拦下，重复请求不产生副作用）")
	}
	// 人机城必须跳过（后台 tick 驱动，1 秒窗口只会误伤连续造兵）
	if !strings.Contains(body[hit-200:hit+200], "city.IsBot == 0") {
		t.Fatalf("请求级去重必须只对真人城生效（city.IsBot == 0）\n%s", body)
	}
}

func TestTrainLockedRechecksFreshQueue(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) trainTroop(city *model.EzfyCity, troopId, count int, split bool) string {")
	lock := strings.Index(body, "lock.Lock()")
	chk := strings.Index(body, "ezfyTrainQueueTooFresh(")
	if chk < 0 {
		t.Fatalf("trainTroop 缺少落库级复核（ezfyTrainQueueTooFresh）：两机同库时内存锁跨不过去\n%s", body)
	}
	if lock < 0 || chk < lock {
		t.Fatal("落库级复核必须在 lock.Lock() 之后（否则并发两个请求会同时通过）")
	}
	if !strings.Contains(body, "city.IsBot == 0") {
		t.Fatalf("落库级复核必须只对真人城生效（city.IsBot == 0）\n%s", body)
	}
}

// TestTrainDupHitWindow 行为级：1 秒窗口真的按「玩家 + 城市」分别计时。
func TestTrainDupHitWindow(t *testing.T) {
	base := time.Now()
	const uidA, uidB = 990001, 990002
	const cityA, cityB = 880001, 880002
	if ezfyTrainDupHit(uidA, cityA, base) {
		t.Fatal("首次请求不应被拦")
	}
	if !ezfyTrainDupHit(uidA, cityA, base.Add(300*time.Millisecond)) {
		t.Fatal("1 秒内的第二次请求必须被拦（否则连点会下两条征兵命令）")
	}
	if !ezfyTrainDupHit(uidA, cityA, base.Add(999*time.Millisecond)) {
		t.Fatal("1 秒窗口边界内（999ms）必须被拦")
	}
	if ezfyTrainDupHit(uidA, cityA, base.Add(1200*time.Millisecond)) {
		t.Fatal("超过 1 秒必须放行")
	}
	if ezfyTrainDupHit(uidB, cityA, base.Add(200*time.Millisecond)) {
		t.Fatal("不同玩家之间不能互相影响")
	}
	if ezfyTrainDupHit(uidA, cityB, base.Add(200*time.Millisecond)) {
		t.Fatal("同一玩家的不同城市之间不能互相影响")
	}
}
