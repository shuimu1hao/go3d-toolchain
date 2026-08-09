// data.go - 五女主 / 物品 / 武功 / 怪物 数据
package main

import "go2dgame/engine"

// ===== 女主 =====
type Girl struct {
	ID       string
	Name     string
	Title    string // 称号
	Desc     string // 简介
	Color    engine.Color
	InitLove int
	// 事件进度
	Event int // 已触发事件数（0-3）
	Love  int // 好感度 0-100
	// 战斗辅助
	JoinBattle bool // 是否已加入战斗
	Atk        int
	Def        int
	Skill      string // 女主技能名
	SkillDesc  string
}

var girls []*Girl

func initGirls() {
	girls = []*Girl{
		{
			ID: "qing", Name: "青儿", Title: "青州药铺千金", Color: engine.Color{R: 110, G: 200, B: 120},
			Desc:     "青州城药铺掌柜的独女，精通岐黄之术，性情温婉，一袭翠衫总带着淡淡药香。",
			InitLove: 12, Event: 0, Love: 12,
			Atk: 10, Def: 6, Skill: "回春术", SkillDesc: "为我方全体恢复气血",
		},
		{
			ID: "luoyao", Name: "洛瑶", Title: "洛阳城女捕快", Color: engine.Color{R: 220, G: 90, B: 80},
			Desc:     "洛阳府衙年轻捕快，一身红衣劲装，嫉恶如仇，办案如神，人称“红衣铁面”。",
			InitLove: 6, Event: 0, Love: 6,
			Atk: 16, Def: 8, Skill: "擒拿手", SkillDesc: "对单个敌人造成重击",
		},
		{
			ID: "yueli", Name: "月璃", Title: "天山雪峰神秘女子", Color: engine.Color{R: 190, G: 200, B: 230},
			Desc:     "常年居于天山的白衣女子，清冷如霜，身负血月教秘密，来去无踪。",
			InitLove: 0, Event: 0, Love: 0,
			Atk: 20, Def: 5, Skill: "寒月斩", SkillDesc: "对单个敌人造成大量伤害",
		},
		{
			ID: "xiaoman", Name: "小蛮", Title: "黑风寨寨主千金", Color: engine.Color{R: 240, G: 170, B: 80},
			Desc:     "黑风寨寨主的掌上明珠，性子跳脱，爱玩爱闹，双马尾一甩就是一颗糖。",
			InitLove: 10, Event: 0, Love: 10,
			Atk: 12, Def: 7, Skill: "蛮牛冲", SkillDesc: "冲撞敌方前排",
		},
		{
			ID: "suxue", Name: "素雪", Title: "古墓守墓人", Color: engine.Color{R: 170, G: 120, B: 200},
			Desc:     "古墓深处守墓的紫衣女子，冷艳孤傲，身世成谜，一柄长剑挑尽过往。",
			InitLove: 4, Event: 0, Love: 4,
			Atk: 14, Def: 10, Skill: "寒玉剑", SkillDesc: "剑气凛然，无视部分防御",
		},
	}
}

func findGirl(id string) *Girl {
	for _, g := range girls {
		if g.ID == id {
			return g
		}
	}
	return nil
}

// ===== 物品 =====
type Item struct {
	ID    string
	Name  string
	Desc  string
	HP    int
	MP    int
	Price int
	// 送礼
	Gift   bool
	GiftID string // 送给谁（"" 表示通用）
	Love   int    // 送礼好感度
}

