package app

import (
	"fmt"
	"time"

	"go2dgame/engine"
	"go3deditor/ui"
)

// uiBtns 按钮命中表（每帧清空重建）。
type uiBtn struct {
	x, y, w, h int
	cb         func()
}

var uiBtns []uiBtn

// drawBtn 绘制按钮并注册命中。
func drawBtn(c *engine.Canvas, x, y, w, h int, label string, active bool, cb func()) {
	col := uiColorBtn
	if active {
		col = uiColorAccent
	}
	c.FillRect(x, y, w, h, col)
	c.Rect(x, y, w, h, uiColorBorder)
	tc := uiColorText
	if active {
		tc = engine.Color{R: 255, G: 255, B: 255}
	}
	ui.DrawText(c, x+8, y+(h-ui.LineH())/2+1, label, tc)
	uiBtns = append(uiBtns, uiBtn{x, y, w, h, cb})
}

// drawToolbar 顶部工具栏（两行，28px 按钮，与建模编辑器同风格）。
func (e *Editor) drawToolbar(c *engine.Canvas) {
	c.FillRect(0, 0, c.W, ToolbarH, uiColorPanel)
	c.FillRect(0, ToolbarH-1, c.W, 1, uiColorBorder)
	const bh = 28
	// 第一行
	x := 10
	y := 5
	ui.DrawText(c, x, y+6, "关卡编辑器", uiColorAccent)
	x += ui.TextW("关卡编辑器") + 14
	sw := ui.TextW("保存") + 14
	lw := ui.TextW("载入") + 14
	drawBtn(c, x, y, ui.TextW("载入建模JSON")+14, bh, "载入建模JSON", false, func() { e.importModelDialog() })
	x += ui.TextW("载入建模JSON") + 20
	drawBtn(c, x, y, ui.TextW("载入OBJ模型")+14, bh, "载入OBJ模型", false, func() { e.importOBJModel() })
	x += ui.TextW("载入OBJ模型") + 20
	drawBtn(c, x, y, ui.TextW("载入素材图")+14, bh, "载入素材图", false, func() { e.importSpriteDialog() })
	x += ui.TextW("载入素材图") + 20
	drawBtn(c, c.W-PropW-10-sw*2-6, y, sw, bh, "保存", false, func() { e.saveDoc() })
	drawBtn(c, c.W-PropW-10-sw, y, lw, bh, "载入", false, func() { e.loadDoc() })
	// 第二行
	y = 41
	x = 10
	tmodes := []struct {
		label string
		m     int
	}{
		{"移动[G]", 0}, {"旋转[R]", 1}, {"缩放[S]", 2},
	}
	for _, b := range tmodes {
		w := ui.TextW(b.label) + 14
		drawBtn(c, x, y, w, bh, b.label, e.mode == b.m, func() { e.mode = b.m; e.drag = -1 })
		x += w + 6
	}
	x += 8
	// 资源快速添加（模型）
	for i, m := range e.level.Models {
		w := ui.TextW(m.Name) + 12
		if x+w > c.W-PropW-320 {
			break
		}
		drawBtn(c, x, y, w, bh, "＋"+m.Name, e.selRes == i, func() { e.AddInstance(i) })
		x += w + 6
	}
	// 运行按钮
	playLabel := "▶ 运行[F5]"
	if e.playing {
		playLabel = "■ 编辑[Esc]"
	}
	drawBtn(c, c.W-PropW-10-ui.TextW(playLabel)-14, y, ui.TextW(playLabel)+14, bh, playLabel, e.playing, func() { e.TogglePlay() })
	ui.DrawText(c, c.W-PropW-10-ui.TextW(playLabel)-16-ui.TextW("F1 帮助"), y+7, "F1 帮助", uiColorDim)
}

