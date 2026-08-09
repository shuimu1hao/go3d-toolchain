// menu.go - 游戏菜单（状态/物品/武功/好感度/存档/帮助）
package main

import "go2dgame/engine"

// 菜单面板区域（与 Draw 布局一致）。
const (
	menuMX, menuMY = 60, 60
	menuMW, menuMH = 220, 420
)

// 菜单右上角触屏按钮。
var (
	menuCloseBtn = Btn{872, 24, 74, 40, "关闭"}
	menuBackBtn  = Btn{872, 24, 74, 40, "返回"}
)

// Menu 菜单状态（背包/装备/武功/任务）。
type Menu struct {
	active  bool
	sel     int
	page    string // "" main / "status" "items" "skills" "girls" "save" "help"
	itemSel int
	itemTop int
}

func (m *Menu) Open(g *Game) {
	m.active = true
	m.sel = 0
	m.page = ""
	m.itemSel = 0
	m.itemTop = 0
}

func (m *Menu) Close(g *Game) {
	m.active = false
	g.mode = "map"
}

// openPage 主菜单项进入子页（i: 0状态 1物品 2武功 3好感 4存档 5帮助）。
func (m *Menu) openPage(g *Game, i int) {
	switch i {
	case 0:
		m.page = "status"
	case 1:
		m.page = "items"
	case 2:
		m.page = "skills"
	case 3:
		m.page = "girls"
	case 4:
		m.page = "save"
	case 5:
		m.page = "help"
	}
}

func (m *Menu) Update(g *Game, dt float64) {
	in := g.eng.Input()
	if m.page == "" {
		// 主菜单
		if in.Pressed(engine.KeyEscape) || touchPressed(in, menuCloseBtn) {
			m.Close(g)
			return
		}
		if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
			m.sel = (m.sel + 6 - 1) % 6
		}
		if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
			m.sel = (m.sel + 1) % 6
		}
		// 触屏：点击左侧菜单项
		if in.MouseLeftPressed {
			for i := 0; i < 6; i++ {
				ry := menuMY + 56 + i*44
				if in.MouseX >= menuMX && in.MouseX < menuMX+menuMW && in.MouseY >= ry-4 && in.MouseY < ry+lineH()+16 {
					m.sel = i
					m.openPage(g, i)
					return
				}
			}
		}
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			m.openPage(g, m.sel)
		}
		return
	}
	// 子页：Esc / 返回按钮 回主菜单
	if in.Pressed(engine.KeyEscape) || touchPressed(in, menuBackBtn) {
		m.page = ""
		return
	}
	// 物品页支持上下滚动 + 回车使用 + 触屏点击行使用
	if m.page == "items" {
		if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
			if m.itemSel > 0 {
				m.itemSel--
			}
		}
		if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
			if m.itemSel < len(g.player.Items)-1 {
				m.itemSel++
			}
		}
		if in.MouseLeftPressed && len(g.player.Items) > 0 {
			// 触屏：点击聚合行 → 映射到第一个该物品下标并直接使用
			ids := groupedItemIDs(g)
			if i := hitRow(in.MouseX, in.MouseY, menuMX+menuMW+30, menuMY+72, 24, 28, len(ids)); i >= 0 {
				target := ids[i]
				for k, id := range g.player.Items {
					if id == target {
						m.itemSel = k
						m.useItemAt(g, k)
						break
					}
				}
				return
			}
		}
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			if len(g.player.Items) > 0 {
				m.useItemAt(g, m.itemSel)
			}
		}
	}
	// 存档页回车保存
	if m.page == "save" {
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			if g.saveGame() {
				g.msg = "存档成功！"
				g.msgT = 2
			} else {
				g.msg = "存档失败！"
				g.msgT = 2
			}
		}
	}
}

// useItemAt 使用/查看玩家背包第 idx 个物品。
func (m *Menu) useItemAt(g *Game, idx int) {
	if idx < 0 || idx >= len(g.player.Items) {
		return
	}
	id := g.player.Items[idx]
	it := findItem(id)
	if it != nil && (it.HP > 0 || it.MP > 0) {
		g.player.removeItem(id)
		if it.HP > 0 {
			g.player.HP += it.HP
			if g.player.HP > g.player.MaxHP {
				g.player.HP = g.player.MaxHP
			}
		}
		if it.MP > 0 {
			g.player.MP += it.MP
			if g.player.MP > g.player.MaxMP {
				g.player.MP = g.player.MaxMP
			}
		}
		g.msg = "使用「" + it.Name + "」"
		g.msgT = 1.5
	} else if it != nil && it.Gift {
		g.msg = "「" + it.Name + "」是礼物，去找女主送给她吧"
		g.msgT = 2
	} else if it != nil {
		g.msg = "「" + it.Name + "」无法使用"
		g.msgT = 1.5
	}
}

