// dialog.go - 对话系统（RPG Maker 风格对话框 + 选项 + 好感度）
package main

import (
	"go2dgame/engine"
)

// 对话框区域（960x540 逻辑坐标），Update 触屏判定和 Draw 渲染共用。
const (
	dialogDX, dialogDY = 40, 392
	dialogDW, dialogDH = 880, 136
)

// Choice 对话选项（文本/好感度影响）。
type Choice struct {
	Text   string
	Love   map[string]int
	Next   string
	Action string // give / join / end / nothing
	Param  string
}

// DialogLine 对话行（说话人 + 文本）。
type DialogLine struct {
	Speaker string // girl id / "hero" / "" 旁白
	Text    string
	Choices []Choice
	Action  string
	Param   string
}

// Dialog 对话状态机。
type Dialog struct {
	active  bool
	lines   []DialogLine
	idx     int
	charIdx int
	charT   float64
	typing  bool
	choices []Choice
	sel     int
	curID   string
	girl    *Girl // 当前对话女主
}

// Start 用静态对话树开始对话。
func (d *Dialog) Start(g *Game, id string) {
	lines, ok := dialogTree[id]
	if !ok {
		lines = []DialogLine{{Speaker: "", Text: "（此处无人应答）"}}
	}
	d.active = true
	d.lines = lines
	d.idx = 0
	d.charIdx = 0
	d.charT = 0
	d.typing = true
	d.choices = nil
	d.curID = id
	d.girl = nil
	g.mode = "dialog"
}

// StartGirl 开始与女主对话（动态生成）。
func (d *Dialog) StartGirl(g *Game, gr *Girl) {
	d.active = true
	d.girl = gr
	d.lines = g.girlDialogLines(gr)
	d.idx = 0
	d.charIdx = 0
	d.charT = 0
	d.typing = true
	d.choices = nil
	d.curID = "girl_" + gr.ID
	g.mode = "dialog"
}

// Update 对话推进。
func (d *Dialog) Update(g *Game, dt float64) {
	in := g.eng.Input()
	if d.typing {
		d.charT += dt
		if d.charT >= 0.028 {
			d.charT = 0
			d.charIdx += 2
			cur := d.lines[d.idx]
			if d.charIdx >= len([]rune(cur.Text)) {
				d.charIdx = len([]rune(cur.Text))
				d.typing = false
			}
		}
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			d.charIdx = len([]rune(d.lines[d.idx].Text))
			d.typing = false
		}
		return
	}
	cur := d.lines[d.idx]
	if len(cur.Choices) > 0 {
		// 触屏：点击选项行直接选中（行布局与 Draw 一致：dialogDY+10 + i*行距）
		if in.MouseLeftPressed {
			rowY := dialogDY + 10
			gap := lineH() + 4
			if i := hitRow(in.MouseX, in.MouseY, dialogDX, rowY, gap, gap, len(cur.Choices)); i >= 0 {
				d.sel = i
				d.applyChoice(g, cur.Choices[i])
				return
			}
		}
		// 选项模式
		if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
			d.sel = (d.sel + len(cur.Choices) - 1) % len(cur.Choices)
		}
		if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
			d.sel = (d.sel + 1) % len(cur.Choices)
		}
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			d.applyChoice(g, cur.Choices[d.sel])
		}
		return
	}
	if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
		d.advance(g)
	}
}

func (d *Dialog) advance(g *Game) {
	d.idx++
	if d.idx >= len(d.lines) {
		d.finish(g)
		return
	}
	d.charIdx = 0
	d.typing = true
	d.choices = nil
	cur := d.lines[d.idx]
	if len(cur.Choices) > 0 {
		d.choices = cur.Choices
		d.sel = 0
	}
}

func (d *Dialog) applyChoice(g *Game, ch Choice) {
	// 好感度变化
	if ch.Love != nil {
		for id, v := range ch.Love {
			gr := findGirl(id)
			if gr != nil {
				gr.Love += v
				if gr.Love < 0 {
					gr.Love = 0
				}
				if gr.Love > 100 {
					gr.Love = 100
				}
				g.player.GirlLove[id] = gr.Love
				if v > 0 {
					g.msg = gr.Name + " 好感度 +" + itoa(v) + "！"
				} else {
					g.msg = gr.Name + " 好感度 " + itoa(v)
				}
				g.msgT = 1.8
			}
		}
	}
	switch ch.Action {
	case "give":
		g.giveGift(d.girl)
		return // give 打开送礼菜单，不推进
	case "join":
		if d.girl != nil {
			d.girl.JoinBattle = true
			g.player.JoinedGirls[d.girl.ID] = true
			g.msg = d.girl.Name + " 加入了队伍！战斗中可召唤支援"
			g.msgT = 2.5
		}
	case "end":
		d.finish(g)
		return
	case "event_end":
		if d.girl != nil {
			d.girl.Event++
			g.player.GirlEvents[d.girl.ID] = d.girl.Event
		}
		d.finish(g)
		return
	case "ending":
		g.battle.endingText = ch.Param
		g.mode = "ending"
		return
	}
	if ch.Next != "" {
		d.Start(g, ch.Next)
	} else {
		d.finish(g)
	}
}

