// lipics_test.go - 立绘加载验证
package main

import (
	"testing"

	"go2dgame/engine"
)

func TestLoadLipics(t *testing.T) {
	// go test 的 cwd 是包目录（cmd/jianghu-rpg），相对路径直接可用
	g := &Game{lipics: map[string]*engine.Sprite{}}
	g.loadLipics()
	if len(g.lipics) != 7 {
		t.Fatalf("应加载 7 张立绘，实际 %d", len(g.lipics))
	}
	ids := []string{"hero", "qing", "luoyao", "yueli", "xiaoman", "suxue", "zhuzhai"}
	for _, id := range ids {
		spr, ok := g.lipics[id]
		if !ok {
			t.Errorf("缺少立绘 %s", id)
			continue
		}
		if spr.W == 0 || spr.H == 0 {
			t.Errorf("%s 立绘尺寸异常 %dx%d", id, spr.W, spr.H)
		}
		t.Logf("%s: %dx%d", id, spr.W, spr.H)
	}
	// Resize 不崩溃且尺寸正确
	small := g.lipics["qing"].Resize(220, 330)
	if small.W != 220 || small.H != 330 {
		t.Errorf("resize 尺寸错误 %dx%d", small.W, small.H)
	}
	// 检查 Resize 后有不透明像素（图片确实有内容）
	opaque := 0
	for i := 3; i < len(small.Pixels); i += 4 {
		if small.Pixels[i] > 0 {
			opaque++
		}
	}
	if opaque < 1000 {
		t.Errorf("缩略图内容太少，不透明像素仅 %d", opaque)
	}
	t.Logf("resize 不透明像素: %d", opaque)
}
