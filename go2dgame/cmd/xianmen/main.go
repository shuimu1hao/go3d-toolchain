// main.go - 仙门物语：桌面版纯 Galgame（go2dgame 引擎）
// 剧情 199 节点 / 6 次好感度选择 / 3 结局 / 立绘·场景·CG / 触屏+键盘
package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"go2dgame/engine"
)

// 屏幕常量（960x540 横屏，与 go2dgame 引擎一致）
const (
	scrW = 960
	scrH = 540
	// 对话框区域
	diaX, diaY = 40, 380
	diaW, diaH = 880, 150
	// 选项区
	optX, optY = 120, 300
	optW, optH = 720, 40
	optGap    = 12
)

type Game struct {
	eng *engine.Engine
	mode string // title / story / ending

	idx    int
	aff    AffMap
	typing bool
	charN  int // 打字机已显示字数
	charT  float64
	sel    int // 选项光标

	scenes   map[string]*engine.Sprite // 背景
	portrait map[string]*engine.Sprite // 半身立绘（用 AI 图）
	cgs      map[string]*engine.Sprite // CG 剧情立绘

	chapterT float64 // 章节横幅计时
	chapter  string
	ending   int // 0师妹bad 1师傅good 2师姐good 3变体bad

	lastClick bool
	startT    time.Time
	frame     int
	autoPlay  bool
	botLine   int
	runs      int
}

func main() {
	e, err := engine.New(engine.DefaultConfig("仙门物语"))
	if err != nil {
		fmt.Println("engine.New:", err)
		return
	}
	auto, line := false, 0
	for _, a := range os.Args {
		if a == "--bot" {
			auto = true
		}
		if len(a) > 6 && a[:6] == "--bot=" {
			auto = true
			fmt.Sscanf(a[6:], "%d", &line)
		}
	}
	g := &Game{
		eng:      e,
		autoPlay: auto,
		botLine:  line,
		mode:     "title",
		aff:      AffMap{},
		scenes:   map[string]*engine.Sprite{},
		portrait: map[string]*engine.Sprite{},
		cgs:      map[string]*engine.Sprite{},
		startT:   time.Now(),
	}
	g.aff["shifu"] = 0
	g.aff["shijie"] = 0
	g.aff["shimei"] = 0

	// 字体（先 assets 后系统 fallback）
	if err := loadCJK("assets/fonts/NotoSansCJK-Regular.ttc", 20); err != nil {
		if err2 := loadCJK("/system/fonts/NotoSansCJK-Regular.ttc", 20); err2 != nil {
			fmt.Println("[font] 警告: 未找到 CJK 字体:", err)
		}
	}

	g.loadSprites()
	if err := e.Run(g); err != nil {
		fmt.Println("engine.Run:", err)
	}
}

// loadSprites 加载全部图片素材。
func (g *Game) loadSprites() {
	scenes := []string{"gate", "cave", "spring", "market", "demon", "battle"}
	for _, id := range scenes {
		if sp := loadSprite("assets/scenes/" + id + ".png"); sp != nil {
			sp = sp.Resize(scrW, scrH)
			g.scenes[id] = sp
		}
	}
	// 半身立绘（对话框用），full 版本用于标题页展示
	portraits := []string{"hero", "shifu", "shijie", "shimei", "hero_full", "shifu_full", "shijie_full", "shimei_full"}
	for _, id := range portraits {
		if sp := loadSprite("assets/portraits/" + id + ".png"); sp != nil {
			sp = sp.Resize(360, 540) // 立绘高 540 全高，宽等比约 360
			g.portrait[id] = sp
		}
	}
	cgs := []string{"cg_gate_meet", "cg_guihua", "cg_spring", "cg_poison", "cg_purple", "cg_demon_lord", "cg_ending1", "cg_ending2", "cg_ending3"}
	for _, id := range cgs {
		if sp := loadSprite("assets/cg/" + id + ".png"); sp != nil {
			sp = sp.Resize(scrW, scrH)
			g.cgs[id] = sp
		}
	}
	fmt.Printf("[assets] scenes=%d portraits=%d cg=%d\n", len(g.scenes), len(g.portrait), len(g.cgs))
}

