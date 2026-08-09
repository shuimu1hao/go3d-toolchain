// battle.go - 回合制战斗（RPG Maker 风格）
package main

import (
	"math/rand"

	"go2dgame/engine"
)

// Buff 持续状态
type Buff struct {
	Name  string
	Kind  string // atk_up def_up burn chill regen
	Val   int
	Turns int
}

// Battle 回合制战斗状态。
type Battle struct {
	active      bool
	boss        bool
	mon         Monster
	monHP       int
	pHP         int
	pMP         int
	cmdIdx      int
	subMenu     string // "" / "skill" / "item"
	subSel      int
	msg         string
	msgT        float64
	turn        string // player / enemy
	animT       float64
	ended       bool
	win         bool
	endingText  string
	helper      *Girl
	helperCD    float64
	fleeTry     int
	defending   bool
	playerBuffs []Buff
	monBuffs    []Buff
}

// Start 普通遇敌。
func (b *Battle) Start(g *Game, monName string) {
	b.active = true
	b.boss = false
	b.mon = findMonster(monName)
	b.monHP = b.mon.HP
	b.pHP = g.player.HP
	b.pMP = g.player.MP
	b.cmdIdx = 0
	b.subMenu = ""
	b.msg = b.mon.Name + " 出现了！"
	b.msgT = 1.2
	b.turn = "player"
	b.ended = false
	b.win = false
	b.fleeTry = 0
	b.playerBuffs = nil
	b.monBuffs = nil
	// 女主支援
	b.helper = nil
	for _, gr := range girls {
		if gr.JoinBattle {
			b.helper = gr
			break
		}
	}
	b.helperCD = 2
}

// StartBoss 剧情 Boss 战。
func (b *Battle) StartBoss(g *Game, monName string) {
	b.Start(g, monName)
	b.boss = true
}

func findMonster(name string) Monster {
	for _, m := range monsters {
		if m.Name == name {
			return m
		}
	}
	return Monster{Name: "野狼", HP: 22, Atk: 6, Def: 1, MDef: 0, XP: 14, Gold: 4, AtkType: "phy"}
}

// ===== Buff 工具 =====
func (b *Battle) addPlayerBuff(kind, name string, val, turns int) {
	for i := range b.playerBuffs {
		if b.playerBuffs[i].Kind == kind {
			b.playerBuffs[i].Val = val
			b.playerBuffs[i].Turns = turns
			return
		}
	}
	b.playerBuffs = append(b.playerBuffs, Buff{Name: name, Kind: kind, Val: val, Turns: turns})
}

func (b *Battle) addMonBuff(kind, name string, val, turns int) {
	for i := range b.monBuffs {
		if b.monBuffs[i].Kind == kind {
			b.monBuffs[i].Val = val
			b.monBuffs[i].Turns = turns
			return
		}
	}
	b.monBuffs = append(b.monBuffs, Buff{Name: name, Kind: kind, Val: val, Turns: turns})
}

func (b *Battle) hasPlayerBuff(kind string) bool {
	for _, bf := range b.playerBuffs {
		if bf.Kind == kind && bf.Turns > 0 {
			return true
		}
	}
	return false
}

func (b *Battle) hasMonBuff(kind string) bool {
	for _, bf := range b.monBuffs {
		if bf.Kind == kind && bf.Turns > 0 {
			return true
		}
	}
	return false
}

// tickBuffs 敌人回合开始结算持续效果（灼烧/再生）。
func (b *Battle) tickBuffs(g *Game) {
	// 灼烧：敌人掉血
	for i := range b.monBuffs {
		bf := &b.monBuffs[i]
		if bf.Kind == "burn" && bf.Turns > 0 {
			b.monHP -= bf.Val
			b.msg = b.mon.Name + " 被灼烧侵蚀，失去 " + itoa(bf.Val) + " 点气血！"
			b.msgT = 1.2
			if b.monHP <= 0 {
				b.monHP = 0
				b.victory(g)
				return
			}
		}
	}
	// 再生：玩家回血
	for i := range b.playerBuffs {
		bf := &b.playerBuffs[i]
		if bf.Kind == "regen" && bf.Turns > 0 {
			b.pHP += bf.Val
			if b.pHP > g.player.MaxHP {
				b.pHP = g.player.MaxHP
			}
			b.msg = "真气流转，你恢复了 " + itoa(bf.Val) + " 点气血！"
			b.msgT = 1.2
		}
	}
	// 回合数递减 & 清理
	b.tickTurns()
}

