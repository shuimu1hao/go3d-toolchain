// assets.go - 程序生成的 RPG Maker 风格像素素材
package main

import (
	"math/rand"

	"go2dgame/engine"
)

// ===== Tile 类型 =====
const (
	TileVoid     = iota // 虚空（不可走）
	TileGrass           // 草地
	TilePath            // 泥土路
	TileWall            // 砖墙
	TileFloor           // 木地板
	TileStone           // 石地板
	TileWater           // 水面
	TileTree            // 树
	TileFlower          // 花丛
	TileRoof            // 屋顶
	TileSnow            // 雪地
	TileCave            // 石壁
	TileWoodPath        // 木桥
	TileDoor            // 门（事件用，可走）
	TileNum
)

var tiles = make(map[int]*engine.Sprite) // 16x16

// tileWalkable 定义哪些 tile 可走。
var tileWalkable = map[int]bool{
	TileGrass: true, TilePath: true, TileFloor: true, TileStone: true,
	TileFlower: true, TileSnow: true, TileWoodPath: true, TileDoor: true,
}

func isWalkable(t int) bool { return tileWalkable[t] }

// noise 返回 [0,n) 伪随机（每 tile 独立 seed）。
func noise(x, y, seed, n int) int {
	r := rand.New(rand.NewSource(int64(x*73856093 ^ y*19349663 ^ seed)))
	return r.Intn(n)
}

