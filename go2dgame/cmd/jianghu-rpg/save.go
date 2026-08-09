// save.go - JSON 存档
package main

import (
	"encoding/json"
	"os"
)

// SaveData 存档数据（JSON 序列化）。
type SaveData struct {
	Player *Player
	Chests map[string]bool
}

func (g *Game) saveGame() bool {
	// 同步全局 girls 状态到 player
	for _, gr := range girls {
		g.player.GirlLove[gr.ID] = gr.Love
		g.player.GirlEvents[gr.ID] = gr.Event
		if gr.JoinBattle {
			g.player.JoinedGirls[gr.ID] = true
		}
	}
	sd := SaveData{Player: g.player, Chests: g.chests}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return false
	}
	if err := os.WriteFile(g.savePath, data, 0644); err != nil {
		return false
	}
	return true
}

func (g *Game) loadSave() bool {
	data, err := os.ReadFile(g.savePath)
	if err != nil {
		return false
	}
	var sd SaveData
	if err := json.Unmarshal(data, &sd); err != nil {
		return false
	}
	if sd.Player == nil {
		return false
	}
	g.player = sd.Player
	// 旧存档兼容：补齐法攻/法防
	if g.player.MAtk == 0 {
		g.player.MAtk = 10
	}
	if g.player.MDef == 0 {
		g.player.MDef = 3
	}
	if sd.Chests != nil {
		g.chests = sd.Chests
	}
	// 恢复全局 girls 状态
	initGirls()
	for _, gr := range girls {
		gr.Love = g.player.GirlLove[gr.ID]
		gr.Event = g.player.GirlEvents[gr.ID]
		gr.JoinBattle = g.player.JoinedGirls[gr.ID]
	}
	g.curMap = findMap(g.player.MapID)
	if g.curMap == nil {
		g.curMap = findMap("qingzhou")
	}
	g.px, g.py = g.player.PX, g.player.PY
	return true
}
