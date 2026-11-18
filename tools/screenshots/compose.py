"""Compose WiFerry screenshots into the README demo GIF, stills and the
GitHub social preview. Usage: python3 compose.py <shots_dir> <out_dir> <icon.png>"""
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter, ImageFont

SHOTS, OUT, ICON = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
OUT.mkdir(parents=True, exist_ok=True)
FONT_B = "/usr/share/fonts/opentype/inter/Inter-Bold.otf"
FONT_M = "/usr/share/fonts/opentype/inter/InterDisplay-Medium.otf"
TEAL, TEXT, MUTED = (13, 127, 138), (15, 20, 25), (90, 100, 120)


def font(path, size):
    return ImageFont.truetype(path, size)


def gradient(w, h, top=(236, 246, 247), bottom=(245, 246, 248)):
    img = Image.new("RGB", (w, h), top)
    d = ImageDraw.Draw(img)
    for y in range(h):
        t = y / h
        d.line([(0, y), (w, y)], fill=tuple(int(top[i] + (bottom[i] - top[i]) * t) for i in range(3)))
    return img


def rounded(img, r):
    mask = Image.new("L", img.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, img.width - 1, img.height - 1], r, fill=255)
    out = Image.new("RGBA", img.size)
    out.paste(img, (0, 0), mask)
    return out


def shadow(canvas, box, r, blur=24, offset=10, alpha=60):
    x0, y0, x1, y1 = box
    sh = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    ImageDraw.Draw(sh).rounded_rectangle([x0, y0 + offset, x1, y1 + offset], r, fill=(16, 24, 40, alpha))
    canvas.alpha_composite(sh.filter(ImageFilter.GaussianBlur(blur)))


