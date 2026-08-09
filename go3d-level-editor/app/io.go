package app

import (
	"encoding/json"
	"os"

	"go3d/math3d"
	"go3deditor/app"
)

// 关卡序列化格式：资源引用 + 实例变换 + 素材路径。
type levelJSON struct {
	Name    string       `json:"name"`
	Models  []modelJSON  `json:"models"`
	Insts   []instJSON   `json:"insts"`
	Sprites []spriteJSON `json:"sprites,omitempty"`
}

type modelJSON struct {
	Name string `json:"name"`
	Path string `json:"path"` // 建模编辑器 JSON 文件（含该模型）
	Idx  int    `json:"idx"`  // 该文件中对象索引
}

type instJSON struct {
	Name       string     `json:"name"`
	Model      string     `json:"model"` // 资源名
	Pos        [3]float32 `json:"pos"`
	Rot        [3]float32 `json:"rot"`
	Scale      float32    `json:"scale"`
	Visible    bool       `json:"visible"`
	IsPlayer   bool       `json:"is_player,omitempty"`
	IsSprite   bool       `json:"is_sprite,omitempty"`
	SpriteName string     `json:"sprite,omitempty"`
}

type spriteJSON struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Save 保存关卡 JSON。
func (e *Editor) Save(path string) error {
	lj := levelJSON{Name: e.level.Name}
	for _, m := range e.level.Models {
		lj.Models = append(lj.Models, modelJSON{Name: m.Name, Path: e.resPaths[m.Name], Idx: e.resIdxs[m.Name]})
	}
	for _, s := range e.level.Sprites {
		lj.Sprites = append(lj.Sprites, spriteJSON{Name: s.Name, Path: e.spritePaths[s.Name]})
	}
	for _, i := range e.level.Insts {
		ij := instJSON{
			Name:     i.Name,
			Pos:      [3]float32{i.Pos.X, i.Pos.Y, i.Pos.Z},
			Rot:      [3]float32{i.RotX, i.RotY, i.RotZ},
			Scale:    i.Scale,
			Visible:  i.Visible,
			IsPlayer: i.IsPlayer,
		}
		if i.Res != nil {
			ij.Model = i.Res.Name
		}
		if i.IsSprite {
			ij.IsSprite = true
			ij.SpriteName = i.GetSpriteName()
		}
		lj.Insts = append(lj.Insts, ij)
	}
	data, err := json.MarshalIndent(lj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load 加载关卡 JSON。
func (e *Editor) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var lj levelJSON
	if err := json.Unmarshal(data, &lj); err != nil {
		return err
	}
	lv := NewLevel(lj.Name)
	e.level = lv
	e.resPaths = map[string]string{}
	e.resIdxs = map[string]int{}
	e.spritePaths = map[string]string{}
	// 模型资源
	for _, mj := range lj.Models {
		doc, err := app.LoadDocumentFile(mj.Path)
		if err != nil || mj.Idx < 0 || mj.Idx >= len(doc.Objs) {
			continue
		}
		o := doc.Objs[mj.Idx]
		res := ModelResFromObject(o)
		lv.AddModel(res)
		e.resPaths[res.Name] = mj.Path
		e.resIdxs[res.Name] = mj.Idx
	}
	// 素材
	for _, sj := range lj.Sprites {
		sp, err := LoadSprite(sj.Path)
		if err != nil {
			continue
		}
		lv.AddSprite(sp)
		e.spritePaths[sp.Name] = sj.Path
	}
	// 实例
	for _, ij := range lj.Insts {
		var inst *Instance
		if ij.IsSprite {
			inst = &Instance{
				Name:      ij.Name,
				Pos:       math3d.Vec3{ij.Pos[0], ij.Pos[1], ij.Pos[2]},
				RotX:      ij.Rot[0],
				RotY:      ij.Rot[1],
				RotZ:      ij.Rot[2],
				Scale:     ij.Scale,
				Visible:   ij.Visible,
				IsPlayer:  ij.IsPlayer,
				IsSprite:  true,
				SpriteName: ij.SpriteName,
			}
			if inst.Scale == 0 {
				inst.Scale = 1
			}
			if sp := lv.FindSprite(ij.SpriteName); sp != nil {
				inst.Sprite = sp
			}
		} else {
			res := lv.FindModel(ij.Model)
			if res == nil {
				continue
			}
			inst = NewInstance(ij.Name, res)
			inst.Pos = math3d.Vec3{ij.Pos[0], ij.Pos[1], ij.Pos[2]}
			inst.RotX, inst.RotY, inst.RotZ = ij.Rot[0], ij.Rot[1], ij.Rot[2]
			inst.Scale = ij.Scale
			if inst.Scale == 0 {
				inst.Scale = 1
			}
			inst.Visible = ij.Visible
			inst.IsPlayer = ij.IsPlayer
		}
		lv.AddInstance(inst)
		if inst.IsPlayer {
			e.player = inst
		}
	}
	return nil
}
