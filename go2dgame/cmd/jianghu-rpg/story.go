// story.go - 五女主剧本 / 事件 / 告白 / 送礼
package main

import "go2dgame/engine"

// girlEventNeed 事件好感度阈值（事件 1/2/3）
var girlEventNeed = map[string][]int{
	"qing":    {20, 45, 65},
	"luoyao":  {15, 40, 60},
	"yueli":   {10, 35, 55},
	"xiaoman": {15, 40, 60},
	"suxue":   {10, 35, 55},
}

// girlChat 闲聊文案（好感度分档 0/1/2/3）
var girlChat = map[string][]string{
	"qing": {
		"（她低头整理药柜，没注意到你）……啊，客官！要买药吗？",
		"青儿抬起头，朝你甜甜一笑：今天风真好，药圃里的花都开了呢。",
		"青儿轻声问：你……要不要尝尝我新熬的桂花糖水？",
		"青儿望着你，脸颊微红：我好像，越来越习惯你来了……",
	},
	"luoyao": {
		"洛瑶抱剑倚在门口，斜眼扫你：又是你？没事别挡道。",
		"洛瑶收起长刀，难得露出一点笑意：你今天来得正好，我正想找人练练手。",
		"洛瑶拍拍你的肩：有你在，办案都安心了不少。",
		"洛瑶别过脸去，声音却温柔：喂，等我闲下来……陪我逛逛集市呗？",
	},
	"yueli": {
		"月璃立在雪中，如霜似雪，并不看你：天山风寒，无事便回吧。",
		"月璃静静望着远山，语气淡了些许：你……倒是个不怕冷的人。",
		"月璃的眼中似乎有光：这雪景，有人共赏，好像也没那么冷了。",
		"月璃轻轻握住你的手，指尖微凉：别走了……就留在天山，好吗？",
	},
	"xiaoman": {
		"小蛮叉着腰：喂！你这家伙怎么又跑我们黑风寨来了！是不是想挨揍！",
		"小蛮蹦到你面前，眼睛亮晶晶的：快看快看！我新编的蚂蚱！送你了！",
		"小蛮嘟着嘴：哼，今天怎么才来……本小姐等你半天啦！",
		"小蛮扑过来抱住你的胳膊，仰头看着你：说好了，你要一直陪我玩的！",
	},
	"suxue": {
		"素雪静立古墓之前，面纱轻扬：古墓清净，施主请止步。",
		"素雪微微颔首：你……倒是第一个不惧此地阴寒之人。",
		"素雪解下腰间的酒囊递来：墓中冷，喝口暖酒吧。",
		"素雪的眸子里映着烛火：若你愿意……这古墓，从此便不再只有我一个人了。",
	},
}