func loadSprite(path string) *engine.Sprite {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return engine.FromImage(img)
}

// ---------- Update ----------
func (g *Game) Update(dt float64) {
	g.frame++
	if g.chapterT > 0 {
		g.chapterT -= dt
	}
	if g.typing {
		g.charT += dt * 22 // 每秒 22 字
		if g.charT >= 1 {
			n := int(g.charT)
			g.charN += n
			g.charT -= float64(n)
			node := script[g.idx]
			if g.charN >= len([]rune(node.TX)) {
				g.charN = len([]rune(node.TX))
				g.typing = false
			}
		}
	}

	in := g.eng.Input()
	// bot 自动测试模式：自动推进对话、选第一个选项
	// 实现：typing 立即完成；普通节点模拟按下；选项选第一个；
	// Chapter/Judge/End 节点不拦截，走下方正常逻辑。
	if g.autoPlay {
		if g.mode == "story" {
			node := script[g.idx]
			if g.frame%200 == 0 {
				fmt.Printf("[bot] frame=%d idx=%d mode=%s chapter=%q judge=%v end=%v choices=%d typing=%v\n",
					g.frame, g.idx, g.mode, node.Chapter, node.Judge, node.End, len(node.Choices), g.typing)
			}
			if len(node.Choices) > 0 {
				if g.botLine < len(node.Choices) {
					g.choose(g.botLine)
				} else {
					g.choose(0)
				}
				time.Sleep(20 * time.Millisecond)
				return
			}
			if g.typing {
				g.typing = false
				g.charN = len([]rune(node.TX))
				return
			}
			if node.Chapter != "" || node.Judge || node.End {
				// 放行给正常逻辑
			} else {
				g.idx++
				g.gotoNode(g.idx)
				time.Sleep(20 * time.Millisecond)
				return
			}
		}
		if g.mode == "ending" {
			g.runs++
			node := script[g.idx]
			fmt.Printf("[bot] 结局: %s good=%v\n", node.Title, node.Good)
			fmt.Printf("[bot] 通关 %d 次, aff=%v\n", g.runs, g.aff)
			if g.runs >= 3 {
				fmt.Println("[bot] 3 次通关验证完成")
				g.eng.Close()
				return
			}
			g.mode = "title"
			g.idx = 0
			time.Sleep(50 * time.Millisecond)
			return
		}
		if g.mode == "title" {
			g.mode = "story"
			g.idx = 0
			g.aff["shifu"] = 0
			g.aff["shijie"] = 0
			g.aff["shimei"] = 0
			g.gotoNode(g.idx)
			return
		}
	}
	click := in.MouseLeftPressed
	press := in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || click
	up := in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp)
	down := in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown)

	switch g.mode {
	case "title":
		if press {
			g.mode = "story"
			g.idx = 0
			g.aff["shifu"] = 0
			g.aff["shijie"] = 0
			g.aff["shimei"] = 0
			g.gotoNode(g.idx)
		}
	case "story":
		node := script[g.idx]
		// 章节横幅
		if node.Chapter != "" {
			g.chapter = node.Chapter
			g.chapterT = 1.6
			g.idx++
			g.gotoNode(g.idx)
			return
		}
		// 结局判定
		if node.Judge {
			r := judgeEnding(g.aff)
			switch r {
			case 0:
				g.idx = node.NextBad
			case 1:
				g.idx = node.NextGood1
			case 2:
				g.idx = node.NextGood2
			default:
				g.idx = node.NextBadAlt
			}
			g.gotoNode(g.idx)
			return
		}
		// 结局节点
		if node.End {
			g.ending = 0
			if node.Good {
				g.ending = 1
			}
			g.mode = "ending"
			g.typing = false
			return
		}
		// 选项
		if len(node.Choices) > 0 {
			if up {
				g.sel--
				if g.sel < 0 {
					g.sel = len(node.Choices) - 1
				}
			}
			if down {
				g.sel++
				if g.sel >= len(node.Choices) {
					g.sel = 0
				}
			}
			// 触屏点击选项
			if click {
				mx, my := in.MouseX, in.MouseY
				for i := range node.Choices {
					cy := optY + i*(optH+optGap)
					if mx >= optX && mx <= optX+optW && my >= cy && my <= cy+optH {
						g.sel = i
						g.choose(i)
						return
					}
				}
			}
			if press && !g.typing && g.sel >= 0 && g.sel < len(node.Choices) {
				g.choose(g.sel)
			}
			return
		}
		// 普通对话推进
		if press {
			if g.typing {
				g.typing = false
				g.charN = len([]rune(node.TX))
			} else {
				g.idx++
				g.gotoNode(g.idx)
			}
		}
	case "ending":
		if press {
			g.mode = "title"
			g.idx = 0
			g.ending = 0
		}
	}
}

