// tilemap.go - 地图数据 / 碰撞 / 传送 / 事件
package main

// 字符到 tile 的映射
var tileMap = map[byte]int{
	'.': TileGrass, ',': TilePath, '#': TileWall, '=': TileFloor,
	'~': TileWater, 'T': TileTree, 'F': TileFlower, '^': TileRoof,
	'*': TileSnow, 'C': TileCave, 'B': TileWoodPath, 'D': TileDoor,
}

// MapDef 一张地图
type MapDef struct {
	ID    string
	Name  string
	Data  []string
	W, H  int
	Music string
	// 遇敌
	Encounters []string // 怪物名列表（可空=安全区）
	EncRate    float64  // 每步遇敌概率
}

var maps = []*MapDef{
	{
		ID: "qingzhou", Name: "青州城",
		Data: []string{
			"####################",
			"#TT..........TT....#",
			"#.................#",
			"#.DD............DD.#",
			"#.^^............^^.#",
			"#...................#",
			"#,,,,,,,,,,,,,,,,,,,#",
			"#...................#",
			"#.DD............DD.#",
			"#.^^............^^.#",
			"#...................#",
			"#...................#",
			"####################",
		},
		Encounters: nil, EncRate: 0,
	},
	{
		ID: "wild", Name: "郊野",
		Data: []string{
			"TTTTTTTTTTTTTTTTTTTT",
			"T..........TT......T",
			"T..,,,,,,,.........T",
			"T..,.....,..TT..F..T",
			"T..,.....T........T",
			"T..,.....T...TT...T",
			"T..,.....T.......F.T",
			"T..,......T.......T",
			"T..,,,,,,,,........T",
			"T.......F....TT....T",
			"T..TT..............T",
			"T.........TT....F..T",
			"TTTTTTTTTTTTTTTTTTTT",
		},
		Encounters: []string{"野狼", "山贼", "毒蛇"}, EncRate: 0.18,
	},
	{
		ID: "luoyang", Name: "洛阳城",
		Data: []string{
			"####################",
			"#..DD..........DD..#",
			"#..^^..........^^..#",
			"#...................#",
			"#,,,,,,,,,,,,,,,,,,,#",
			"#...................#",
			"#..DD..........DD..#",
			"#..^^..........^^..#",
			"#...................#",
			"#,,,,,,,,,,,,,,,,,,,#",
			"#...................#",
			"#...................#",
			"####################",
		},
		Encounters: nil, EncRate: 0,
	},
	{
		ID: "heifeng", Name: "黑风寨",
		Data: []string{
			"####################",
			"#........TT........#",
			"#..CCCCCCCCCCCC....#",
			"#..C==========C....#",
			"#..C==========C....#",
			"#..CCCCCCCCCCCC....#",
			"#...................#",
			"#,,,,,,,,,,,,,,,,,,,#",
			"#...................#",
			"#..T............T..#",
			"#...................#",
			"#TT..............TT#",
			"####################",
		},
		Encounters: []string{"黑风寨兵"}, EncRate: 0.12,
	},
	{
		ID: "tianshan", Name: "天山",
		Data: []string{
			"********************",
			"*TT..............TT*",
			"*........TT........*",
			"*..,................*",
			"*..,....TT.........*",
			"*..,................*",
			"*..,................*",
			"*..,,,,,,...........*",
			"*........TT........*",
			"*......F......TT...*",
			"*...................*",
			"*...................*",
			"********************",
		},
		Encounters: []string{"血月教徒", "野狼"}, EncRate: 0.15,
	},
	{
		ID: "gumu", Name: "古墓",
		Data: []string{
			"CCCCCCCCCCCCCCCCCCCC",
			"C..................C",
			"C..CCCCCC..CCCCCC..C",
			"C..C====C..C====C..C",
			"C..C====C..C====C..C",
			"C..CCCCCC..CCCCCC..C",
			"C..................C",
			"C,,,............,,,C",
			"C..................C",
			"C..CCCCCCCCCCCCCC..C",
			"C..C============C..C",
			"C..C============C..C",
			"CCCCCCCCCCCCCCCCCCCC",
		},
		Encounters: []string{"古墓尸卫", "毒蛇"}, EncRate: 0.15,
	},
}

// tileAt 返回地图坐标的 tile 类型（越界视为不可走虚空）。
func (m *MapDef) tileAt(x, y int) int {
	if x < 0 || y < 0 || y >= len(m.Data) || x >= len(m.Data[y]) {
		return TileVoid
	}
	ch := m.Data[y][x]
	if t, ok := tileMap[ch]; ok {
		return t
	}
	return TileGrass
}