func (b *Battle) tickTurns() {
	alive := b.playerBuffs[:0]
	for _, bf := range b.playerBuffs {
		bf.Turns--
		if bf.Turns > 0 {
			alive = append(alive, bf)
		}
	}
	b.playerBuffs = alive
	alive2 := b.monBuffs[:0]
	for _, bf := range b.monBuffs {
		bf.Turns--
		if bf.Turns > 0 {
			alive2 = append(alive2, bf)
		}
	}
	b.monBuffs = alive2
}

// calcPhys 武学物理伤害（受物抗影响）。
func (b *Battle) calcPhys(g *Game, base, rnd int) int {
	dmg := g.player.Atk/2 + base + rand.Intn(rnd+1) - b.mon.Def
	if b.hasPlayerBuff("atk_up") {
		dmg = dmg * 140 / 100
	}
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// calcMagic 功法魔法伤害（受魔抗影响）。
func (b *Battle) calcMagic(g *Game, base, rnd int) int {
	dmg := g.player.MAtk/2 + base + rand.Intn(rnd+1) - b.mon.MDef
	if b.hasPlayerBuff("atk_up") {
		dmg = dmg * 140 / 100
	}
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// applyEffect 功法附带效果（灼烧/冰寒）。
func (b *Battle) applyEffect(g *Game, m *Martial) {
	if m.Effect == "" {
		return
	}
	switch m.Effect {
	case "burn":
		if rand.Float64() < 0.45 {
			b.addMonBuff("burn", "灼烧", m.EffVal+g.player.Level, 3)
			b.msg += " 敌人燃起烈焰！"
		}
	case "chill":
		if rand.Float64() < 0.45 {
			b.addMonBuff("chill", "冰寒", m.EffVal, 2)
			b.msg += " 敌人被寒气缠身，速度大减！"
		}
	}
}

// Update 战斗逻辑。
func (b *Battle) Update(g *Game, dt float64) {
	b.animT += dt
	if b.msgT > 0 {
		b.msgT -= dt
	}
	if b.ended {
		if g.eng.Input().Pressed(engine.KeyEnter) || g.eng.Input().Pressed(engine.KeySpace) || g.eng.Input().MouseLeftPressed {
			if b.win && b.boss {
				g.markBossDefeated(b.mon.Name)
				g.msg = "击败了 " + b.mon.Name + "！"
				g.msgT = 2
			}
			b.active = false
			g.mode = "map"
		}
		return
	}
	in := g.eng.Input()
	if b.turn == "enemy" {
		b.animT += dt
		// 敌人攻击动画延迟
		if b.animT > 0.9 {
			b.enemyAttack(g)
		}
		return
	}
	// ===== 玩家回合 =====
	if b.subMenu == "" {
		// 主菜单
		if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
			b.cmdIdx = (b.cmdIdx + 5 - 1) % 5
		}
		if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
			b.cmdIdx = (b.cmdIdx + 1) % 5
		}
		// 触屏：点击指令行直接执行（行布局：my+12+i*32）
		if in.MouseLeftPressed {
			if i := hitRow(in.MouseX, in.MouseY, viewX+viewW-190, viewY+viewH-178, 28, 32, 5); i >= 0 {
				b.cmdIdx = i
				b.applyCmd(g, i)
				return
			}
		}
		if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
			b.applyCmd(g, b.cmdIdx)
		}
		return
	}
	// 子菜单
	var list []string
	if b.subMenu == "skill" {
		list = g.player.Skills
	} else {
		for _, id := range g.player.Items {
			it := findItem(id)
			if it != nil && (it.HP > 0 || it.MP > 0) {
				list = append(list, id)
			}
		}
	}
	if len(list) == 0 {
		b.subMenu = ""
		b.msg = "没有可用的选项"
		b.msgT = 1
		return
	}
	if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
		b.subSel = (b.subSel + len(list) - 1) % len(list)
	}
	if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
		b.subSel = (b.subSel + 1) % len(list)
	}
	if in.Pressed(engine.KeyEscape) || in.Pressed(engine.KeyChar('q')) || in.Pressed(engine.KeyChar('Q')) {
		b.subMenu = ""
		return
	}
	// 触屏：点击子菜单行直接使用；点击面板外返回
	if in.MouseLeftPressed {
		if i := hitRow(in.MouseX, in.MouseY, 490, 302, 26, 30, len(list)); i >= 0 {
			b.subSel = i
			if b.subMenu == "skill" {
				b.useSkill(g, list[i])
			} else {
				b.useItem(g, list[i])
			}
			return
		}
		if in.MouseX < 490 || in.MouseX > 790 || in.MouseY < 268 || in.MouseY > 268+30+len(list)*30 {
			b.subMenu = ""
			return
		}
	}
	if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
		if b.subMenu == "skill" {
			b.useSkill(g, list[b.subSel])
		} else {
			b.useItem(g, list[b.subSel])
		}
	}
}