// choose 处理选项选择。
func (g *Game) choose(i int) {
	node := script[g.idx]
	c := node.Choices[i]
	for k, v := range c.Aff {
		g.aff[k] += v
	}
	g.idx = node.NextIdx
	g.gotoNode(g.idx)
}

// gotoNode 初始化进入某节点。
func (g *Game) gotoNode(idx int) {
	g.idx = idx
	if idx < 0 || idx >= len(script) {
		g.mode = "title"
		return
	}
	node := script[idx]
	g.charN = 0
	g.charT = 0
	g.typing = node.TX != "" && len(node.Choices) == 0 && !node.End && !node.Judge && node.Chapter == ""
	g.sel = 0
}

// ---------- Draw ----------
func (g *Game) Draw(c *engine.Canvas) {
	c.Clear(engine.Color{R: 8, G: 8, B: 16})

	switch g.mode {
	case "title":
		g.drawTitle(c)
	case "story":
		g.drawStory(c)
	case "ending":
		g.drawEnding(c)
	}
}

func (g *Game) drawTitle(c *engine.Canvas) {
	if bg, ok := g.scenes["gate"]; ok {
		bg.Draw(c, 0, 0)
	}
	// 上下暗角（不盖全屏，保留背景可见）
	c.FillRect(0, 0, scrW, 60, engine.Color{R: 6, G: 6, B: 14})
	c.FillRect(0, scrH-80, scrW, 80, engine.Color{R: 6, G: 6, B: 14})
	// 标题文字带深色底衬保证可读
	c.FillRect(280, 118, 400, 72, engine.Color{R: 8, G: 8, B: 18})
	drawTextShadowCenter(c, scrW/2, 140, "仙 门 物 语", engine.Color{R: 255, G: 233, B: 184})
	drawTextShadowCenter(c, scrW/2, 200, "—— 三线仙缘 · 一念成魔 ——", engine.Color{R: 180, G: 185, B: 220})
	drawTextShadowCenter(c, scrW/2, 330, "【 按 Enter / 空格 / 点击屏幕 开始 】", engine.Color{R: 255, G: 210, B: 122})
	drawTextShadowCenter(c, scrW/2, 470, "师傅 · 师姐 · 师妹 — 6 次抉择决定结局", engine.Color{R: 130, G: 135, B: 170})
}