// groupedItemIDs 物品按名称聚合后的 ID 列表（与 drawItems 显示行序一致）。
func groupedItemIDs(g *Game) []string {
	var ids []string
	seen := map[string]bool{}
	for _, id := range g.player.Items {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *Menu) Draw(c *engine.Canvas, g *Game) {
	// 半透明遮罩
	c.FillRect(0, 0, 960, 540, engine.Color{R: 0, G: 0, B: 0})
	// 左侧菜单栏
	mx, my, mw, mh := 60, 60, 220, 420
	c.FillRect(mx, my, mw, mh, engine.Color{R: 30, G: 28, B: 44})
	c.Rect(mx, my, mw, mh, engine.Color{R: 200, G: 200, B: 220})
	drawTextShadow(c, mx+16, my+14, "菜 单", engine.Color{R: 255, G: 230, B: 130})
	items := []string{"状态", "物品", "武功", "好感度", "存档", "帮助"}
	for i, s := range items {
		col := engine.Color{R: 220, G: 220, B: 240}
		mark := "  "
		if m.page == "" && i == m.sel {
			col = engine.Color{R: 255, G: 235, B: 130}
			mark = "▶"
		} else if m.page != "" && i == m.pageIdx() {
			col = engine.Color{R: 160, G: 160, B: 200}
			mark = "·"
		}
		drawTextShadow(c, mx+20, my+56+i*44, mark+" "+s, col)
	}
	// 右侧详情
	dx := mx + mw + 30
	dy := my
	switch m.page {
	case "":
		// 未选时显示玩家状态概览
		drawTextShadow(c, dx, dy+10, "—— 女侠小传 ——", engine.Color{R: 255, G: 230, B: 130})
		drawText(c, dx, dy+52, "姓名："+g.player.Name, engine.ColWhite)
		drawText(c, dx, dy+86, "等级："+itoa(g.player.Level), engine.ColWhite)
		drawText(c, dx, dy+120, "气血："+itoa(g.player.HP)+"/"+itoa(g.player.MaxHP), engine.ColWhite)
		drawText(c, dx, dy+154, "内力："+itoa(g.player.MP)+"/"+itoa(g.player.MaxMP), engine.ColWhite)
		drawText(c, dx, dy+188, "银两："+itoa(g.player.Gold), engine.Color{R: 255, G: 220, B: 120})
		drawText(c, dx, dy+222, "所在："+g.curMap.Name, engine.ColWhite)
		drawText(c, dx, dy+300, "W/S 选择  回车 进入  Esc 返回", engine.Color{R: 140, G: 140, B: 170})
	case "status":
		m.drawStatus(c, g, dx, dy)
	case "items":
		m.drawItems(c, g, dx, dy)
	case "skills":
		m.drawSkills(c, g, dx, dy)
	case "girls":
		m.drawGirls(c, g, dx, dy)
	case "save":
		m.drawSave(c, g, dx, dy)
	case "help":
		m.drawHelp(c, g, dx, dy)
	}
	// 右上角触屏按钮
	if m.page == "" {
		drawTouchBtn(c, menuCloseBtn)
	} else {
		drawTouchBtn(c, menuBackBtn)
	}
}

func (m *Menu) pageIdx() int {
	switch m.page {
	case "status":
		return 0
	case "items":
		return 1
	case "skills":
		return 2
	case "girls":
		return 3
	case "save":
		return 4
	case "help":
		return 5
	}
	return -1
}

func (m *Menu) drawStatus(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 状态 ——", engine.Color{R: 255, G: 230, B: 130})
	rows := []string{
		"姓名：" + g.player.Name,
		"等级：" + itoa(g.player.Level),
		"经验：" + itoa(g.player.XP) + " / " + itoa(g.player.xpToNext()),
		"气血：" + itoa(g.player.HP) + " / " + itoa(g.player.MaxHP),
		"内力：" + itoa(g.player.MP) + " / " + itoa(g.player.MaxMP),
		"物攻：" + itoa(g.player.Atk) + "  法攻：" + itoa(g.player.MAtk),
		"物防：" + itoa(g.player.Def) + "  法防：" + itoa(g.player.MDef),
		"银两：" + itoa(g.player.Gold),
		"所在：" + g.curMap.Name,
	}
	for i, s := range rows {
		drawText(c, dx, dy+52+i*30, s, engine.ColWhite)
	}
}

func (m *Menu) drawItems(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 物品 ——", engine.Color{R: 255, G: 230, B: 130})
	drawText(c, dx, dy+40, "回车 使用/查看  Esc 返回", engine.Color{R: 140, G: 140, B: 170})
	// 物品（按名称聚合显示）
	type kv struct {
		it  *Item
		cnt int
	}
	var list []kv
	for _, id := range g.player.Items {
		it := findItem(id)
		if it == nil {
			continue
		}
		found := false
		for i := range list {
			if list[i].it.ID == it.ID {
				list[i].cnt++
				found = true
				break
			}
		}
		if !found {
			list = append(list, kv{it, 1})
		}
	}
	y := dy + 72
	if len(list) == 0 {
		drawText(c, dx, y, "（空空如也）", engine.Color{R: 160, G: 160, B: 180})
		return
	}
	for i, kv := range list {
		col := engine.Color{R: 220, G: 220, B: 240}
		if i == m.itemSel {
			col = engine.Color{R: 255, G: 235, B: 130}
			drawTextShadow(c, dx, y, "▶ "+kv.it.Name+" ×"+itoa(kv.cnt), col)
			drawText(c, dx+260, y, kv.it.Desc, engine.Color{R: 150, G: 150, B: 170})
		} else {
			drawTextShadow(c, dx, y, "  "+kv.it.Name+" ×"+itoa(kv.cnt), col)
		}
		y += 28
	}
}

func (m *Menu) drawSkills(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 武功 ——", engine.Color{R: 255, G: 230, B: 130})
	y := dy + 52
	for _, name := range g.player.Skills {
		var m2 *Martial
		for i := range martials {
			if martials[i].Name == name {
				m2 = &martials[i]
				break
			}
		}
		if m2 == nil {
			continue
		}
		drawTextShadow(c, dx, y, "★ "+name+"（内力"+itoa(m2.Cost)+"）", engine.Color{R: 230, G: 230, B: 250})
		drawText(c, dx+20, y+24, m2.Desc, engine.Color{R: 150, G: 150, B: 170})
		y += 56
	}
}

func (m *Menu) drawGirls(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 红颜知己 ——", engine.Color{R: 255, G: 230, B: 130})
	y := dy + 52
	for _, gr := range girls {
		// 进度
		prog := "未相遇"
		need := ""
		if gr.Event < 3 {
			if gr.Event > 0 {
				prog = "羁绊" + itoa(gr.Event) + "/3"
			}
			if gr.Love < girlEventNeed[gr.ID][gr.Event] {
				need = "（好感 " + itoa(girlEventNeed[gr.ID][gr.Event]) + " 触发）"
			}
		} else {
			prog = "羁绊已满"
		}
		// 状态
		state := ""
		if gr.JoinBattle {
			state = "  [同行]"
		}
		drawTextShadow(c, dx, y, gr.Name+" "+gr.Title, gr.Color)
		drawText(c, dx+20, y+24, "好感度 "+itoa(gr.Love)+"  "+prog+need+state, engine.Color{R: 190, G: 190, B: 210})
		y += 54
	}
}

func (m *Menu) drawSave(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 存档 ——", engine.Color{R: 255, G: 230, B: 130})
	drawText(c, dx, dy+60, "按 回车 保存当前进度", engine.ColWhite)
	drawText(c, dx, dy+100, "（标题画面按 L 读取存档）", engine.Color{R: 150, G: 150, B: 170})
	drawText(c, dx, dy+160, "当前：Lv."+itoa(g.player.Level)+" "+g.curMap.Name, engine.Color{R: 200, G: 220, B: 255})
}

func (m *Menu) drawHelp(c *engine.Canvas, g *Game, dx, dy int) {
	drawTextShadow(c, dx, dy+10, "—— 操作帮助 ——", engine.Color{R: 255, G: 230, B: 130})
	help := []string{
		"WASD / 方向键  移动",
		"E / 空格        对话·交互",
		"回车            确认",
		"Esc            菜单 / 返回",
		"",
		"战斗：",
		"W/S 选指令，回车确认",
		"攻击 · 武功 · 物品 · 防御 · 逃跑",
		"",
		"攻略：",
		"与女主对话触发剧情，好感度上升",
		"送礼物可增加好感度",
		"好感≥40 可邀她同行（战斗支援）",
		"好感≥80 且羁绊3/3 可告白",
	}
	for i, s := range help {
		drawText(c, dx, dy+52+i*26, s, engine.ColWhite)
	}
}
