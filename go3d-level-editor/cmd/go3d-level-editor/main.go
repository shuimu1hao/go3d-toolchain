// go3d-level-editor：模型/动画 → 场景搭建/素材/按键设置/运行 的关卡编辑器。
package main

import (
	"flag"
	"fmt"
	"os"

	"go2dgame/engine"
	"go3dlevel/app"
	"go3deditor/ui"
)

// gameAdapter 把 Editor 适配为 engine.Game。
type gameAdapter struct {
	ed *app.Editor
}

func (g *gameAdapter) Update(dt float64) { g.ed.Update(dt) }
func (g *gameAdapter) Draw(c *engine.Canvas) { g.ed.Draw(c) }

func main() {
	var (
		res  = flag.String("res", "", "窗口分辨率 WxH（默认 1100x680）")
		load = flag.String("load", "", "启动时加载关卡 JSON 文件")
	)
	flag.Parse()

	w, h := 1280, 780
	if *res != "" {
		var rw, rh int
		if _, err := fmt.Sscanf(*res, "%dx%d", &rw, &rh); err == nil && rw > 0 && rh > 0 {
			w, h = rw, rh
		}
	}
	// 窗口尺寸 clamp 到实际屏幕（与建模编辑器一致，避免右端按钮被裁剪）
	if sw, sh, ok := engine.ScreenSize(); ok && sw > 0 && sh > 0 {
		if w > sw {
			w = sw
		}
		if h > sh {
			h = sh
		}
		fmt.Printf("[main] 屏幕 %dx%d，窗口 %dx%d\n", sw, sh, w, h)
	}

	// 中文字体（系统 NotoSansCJK）
	fontPaths := []string{
		"/system/fonts/NotoSansCJK-Regular.ttc",
		"/system/fonts/NotoSansCJKsc-Regular.otf",
		"/system/fonts/DroidSansFallback.ttf",
	}
	loaded := false
	for _, p := range fontPaths {
		if err := ui.LoadCJK(p, 18); err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		fmt.Println("[ui] 警告：未找到中文字体，界面将使用 ASCII 位图字体")
	}

	cfg := engine.DefaultConfig("go3d-level-editor")
	cfg.Width, cfg.Height = w, h
	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}
	defer eng.Close()

	ed := app.New(w, h)
	ed.SetEngine(eng)
	if *load != "" {
		if err := ed.Load(*load); err != nil {
			fmt.Fprintln(os.Stderr, "load:", err)
			os.Exit(1)
		}
	}

	if err := eng.Run(&gameAdapter{ed: ed}); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