// 女主事件剧情
func (g *Game) girlEvent(gr *Girl) []DialogLine {
	var ls []DialogLine
	add := func(spk, txt string) { ls = append(ls, DialogLine{Speaker: spk, Text: txt}) }
	switch gr.ID {
	case "qing":
		switch gr.Event {
		case 0:
			add("qing", "哎呀，你来得正好！药圃的灵芝熟了，可我一个人不敢去郊野采……")
			add("qing", "郊野的野狼最近可凶了。你能……陪我一起去采一株千年灵芝吗？")
			add("", "你答应了青儿的请求。")
			add("qing", "（郊野采到灵芝后）谢谢你！这个给你，是我娘留给我的玉簪，就当谢礼吧！")
			gr.Love += 8
			g.player.addItem("lingzhi")
			g.player.addItem("yuzan")
		case 1:
			add("qing", "糟了糟了！山贼抢走了我进的一批药材，还打伤了伙计……")
			add("qing", "他们说要把药材卖到黑风寨去！我……我不知道该怎么办了……")
			add("hero", "别怕，我去帮你讨回来。")
			add("qing", "（黑风寨一行，你帮她夺回药材）你……你真的做到了！这个手帕你收着，是我绣的！")
			gr.Love += 10
			g.player.addItem("juan")
		case 2:
			add("qing", "药铺的定期药材被黑风寨扣押了，再不交货，爹爹的药铺就开不下去了……")
			add("hero", "我陪你去黑风寨说清楚。")
			add("qing", "（你们一起说服了寨主）太好了……有你在，我什么都不怕了。")
			add("qing", "这是我存的千年灵芝，送给你！以后……也请你多多关照了。")
			gr.Love += 12
			g.player.addItem("lingzhi")
		}
	case "luoyao":
		switch gr.Event {
		case 0:
			add("luoyao", "喂，你来得正好。最近城里闹采花贼，我缺个帮手。")
			add("hero", "要我怎么帮？")
			add("luoyao", "帮我盯住城南的巷子，见到形迹可疑的人就喊一声。")
			add("luoyao", "（一番蹲守后你协助抓住了贼人）干得漂亮！这个令牌送你，凭它可以找我帮忙。")
			gr.Love += 8
			g.player.addItem("zhetie")
		case 1:
			add("luoyao", "案子查到头了——黑风寨里有人给采花贼通风报信！")
			add("hero", "我陪你去黑风寨走一趟。")
			add("luoyao", "（你们揪出了内应）痛快！跟你并肩办案，是我近来最顺心的事。")
			add("luoyao", "喏，这壶女儿红，当我谢你的！")
			gr.Love += 10
			g.player.addItem("nvhong")
		case 2:
			add("luoyao", "我怀疑黑风寨与血月教暗中勾结……可这案子太大，我一个人的话没人信。")
			add("hero", "我信你。我陪你把证据找齐。")
			add("luoyao", "（你们在天山截获了血月教的密信）……谢谢你。")
			add("luoyao", "等这案子结了……我想跟你一起，把这江湖再走一遍。")
			gr.Love += 12
			g.player.addItem("mixin")
		}
	case "yueli":
		switch gr.Event {
		case 0:
			add("yueli", "……你三番两次上天山，究竟为了什么？")
			add("hero", "我听说天山雪莲能救人，想求一株。")
			add("yueli", "（她沉默片刻，摘下一朵雪莲递来）……拿去吧。下次，不必再来了。")
			add("yueli", "（她目送你离开，眼底却有一丝不易察觉的波动）")
			gr.Love += 6
			g.player.addItem("ganlu")
		case 1:
			add("yueli", "你……真的想听我的故事？")
			add("yueli", "我原是血月教的圣女。教中血祭无道，我叛教出逃，躲在这天山之上。")
			add("hero", "血月教的事，与你无关。你只做你自己就好。")
			add("yueli", "（她怔怔看着你，声音微微发颤）……从未有人，对我说过这样的话。")
			gr.Love += 9
		case 2:
			add("", "血月护法追上天山，剑光如血！")
			add("hero", "小心！")
			add("yueli", "（她与你并肩而战，击败护法后靠在崖边喘息）……谢你。")
			add("yueli", "这天山之上，从此……有你，我便不再孤单了。")
			gr.Love += 12
		}
	case "xiaoman":
		switch gr.Event {
		case 0:
			add("xiaoman", "喂！陪本小姐玩嘛！整天就知道打打杀杀，多没意思！")
			add("hero", "想玩什么？")
			add("xiaoman", "放纸鸢！我在集市看到一只好漂亮的花纸鸢，你给我买来，我教你放！")
			add("xiaoman", "（纸鸢飞上天，她笑得前仰后合）哈哈！你这呆子，跑得还没纸鸢快！")
			gr.Love += 7
			g.player.addItem("zhihua")
		case 1:
			add("xiaoman", "哼！寨里那帮家伙又笑我“大小姐不会武功”，气死我了！")
			add("hero", "我教你几招防身的吧。")
			add("xiaoman", "（练了一下午，她累得直喘气却不肯认输）……哼，明天继续！你可得来！")
			add("xiaoman", "喏！这是我爹的藏酒，我偷了一坛给你！不许说出去！")
			gr.Love += 9
			g.player.addItem("nvhong")
		case 2:
			add("xiaoman", "娘要和血月教做买卖……我总觉得不对。你陪我去劝劝她好不好？")
			add("hero", "好，我陪你去。")
			add("xiaoman", "（你帮她劝住了寨主）娘亲终于肯听我说话了……都是因为你！")
			add("xiaoman", "本小姐宣布！从今天起，你就是我小蛮罩着的人了！")
			gr.Love += 11
		}
	case "suxue":
		switch gr.Event {
		case 0:
			add("", "古墓深处传来幽幽琴声，你不自觉走了进去，却迷了路……")
			add("suxue", "（一袭紫衣现于暗处）……迷路之人，随我来。")
			add("suxue", "（她引你走出古墓，转身欲走）……古墓阴寒，莫要再来。")
			add("hero", "等等——你叫什么名字？")
			add("suxue", "……素雪。")
			gr.Love += 7
		case 1:
			add("suxue", "你受伤了。墓中寒玉床可疗伤，……随我来。")
			add("", "寒玉床上，她的指尖为你推宫过血，一股暖流涌遍全身。")
			add("suxue", "（她背过身去，声音低了几分）……明日，还是别再来了。")
			add("hero", "可我想来看看你。")
			add("suxue", "（她沉默良久）……随你。")
			gr.Love += 9
		case 2:
			add("", "古墓震动！沉睡的尸王破棺而出！")
			add("suxue", "（拔剑护在你身前）快走！这不是你能应付的！")
			add("hero", "我留下，与你并肩！")
			add("suxue", "（激战之后，她望着你染血的衣襟，声音第一次带了温度）……傻子。")
			add("suxue", "……这古墓，好像也没那么冷了。")
			gr.Love += 12
		}
	}
	// 事件结束标记（由 Choice 的 event_end action 推进）
	ls = append(ls, DialogLine{
		Speaker: "",
		Text:    "（与" + gr.Name + "的羁绊加深了）",
		Choices: []Choice{{
			Text: "继续聊", Love: nil,
			Action: "event_end",
		}},
	})
	return ls
}