// applyCmd 执行指令菜单项（0攻击 1武功 2物品 3防御 4逃跑）。
func (b *Battle) applyCmd(g *Game, i int) {
	switch i {
	case 0:
		b.playerAttack(g)
	case 1:
		b.subMenu = "skill"
		b.subSel = 0
	case 2:
		b.subMenu = "item"
		b.subSel = 0
	case 3:
		b.playerDefend(g)
	case 4:
		b.playerFlee(g)
	}
}

func (b *Battle) playerAttack(g *Game) {
	dmg := b.calcPhys(g, 6, 5)
	b.monHP -= dmg
	b.msg = "你挥剑攻击，造成 " + itoa(dmg) + " 点伤害！"
	b.msgT = 1.4
	if b.monHP <= 0 {
		b.victory(g)
		return
	}
	b.endPlayerTurn()
}

func (b *Battle) pickWeapon() string {
	return "基础剑法"
}

// findMartial 按名查找武功。
func findMartial(name string) *Martial {
	for i := range martials {
		if martials[i].Name == name {
			return &martials[i]
		}
	}
	return nil
}

// typeTag 技能类型标记。
func typeTag(m *Martial) string {
	switch m.Type {
	case "wu":
		return "[武]"
	case "gong":
		return "[法]"
	case "heal":
		return "[医]"
	}
	return "[?]"
}

// buffTag buff 显示名。
func buffTag(kind string) string {
	switch kind {
	case "atk_up":
		return "攻↑"
	case "def_up":
		return "防↑"
	case "burn":
		return "灼"
	case "chill":
		return "冰"
	case "regen":
		return "愈"
	}
	return kind
}

func (b *Battle) playerDefend(g *Game) {
	b.addPlayerBuff("def_up", "守势", 40, 3)
	b.msg = "你凝神运功，进入守势（减伤40%，持续3回合）！"
	b.msgT = 1.4
	b.endPlayerTurnDef()
}

func (b *Battle) playerFlee(g *Game) {
	if b.boss {
		b.msg = "Boss 战无法逃跑！"
		b.msgT = 1.2
		return
	}
	if rand.Float64() < 0.6 {
		b.msg = "你成功逃跑了！"
		b.msgT = 1.2
		b.ended = true
		b.win = false
		b.active = false
		g.mode = "map"
	} else {
		b.fleeTry++
		b.msg = "逃跑失败！"
		b.msgT = 1.0
		b.endPlayerTurn()
	}
}

