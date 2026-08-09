#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""根据 zoombase/result 下的角色立绘 PNG 生成画廊 HTML"""
import base64, os

OUT = "/data/data/com.termux/files/home/storage/shared/zoombase/result"
files = ["01_hero_taoyao.png", "02_qing.png", "03_luoyao.png", "04_yueli.png",
         "05_xiaoman.png", "06_suxue.png", "07_zhuzhai_feng.png"]
names = ["桃夭（主角·贫乳）", "青儿", "洛瑶", "月璃", "小蛮", "素雪", "黑风女寨主"]
tips = ["黑长直粉蝴蝶结·蓝色劲装", "翠衫药铺千金·温柔", "红衣女捕快·高马尾",
        "白衣银发·天山仙影", "橙衣双马尾·活泼", "紫衣古墓守墓人", "红发紫衣·御姐寨主"]

cards = ""
for n, d, tip in zip(names, files, tips):
    p = os.path.join(OUT, n if os.path.exists(os.path.join(OUT, n)) else d)
    path = os.path.join(OUT, d)
    b64 = base64.b64encode(open(path, 'rb').read()).decode()
    cards += (f'<div class="card"><img src="data:image/png;base64,{b64}">'
              f'<div class="nm">{n}</div><div class="tp">{tip}</div></div>\n')

html = f'''<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>江湖行·红颜劫 - 角色立绘</title>
<style>
body {{ margin:0; padding:24px; background:#12101e; color:#eee; font-family:sans-serif; }}
h1 {{ text-align:center; color:#ffd878; margin-bottom:8px; }}
p.sub {{ text-align:center; color:#999; margin-bottom:24px; }}
.grid {{ display:flex; flex-wrap:wrap; justify-content:center; gap:20px; }}
.card {{ background:#1e1a30; border-radius:12px; padding:12px; text-align:center; box-shadow:0 4px 16px rgba(0,0,0,.4); }}
.card img {{ height:480px; border-radius:8px; display:block; }}
.nm {{ margin-top:10px; color:#ffd878; font-weight:bold; font-size:18px; }}
.tp {{ margin-top:4px; color:#999; font-size:13px; }}
</style></head><body>
<h1>🌸 江湖行·红颜劫 🌸</h1>
<p class="sub">角色立绘 · AI生成 · 百合向 · 主角贫乳 女主大胸</p>
<div class="grid">
{cards}
</div>
</body></html>'''

with open(os.path.join(OUT, "立绘画廊.html"), "w", encoding="utf-8") as f:
    f.write(html)
print("gallery updated:", os.path.getsize(os.path.join(OUT, "立绘画廊.html")), "bytes")