// genTiles 生成全部 tile 精灵。
func genTiles() {
	grass := engine.Color{R: 96, G: 168, B: 84}
	grassD := engine.Color{R: 82, G: 150, B: 72}
	path := engine.Color{R: 176, G: 140, B: 96}
	pathD := engine.Color{R: 158, G: 124, B: 84}
	wall := engine.Color{R: 128, G: 108, B: 92}
	wallD := engine.Color{R: 104, G: 86, B: 74}
	floor := engine.Color{R: 178, G: 138, B: 96}
	floorD := engine.Color{R: 152, G: 116, B: 82}
	stone := engine.Color{R: 148, G: 148, B: 150}
	stoneD := engine.Color{R: 124, G: 124, B: 128}
	water := engine.Color{R: 60, G: 110, B: 190}
	waterD := engine.Color{R: 48, G: 92, B: 164}
	roof := engine.Color{R: 120, G: 82, B: 66}
	roofD := engine.Color{R: 98, G: 64, B: 52}
	snow := engine.Color{R: 232, G: 238, B: 244}
	snowD := engine.Color{R: 206, G: 214, B: 224}
	cave := engine.Color{R: 88, G: 82, B: 92}
	caveD := engine.Color{R: 70, G: 64, B: 74}
	wood := engine.Color{R: 140, G: 96, B: 56}
	woodD := engine.Color{R: 116, G: 78, B: 46}

	t := func(tp int, s *engine.Sprite) { tiles[tp] = s }

	// 虚空
	t(TileVoid, solidTile(engine.Color{R: 10, G: 12, B: 16}))

	// 草地：绿底 + 深色斑点 + 几根草
	{
		s := solidTile(grass)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if noise(x, y, 1, 10) < 2 {
					s.Set(x, y, grassD, 255)
				}
			}
		}
		for i := 0; i < 4; i++ {
			x := noise(i, 1, 2, 16)
			y := noise(i, 2, 3, 14) + 1
			s.Set(x, y, engine.Color{R: 70, G: 140, B: 60}, 255)
			if y+1 < 16 {
				s.Set(x, y+1, engine.Color{R: 70, G: 140, B: 60}, 255)
			}
		}
		t(TileGrass, s)
	}
	// 泥土路
	{
		s := solidTile(path)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if noise(x, y, 4, 8) < 2 {
					s.Set(x, y, pathD, 255)
				}
			}
		}
		t(TilePath, s)
	}
	// 砖墙
	{
		s := solidTile(wall)
		for y := 0; y < 16; y += 4 {
			for x := 0; x < 16; x++ {
				off := 0
				if (y/4)%2 == 1 {
					off = 4
				}
				// 砖缝
				if x == (3+off)%8 || x == (11+off)%8 {
					s.Set(x, y, wallD, 255)
					if y+1 < 16 {
						s.Set(x, y+1, wallD, 255)
					}
				}
			}
			if y+3 < 16 {
				for x := 0; x < 16; x++ {
					s.Set(x, y+3, wallD, 255)
				}
			}
		}
		t(TileWall, s)
	}
	// 木地板
	{
		s := solidTile(floor)
		for y := 0; y < 16; y += 4 {
			for x := 0; x < 16; x++ {
				if y+3 < 16 {
					s.Set(x, y+3, floorD, 255)
				}
				if noise(x, y, 5, 12) == 0 {
					s.Set(x, y, floorD, 255)
				}
			}
		}
		t(TileFloor, s)
	}
	// 石地板
	{
		s := solidTile(stone)
		for y := 0; y < 16; y += 4 {
			for x := 0; x < 16; x++ {
				if (x/4+y/4)%2 == 0 {
					if y+3 < 16 {
						s.Set(x, y+3, stoneD, 255)
					}
					if x == 0 || x == 4 || x == 8 || x == 12 {
						s.Set(x, y, stoneD, 255)
					}
				} else {
					if x == 2 || x == 6 || x == 10 || x == 14 {
						s.Set(x, y, stoneD, 255)
					}
				}
			}
		}
		t(TileStone, s)
	}
	// 水面
	{
		s := solidTile(water)
		for x := 0; x < 16; x += 4 {
			yy := noise(x, 7, 6, 14)
			for i := 0; i < 3; i++ {
				s.Set(x+i, yy, waterD, 255)
			}
			yy2 := noise(x, 8, 7, 14)
			for i := 0; i < 2; i++ {
				s.Set(x+i, yy2, engine.Color{R: 84, G: 136, B: 216}, 255)
			}
		}
		t(TileWater, s)
	}
	// 树（16x16：树干 + 树冠）
	{
		s := engine.NewSprite(16, 16)
		// 树冠
		for y := 1; y < 11; y++ {
			for x := 2; x < 14; x++ {
				dx := x - 8
				dy := y - 5
				if dx*dx/4+dy*dy < 16 && noise(x, y, 8, 5) < 4 {
					col := engine.Color{R: 56, G: 132, B: 52}
					if noise(x, y, 9, 5) == 0 {
						col = engine.Color{R: 44, G: 112, B: 44}
					}
					s.Set(x, y, col, 255)
				}
			}
		}
		// 树干
		for y := 10; y < 16; y++ {
			s.Set(7, y, engine.Color{R: 96, G: 66, B: 40}, 255)
			s.Set(8, y, engine.Color{R: 110, G: 76, B: 46}, 255)
		}
		t(TileTree, s)
	}
	// 花丛
	{
		s := solidTile(grass)
		for i := 0; i < 5; i++ {
			x := noise(i, 3, 10, 14) + 1
			y := noise(i, 4, 11, 14) + 1
			cols := []engine.Color{
				{R: 240, G: 120, B: 160}, {R: 250, G: 220, B: 90},
				{R: 200, G: 120, B: 240}, {R: 250, G: 150, B: 120},
			}
			s.Set(x, y, cols[i%len(cols)], 255)
			s.Set(x+1, y, cols[(i+1)%len(cols)], 255)
		}
		t(TileFlower, s)
	}
	// 屋顶
	{
		s := solidTile(roof)
		for y := 0; y < 16; y += 4 {
			for x := 0; x < 16; x++ {
				if (x/4+y/4)%2 == 1 {
					s.Set(x, y, roofD, 255)
				}
				if y+3 < 16 {
					s.Set(x, y+3, roofD, 255)
				}
			}
		}
		t(TileRoof, s)
	}
	// 雪地
	{
		s := solidTile(snow)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if noise(x, y, 12, 10) < 2 {
					s.Set(x, y, snowD, 255)
				}
			}
		}
		t(TileSnow, s)
	}
	// 石壁（地宫）
	{
		s := solidTile(cave)
		for y := 0; y < 16; y += 5 {
			for x := 0; x < 16; x++ {
				if noise(x, y, 13, 6) < 2 {
					s.Set(x, y, caveD, 255)
				}
			}
		}
		for i := 0; i < 3; i++ {
			s.Set(noise(i, 5, 14, 16), noise(i, 6, 15, 16), caveD, 255)
		}
		t(TileCave, s)
	}
	// 木桥
	{
		s := solidTile(wood)
		for x := 0; x < 16; x++ {
			if x%4 == 3 {
				for y := 0; y < 16; y++ {
					s.Set(x, y, woodD, 255)
				}
			}
			if x == 7 || x == 8 {
				s.Set(x, 15, woodD, 255)
			}
		}
		t(TileWoodPath, s)
	}
	// 门（深色木门）
	{
		s := solidTile(engine.Color{R: 96, G: 66, B: 44})
		for y := 0; y < 16; y += 4 {
			for x := 0; x < 16; x++ {
				if y+3 < 16 {
					s.Set(x, y+3, engine.Color{R: 76, G: 52, B: 36}, 255)
				}
			}
		}
		s.Set(7, 10, engine.Color{R: 220, G: 180, B: 90}, 255) // 门环
		t(TileDoor, s)
	}
}