// drawLeft 左侧面板：资源列表 + 场景树（行高 30，与建模编辑器模型树一致）。
func (e *Editor) drawLeft(c *engine.Canvas) {
	c.FillRect(0, ToolbarH, TreeW, c.H-ToolbarH, uiColorPanel2)
	c.FillRect(TreeW-1, ToolbarH, 1, c.H-ToolbarH, uiColorBorder)
	const rowH = 30
	// 资源列表
	y := ToolbarH + 10
	ui.DrawText(c, 10, y, "模型资源", uiColorAccent)
	y += 26
	for i, m := range e.level.Models {
		rowY := y + i*rowH
		col := uiColorText
		if e.selRes == i {
			c.FillRect(2, rowY, TreeW-4, rowH, uiColorSel)
		}
		ui.DrawText(c, 14, rowY+8, "▣ "+m.Name, col)
	}
	y += len(e.level.Models)*rowH + 8
	// 素材
	ui.DrawText(c, 10, y, "贴图素材", uiColorAccent)
	y += 26
	for i, s := range e.level.Sprites {
		rowY := y + i*rowH
		ui.DrawText(c, 14, rowY+8, "🖼 "+s.Name, uiColorText)
	}
	y += len(e.level.Sprites)*rowH + 12
	// 场景树
	ui.DrawText(c, 10, y, "场景实例", uiColorAccent)
	y += 28
	for i, inst := range e.level.Insts {
		rowY := y + i*rowH
		if i == e.sel {
			c.FillRect(2, rowY, TreeW-4, rowH, uiColorSel)
		}
		mark := "○"
		if inst.IsPlayer {
			mark = "★"
		}
		if inst.IsSprite {
			mark = "◈"
		}
		ui.DrawText(c, 14, rowY+8, fmt.Sprintf("%s %s", mark, inst.Name), uiColorText)
	}
}

// treeClick 左侧面板点击。
func (e *Editor) treeClick(mx, my int) {
	const rowH = 30
	y := ToolbarH + 10 + 26
	// 模型资源
	for i := range e.level.Models {
		if my >= y+i*rowH && my < y+i*rowH+rowH {
			e.selRes = i
			return
		}
	}
	y += len(e.level.Models)*rowH + 8 + 26
	// 素材：点击 → 添加精灵实例
	for i := range e.level.Sprites {
		if my >= y+i*rowH && my < y+i*rowH+rowH {
			e.AddSpriteInst(i)
			return
		}
	}
	y += len(e.level.Sprites)*rowH + 12 + 28
	// 场景实例
	for i := range e.level.Insts {
		if my >= y+i*rowH && my < y+i*rowH+rowH {
			e.sel = i
			return
		}
	}
}

