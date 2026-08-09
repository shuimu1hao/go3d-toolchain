package app

import (
	"bytes"
	"go3d/math3d"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// 测试用建模场景：立方体 + 带骨骼动画的球。
func makeModelJSON(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "model.json")
	// 直接构造：两个对象（立方体、带骨骼动画的拉伸体）
	content := `{
  "name": "测试模型",
  "objs": [
    {
      "name": "平台",
      "type": 0,
      "pos": [0, 0, 0],
      "rot": [0, 0, 0],
      "scale": 1,
      "color": [120, 120, 160],
      "visible": true,
      "custom_verts": [[-2,0,-2],[2,0,-2],[2,0,2],[-2,0,2],[-2,0.4,-2],[2,0.4,-2],[2,0.4,2],[-2,0.4,2]],
      "custom_faces": [[0,1,2],[0,2,3],[4,5,6],[4,6,7],[0,1,5],[0,5,4],[1,2,6],[1,6,5],[2,3,7],[2,7,6],[3,0,4],[3,4,7]]
    },
    {
      "name": "角色",
      "type": 1,
      "pos": [0, 0.5, 0],
      "rot": [0, 0, 0],
      "scale": 0.5,
      "color": [200, 120, 80],
      "visible": true,
      "bones": [
        {"name": "root", "parent": -1, "pos": [0, 0, 0], "rot": [0, 0, 0]},
        {"name": "头", "parent": 0, "pos": [0, 1, 0], "rot": [0, 0, 0]}
      ],
      "anim": {
        "name": "摇头",
        "duration": 1.0,
        "loop": true,
        "tracks": [
          {"bone": 1, "keys": [
            {"t": 0, "pos": [0, 1, 0], "rot": [0, 0, 0]},
            {"t": 0.5, "pos": [0, 1, 0], "rot": [0, 0, 0.8]},
            {"t": 1, "pos": [0, 1, 0], "rot": [0, 0, 0]}
          ]}
        ]
      }
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportModelDoc(t *testing.T) {
	e := New(1100, 680)
	path := makeModelJSON(t, t.TempDir())
	if err := e.ImportModelDoc(path); err != nil {
		t.Fatal("import:", err)
	}
	if len(e.level.Models) != 2 {
		t.Fatalf("should import 2 models, got %d", len(e.level.Models))
	}
	// 角色有动画
	role := e.level.FindModel("角色")
	if role == nil || role.Anim == nil {
		t.Fatal("role model should have anim")
	}
	if role.Skeleton == nil || len(role.Skeleton.Bones) != 2 {
		t.Fatal("role should have 2 bones")
	}
}

func TestInstanceAnimIndependent(t *testing.T) {
	e := New(1100, 680)
	path := makeModelJSON(t, t.TempDir())
	_ = e.ImportModelDoc(path)
	// 两个角色实例，动画独立播放
	i1 := NewInstance("r1", e.level.FindModel("角色"))
	i2 := NewInstance("r2", e.level.FindModel("角色"))
	if i1.Skel == nil || i2.Skel == nil {
		t.Fatal("instances should have independent skeletons")
	}
	// 指针不同（深拷贝）
	if i1.Skel == i2.Skel {
		t.Fatal("skeletons should be deep copies")
	}
	i1.AnimPlaying = true
	i2.AnimPlaying = true
	i1.UpdateAnim(0.1)
	i2.UpdateAnim(0.6)
	// 头骨骼旋转应不同（0→0.8→0 曲线，0.1 处 0.16，0.6 处 0.64）
	if i1.Skel.Bones[1].RotZ == i2.Skel.Bones[1].RotZ {
		t.Fatalf("anims should diverge: %v vs %v", i1.Skel.Bones[1].RotZ, i2.Skel.Bones[1].RotZ)
	}
}

func TestLevelSaveLoad(t *testing.T) {
	e := New(1100, 680)
	path := makeModelJSON(t, t.TempDir())
	_ = e.ImportModelDoc(path)
	// 添加实例 + 玩家 + 精灵
	e.AddInstance(0) // 平台
	e.AddInstance(1) // 角色
	e.sel = 1
	e.SetPlayer()
	e.AddInstance(1)
	// 保存
	levelPath := filepath.Join(t.TempDir(), "level.json")
	if err := e.Save(levelPath); err != nil {
		t.Fatal("save:", err)
	}
	// 载入
	e2 := New(1100, 680)
	if err := e2.Load(levelPath); err != nil {
		t.Fatal("load:", err)
	}
	if len(e2.level.Models) != 2 {
		t.Fatalf("models not restored: %d", len(e2.level.Models))
	}
	if len(e2.level.Insts) != 3 {
		t.Fatalf("insts not restored: %d", len(e2.level.Insts))
	}
	if e2.player == nil {
		t.Fatal("player not restored")
	}
	// 实例骨骼独立
	if e2.level.Insts[1].Skel == nil || e2.level.Insts[1].Skel == e2.level.Insts[2].Skel {
		t.Fatal("inst skeletons should be independent")
	}
}

func TestInputMapDefaults(t *testing.T) {
	m := NewInputMap()
	if m.Keys[ActForward] != "w" || m.Keys[ActJump] != " " {
		t.Fatal("default mapping wrong")
	}
	if ActionName(ActRight) != "右移" {
		t.Fatal("action name wrong")
	}
}

func TestSpriteLoadDraw(t *testing.T) {
	// 生成 4x4 PNG
	img := newTestPNG(4, 4)
	path := filepath.Join(t.TempDir(), "s.png")
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := LoadSprite(path)
	if err != nil {
		t.Fatal("load:", err)
	}
	if sp.W != 4 || sp.H != 4 || len(sp.RGBA) != 4*4*4 {
		t.Fatalf("sprite size wrong: %dx%d len %d", sp.W, sp.H, len(sp.RGBA))
	}
}

// newTestPNG 生成单色 PNG（image/png 编码）。
func newTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 200
		img.Pix[i+1] = 100
		img.Pix[i+2] = 50
		img.Pix[i+3] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestSpriteSaveLoad 精灵实例保存/载入。
func TestSpriteSaveLoad(t *testing.T) {
	e := New(1100, 680)
	// 真实 PNG 文件
	spPath := filepath.Join(t.TempDir(), "box.png")
	if err := os.WriteFile(spPath, newTestPNG(8, 8), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := LoadSprite(spPath)
	if err != nil {
		t.Fatal("load sprite:", err)
	}
	e.level.AddSprite(sp)
	e.spritePaths["box.png"] = spPath
	inst := &Instance{Name: "木箱_精灵", Pos: math3d.Vec3{1, 2, 3}, Scale: 1, Visible: true, IsSprite: true, Sprite: sp, SpriteName: "box.png"}
	e.level.AddInstance(inst)

	path := filepath.Join(t.TempDir(), "lvl.json")
	if err := e.Save(path); err != nil {
		t.Fatal("save:", err)
	}
	e2 := New(1100, 680)
	e2.spritePaths = map[string]string{}
	if err := e2.Load(path); err != nil {
		t.Fatal("load:", err)
	}
	if len(e2.level.Insts) != 1 {
		t.Fatalf("inst not restored: %d", len(e2.level.Insts))
	}
	s := e2.level.Insts[0]
	if !s.IsSprite || s.Sprite == nil {
		t.Fatal("sprite instance not restored")
	}
	if s.Pos.X != 1 || s.Pos.Y != 2 || s.Pos.Z != 3 {
		t.Fatal("sprite pos not restored")
	}
}