// solidTile 生成纯色 16x16 精灵。
func solidTile(col engine.Color) *engine.Sprite {
	s := engine.NewSprite(16, 16)
	s.Fill(col)
	return s
}

// ===== 角色精灵（16x16 chibi，4 方向 × 2 帧）=====
type CharColors struct {
	Hair   engine.Color // 头发
	HairD  engine.Color // 头发阴影
	Skin   engine.Color // 皮肤
	Cloth  engine.Color // 衣服
	ClothD engine.Color // 衣服阴影
	Pants  engine.Color // 裤子/裙
	Shoes  engine.Color // 鞋
	Acc    engine.Color // 发饰/眼睛点缀
	Bust   int          // 0 贫乳 / 1+ 大胸
	// 发型：0短发 1双马尾 2高马尾 3长发 4发髻
	HairStyle int
}

// charSprites 返回 map["d0"|"d1"|"u0"|"u1"|"l0"|"l1"|"r0"|"r1"]*Sprite
// makeWarpSprite 传送门精灵（16x16）。bright=true 亮帧 / false 暗帧，交替闪烁。
func makeWarpSprite(bright bool) *engine.Sprite {
	s := engine.NewSprite(16, 16)
	// 字符：F=门框紫 V=内圈紫 W=漩涡亮白
	frame := []string{
		"................",
		".FFF FFFFFFFF...",
		"FFVVVVVVVVVVVFF.",
		"FVWWWWWWWWWWWVF.",
		"FVWWWWWWWWWWWVF.",
		"FVWWW WWW WWWVF.",
		"FVWWW WWW WWWVF.",
		"FVWWWWWWWWWWWVF.",
		"FVWWWWWWWWWWWVF.",
		"FVWWW WWW WWWVF.",
		"FVWWW WWW WWWVF.",
		"FVWWWWWWWWWWWVF.",
		"FFVVVVVVVVVVVFF.",
		".FFF FFFFFFFF...",
		"................",
		"................",
	}
	var fC, vC, wC engine.Color
	if bright {
		fC = engine.Color{R: 168, G: 96, B: 232}  // 亮紫边框
		vC = engine.Color{R: 120, G: 76, B: 200}  // 内圈紫
		wC = engine.Color{R: 230, G: 238, B: 255} // 漩涡亮白
	} else {
		fC = engine.Color{R: 96, G: 56, B: 150}   // 暗紫边框
		vC = engine.Color{R: 70, G: 46, B: 120}
		wC = engine.Color{R: 150, G: 165, B: 200}
	}
	for y, row := range frame {
		for x := 0; x < len(row) && x < 16; x++ {
			switch row[x] {
			case 'F':
				s.Set(x, y, fC, 255)
			case 'V':
				s.Set(x, y, vC, 255)
			case 'W':
				s.Set(x, y, wC, 255)
			}
		}
	}
	return s
}

