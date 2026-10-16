package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 军事区/资源区「建筑详情页(bdetail)」口径守卫 ============
//
// ★ 2026-10-16 需求（用户两轮纠正后的**最终口径**，别改回去）：
//
//   - 原本**点不动**的建筑（民居/农田/炼钢厂/石油基地/稀矿厂/轻工厂/重工厂/停机坪/航海协会/
//     运输站/雷达站/围墙…）点名字 → **跳转独立页面**（不是原地展开），展示
//     简介 + 当前级加成效果 + 下一级效果；
//   - 原本**有功能页**的建筑（市政厅/城防/科研中心/军校/参谋部/交易所/仓库/司令部/军工厂/
//     联络中心）名字仍**直接进原功能页**，且**不加任何额外按钮/标签**
//     （中途试过在名字后加 `[造兵]` 这类短标签，用户明确否掉：「不用多加按钮，直接原来那样」）。
func TestBuildDetailPageContract(t *testing.T) {
	read := func(rel string) string {
		b, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("读前端文件失败 %s: %v", rel, err)
		}
		return string(b)
	}
	shell := read("../../../../web/src/views/ezfy/Ezfy.vue")
	mod := read("../../../../web/src/views/ezfy/modules/EzfyBuild.vue")

	// ① 新页面必须挂在 ezfy-build 模块上，否则 go('bdetail') 落到空白页
	if !strings.Contains(shell, `cur === 'buildm' || cur === 'builds' || cur === 'bdetail'`) {
		t.Fatal("Ezfy.vue 的 ezfy-build 挂载条件缺 `cur === 'bdetail'` → 建筑详情页会白屏")
	}
	// ② 进页必须拉建筑数据（buildInfo 是按 id 从 buildings 现取的）
	if !strings.Contains(shell, `t === 'cityhall' || t === 'bdetail'`) {
		t.Fatal("go() 里进 bdetail 没触发 loadBuildings() → buildInfo 找不到建筑")
	}
	// ③ 刷新要能恢复：建筑 id 写进 URL(bid) + 读回来
	if !strings.Contains(shell, `params.set('bid'`) || !strings.Contains(shell, `params.get('bid')`) {
		t.Fatal("bdetail 的建筑 id 没走 URL(bid) → 刷新后详情页丢失")
	}
	// ④ 点名字是**跳页**，不是原地展开
	if !strings.Contains(shell, `this.go('bdetail')`) {
		t.Fatal("openBuildInfo 必须跳转 bdetail 页（用户要求独立页面，不是原地展开）")
	}
	// ⑤ 原本有功能页的建筑保持原样：名字仍直接进功能页，且没有短标签按钮
	if !strings.Contains(mod, "ezfy.goEntry(b.building_id)") {
		t.Fatal("EzfyBuild.vue 里原本能点的建筑名字没有继续直接进功能页")
	}
	if strings.Contains(shell, "short:") || strings.Contains(mod, "bEntry(b.building_id).short") {
		t.Fatal("又给有功能页的建筑加了短标签按钮 —— 用户明确要求「不用多加按钮，直接原来那样」")
	}
	// ⑥ 详情页三件套：简介 / 当前级效果 / 下一级效果
	if !strings.Contains(mod, `ezfy.cur === 'bdetail'`) {
		t.Fatal("EzfyBuild.vue 里没有 bdetail 页面分支")
	}
	for _, want := range []string{"buildInfo.des", "buildInfo.effect", "buildInfo.next_effect"} {
		if !strings.Contains(mod, want) {
			t.Fatalf("建筑详情页缺 %s（简介 / 当前级效果 / 下一级效果）", want)
		}
	}
}