func (g *Game) drawStory(c *engine.Canvas) {
	node := script[g.idx]

	// 背景
	bgName := node.BG
	if bgName == "" {
		bgName = "gate"
	}
	if bg, ok := g.scenes[bgName]; ok {
		bg.Draw(c, 0, 0)
	} else {
		c.FillRect(0, 0, scrW, scrH, engine.Color{R: 20, G: 24, B: 40})
	}
	// 压暗
	// 压暗（Canvas 无 alpha，用深色遮罩简化）
	// 用半透明黑遮罩（Canvas 无 alpha，直接画深色渐变简化）
	c.FillRect(0, 0, scrW, 40, engine.Color{R: 10, G: 10, B: 20})
	c.FillRect(0, scrH-190, scrW, 190, engine.Color{R: 10, G: 10, B: 20})

	// CG 全屏（若有）
	if node.CG != "" {
		if cg, ok := g.cgs[node.CG]; ok {
			cg.Draw(c, 0, 0)
			c.FillRect(0, scrH-190, scrW, 190, engine.Color{R: 8, G: 8, B: 16})
		}
	}

	// 角色立绘（对话框右侧）
	portraitID := ""
	switch node.Sp {
	case "白霜华":
		portraitID = "shifu"
	case "严吟烽":
		portraitID = "shijie"
	case "陆倩瑶":
		portraitID = "shimei"
	case "陆亘满":
		portraitID = "hero"
	}
	if portraitID != "" {
		if sp, ok := g.portrait[portraitID]; ok {
			// 立绘放右侧（主角居中）
			x := scrW - 340
			if portraitID == "hero" {
				x = scrW - 300
			}
			sp.Draw(c, x, scrH-540+30)
		}
	}

	// 章节横幅
	if g.chapterT > 0 && g.chapter != "" {
		c.FillRect(0, 200, scrW, 140, engine.Color{R: 6, G: 6, B: 14})
		drawTextShadowCenter(c, scrW/2, 250, g.chapter, engine.Color{R: 255, G: 233, B: 184})
	}

	// 选项
	if len(node.Choices) > 0 {
		c.FillRect(0, 0, scrW, scrH, engine.Color{R: 6, G: 6, B: 14})
		drawTextShadowCenter(c, scrW/2, 120, node.TX, engine.Color{R: 255, G: 240, B: 200})
		for i, ch := range node.Choices {
			cy := optY + i*(optH+optGap)
			col := engine.Color{R: 40, G: 44, B: 80}
			if i == g.sel {
				col = engine.Color{R: 90, G: 74, B: 140}
			}
			c.FillRect(optX-6, cy-6, optW+12, optH+12, engine.Color{R: 20, G: 22, B: 40})
			c.Rect(optX-6, cy-6, optW+12, optH+12, col)
			drawText(c, optX, cy+10, ch.Text, engine.Color{R: 230, G: 232, B: 255})
			if ch.AffHint != "" {
				drawText(c, optX+optW-160, cy+10, ch.AffHint, engine.Color{R: 150, G: 155, B: 200})
			}
		}
		drawTextCenter(c, scrW/2, 500, "W/S 选择 · Enter/点击 确认", engine.Color{R: 120, G: 125, B: 160})
		return
	}

	// 对话框
	c.FillRect(diaX, diaY, diaW, diaH, engine.Color{R: 14, G: 16, B: 34})
	c.Rect(diaX, diaY, diaW, diaH, engine.Color{R: 150, G: 140, B: 220})
	// 说话人
	name := ""
	switch node.Sp {
	case "白霜华":
		name = "白霜华 · 师傅"
	case "严吟烽":
		name = "严吟烽 · 师姐"
	case "陆倩瑶":
		name = "陆倩瑶 · 师妹"
	case "陆亘满":
		name = "陆亘满"
	case "魔尊":
		name = "魔尊"
	}
	if name != "" {
		c.FillRect(diaX+10, diaY+6, textW(name)+24, 30, engine.Color{R: 60, G: 52, B: 100})
		drawText(c, diaX+22, diaY+10, name, engine.Color{R: 255, G: 210, B: 122})
	}
	// 正文（打字机）
	text := node.TX
	if g.typing {
		rs := []rune(text)
		if g.charN > len(rs) {
			g.charN = len(rs)
		}
		text = string(rs[:g.charN])
	}
	drawWrapped(c, diaX+22, diaY+44, diaW-44, text, engine.Color{R: 238, G: 240, B: 255})
	drawText(c, diaX+diaW-70, diaY+diaH-26, "▼", engine.Color{R: 255, G: 210, B: 122})

	// 好感度 HUD
	g.drawHUD(c)
}