func (d *Dialog) finish(g *Game) {
	d.active = false
	g.mode = "map"
}

// giveGift 打开送礼选择。
func (g *Game) giveGift(gr *Girl) {
	g.startGift(gr)
}

// Draw 对话框渲染。
func (d *Dialog) Draw(c *engine.Canvas, g *Game) {
	if !d.active {
		return
	}
	cur := d.lines[d.idx]
	// 说话者大立绘（右侧）
	spk0 := cur.Speaker
	var lip *engine.Sprite
	if spk0 == "hero" {
		lip = g.lipics["hero"]
	} else if spk0 == "zhuzhai" {
		lip = g.lipics["zhuzhai"]
	} else if spk0 != "" {
		if gr := findGirl(spk0); gr != nil {
			lip = g.lipics[gr.ID]
		}
	}
	if lip != nil {
		c.FillRect(646, 50, 272, 340, engine.Color{R: 12, G: 12, B: 20})
		c.Rect(646, 50, 272, 340, engine.Color{R: 100, G: 100, B: 130})
		lip.DrawResized(c, 672, 56, 220, 328)
	}
	// 底部对话框
	dx, dy, dw, dh := dialogDX, dialogDY, dialogDW, dialogDH
	c.FillRect(dx, dy, dw, dh, engine.Color{R: 18, G: 18, B: 30})
	c.Rect(dx, dy, dw, dh, engine.Color{R: 255, G: 255, B: 255})
	// 名字框
	spk := cur.Speaker
	name := ""
	port := (*engine.Sprite)(nil)
	if spk == "hero" {
		name = g.player.Name
		port = g.portraits["hero"]
	} else if spk != "" {
		gr := findGirl(spk)
		if gr != nil {
			name = gr.Name
			port = g.portraits[spk]
		} else if spk == "npc" {
			name = "客栈老板娘"
			port = g.portraits["npc"]
		}
	}
	if name != "" {
		nw := textW(name) + 24
		c.FillRect(dx+6, dy-26, nw, 26, engine.Color{R: 220, G: 190, B: 120})
		c.Rect(dx+6, dy-26, nw, 26, engine.Color{R: 255, G: 240, B: 200})
		drawText(c, dx+16, dy-21, name, engine.Color{R: 60, G: 40, B: 20})
	}
	// 头像
	tx := dx + 14
	if port != nil {
		port.DrawScaled(c, dx+12, dy+12, 2)
		tx = dx + 90
	}
	// 文本（打字机）
	text := cur.Text
	runes := []rune(text)
	if d.typing && d.charIdx < len(runes) {
		text = string(runes[:d.charIdx])
	}
	// 自动折行（每行最多 22 个中文字符 ≈ 22*20=440px）
	lines := wrapText(text, 22)
	ty := dy + 14
	for i, ln := range lines {
		if i > 2 {
			break
		}
		drawTextShadow(c, tx, ty, ln, engine.Color{R: 250, G: 250, B: 240})
		ty += lineH() + 4
	}
	// 选项
	if !d.typing && len(cur.Choices) > 0 {
		oy := dy + 10
		for i, ch := range cur.Choices {
			col := engine.Color{R: 220, G: 220, B: 240}
			if i == d.sel {
				col = engine.Color{R: 255, G: 230, B: 120}
				drawTextShadow(c, dx+14, oy, "▶ "+ch.Text, col)
			} else {
				drawTextShadow(c, dx+26, oy, "  "+ch.Text, col)
			}
			oy += lineH() + 4
		}
		// 功能键提示（触屏点选项行 / 键盘 W/S）
		drawText(c, dx+dw-textW("W/S 选择  点击选项"), dy+dh-24, "W/S 选择  点击选项", engine.Color{R: 150, G: 150, B: 175})
	} else if !d.typing {
		// 继续提示
		if int(g.animT*2)%2 == 0 {
			drawText(c, dx+dw-30, dy+dh-24, "▼", engine.ColWhite)
		}
		drawText(c, dx+12, dy+dh-24, "点击/回车 继续", engine.Color{R: 150, G: 150, B: 175})
	}
}

// wrapText 按宽度折行（width 为字符数，中文按 1 个算）。
func wrapText(s string, width int) []string {
	runes := []rune(s)
	var lines []string
	for len(runes) > 0 {
		if len(runes) <= width {
			lines = append(lines, string(runes))
			break
		}
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	return lines
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return itoa(n/10) + string(digits[n%10])
}