func makeChar(cc CharColors) map[string]*engine.Sprite {
	out := map[string]*engine.Sprite{}
	// ===== 发型模板（head = 前 10 行：头+脸；body = 后 6 行：衣裤鞋）=====
	// 字符：.空 H发 S肤 E眼白 C衣 P裤 B鞋 A发饰
	headDefault := []string{
		"................",
		".....HHHHHH.....",
		"...HHHHHHHHHH...",
		"..HHHSSSSSSHHH..",
		"..HHSSSSSSSSHH..",
		"..HSSSSSSSSSSH..",
		"..SSSESSSSESSS..",
		"..SSSESSSSESSS..",
		"...SSSSSSSSSS...",
		"...SSSSSSSSSS...",
	}
	headTwin := []string{ // 双马尾：两侧垂发
		"................",
		".....HHHHHH.....",
		"..HHHHHHHHHHHH..",
		".HHHHSSSSSSHHHH.",
		".HHHHSSSSSSHHHH.",
		".HHHSSSSSSSSHHH.",
		".HHHSSESSSESSH..",
		".HHHSSESSSESSH..",
		"..HHSSSSSSSSHH..",
		"..H.SSSSSSSS.H..",
	}
	headPony := []string{ // 高马尾：头顶束发上翘
		"........H.......",
		".......HHH......",
		"......HHHHH.....",
		".....HHHHHHH....",
		"...HHHSSSSSHHH..",
		"..HHSSSSSSSSHH..",
		"..HSSSSSSSSSSH..",
		"..SSSESSSSESSS..",
		"..SSSESSSSESSS..",
		"...SSSSSSSSSS...",
	}
	headLong := []string{ // 长发：垂到胸两侧
		"................",
		".....HHHHHH.....",
		"...HHHHHHHHHH...",
		"..HHHSSSSSSHHH..",
		".HHHSSSSSSSSHHH.",
		".HHSSSSSSSSSSHH.",
		".HHSSSESSSSESSH.",
		".HHSSSESSSSESSH.",
		".HH.SSSSSSSS.HH.",
		".HH.SSSSSSSS.HH.",
	}
	headBun := []string{ // 发髻：头顶圆髻（A=发饰）
		"......AAAA......",
		"......HHHH......",
		"...HHHHHHHHHH...",
		"..HHHSSSSSSHHH..",
		"..HHSSSSSSSSHH..",
		"..HSSSSSSSSSSH..",
		"..SSSESSSSESSS..",
		"..SSSESSSSESSS..",
		"...SSSSSSSSSS...",
		"...SSSSSSSSSS...",
	}
	heads := [5][]string{headDefault, headTwin, headPony, headLong, headBun}

	// side 变体（侧视，脑后=右侧 x11-13）
	sideDefault := []string{
		"................",
		"......HHHH......",
		"....HHHHHHHH....",
		"...HHHSSSSSHH...",
		"...HHSSSSSSSH...",
		"....SSSSSSSSS...",
		"....SESSSSSS....",
		"....SESSSSSS....",
		"....SSSSSSSS....",
		"....SSSSSSSS....",
	}
	sideTwin := []string{ // 双马尾：脑后一束垂发
		"................",
		"......HHHH......",
		"....HHHHHHHH....",
		"...HHHSSSSSHH...",
		"...HHSSSSSSHH...",
		"...HSSSSSSSHH...",
		"...HSSSSSSSHH...",
		"...HSSSSSSSHH...",
		"....SSSSSSSHH...",
		"....SSSSSSSS....",
	}
	sideLong := []string{ // 长发：前后垂发
		"................",
		"......HHHH......",
		"....HHHHHHHH....",
		"...HHHSSSSSHH...",
		"..HHSSSSSSSHH...",
		"..HHSSSSSSSSH...",
		"..HHSSSSSSSSH...",
		"..HHSSSSSSSSH...",
		"..HH.SSSSSSSH...",
		"..HH.SSSSSSS....",
	}
	sideHeads := [5][]string{sideDefault, sideTwin, sideDefault, sideLong, sideDefault}

	// up 变体（背面全头发）
	upDefault := []string{
		"................",
		".....HHHHHH.....",
		"...HHHHHHHHHH...",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"...HHHHHHHHHH...",
		"...HHHHHHHHHH...",
	}
	upLong := []string{ // 长发：更长
		"................",
		".....HHHHHH.....",
		"...HHHHHHHHHH...",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		"..HHHHHHHHHHHH..",
	}
	upTwin := []string{ // 双马尾：两侧垂下
		"................",
		".....HHHHHH.....",
		"..HHHHHHHHHHHH..",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		".HHHHHHHHHHHHHH.",
		"..HHHHHHHHHHHH..",
		"..H.HHHHHHHH.H..",
	}
	upBun := []string{ // 发髻：头顶圆髻
		"......HHHH......",
		"......HHHH......",
		"...HHHHHHHHHH...",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"...HHHHHHHHHH...",
		"...HHHHHHHHHH...",
	}
	upHeads := [5][]string{upDefault, upTwin, upDefault, upLong, upBun}

	// 身体模板（后 6 行：row10-15）
	bodyDown := []string{
		"..CCCCCCCCCCCC..",
		".CCCCCCCCCCCCCC.",
		".CCCCCCCCCCCCCC.",
		".CCCCCCCCCCCCCC.",
		"..PPPPPPPPPPPP..",
		"...BBBBBBBBBB...",
	}
	bodyBustDown := []string{
		"..CCCCCCCCCCCC..",
		".CCCCCCCCCCCCCC.",
		".CCCDCCCCCCCDCC.",
		"..CCCCCCCCCCCC..",
		"..PPPPPPPPPPPP..",
		"...BBBBBBBBBB...",
	}
	bodyUp := []string{
		"..CCCCCCCCCCCC..",
		".CCCCCCCCCCCCCC.",
		".CCCCCCCCCCCCCC.",
		".CCCCCCCCCCCCCC.",
		"..PPPPPPPPPPPP..",
		"...BBBBBBBBBB...",
	}
	bodySide := []string{
		"...CCCCCCCCCC...",
		"..CCCCCCCCCCCC..",
		"..CCCCCCCCCCCC..",
		"..CCCCCCCCCCCC..",
		"...PPPPPPPPPP...",
		"....BBBBBBBB....",
	}
	bodyBustSide := []string{
		"...CCCCCCCCCC...",
		"..CCCCCCCCCCCC..",
		"..CCCDCCCCCCCC..",
		"..CCCCCCCCCCCC..",
		"...PPPPPPPPPP...",
		"....BBBBBBBB....",
	}

	hs := cc.HairStyle
	if hs < 0 || hs >= 5 {
		hs = 0
	}
	headD := heads[hs]
	headS := sideHeads[hs]
	headU := upHeads[hs]
	bodyD, bodyS, bodyU := bodyDown, bodySide, bodyUp
	if cc.Bust >= 1 {
		bodyD = bodyBustDown
		bodyS = bodyBustSide
	}
	down := append(append([]string{}, headD...), bodyD...)
	side := append(append([]string{}, headS...), bodyS...)
	up := append(append([]string{}, headU...), bodyU...)

	mk := func(rows []string, suffix string) *engine.Sprite {
		s := engine.NewSprite(16, 16)
		for y, row := range rows {
			for x := 0; x < len(row) && x < 16; x++ {
				ch := row[x]
				switch ch {
				case 'H':
					s.Set(x, y, cc.Hair, 255)
					if noise(x, y, 20, 4) == 0 {
						s.Set(x, y, cc.HairD, 255)
					}
				case 'S':
					s.Set(x, y, cc.Skin, 255)
				case 'E':
					s.Set(x, y, engine.ColWhite, 255)
				case 'C':
					s.Set(x, y, cc.Cloth, 255)
					if noise(x, y, 21, 5) == 0 {
						s.Set(x, y, cc.ClothD, 255)
					}
				case 'P':
					s.Set(x, y, cc.Pants, 255)
				case 'B':
					s.Set(x, y, cc.Shoes, 255)
				case 'A':
					s.Set(x, y, cc.Acc, 255)
				}
			}
		}
		_ = suffix
		return s
	}
	out["d0"] = mk(down, "d0")
	out["u0"] = mk(up, "u0")
	out["l0"] = mk(side, "l0")
	out["r0"] = mk(side, "r0")
	// 走路帧：整体下移 1 行（顶部补空），脚步效果
	step := func(base *engine.Sprite) *engine.Sprite {
		s := engine.NewSprite(16, 16)
		for y := 0; y < 15; y++ {
			for x := 0; x < 16; x++ {
				px := base
				i := (y*16 + x) * 4
				a := px.Pixels[i+3]
				if a == 0 {
					continue
				}
				s.Pixels[((y+1)*16+x)*4] = px.Pixels[i]
				s.Pixels[((y+1)*16+x)*4+1] = px.Pixels[i+1]
				s.Pixels[((y+1)*16+x)*4+2] = px.Pixels[i+2]
				s.Pixels[((y+1)*16+x)*4+3] = px.Pixels[i+3]
			}
		}
		return s
	}
	out["d1"] = step(out["d0"])
	out["u1"] = step(out["u0"])
	out["l1"] = step(out["l0"])
	out["r1"] = step(out["r0"])
	return out
}