func (b *Battle) useSkill(g *Game, name string) {
	var m *Martial
	for i := range martials {
		if martials[i].Name == name {
			m = &martials[i]
			break
		}
	}
	if m == nil {
		b.msg = "没有这个武功"
		b.msgT = 1
		return
	}
	if b.pMP < m.Cost {
		b.msg = "内力不足！"
		b.msgT = 1
		return
	}
	b.pMP -= m.Cost
	switch m.Kind {
	case "heal":
		heal := m.Heal + g.player.Level
		b.pHP += heal
		if b.pHP > g.player.MaxHP {
			b.pHP = g.player.MaxHP
		}
		b.msg = "你运功「" + m.Name + "」，恢复 " + itoa(heal) + " 点气血！"
	case "combo":
		total := 0
		for i := 0; i < m.Combo; i++ {
			var dmg int
			if m.Type == "gong" {
				dmg = b.calcMagic(g, m.Base/3, m.Rand)
			} else {
				dmg = b.calcPhys(g, m.Base/3, m.Rand)
			}
			total += dmg
			b.monHP -= dmg
		}
		b.msg = "「" + m.Name + "」连击 " + itoa(m.Combo) + " 次，共造成 " + itoa(total) + " 点伤害！"
	default:
		var dmg int
		if m.Type == "gong" {
			dmg = b.calcMagic(g, m.Base, m.Rand)
		} else {
			dmg = b.calcPhys(g, m.Base, m.Rand)
		}
		if m.Kind == "strong" {
			dmg += 4
		}
		b.monHP -= dmg
		b.msg = "「" + m.Name + "」造成 " + itoa(dmg) + " 点伤害！"
	}
	// 功法附带效果
	b.applyEffect(g, m)
	b.msgT = 1.4
	if b.monHP <= 0 {
		b.victory(g)
		return
	}
	b.endPlayerTurn()
}

func (b *Battle) useItem(g *Game, id string) {
	it := findItem(id)
	if it == nil {
		b.subMenu = ""
		return
	}
	g.player.removeItem(id)
	if it.HP > 0 {
		b.pHP += it.HP
		if b.pHP > g.player.MaxHP {
			b.pHP = g.player.MaxHP
		}
	}
	if it.MP > 0 {
		b.pMP += it.MP
		if b.pMP > g.player.MaxMP {
			b.pMP = g.player.MaxMP
		}
	}
	b.msg = "使用「" + it.Name + "」！"
	b.msgT = 1.2
	b.subMenu = ""
	b.endPlayerTurn()
}

// endPlayerTurn 玩家行动结束 → 敌人回合（无防御加成）。
func (b *Battle) endPlayerTurn() {
	b.turn = "enemy"
	b.animT = 0
}

func (b *Battle) endPlayerTurnDef() {
	b.turn = "enemy"
	b.defending = true
	b.animT = 0
}

func (b *Battle) enemyAttack(g *Game) {
	// 持续效果结算（灼烧/再生）
	b.tickBuffs(g)
	if b.ended {
		return
	}
	// 女主支援（每 2 回合一次）
	if b.helper != nil && b.helperCD <= 0 {
		b.helperCD = 2
		b.helperAction(g)
		return
	}
	b.helperCD -= 1
	var dmg int
	if b.mon.AtkType == "mag" {
		dmg = b.mon.Atk - g.player.MDef
	} else {
		dmg = b.mon.Atk - g.player.Def
	}
	if b.defending {
		dmg = dmg/2 - 1
	}
	b.defending = false
	if b.hasPlayerBuff("def_up") {
		dmg = dmg * 60 / 100
	}
	if b.hasMonBuff("chill") {
		dmg = dmg * 60 / 100
	}
	if dmg < 1 {
		dmg = 1
	}
	b.pHP -= dmg
	if b.mon.AtkType == "mag" {
		b.msg = b.mon.Name + " 施展邪法，造成 " + itoa(dmg) + " 点伤害！"
	} else {
		b.msg = b.mon.Name + " 攻击了你，造成 " + itoa(dmg) + " 点伤害！"
	}
	b.msgT = 1.4
	if b.pHP <= 0 {
		b.pHP = 0
		b.defeat(g)
		return
	}
	b.turn = "player"
	b.subMenu = ""
	b.animT = 0
}