// drawProps 右侧属性面板（行距与建模编辑器统一）。
func (e *Editor) drawProps(c *engine.Canvas) {
	px := c.W - PropW
	c.FillRect(px, ToolbarH, PropW, c.H-ToolbarH, uiColorPanel2)
	c.FillRect(px, ToolbarH, 1, c.H-ToolbarH, uiColorBorder)
	ui.DrawText(c, px+10, ToolbarH+10, "属性", uiColorAccent)
	if e.sel < 0 || e.sel >= len(e.level.Insts) {
		ui.DrawText(c, px+12, ToolbarH+42, "未选中实例", uiColorDim)
		return
	}
	inst := e.level.Insts[e.sel]
	y := ToolbarH + 36
	ui.DrawText(c, px+12, y, "名称:", uiColorDim)
	ui.DrawText(c, px+80, y, inst.Name, uiColorText)
	y += 30
	ui.DrawText(c, px+12, y, "资源:", uiColorDim)
	resName := "贴图素材"
	if inst.Res != nil {
		resName = inst.Res.Name
	}
	ui.DrawText(c, px+80, y, resName, uiColorText)
	y += 36
	labels := []string{"位置X:", "位置Y:", "位置Z:"}
	pos := []float32{inst.Pos.X, inst.Pos.Y, inst.Pos.Z}
	for i := 0; i < 3; i++ {
		ui.DrawText(c, px+12, y, labels[i], uiColorDim)
		ui.DrawText(c, px+80, y, fmt.Sprintf("%.3f", pos[i]), uiColorText)
		y += 28
	}
	y += 10
	ui.DrawText(c, px+12, y, "旋转°:", uiColorDim)
	ui.DrawText(c, px+80, y, fmt.Sprintf("%.1f %.1f %.1f", inst.RotX*180/3.14159, inst.RotY*180/3.14159, inst.RotZ*180/3.14159), uiColorText)
	y += 30
	ui.DrawText(c, px+12, y, "缩放:", uiColorDim)
	ui.DrawText(c, px+80, y, fmt.Sprintf("%.2f", inst.Scale), uiColorText)
	y += 38
	// 按钮
	drawBtn(c, px+16, y, 100, 28, "设为玩家", false, func() { e.SetPlayer() })
	drawBtn(c, px+124, y, 100, 28, "删除实例", false, func() { e.DeleteInstance() })
	y += 38
	if inst.Res != nil && inst.Res.Anim != nil && !inst.IsSprite {
		playLabel := "播放动画"
		if inst.AnimPlaying {
			playLabel = "暂停动画"
		}
		drawBtn(c, px+16, y, 100, 28, playLabel, inst.AnimPlaying, func() { inst.AnimPlaying = !inst.AnimPlaying })
		drawBtn(c, px+124, y, 100, 28, "重置t=0", false, func() { inst.AnimTime = 0; inst.AnimPlaying = false; if inst.Skel != nil { inst.Res.Anim.ApplyToSkeleton(inst.Skel, 0) } })
	}
	y += 44
	ui.DrawText(c, px+12, y, "按键设置:", uiColorDim)
	y += 28
	for a := ActForward; a <= ActRun; a++ {
		ui.DrawText(c, px+16, y, fmt.Sprintf("%s: %s", ActionName(a), KeyLabel(e.inputMap.Keys[a])), uiColorText)
		y += 27
	}
}

// drawStatus 底部状态栏。
func (e *Editor) drawStatus(c *engine.Canvas) {
	y := c.H - StatusH
	c.FillRect(0, y, c.W, StatusH, uiColorPanel)
	c.FillRect(0, y, c.W, 1, uiColorBorder)
	msg := e.msg
	if timeSince(e.msgT) > 5 {
		msg = ""
	}
	if msg == "" {
		if e.playing {
			msg = "运行模式：WASD 移动 / 空格跳跃 / Shift 加速 / Esc 返回"
		} else {
			msg = "编辑模式：点击选中 / Gizmo 变换 / F5 运行"
		}
	}
	ui.DrawText(c, 10, y+7, msg, uiColorText)
	ui.DrawText(c, c.W-200, y+7, fmt.Sprintf("实例 %d  FPS %d", len(e.level.Insts), e.fps), uiColorDim)
}

// drawHelp F1 帮助。
func (e *Editor) drawHelp(c *engine.Canvas) {
	c.FillRect(c.W/2-240, c.H/2-160, 480, 320, uiColorPanel2)
	c.Rect(c.W/2-240, c.H/2-160, 480, 320, uiColorBorder)
	lines := []string{
		"go3d 关卡编辑器",
		"",
		"载入建模JSON → 模型资源出现在左面板",
		"点击资源名 → 工具栏出现 ＋按钮 添加实例",
		"载入素材图 → 添加为精灵公告板",
		"选中实例 → Gizmo 移动/旋转/缩放 (G/R/S)",
		"属性面板 设为玩家 → F5 运行",
		"运行模式：WASD 移动 空格跳跃 Shift加速 Esc返回",
		"Ctrl+S 保存关卡  Ctrl+O 载入关卡",
	}
	for i, l := range lines {
		ui.DrawText(c, c.W/2-220, c.H/2-140+i*27, l, uiColorText)
	}
}

func timeSince(t time.Time) float64 { return time.Since(t).Seconds() }
