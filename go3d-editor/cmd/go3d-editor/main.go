// go3d-editor：SolidWorks 风格模型树 + Blender 风格视口的 3D 建模编辑器。
package main

import (
	"flag"
	"fmt"
	"os"

	"go2dgame/engine"
	"go3deditor/app"
	"go3deditor/ui"
)

// gameAdapter 把 Editor 适配为 engine.Game。
type gameAdapter struct {
	ed *app.Editor
}

func (g *gameAdapter) Update(dt float64) {
	g.ed.Update(dt)
}

func (g *gameAdapter) Draw(c *engine.Canvas) {
	g.ed.Draw(c)
}

func main() {
	var (
		res  = flag.String("res", "", "窗口分辨率 WxH（默认 980x620）")
		load = flag.String("load", "", "启动时加载场景 JSON 文件")
	)
	flag.Parse()

	w, h := 1100, 680
	if *res != "" {
		var rw, rh int
		if _, err := fmt.Sscanf(*res, "%dx%d", &rw, &rh); err == nil && rw > 0 && rh > 0 {
			w, h = rw, rh
		} else {
			fmt.Fprintln(os.Stderr, "bad res:", *res)
			os.Exit(2)
		}
	}

	// 中文字体（系统 NotoSansCJK）
	fontPaths := []string{
		"/system/fonts/NotoSansCJK-Regular.ttc",
		"/system/fonts/NotoSansCJKsc-Regular.otf",
		"/system/fonts/DroidSansFallback.ttf",
	}
	loaded := false
	for _, p := range fontPaths {
		if err := ui.LoadCJK(p, 15); err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		fmt.Println("[ui] 警告：未找到中文字体，界面将使用 ASCII 位图字体")
	}

	cfg := engine.DefaultConfig("go3d-editor")
	cfg.Width, cfg.Height = w, h
	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}
	defer eng.Close()

	ed := app.New(w, h)
	ed.SetEngine(eng)
	// 注册插件（扩展点：未来深入发展在此追加）
	app.RegisterPlugin(app.StatsPlugin{})
	if *load != "" {
		if err := ed.Load(*load); err != nil {
			// 加载失败仅警告，不退出（场景文件可能被清理，编辑器照常可用）
			fmt.Fprintln(os.Stderr, "load warning:", err)
		}
	}

	if err := eng.Run(&gameAdapter{ed: ed}); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