func (b *Battle) helperAction(g *Game) {
	h := b.helper
	// 女主技能（区分物魔）
	switch h.ID {
	case "qing":
		heal := 20 + g.player.Level*2
		b.pHP += heal
		if b.pHP > g.player.MaxHP {
			b.pHP = g.player.MaxHP
		}
		b.addPlayerBuff("regen", "回春", 6+g.player.Level, 3)
		b.msg = h.Name + " 使出「" + h.Skill + "」，恢复 " + itoa(heal) + " 点气血，并持续回血3回合！"
	case "luoyao":
		dmg := 18 + g.player.Level*2 - b.mon.Def
		if dmg < 1 {
			dmg = 1
		}
		b.monHP -= dmg
		b.msg = h.Name + " 使出「" + h.Skill + "」（武学），造成 " + itoa(dmg) + " 点伤害！"
	case "yueli":
		dmg := 22 + g.player.Level*2 - b.mon.MDef
		if dmg < 1 {
			dmg = 1
		}
		b.monHP -= dmg
		b.msg = h.Name + " 使出「" + h.Skill + "」（功法），造成 " + itoa(dmg) + " 点伤害！"
		if rand.Float64() < 0.35 {
			b.addMonBuff("chill", "冰寒", 40, 2)
			b.msg += " 敌人被冰寒缠身！"
		}
	case "xiaoman":
		dmg := 16 + g.player.Level*2 - b.mon.Def
		if dmg < 1 {
			dmg = 1
		}
		b.monHP -= dmg
		b.msg = h.Name + " 使出「" + h.Skill + "」（武学），造成 " + itoa(dmg) + " 点伤害！"
	case "suxue":
		dmg := 20 + g.player.Level*2 - b.mon.Def/2
		if dmg < 1 {
			dmg = 1
		}
		b.monHP -= dmg
		b.msg = h.Name + " 使出「" + h.Skill + "」，无视部分防御，造成 " + itoa(dmg) + " 点伤害！"
	}
	b.msgT = 1.4
	if b.monHP <= 0 {
		b.victory(g)
		return
	}
	// 女主行动完继续敌人回合
	b.turn = "enemy"
	b.animT = 0
}

func (b *Battle) victory(g *Game) {
	b.ended = true
	b.win = true
	xp := b.mon.XP
	gold := b.mon.Gold + rand.Intn(5)
	g.player.addXP(xp)
	g.player.Gold += gold
	g.player.HP = b.pHP
	g.player.MP = b.pMP
	if b.helper != nil {
		b.helper.Love += 2
		if b.helper.Love > 100 {
			b.helper.Love = 100
		}
		g.player.GirlLove[b.helper.ID] = b.helper.Love
		b.msg = "胜利！获得 " + itoa(xp) + " 经验、" + itoa(gold) + " 两银子" + b.loveMsg()
	} else {
		b.msg = "胜利！获得 " + itoa(xp) + " 经验、" + itoa(gold) + " 两银子"
	}
	b.msgT = 2.5
}

func (b *Battle) loveMsg() string {
	if b.helper != nil {
		return "，" + b.helper.Name + " 好感度 +2"
	}
	return ""
}

func (b *Battle) defeat(g *Game) {
	b.ended = true
	b.win = false
	b.pHP = 0
	b.msg = "你倒下了……"
	b.msgT = 2
	// 死亡惩罚：扣一半钱，回青州
	g.player.Gold /= 2
	g.player.HP = g.player.MaxHP / 2
	g.player.MP = g.player.MaxMP / 2
	g.curMap = findMap("qingzhou")
	g.player.MapID = "qingzhou"
	g.player.PX, g.player.PY = 10, 6
	g.px, g.py = 10, 6
}