// walkable 判断坐标可否走。
func (m *MapDef) walkable(x, y int) bool {
	if x < 0 || y < 0 || y >= len(m.Data) || x >= len(m.Data[y]) {
		return false
	}
	return isWalkable(m.tileAt(x, y))
}

func findMap(id string) *MapDef {
	for _, m := range maps {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// ===== 事件 =====
type Event struct {
	MapID string
	X, Y  int
	Kind  string // warp / npc / boss / chest
	Param string // npc: girl id 或 npc id；boss: 怪物名；warp: 目标地图
	TX    int    // warp 目标 x
	TY    int    // warp 目标 y
}

// events 全局事件表。
var events = []Event{
	// 传送门（地图边缘）
	{MapID: "qingzhou", X: 9, Y: 12, Kind: "warp", Param: "wild", TX: 9, TY: 1},
	{MapID: "qingzhou", X: 10, Y: 12, Kind: "warp", Param: "wild", TX: 10, TY: 1},
	{MapID: "wild", X: 9, Y: 0, Kind: "warp", Param: "qingzhou", TX: 9, TY: 11},
	{MapID: "wild", X: 10, Y: 0, Kind: "warp", Param: "qingzhou", TX: 10, TY: 11},
	{MapID: "wild", X: 19, Y: 6, Kind: "warp", Param: "luoyang", TX: 1, TY: 6},
	{MapID: "luoyang", X: 0, Y: 6, Kind: "warp", Param: "wild", TX: 18, TY: 6},
	{MapID: "wild", X: 0, Y: 6, Kind: "warp", Param: "heifeng", TX: 18, TY: 6},
	{MapID: "heifeng", X: 19, Y: 6, Kind: "warp", Param: "wild", TX: 1, TY: 6},
	{MapID: "wild", X: 5, Y: 0, Kind: "warp", Param: "tianshan", TX: 5, TY: 11},
	{MapID: "tianshan", X: 5, Y: 12, Kind: "warp", Param: "wild", TX: 5, TY: 1},
	{MapID: "wild", X: 15, Y: 0, Kind: "warp", Param: "gumu", TX: 15, TY: 11},
	{MapID: "gumu", X: 15, Y: 12, Kind: "warp", Param: "wild", TX: 15, TY: 1},

	// NPC（女主）
	{MapID: "qingzhou", X: 3, Y: 6, Kind: "npc", Param: "qing"},
	{MapID: "luoyang", X: 3, Y: 4, Kind: "npc", Param: "luoyao"},
	{MapID: "heifeng", X: 4, Y: 8, Kind: "npc", Param: "xiaoman"},
	{MapID: "tianshan", X: 10, Y: 5, Kind: "npc", Param: "yueli"},
	{MapID: "gumu", X: 10, Y: 7, Kind: "npc", Param: "suxue"},
	// 客栈老板娘（休息+送礼）
	{MapID: "qingzhou", X: 16, Y: 6, Kind: "npc", Param: "inn"},
	{MapID: "luoyang", X: 16, Y: 4, Kind: "npc", Param: "inn"},

	// Boss 剧情战
	{MapID: "heifeng", X: 10, Y: 7, Kind: "boss", Param: "黑风女寨主"},
	{MapID: "gumu", X: 10, Y: 11, Kind: "boss", Param: "尸王"},
	{MapID: "tianshan", X: 10, Y: 4, Kind: "boss", Param: "血月护法"},

	// 宝箱
	{MapID: "wild", X: 4, Y: 4, Kind: "chest", Param: "yuzan"},
	{MapID: "wild", X: 15, Y: 10, Kind: "chest", Param: "nvhong"},
	{MapID: "heifeng", X: 2, Y: 2, Kind: "chest", Param: "juan"},
	{MapID: "tianshan", X: 15, Y: 8, Kind: "chest", Param: "ganlu"},
	{MapID: "gumu", X: 2, Y: 7, Kind: "chest", Param: "nuanyu"},
	{MapID: "luoyang", X: 17, Y: 8, Kind: "chest", Param: "zhetie"},
}

// eventAt 返回地图坐标处的事件（可走交互）。
func eventAt(mapID string, x, y int) *Event {
	for i := range events {
		e := &events[i]
		if e.MapID == mapID && e.X == x && e.Y == y {
			return e
		}
	}
	return nil
}