// ===== 女主头像（32x32 像素）=====
// 简化 chibi 头像：头发+脸+眼睛+腮红
func makePortrait(hair, hairD, skin, eye, blush engine.Color) *engine.Sprite {
	s := engine.NewSprite(32, 32)
	// 背景透明，画脸（16x16 区域居中放大 2x）
	face := []string{
		"................",
		"..HHHHHHHHHHHH..",
		".HHHHHHHHHHHHHH.",
		".HHSSSSSSSSSSHH.",
		"HHSSSSSSSSSSSSHH",
		"HHSSSSSSSSSSSSHH",
		"HHSSESSSSSSESSHH",
		"HHSSESSSSSSESSHH",
		"HHSSSSSSSSSSSSHH",
		"HHSSSSSSSSSSSSHH",
		"HHSSSSSSSSSSSSHH",
		".HHSSSSSSSSSSHH.",
		".HHSSSSSSSSSSHH.",
		"..HHHHHHHHHHHH..",
		"..HHHHHHHHHHHH..",
		"................",
	}
	// 放大 2 倍画
	for y, row := range face {
		for x := 0; x < len(row) && x < 16; x++ {
			var col engine.Color
			switch row[x] {
			case 'H':
				col = hair
			case 'S':
				col = skin
			case 'E':
				col = eye
			default:
				continue
			}
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					s.Set(x*2+dx, y*2+dy, col, 255)
				}
			}
		}
	}
	// 腮红（2x2 在脸颊）
	for _, p := range [][2]int{{4, 10}, {25, 10}} {
		s.Set(p[0], p[1], blush, 255)
		s.Set(p[0]+1, p[1], blush, 255)
		s.Set(p[0], p[1]+1, blush, 255)
		s.Set(p[0]+1, p[1]+1, blush, 255)
	}
	// 头发阴影 + 发饰点缀
	for i := 0; i < 8; i++ {
		x := noise(i, 30, 31, 28) + 2
		y := noise(i, 31, 32, 4) + 2
		s.Set(x, y, hairD, 255)
	}
	return s
}