// Draw 战斗渲染。
func (b *Battle) Draw(c *engine.Canvas, g *Game) {
	// 背景
	bgX, bgY := viewX, viewY
	g.battleBg.Draw(c, bgX, bgY)
	// 边框
	c.Rect(bgX-4, bgY-4, viewW+8, viewH+8, engine.Color{R: 90, G: 90, B: 110})
	// 怪物（右侧上方）
	monSpr := makeMonsterSprite(b.mon.Name)
	if monSpr != nil {
		ms := 3
		if b.mon.Boss {
			ms = 4
		}
		monSpr.DrawScaled(c, bgX+viewW/2+60, bgY+50, ms)
	}
	// 玩家（左侧下方）
	pSpr := g.charSprites["hero"]["down0"]
	if pSpr == nil {
		pSpr = g.charSprites["hero"]["d0"]
	}
	if pSpr != nil {
		pSpr.DrawScaled(c, bgX+70, bgY+viewH-150, 3)
	}
	// 女主支援
	if b.helper != nil {
		hSpr := g.charSprites[b.helper.ID]["d0"]
		if hSpr != nil {
			hSpr.DrawScaled(c, bgX+140, bgY+viewH-120, 2)
		}
	}
	// 血条
	drawBar(c, bgX+viewW/2+20, bgY+20, 260, b.mon.Name, b.monHP, b.mon.HP, engine.Color{R: 230, G: 90, B: 70})
	drawBar(c, bgX+30, bgY+viewH-100, 240, g.player.Name, b.pHP, g.player.MaxHP, engine.Color{R: 90, G: 200, B: 110})
	// 蓝条
	drawMPBar(c, bgX+30, bgY+viewH-74, 240, b.pMP, g.player.MaxMP)
	// Buff 显示（怪物 + 玩家）
	bx := bgX + viewW/2 + 20
	for _, bf := range b.monBuffs {
		if bf.Turns > 0 {
			drawText(c, bx, bgY+40, buffTag(bf.Kind)+"x"+itoa(bf.Turns), engine.Color{R: 255, G: 150, B: 120})
			bx += 46
		}
	}
	px2 := bgX + 30
	for _, bf := range b.playerBuffs {
		if bf.Turns > 0 {
			drawText(c, px2, bgY+viewH-58, buffTag(bf.Kind)+"x"+itoa(bf.Turns), engine.Color{R: 130, G: 220, B: 255})
			px2 += 46
		}
	}
	// 战斗消息
	if b.msgT > 0 {
		w := textW(b.msg) + 20
		mx := 480 - w/2
		my := bgY + 20
		c.FillRect(mx, my, w, 28, engine.Color{R: 15, G: 15, B: 24})
		c.Rect(mx, my, w, 28, engine.Color{R: 200, G: 200, B: 220})
		drawText(c, mx+10, my+5, b.msg, engine.Color{R: 255, G: 250, B: 230})
	}
	// 结束画面
	if b.ended {
		c.FillRect(280, 200, 400, 140, engine.Color{R: 22, G: 22, B: 34})
		c.Rect(280, 200, 400, 140, engine.Color{R: 255, G: 255, B: 255})
		if b.win {
			drawTextShadowCenter(c, 480, 230, "战斗胜利！", engine.Color{R: 255, G: 220, B: 100})
			drawTextCenter(c, 480, 268, b.msg, engine.Color{R: 230, G: 230, B: 240})
			drawTextCenter(c, 480, 300, "按 回车/空格 继续", engine.Color{R: 160, G: 160, B: 180})
		} else {
			drawTextShadowCenter(c, 480, 230, "战斗失败……", engine.Color{R: 240, G: 100, B: 100})
			drawTextCenter(c, 480, 268, b.msg, engine.Color{R: 230, G: 230, B: 240})
			drawTextCenter(c, 480, 300, "按 回车/空格 返回青州城", engine.Color{R: 160, G: 160, B: 180})
		}
		return
	}
	// 指令菜单（右下）
	if b.turn == "player" && b.subMenu == "" {
		mx, my := bgX+viewW-190, bgY+viewH-190
		c.FillRect(mx, my, 180, 180, engine.Color{R: 20, G: 20, B: 32})
		c.Rect(mx, my, 180, 180, engine.Color{R: 220, G: 220, B: 240})
		cmds := []string{"攻击", "武功", "物品", "防御", "逃跑"}
		for i, s := range cmds {
			col := engine.Color{R: 220, G: 220, B: 240}
			mark := "  "
			if i == b.cmdIdx {
				col = engine.Color{R: 255, G: 235, B: 130}
				mark = "▶"
			}
			drawTextShadow(c, mx+14, my+12+i*32, mark+" "+s, col)
		}
	}
	// 子菜单
	if b.turn == "player" && b.subMenu != "" {
		var list []string
		if b.subMenu == "skill" {
			list = g.player.Skills
		} else {
			for _, id := range g.player.Items {
				it := findItem(id)
				if it != nil && (it.HP > 0 || it.MP > 0) {
					list = append(list, it.Name)
				}
			}
		}
		mx, my := bgX+viewW-310, bgY+viewH-210
		h := 30 + len(list)*30
		if h > 260 {
			h = 260
		}
		c.FillRect(mx, my, 300, h, engine.Color{R: 22, G: 22, B: 34})
		c.Rect(mx, my, 300, h, engine.Color{R: 220, G: 220, B: 240})
		title := "选择武功"
		if b.subMenu == "item" {
			title = "选择物品"
		}
		drawTextShadow(c, mx+12, my+6, title, engine.Color{R: 255, G: 230, B: 130})
		for i, s := range list {
			col := engine.Color{R: 220, G: 220, B: 240}
			mark := "  "
			if i == b.subSel {
				col = engine.Color{R: 255, G: 235, B: 130}
				mark = "▶"
			}
			disp := s
			if b.subMenu == "skill" {
				if m := findMartial(s); m != nil {
					disp = typeTag(m) + s
				}
			}
			drawTextShadow(c, mx+14, my+34+i*30, mark+" "+disp, col)
		}
		drawText(c, mx+12, my+h-24, "W/S选择 Esc返回", engine.Color{R: 150, G: 150, B: 170})
	}
}