// 女主告白（好感 ≥80 且 3 事件完成）
func (g *Game) girlConfess(gr *Girl) []DialogLine {
	var ls []DialogLine
	ending := ""
	switch gr.ID {
	case "qing":
		ls = append(ls,
			DialogLine{Speaker: "qing", Text: "（青州城的月夜，药香氤氲，她红着脸拦住你）"},
			DialogLine{Speaker: "qing", Text: "我……我有句话，藏在心里好久了。这一路的灵芝、玉簪、手帕，都是我的心意。"},
			DialogLine{Speaker: "hero", Text: "青儿，我也是。"},
			DialogLine{Speaker: "qing", Text: "（她扑进你怀里，泪光盈盈）那……那以后，青儿的药铺，就是你的家了！"},
		)
		ending = "青儿结局：青州城药铺的门口，从此多了一双忙碌的身影。药香氤氲间，江湖路远，有你相伴。"
	case "luoyao":
		ls = append(ls,
			DialogLine{Speaker: "luoyao", Text: "（洛阳城头，晚风猎猎，她抱剑而立，头也不回）"},
			DialogLine{Speaker: "luoyao", Text: "喂，我这个人不会说软话……但这江湖，我想跟你一起走。"},
			DialogLine{Speaker: "hero", Text: "一言为定。"},
			DialogLine{Speaker: "luoyao", Text: "（她转身，月光下笑得肆意）那说好了——你惹的祸，我兜着；我办的案，你跟着！"},
		)
		ending = "洛瑶结局：洛阳府的案卷上，从此多了“侠侣双飞”的传说。红衣飒沓，快意恩仇，江湖皆闻其名。"
	case "yueli":
		ls = append(ls,
			DialogLine{Speaker: "yueli", Text: "（天山之巅，雪落无声，她静静立在月下）"},
			DialogLine{Speaker: "yueli", Text: "我曾以为，这世间再无人懂我……直到你踏雪而来。"},
			DialogLine{Speaker: "hero", Text: "月璃，跟我下山吧。"},
			DialogLine{Speaker: "yueli", Text: "（她闭上眼，再睁开时，眼底的霜雪化成了春水）……好。天涯海角，我都随你去。"},
		)
		ending = "月璃结局：天山雪融，血月教终成过往。白衣女子挽着你的手走下雪山，从此江湖只余清风明月。"
	case "xiaoman":
		ls = append(ls,
			DialogLine{Speaker: "xiaoman", Text: "（黑风寨的晒谷场，她背着手踱来踱去，终于一跺脚）"},
			DialogLine{Speaker: "xiaoman", Text: "本小姐决定了！以后你去哪，我就去哪！谁欺负你，我就打谁！"},
			DialogLine{Speaker: "hero", Text: "小蛮，你认真的？"},
			DialogLine{Speaker: "xiaoman", Text: "（她仰起头，脸颊通红）认真的！比真金还真！反正……反正我认定你了！"},
		)
		ending = "小蛮结局：黑风寨的千金大小姐，从此跟着你闯荡江湖。一路鸡飞狗跳，却也是最热闹的人间烟火。"
	case "suxue":
		ls = append(ls,
			DialogLine{Speaker: "suxue", Text: "（古墓烛火摇曳，她缓缓摘下常年遮掩的面纱）"},
			DialogLine{Speaker: "suxue", Text: "我守了这古墓十年，本以为自己会孤独终老……直到遇见了你。"},
			DialogLine{Speaker: "hero", Text: "素雪，让我陪你，不再守着冰冷的长夜。"},
			DialogLine{Speaker: "suxue", Text: "（她低垂眼帘，轻声）……好。从此，古墓之外，也有了一盏为我而亮的灯。"},
		)
		ending = "素雪结局：古墓的守墓人，终于走出了那扇石门。紫衣长剑，江湖寻你，从此不再一人。"
	}
	ls = append(ls, DialogLine{
		Speaker: "",
		Text:    "—— " + gr.Name + " 攻略完成！——",
		Choices: []Choice{{
			Text: "与她携手江湖", Action: "ending", Param: ending,
		}},
	})
	return ls
}

