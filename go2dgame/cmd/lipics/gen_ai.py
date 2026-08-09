#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""重新生成 7 张半身像立绘：面部更精致（Pollinations.ai 串行+重试）"""
import io, os, time, urllib.parse, urllib.request
from PIL import Image

OUT = "/data/data/com.termux/files/home/storage/shared/zoombase/result"

FACE = "detailed beautiful face, large sparkling eyes, sharp facial features, delicate skin, "

def gen(name, prompt, seed):
    url = ("https://image.pollinations.ai/prompt/" +
           urllib.parse.quote(prompt) +
           f"?width=1024&height=1536&model=flux&nologo=true&seed={seed}")
    for attempt in range(4):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req, timeout=240) as r:
                data = r.read()
            im = Image.open(io.BytesIO(data))
            im.load()
            # 轻度锐化面部：直接保存原始
            path = os.path.join(OUT, name + ".png")
            with open(path, "wb") as f:
                f.write(data)
            return f"{name}: OK {im.size} {len(data)}B"
        except Exception as e:
            if attempt < 3:
                print(f"  {name} 第{attempt+1}次失败({e})，等15秒重试...")
                time.sleep(15)
            else:
                return f"{name}: FAIL {e}"
    return f"{name}: FAIL"

jobs = [
    ("01_hero_taoyao", "anime style 2D game character portrait, chest-up bust shot, cute young girl heroine, petite flat chest, long black hair with pink ribbon bow, blue and white wuxia martial arts outfit, gentle shy smile, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2001),
    ("02_qing", "anime style 2D game character portrait, chest-up bust shot, beautiful young woman with large breasts, long brown hair, green hanfu dress with white sash, gentle warm smile, herbal medicine shop girl, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2002),
    ("03_luoyao", "anime style 2D game character portrait, chest-up bust shot, beautiful young woman with large breasts, black hair in high ponytail with gold hairpin, red chinese constable uniform, confident smile, swordswoman, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2003),
    ("04_yueli", "anime style 2D game character portrait, chest-up bust shot, elegant beautiful woman with large breasts, long silver white hair, white flowing dress, cold ethereal beauty, snow immortal, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2004),
    ("05_xiaoman", "anime style 2D game character portrait, chest-up bust shot, cute lively girl with large breasts, orange dress, brown hair in twin tails, cheerful energetic expression, bandit princess, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2005),
    ("06_suxue", "anime style 2D game character portrait, chest-up bust shot, beautiful cold mysterious woman with large breasts, long purple hair, purple dress, swordswoman guardian of ancient tomb, calm expression, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2006),
    ("07_zhuzhai_feng", "anime style 2D game character portrait, chest-up bust shot, mature seductive woman with huge breasts, long red hair, dark purple chinese dress, female bandit leader, confident smirk, " + FACE + "clean soft background, game CG illustration, high quality, masterpiece", 2007),
]

for i, (n, p, s) in enumerate(jobs):
    print(f"[{i+1}/7] 生成 {n} ...")
    print("  " + gen(n, p, s))
    if i < len(jobs) - 1:
        time.sleep(8)
print("DONE")
