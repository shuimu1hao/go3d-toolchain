// battle_test.go - 战斗系统无头验证（武学/功法/物魔抗/buff）
package main

import (
	"math/rand"
	"testing"
)

// 构造无引擎 Game
func testGame() *Game {
	g := &Game{player: newPlayer()}
	return g
}

func TestPhysicalVsMagic(t *testing.T) {
	g := testGame()

	// 山贼：物抗高(4) 魔抗低(1) → 功法应比武学更痛
	shanzei := findMonster("山贼")
	b1 := &Battle{mon: shanzei}
	total1 := 0
	for i := 0; i < 200; i++ {
		total1 += b1.calcPhys(g, 13, 6) // 烈火掌基准
	}
	avgPhys := total1 / 200

	b2 := &Battle{mon: shanzei}
	total2 := 0
	for i := 0; i < 200; i++ {
		total2 += b2.calcMagic(g, 13, 6)
	}
	avgMag := total2 / 200

	t.Logf("山贼 物理均伤=%d 魔法均伤=%d", avgPhys, avgMag)
	if avgMag <= avgPhys {
		t.Errorf("山贼魔抗应低于物抗：魔法应更痛 avgPhys=%d avgMag=%d", avgPhys, avgMag)
	}

	// 血月教徒：物抗低(2) 魔抗高(8) → 武学应比功法更痛
	jiaotu := findMonster("血月教徒")
	b3 := &Battle{mon: jiaotu}
	total3 := 0
	for i := 0; i < 200; i++ {
		total3 += b3.calcPhys(g, 13, 6)
	}
	avgPhys2 := total3 / 200

	b4 := &Battle{mon: jiaotu}
	total4 := 0
	for i := 0; i < 200; i++ {
		total4 += b4.calcMagic(g, 13, 6)
	}
	avgMag2 := total4 / 200

	t.Logf("血月教徒 物理均伤=%d 魔法均伤=%d", avgPhys2, avgMag2)
	if avgPhys2 <= avgMag2 {
		t.Errorf("血月教徒魔抗应高于物抗：武学应更痛 avgPhys=%d avgMag=%d", avgPhys2, avgMag2)
	}
}

func TestBuffDefendAndBurn(t *testing.T) {
	g := testGame()
	mon := findMonster("野狼")
	b := &Battle{mon: mon, monHP: mon.HP, pHP: g.player.HP}

	// 防御 → def_up buff 3 回合
	b.playerDefend(g)
	if !b.hasPlayerBuff("def_up") {
		t.Fatal("防御后应有 def_up buff")
	}
	// def_up 减伤 40%
	before := b.pHP
	b.enemyAttack(g)
	after := b.pHP
	t.Logf("def_up 下受击损失=%d", before-after)
	if before-after >= mon.Atk {
		t.Errorf("def_up 应显著减伤：损失 %d 应小于 %d", before-after, mon.Atk)
	}

	// 灼烧：tickBuffs 敌人掉血
	b2 := &Battle{mon: mon, monHP: mon.HP, pHP: g.player.HP, playerBuffs: nil}
	b2.addMonBuff("burn", "灼烧", 10, 3)
	hp0 := b2.monHP
	b2.tickBuffs(g)
	if b2.monHP != hp0-10 {
		t.Errorf("灼烧应每回合扣 10 血：%d -> %d", hp0, b2.monHP)
	}
	if b2.monBuffs[0].Turns != 2 {
		t.Errorf("灼烧回合应减为 2，实际 %d", b2.monBuffs[0].Turns)
	}

	// 再生：tickBuffs 玩家回血
	b3 := &Battle{mon: mon, monHP: mon.HP, pHP: 50, playerBuffs: nil, monBuffs: nil}
	b3.addPlayerBuff("regen", "回春", 6, 3)
	hp1 := b3.pHP
	b3.tickBuffs(g)
	if b3.pHP != hp1+6 {
		t.Errorf("再生应每回合回 6 血：%d -> %d", hp1, b3.pHP)
	}
}

func TestSkillTypeRouting(t *testing.T) {
	g := testGame()
	mon := findMonster("山贼")
	b := &Battle{mon: mon, monHP: mon.HP, pHP: 100, pMP: 100, playerBuffs: nil, monBuffs: nil}

	// 武学基础剑法（物理）打山贼
	hp0 := b.monHP
	m := findMartial("基础剑法")
	if m == nil || m.Type != "wu" {
		t.Fatal("基础剑法应为 wu 类型")
	}
	_ = m
	// 直接走 useSkill 路径
	b.useSkill(g, "基础剑法")
	if b.monHP >= hp0 {
		t.Error("武学应造成伤害")
	}
	t.Logf("基础剑法打山贼 掉血=%d", hp0-b.monHP)

	// 功法烈火掌打山贼（魔抗低，应更痛）且可能附灼烧
	b2 := &Battle{mon: findMonster("山贼"), monHP: findMonster("山贼").HP, pHP: 100, pMP: 100}
	hp1 := b2.monHP
	b2.useSkill(g, "烈火掌")
	loss := hp1 - b2.monHP
	t.Logf("烈火掌打山贼 掉血=%d 有灼烧=%v", loss, b2.hasMonBuff("burn"))
	if loss <= 0 {
		t.Error("功法应造成伤害")
	}
}

func TestRandomSeedStable(t *testing.T) {
	// 随机源可用性
	rand.Seed(1)
	if rand.Intn(10) < 0 {
		t.Fatal("rand broken")
	}
}