// drawBar 血条。
func drawBar(c *engine.Canvas, x, y, w int, name string, cur, max int, col engine.Color) {
	drawTextShadow(c, x, y-20, name, engine.ColWhite)
	c.FillRect(x, y, w, 12, engine.Color{R: 60, G: 30, B: 30})
	if max > 0 {
		fw := w * cur / max
		if fw > 0 {
			c.FillRect(x, y, fw, 12, col)
		}
	}
	c.Rect(x, y, w, 12, engine.ColWhite)
	drawText(c, x+w+8, y-2, itoa(cur)+"/"+itoa(max), engine.ColWhite)
}

func drawMPBar(c *engine.Canvas, x, y, w int, cur, max int) {
	drawTextShadow(c, x, y-18, "内力", engine.Color{R: 150, G: 200, B: 255})
	c.FillRect(x, y, w, 8, engine.Color{R: 20, G: 30, B: 50})
	if max > 0 {
		fw := w * cur / max
		if fw > 0 {
			c.FillRect(x, y, fw, 8, engine.Color{R: 80, G: 140, B: 255})
		}
	}
	c.Rect(x, y, w, 8, engine.ColWhite)
	drawText(c, x+w+8, y-4, itoa(cur)+"/"+itoa(max), engine.Color{R: 150, G: 200, B: 255})
}