// 女主对话主流程
func (g *Game) girlDialogLines(gr *Girl) []DialogLine {
	var lines []DialogLine
	// 欢迎语分档
	band := 0
	if gr.Love >= 30 {
		band = 1
	}
	if gr.Love >= 55 {
		band = 2
	}
	if gr.Love >= 75 {
		band = 3
	}
	lines = append(lines, DialogLine{Speaker: gr.ID, Text: girlChat[gr.ID][band]})

	// 告白检测（事件全完成 + 好感 ≥80）
	if gr.Event >= 3 && gr.Love >= 80 && !g.girlEnded(gr.ID) {
		lines = append(lines, g.girlConfess(gr)...)
		return lines
	}
	// 事件检测
	if gr.Event < 3 {
		need := girlEventNeed[gr.ID][gr.Event]
		if gr.Love >= need {
			lines = append(lines, g.girlEvent(gr)...)
			return lines
		}
	}
	// 主菜单选项
	lines = append(lines, DialogLine{
		Speaker: "",
		Text:    "",
		Choices: g.mainChoices(gr),
	})
	return lines
}

func (g *Game) girlEnded(id string) bool {
	return g.player.GirlEvents[id] >= 100 // 标记结局已触发
}

func (g *Game) mainChoices(gr *Girl) []Choice {
	cs := []Choice{
		{Text: "送她一件礼物", Action: "give"},
		{Text: "告辞", Action: "end"},
	}
	if !gr.JoinBattle && gr.Love >= 40 {
		cs = append([]Choice{{Text: "邀她同行（加入战斗）", Action: "join"}}, cs...)
	}
	return cs
}

// ===== 静态对话树 =====
var dialogTree = map[string][]DialogLine{
	"intro": {
		{Speaker: "", Text: "（青州城外，桃花正盛。你初入江湖，还只是个懵懂的少女桃夭。）"},
		{Speaker: "", Text: "（城中药铺门前，一名翠衫少女正踮着脚晾晒药材。）"},
		{Speaker: "qing", Text: "咦？这位姑娘面生得很，是刚来青州城吧？"},
		{Speaker: "hero", Text: "姑娘好，小女子初到贵地，想找个地方歇脚。"},
		{Speaker: "qing", Text: "那去城东客栈吧！老板娘人可好啦。对了，我叫青儿，就在这家药铺帮忙。"},
		{Speaker: "qing", Text: "若是伤了病了，尽管来找我！青儿的药，保你生龙活虎！"},
		{Speaker: "", Text: "（江湖路远，红颜初遇。你的故事，从这座小城开始。）"},
		{Speaker: "", Text: "操作提示：WASD 移动，靠近 NPC 按 E 对话。"},
	},
	"npc_inn": {
		{Speaker: "npc", Text: "客官，住店吗？一晚十两银子，包吃包住，还能舒舒服服睡一觉！"},
		{Speaker: "", Text: "（你决定在此歇脚）"},
		{Speaker: "npc", Text: "好嘞！给您上房一间！明早起来，保准龙精虎猛！"},
	},
}