var items = []Item{
	{"jinchuang", "金疮药", "疗伤圣药，恢复60点气血", 60, 0, 30, false, "", 0},
	{"dahuan", "大还丹", "少林秘药，恢复150点气血", 150, 0, 90, false, "", 0},
	{"neili", "内力丹", "回复40点内力", 0, 40, 50, false, "", 0},
	{"lingzhi", "千年灵芝", "药铺掌柜重金收购的灵药", 0, 0, 60, true, "", 4},
	{"nvhong", "女儿红", "洛阳陈酿，酒香醇厚", 0, 0, 40, true, "", 4},
	{"zhihua", "纸鸢", "集市上买的花纸鸢，色彩斑斓", 0, 0, 20, true, "", 3},
	{"yuzan", "玉簪", "精致的玉簪，做工考究", 0, 0, 80, true, "", 6},
	{"juan", "锦帕", "绣着兰花的锦帕", 0, 0, 35, true, "", 3},
	{"zhetie", "折扇", "文人墨客的折扇", 0, 0, 25, true, "", 3},
	{"ganlu", "甘露", "天山雪水化成的甘露", 0, 0, 70, true, "", 5},
	{"mixin", "密信", "血月教密信，字迹诡谲（任务）", 0, 0, 0, false, "", 0},
	{"nuanyu", "暖玉", "古墓中发现的奇异暖玉（任务）", 0, 0, 0, false, "", 0},
	{"lingpai", "血月令牌", "刻着血色弯月的令牌（任务）", 0, 0, 0, false, "", 0},
	{"tiancan", "天蚕神功残卷", "传说中的绝世心法（任务）", 0, 0, 0, false, "", 0},
}