// makeMonsterSprite 生成怪物精灵（16x16 程序像素）。
func makeMonsterSprite(name string) *engine.Sprite {
	s := engine.NewSprite(16, 16)
	paint := func(rows []string, pal map[byte]engine.Color) {
		for y, row := range rows {
			for x := 0; x < len(row) && x < 16; x++ {
				if col, ok := pal[row[x]]; ok {
					s.Set(x, y, col, 255)
				}
			}
		}
	}
	switch name {
	case "野狼":
		paint([]string{
			"................",
			"..B....BB....B..",
			"..BBBBBBBBBBBB..",
			"...BBBBBBBBBB...",
			"...BBBRBBBBBB...",
			"....BBBBBBBB....",
			".....BBBBBB.....",
			"....BBBBBBBB....",
			"...BB.BBBB.BB...",
		}, map[byte]engine.Color{
			'B': {R: 120, G: 120, B: 128}, 'R': {R: 220, G: 60, B: 60},
		})
	case "山贼", "黑风寨兵":
		paint([]string{
			"......RR........",
			".....RRRR.......",
			".....RSSSR......",
			".....SESES......",
			".....SSSSS......",
			".....CCCCC......",
			"....CCCCCCC.....",
			"....CCCCCCC.....",
			".....CCCCC......",
			".....PP.PP......",
		}, map[byte]engine.Color{
			'R': {R: 150, G: 60, B: 40}, 'S': {R: 230, G: 190, B: 160},
			'E': {R: 30, G: 30, B: 30}, 'C': {R: 120, G: 80, B: 50},
			'P': {R: 70, G: 50, B: 40},
		})
	case "毒蛇":
		paint([]string{
			"................",
			"....GGGGGG......",
			"...GGGGGGGGG....",
			"..GGGGGGGGGGG...",
			"..GGRGGGGGGGG...",
			"...GGGGGGGG.....",
			".....GGGGG......",
		}, map[byte]engine.Color{
			'G': {R: 60, G: 160, B: 70}, 'R': {R: 220, G: 40, B: 40},
		})
	case "古墓尸卫", "尸王":
		paint([]string{
			"......WW........",
			".....WWWW.......",
			"....WWWWWW......",
			"....WWRRWW......",
			"....WWWWWW......",
			".....GGGG.......",
			"....GGGGGG......",
			"....GG.GG.......",
			"....GG.GG.......",
		}, map[byte]engine.Color{
			'W': {R: 180, G: 190, B: 180}, 'R': {R: 200, G: 50, B: 50},
			'G': {R: 90, G: 100, B: 90},
		})
	case "血月教徒", "血月护法", "血月教主":
		paint([]string{
			"......PP........",
			".....PPPP.......",
			".....PSSSP......",
			".....SESES......",
			".....SSSSS......",
			".....MMMMM......",
			"....MMMMMMMM....",
			"...MMMMMMMMMM...",
			"....MM..MM......",
		}, map[byte]engine.Color{
			'P': {R: 120, G: 40, B: 60}, 'S': {R: 220, G: 190, B: 170},
			'E': {R: 200, G: 30, B: 30}, 'M': {R: 110, G: 40, B: 90},
		})
	case "黑风女寨主":
		paint([]string{
			"......HH........",
			".....HHHH.......",
			"....HHHSSHH.....",
			".....SESES......",
			"....SSSSSSS.....",
			"....CCCCCCC.....",
			"...CCCCCCCCC....",
			"..CCCCCCCCCCC...",
			"..CCCDCCCCCDCC..",
			"..CCCCCCCCCCC...",
			"....CC.CC.......",
			"....SS.SS.......",
		}, map[byte]engine.Color{
			'H': {R: 176, G: 60, B: 56}, 'S': {R: 236, G: 198, B: 168},
			'E': {R: 60, G: 30, B: 30}, 'C': {R: 120, G: 52, B: 96},
		})
	default:
		paint([]string{
			"......GG........",
			".....GGGG.......",
			"....GGGGGG......",
			"....GGRRGG......",
			"....GGGGGG......",
			".....GGGG.......",
			"....GGGGGG......",
			"....GG.GG.......",
		}, map[byte]engine.Color{
			'G': {R: 100, G: 150, B: 90}, 'R': {R: 200, G: 60, B: 40},
		})
	}
	return s
}