def browser(host_png, width):
    """PC screenshot inside a minimal browser window."""
    shot = Image.open(host_png).convert("RGB")
    shot = shot.crop((0, 0, shot.width - 34, shot.height))  # drop scrollbar
    h = round(shot.height * width / shot.width)
    shot = shot.resize((width, h), Image.LANCZOS)
    bar = 34
    win = Image.new("RGB", (width, h + bar), (255, 255, 255))
    d = ImageDraw.Draw(win)
    d.rectangle([0, 0, width, bar], fill=(232, 235, 239))
    for i, c in enumerate([(255, 95, 87), (254, 188, 46), (40, 200, 64)]):
        d.ellipse([14 + i * 20, 11, 26 + i * 20, 23], fill=c)
    d.rounded_rectangle([90, 7, width - 90, bar - 7], 9, fill=(255, 255, 255))
    d.text((width // 2, bar // 2), "127.0.0.1:8642/host", font=font(FONT_M, 13), fill=MUTED, anchor="mm")
    win.paste(shot, (0, bar))
    return rounded(win, 12)


def phone(phone_png, height):
    """Phone screenshot inside a simple device frame."""
    raw = Image.open(phone_png).convert("RGB")
    bezel = 11
    inner_h = height - 2 * bezel
    # Status bar strip above the page, like Safari on a real iPhone.
    bar_h = round(inner_h * 0.065)
    inner_w = round(raw.width * inner_h / raw.height)
    page = raw.resize((inner_w, round(raw.height * inner_w / raw.width)), Image.LANCZOS)
    bg = raw.getpixel((4, 4))
    fg = (255, 255, 255) if sum(bg) < 300 else TEXT
    shot = Image.new("RGB", (inner_w, inner_h), bg)
    shot.paste(page.crop((0, 0, inner_w, inner_h - bar_h)), (0, bar_h))
    sd = ImageDraw.Draw(shot)
    fs = max(11, round(bar_h * 0.42))
    sd.text((round(inner_w * 0.13), bar_h // 2 + 2), "9:41", font=font(FONT_B, fs), fill=fg, anchor="mm")
    bx = inner_w - round(inner_w * 0.13)
    sd.rounded_rectangle([bx - fs, bar_h // 2 - fs // 3 + 2, bx + fs // 3, bar_h // 2 + fs // 3 + 2], 3, outline=fg, width=2)
    sd.rectangle([bx - fs + 3, bar_h // 2 - fs // 3 + 5, bx + fs // 3 - 4, bar_h // 2 + fs // 3 - 1], fill=fg)
    shot = rounded(shot, 34)
    dev = Image.new("RGBA", (inner_w + 2 * bezel, height), (0, 0, 0, 0))
    ImageDraw.Draw(dev).rounded_rectangle([0, 0, dev.width - 1, height - 1], 44, fill=(22, 24, 28, 255))
    dev.alpha_composite(shot, (bezel, bezel))
    d = ImageDraw.Draw(dev)
    isl = round(dev.width * 0.13); d.rounded_rectangle([dev.width // 2 - isl, bezel + 8, dev.width // 2 + isl, bezel + 8 + round(inner_h * 0.045)], 14, fill=(22, 24, 28, 255))
    return dev


def scene(step, host_png, phone_png, title, sub, size=(1600, 900)):
    W, H = size
    canvas = gradient(W, H).convert("RGBA")
    d = ImageDraw.Draw(canvas)
    # Step badge + caption.
    d.ellipse([60, 44, 108, 92], fill=TEAL)
    d.text((84, 68), str(step), font=font(FONT_B, 26), fill="white", anchor="mm")
    d.text((128, 50), title, font=font(FONT_B, 38), fill=TEXT)
    d.text((130, 98), sub, font=font(FONT_M, 22), fill=MUTED)
    win = browser(host_png, 1040)
    shadow(canvas, (60, 160, 60 + win.width, 160 + win.height), 12)
    canvas.alpha_composite(win, (60, 160))
    dev = phone(phone_png, 740)
    x = W - dev.width - 70
    shadow(canvas, (x, 140, x + dev.width, 140 + dev.height), 44, alpha=80)
    canvas.alpha_composite(dev, (x, 140))
    return canvas.convert("RGB")


steps = [
    ("Scan the QR code with the iPhone Camera", "Same Wi-Fi or hotspot. No app to install, no internet needed."),
    ("Send photos and videos from the iPhone", "Pick from Photos or Files. Big videos stream straight to disk."),
    ("They land on your PC", "Local Wi-Fi only. Nothing goes through the cloud."),
    ("Save it on the PC and it's gone from both sides", "Files are handed off, not left lying around."),
    ("Drop files on the PC to send them to the phone", "They appear on the iPhone within a couple of seconds."),
    ("Tap to download on the iPhone. Done.", "Saved to Files on the phone. Gone from the PC's list."),
]
frames = []
for i, (title, sub) in enumerate(steps, 1):
    f = scene(i, SHOTS / f"0{i}-host.png", SHOTS / f"0{i}-phone.png", title, sub)
    frames.append(f)
    if i == 3:
        f.save(OUT / "hero.png", optimize=True)

# Demo GIF (scaled down to keep it light).
small = [f.resize((1200, 675), Image.LANCZOS).quantize(colors=128, method=Image.MEDIANCUT, dither=Image.NONE) for f in frames]
small[0].save(OUT / "demo.gif", save_all=True, append_images=small[1:], duration=[2600] * 5 + [3600], loop=0, optimize=True)

# Stills.
def still(src, dst, width):
    im = Image.open(src).convert("RGB")
    if "host" in src.name:
        im = im.crop((0, 0, im.width - 34, im.height))
    im.resize((width, round(im.height * width / im.width)), Image.LANCZOS).save(dst, optimize=True)

still(SHOTS / "03-host.png", OUT / "desktop.png", 1280)
still(SHOTS / "dark-host.png", OUT / "desktop-dark.png", 1280)
for name, src in [("phone.png", "05-phone.png"), ("phone-dark.png", "dark-phone.png")]:
    phone(SHOTS / src, 900).save(OUT / name, optimize=True)

# Social preview (GitHub recommends 1280x640).
W, H = 1280, 640
card = gradient(W, H, (230, 245, 246), (244, 246, 248)).convert("RGBA")
d = ImageDraw.Draw(card)
icon = Image.open(ICON).convert("RGBA").resize((112, 112), Image.LANCZOS)
card.alpha_composite(icon, (72, 96))
d.text((72, 236), "WiFerry", font=font(FONT_B, 84), fill=TEXT)
lines = ["Send photos and files between", "your iPhone and Windows or Linux."]
for i, line in enumerate(lines):
    d.text((76, 350 + i * 46), line, font=font(FONT_M, 36), fill=TEXT)
d.text((76, 466), "No app on the phone  ·  No internet  ·  One file on the PC", font=font(FONT_M, 24), fill=TEAL)
d.text((76, 540), "github.com/pramesh3/wiferry", font=font(FONT_M, 22), fill=MUTED)
dev = phone(SHOTS / "03-phone.png", 560)
shadow(card, (W - dev.width - 90, 40, W - 90, 40 + dev.height), 44, alpha=80)
card.alpha_composite(dev, (W - dev.width - 90, 40))
card.convert("RGB").save(OUT / "social-preview.png", optimize=True)

for p in sorted(OUT.iterdir()):
    print(f"{p.name:22} {p.stat().st_size // 1024:5} KB")