func (g *Game) drawEnding(c *engine.Canvas) {
	node := script[g.idx]
	bgName := node.BG
	if bgName == "" {
		bgName = "gate"
	}
	if bg, ok := g.scenes[bgName]; ok {
		bg.Draw(c, 0, 0)
	}
	// CG
	if node.CG != "" {
		if cg, ok := g.cgs[node.CG]; ok {
			cg.Draw(c, 0, 0)
		}
	}
	title := node.Title
	sub := node.Sub
	col := engine.Color{R: 255, G: 141, B: 154}
	if node.Good {
		col = engine.Color{R: 141, G: 255, B: 176}
	}
	drawTextShadowCenter(c, scrW/2, 80, "【 "+title+" 】", col)
	drawTextCenter(c, scrW/2, 130, sub, engine.Color{R: 160, G: 165, B: 200})

	// 结局正文滚动显示（直接全部显示，可读）
	drawWrappedSmall(c, 120, 175, scrW-240, node.TX, engine.Color{R: 222, G: 225, B: 255})

	drawTextCenter(c, scrW/2, 500, "按 Enter / 点击 返回标题", engine.Color{R: 255, G: 210, B: 122})
}

// drawHUD 好感度指示。
func (g *Game) drawHUD(c *engine.Canvas) {
	drawHudItem(c, 10, 8, "师", g.aff["shifu"], engine.Color{R: 200, G: 220, B: 255})
	drawHudItem(c, 130, 8, "姐", g.aff["shijie"], engine.Color{R: 255, G: 180, B: 170})
	drawHudItem(c, 250, 8, "妹", g.aff["shimei"], engine.Color{R: 190, G: 170, B: 255})
}

func drawHudItem(c *engine.Canvas, x, y int, label string, n int, col engine.Color) {
	c.FillRect(x, y, 110, 26, engine.Color{R: 8, G: 10, B: 24})
	c.Rect(x, y, 110, 26, engine.Color{R: 60, G: 60, B: 100})
	drawText(c, x+8, y+4, label, col)
	dot := ""
	for i := 0; i < 5; i++ {
		if i < n {
			dot += "●"
		} else {
			dot += "○"
		}
	}
	drawText(c, x+34, y+4, dot, engine.Color{R: 255, G: 184, B: 77})
}

// drawWrapped 按宽度折行绘制（20px 字号）。
func drawWrapped(c *engine.Canvas, x, y, maxW int, s string, col engine.Color) {
	rs := []rune(s)
	cur := ""
	cx, cy := x, y
	lh := lineH() + 4
	for _, r := range rs {
		if r == '\n' {
			drawText(c, cx, cy, cur, col)
			cur = ""
			cy += lh
			if cy > diaY+diaH-20 {
				break
			}
			continue
		}
		if textW(cur+string(r)) > maxW {
			drawText(c, cx, cy, cur, col)
			cur = ""
			cy += lh
			if cy > diaY+diaH-20 {
				break
			}
		}
		cur += string(r)
	}
	if cur != "" {
		drawText(c, cx, cy, cur, col)
	}
}

// drawWrappedSmall 结局正文折行（16px）。
func drawWrappedSmall(c *engine.Canvas, x, y, maxW int, s string, col engine.Color) {
	rs := []rune(s)
	cur := ""
	cy := y
	lh := lineH() - 2
	for _, r := range rs {
		if r == '\n' {
			drawText(c, x, cy, cur, col)
			cur = ""
			cy += lh
			if cy > scrH-60 {
				break
			}
			continue
		}
		if textW(cur+string(r)) > maxW {
			drawText(c, x, cy, cur, col)
			cur = ""
			cy += lh
			if cy > scrH-60 {
				break
			}
		}
		cur += string(r)
	}
	if cur != "" {
		drawText(c, x, cy, cur, col)
	}
}