// makeBattleBg 战斗背景（横向渐变 + 远山剪影）。
func makeBattleBg(w, h int) *engine.Sprite {
	s := engine.NewSprite(w, h)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		// 天空：上深蓝下浅
		r := uint8(30 + (90-30)*t)
		g := uint8(40 + (120-40)*t)
		b := uint8(70 + (170-70)*t)
		for x := 0; x < w; x++ {
			s.Set(x, y, engine.Color{R: r, G: g, B: b}, 255)
		}
	}
	// 远山剪影
	for y := 0; y < h/3; y++ {
		for x := 0; x < w; x++ {
			peak := int(float64(w)/2.0 + 120*sin6(float64(x)/float64(w)*3.14159*2))
			_ = peak
			// 简单双山
			m1 := 140 - int(60*absSin(float64(x)*0.03))
			m2 := 220 - int(50*absSin(float64(x)*0.02+1.7))
			if y > h-140+m1 || y > h-160+m2 {
				s.Set(x, y, engine.Color{R: 40, G: 48, B: 66}, 255)
			}
		}
	}
	// 地面
	for y := h - 90; y < h; y++ {
		for x := 0; x < w; x++ {
			s.Set(x, y, engine.Color{R: 58, G: 74, B: 60}, 255)
		}
	}
	return s
}

func absSin(v float64) float64 {
	v = v - float64(int(v/(3.14159*2)))*(3.14159*2)
	if v < 0 {
		v = -v
	}
	if v > 3.14159 {
		v = 2*3.14159 - v
	}
	return v
}

func sin6(v float64) float64 {
	v = v - float64(int(v/(3.14159*2)))*(3.14159*2)
	return v
}