// ===== 送礼菜单 =====
type GiftMenu struct {
	active bool
	items  []*Item
	sel    int
	girl   *Girl
}

func (g *Game) startGift(gr *Girl) {
	var gifts []*Item
	for _, id := range g.player.Items {
		it := findItem(id)
		if it != nil && it.Gift {
			gifts = append(gifts, it)
		}
	}
	if len(gifts) == 0 {
		g.msg = "没有可以送的礼物"
		g.msgT = 1.5
		return
	}
	g.giftMenu = GiftMenu{active: true, items: gifts, sel: 0, girl: gr}
}

// giftUpdate 送礼菜单逻辑（在 dialog.Update 前调用）。
func (g *Game) giftUpdate(dt float64) bool {
	in := g.eng.Input()
	gd := &g.giftMenu
	if in.Pressed(engine.KeyEscape) || in.Pressed(engine.KeyChar('q')) || in.Pressed(engine.KeyChar('Q')) {
		gd.active = false
		return false
	}
	if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
		gd.sel = (gd.sel + len(gd.items) - 1) % len(gd.items)
	}
	if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
		gd.sel = (gd.sel + 1) % len(gd.items)
	}
	// 触屏：点击礼物行直接送出；点击面板外取消
	if in.MouseLeftPressed {
		if i := hitRow(in.MouseX, in.MouseY, 200, 176, 24, 28, len(gd.items)); i >= 0 {
			gd.sel = i
			if g.sendGift(gd) {
				return true
			}
			return false
		}
		if in.MouseX < 200 || in.MouseX > 760 || in.MouseY < 120 || in.MouseY > 420 {
			gd.active = false
			return false
		}
	}
	if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || in.MouseLeftPressed {
		if g.sendGift(gd) {
			return true
		}
	}
	return false
}

// sendGift 送出当前选中的礼物，成功返回 true。
func (g *Game) sendGift(gd *GiftMenu) bool {
	it := gd.items[gd.sel]
	if g.player.removeItem(it.ID) {
		gd.girl.Love += it.Love
		if gd.girl.Love > 100 {
			gd.girl.Love = 100
		}
		g.player.GirlLove[gd.girl.ID] = gd.girl.Love
		g.msg = gd.girl.Name + " 收下「" + it.Name + "」，好感度 +" + itoa(it.Love) + "！"
		g.msgT = 2.5
		gd.active = false
		// 送礼后可再聊
		g.dialog.StartGirl(g, gd.girl)
		return true
	}
	return false
}

// giftDraw 送礼菜单渲染。
func (g *Game) giftDraw(c *engine.Canvas) {
	gd := &g.giftMenu
	dx, dy, dw, dh := 200, 120, 560, 300
	c.FillRect(dx, dy, dw, dh, engine.Color{R: 22, G: 22, B: 34})
	c.Rect(dx, dy, dw, dh, engine.Color{R: 200, G: 200, B: 220})
	drawTextShadow(c, dx+20, dy+16, "选择礼物送给 "+gd.girl.Name, engine.Color{R: 255, G: 230, B: 130})
	y := dy + 56
	for i, it := range gd.items {
		col := engine.Color{R: 220, G: 220, B: 240}
		mark := "  "
		if i == gd.sel {
			col = engine.Color{R: 255, G: 235, B: 130}
			mark = "▶"
		}
		drawTextShadow(c, dx+30, y, mark+" "+it.Name+"（好感+"+itoa(it.Love)+"）", col)
		drawText(c, dx+250, y, it.Desc, engine.Color{R: 150, G: 150, B: 170})
		y += lineH() + 8
	}
	drawText(c, dx+20, dy+dh-34, "W/S 选择  回车 送出  Esc 取消；触屏点击行送出，点空白取消", engine.Color{R: 150, G: 150, B: 170})
}