func findItem(id string) *Item {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

// ===== 武功 =====
type Martial struct {
	Name   string
	Cost   int    // 内力消耗
	Base   int    // 基础伤害
	Rand   int    // 随机浮动
	Acc    int    // 命中率（百分数）
	Kind   string // normal strong combo heal
	Type   string // wu 武学(物理) / gong 功法(魔法) / heal 治疗
	Heal   int
	Combo  int
	Desc   string
	Effect string // "" / burn 灼烧 / chill 冰寒(减敌攻) / atkup 攻击提升 / defup 防御提升
	EffVal int    // 效果数值
}

var martials = []Martial{
	{Name: "江湖把式", Cost: 0, Base: 6, Rand: 4, Acc: 90, Kind: "normal", Type: "wu", Desc: "市井拳脚，无门无派"},
	{Name: "基础剑法", Cost: 3, Base: 9, Rand: 5, Acc: 90, Kind: "normal", Type: "wu", Desc: "江湖通用入门剑法"},
	{Name: "青锋剑法", Cost: 8, Base: 16, Rand: 7, Acc: 88, Kind: "normal", Type: "wu", Desc: "快剑无痕，凌厉非常"},
	{Name: "铁血刀法", Cost: 12, Base: 24, Rand: 10, Acc: 85, Kind: "strong", Type: "wu", Desc: "势大力沉，破甲一击"},
	{Name: "基础内功", Cost: 5, Base: 0, Rand: 0, Acc: 100, Kind: "heal", Type: "heal", Heal: 25, Desc: "运功调息，恢复25点气血"},
	{Name: "烈火掌", Cost: 8, Base: 13, Rand: 6, Acc: 88, Kind: "normal", Type: "gong", Desc: "火系功法，有几率灼烧敌人", Effect: "burn", EffVal: 8},
	{Name: "寒冰诀", Cost: 10, Base: 15, Rand: 7, Acc: 86, Kind: "normal", Type: "gong", Desc: "冰系功法，有几率冰寒减速敌人", Effect: "chill", EffVal: 40},
	{Name: "紫电心法", Cost: 14, Base: 30, Rand: 12, Acc: 92, Kind: "combo", Type: "gong", Combo: 3, Desc: "雷系功法，紫电三连"},
	{Name: "天蚕神功", Cost: 16, Base: 26, Rand: 12, Acc: 90, Kind: "strong", Type: "gong", Desc: "天蚕真气，破魔一击"},
}

var playerMartials = []string{"江湖把式", "基础剑法", "基础内功", "烈火掌", "寒冰诀"}

// ===== 怪物 =====
type Monster struct {
	Name    string
	HP      int
	Atk     int
	Def     int // 物抗
	MDef    int // 魔抗
	XP      int
	Gold    int
	Boss    bool
	AtkType string // phy 物理 / mag 魔法
}

var monsters = []Monster{
	{Name: "野狼", HP: 22, Atk: 6, Def: 1, MDef: 0, XP: 14, Gold: 4, AtkType: "phy"},
	{Name: "山贼", HP: 30, Atk: 7, Def: 4, MDef: 1, XP: 20, Gold: 6, AtkType: "phy"},
	{Name: "毒蛇", HP: 18, Atk: 8, Def: 0, MDef: 2, XP: 18, Gold: 5, AtkType: "phy"},
	{Name: "黑风寨兵", HP: 52, Atk: 11, Def: 5, MDef: 2, XP: 40, Gold: 12, AtkType: "phy"},
	{Name: "古墓尸卫", HP: 60, Atk: 13, Def: 6, MDef: 7, XP: 45, Gold: 16, AtkType: "phy"},
	{Name: "血月教徒", HP: 70, Atk: 15, Def: 2, MDef: 8, XP: 60, Gold: 20, AtkType: "mag"},
	{Name: "黑风女寨主", HP: 150, Atk: 16, Def: 6, MDef: 5, XP: 130, Gold: 55, Boss: true, AtkType: "phy"},
	{Name: "尸王", HP: 200, Atk: 20, Def: 9, MDef: 10, XP: 200, Gold: 80, Boss: true, AtkType: "phy"},
	{Name: "血月护法", HP: 120, Atk: 20, Def: 4, MDef: 10, XP: 120, Gold: 40, Boss: true, AtkType: "mag"},
	{Name: "血月教主", HP: 320, Atk: 26, Def: 8, MDef: 14, XP: 400, Gold: 200, Boss: true, AtkType: "mag"},
}

// ===== 玩家 =====
type Player struct {
	Name   string
	Level  int
	XP     int
	HP     int
	MaxHP  int
	MP     int
	MaxMP  int
	Atk    int
	Def    int
	MAtk   int
	MDef   int
	Gold   int
	Items  []string // 物品 ID
	Skills []string // 武功名
	// 地图状态
	MapID string
	PX    int
	PY    int
	// 进度
	MainProgress int
	GirlEvents   map[string]int // girlID -> 事件数
	GirlLove     map[string]int
	JoinedGirls  map[string]bool
}

func newPlayer() *Player {
	return &Player{
		Name: "桃夭", Level: 1, XP: 0,
		HP: 80, MaxHP: 80, MP: 20, MaxMP: 20,
		Atk: 8, Def: 2, MAtk: 10, MDef: 3, Gold: 50,
		Items:  []string{"jinchuang", "jinchuang", "neili"},
		Skills: []string{"江湖把式", "基础剑法"},
		MapID:  "qingzhou",
		PX:     10, PY: 10,
		GirlEvents:  map[string]int{},
		GirlLove:    map[string]int{},
		JoinedGirls: map[string]bool{},
	}
}

func (p *Player) xpToNext() int { return 30 + p.Level*20 }

func (p *Player) addXP(n int) {
	p.XP += n
	for p.XP >= p.xpToNext() {
		p.XP -= p.xpToNext()
		p.Level++
		p.MaxHP += 18
		p.MaxMP += 6
		p.HP = p.MaxHP
		p.MP = p.MaxMP
		p.Atk += 3
		p.Def += 2
		p.MAtk += 3
		p.MDef += 2
	}
}

func (p *Player) hasItem(id string) bool {
	for _, it := range p.Items {
		if it == id {
			return true
		}
	}
	return false
}

func (p *Player) addItem(id string) { p.Items = append(p.Items, id) }

func (p *Player) removeItem(id string) bool {
	for i, it := range p.Items {
		if it == id {
			p.Items = append(p.Items[:i], p.Items[i+1:]...)
			return true
		}
	}
	return false
}

func (p *Player) hasSkill(name string) bool {
	for _, s := range p.Skills {
		if s == name {
			return true
		}
	}
	return false
}
