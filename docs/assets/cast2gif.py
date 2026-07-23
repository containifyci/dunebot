#!/usr/bin/env python3
"""Render an asciicast v2 file (.cast) to an animated GIF.

Implements a minimal VT100/ANSI terminal emulator sufficient to replay
asciinema recordings: cursor movement, line feed, carriage return, backspace,
erase display/line, scroll, SGR (colors), and OSC title stripping.
"""
import json
import sys
import os
from PIL import Image, ImageDraw, ImageFont

FONT_REGULAR = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
FONT_BOLD = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf"
FONT_ITALIC = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Oblique.ttf"

# ── palette ────────────────────────────────────────────────────────────────
DEFAULT_FG = (220, 223, 228)   # light grey
DEFAULT_BG = (30, 32, 38)      # dark slate
COL = {
    0: (90, 92, 96),       # black (dimmed so it's visible on dark bg)
    1: (224, 108, 117),    # red
    2: (152, 195, 121),    # green
    3: (209, 154, 102),    # yellow
    4: (97, 175, 239),     # blue
    5: (198, 120, 221),    # magenta
    6: (86, 182, 194),     # cyan
    7: (220, 223, 228),    # white
}
# bright variants
BRIGHT = {
    0: (130, 132, 136),
    1: (240, 128, 137),
    2: (172, 215, 141),
    3: (229, 174, 122),
    4: (117, 195, 255),
    5: (218, 140, 241),
    6: (106, 202, 214),
    7: (240, 243, 248),
}


class Terminal:
    def __init__(self, width, height):
        self.w = width
        self.h = height
        self.cx = 0
        self.cy = 0
        self.cells = [[Cell() for _ in range(width)] for _ in range(height)]
        self.fg = None
        self.bg = None
        self.bold = False
        self.italic = False
        self.cur_visible = True

    def _cur_style(self):
        return (self.fg, self.bg, self.bold, self.italic)

    def put(self, ch):
        if self.cx >= self.w:
            self.cx = 0
            self.linefeed()
        if self.cy >= self.h:
            self.cy = self.h - 1
        # skip combining chars / non-printable
        cell = self.cells[self.cy][self.cx]
        cell.ch = ch
        cell.fg = self.fg
        cell.bg = self.bg
        cell.bold = self.bold
        cell.italic = self.italic
        self.cx += 1

    def linefeed(self):
        self.cy += 1
        if self.cy >= self.h:
            self.cy = self.h - 1
            # scroll up one line
            self.cells.pop(0)
            self.cells.append([Cell() for _ in range(self.w)])

    def carriage_return(self):
        self.cx = 0

    def backspace(self):
        if self.cx > 0:
            self.cx -= 1

    def move(self, x, y):
        self.cx = max(0, min(self.w - 1, x))
        self.cy = max(0, min(self.h - 1, y))

    def erase_display(self, mode):
        if mode == 0:  # cursor to end
            for x in range(self.cx, self.w):
                self.cells[self.cy][x] = Cell()
            for y in range(self.cy + 1, self.h):
                for x in range(self.w):
                    self.cells[y][x] = Cell()
        elif mode == 2 or mode == 3:  # whole screen
            self.cells = [[Cell() for _ in range(self.w)] for _ in range(self.h)]
            # 3 also clears scrollback — we have none
        elif mode == 1:  # start to cursor
            for y in range(0, self.cy):
                for x in range(self.w):
                    self.cells[y][x] = Cell()
            for x in range(0, self.cx + 1):
                self.cells[self.cy][x] = Cell()

    def erase_line(self, mode):
        if mode == 0:
            for x in range(self.cx, self.w):
                self.cells[self.cy][x] = Cell()
        elif mode == 2:
            for x in range(self.w):
                self.cells[self.cy][x] = Cell()
        elif mode == 1:
            for x in range(0, self.cx + 1):
                self.cells[self.cy][x] = Cell()


class Cell:
    __slots__ = ("ch", "fg", "bg", "bold", "italic")

    def __init__(self):
        self.ch = " "
        self.fg = None
        self.bg = None
        self.bold = False
        self.italic = False


def parse_color(param, is_bg):
    """Parse SGR color params starting at index; return (fg/bg, index consumed)."""
    # simplified: handles 30-37, 40-47, 90-97, 100-107, 38;5;N, 48;5;N, 38;2;r;g;b, 48;2
    return None, 0


def apply_sgr(term, params):
    i = 0
    while i < len(params):
        p = params[i]
        if p == 0:
            term.fg = None; term.bg = None; term.bold = False; term.italic = False
        elif p == 1:
            term.bold = True
        elif p == 3:
            term.italic = True
        elif p == 22:
            term.bold = False
        elif p == 23:
            term.italic = False
        elif 30 <= p <= 37:
            term.fg = ("std", p - 30)
        elif 40 <= p <= 47:
            term.bg = ("std", p - 40)
        elif 90 <= p <= 97:
            term.fg = ("bright", p - 90)
        elif 100 <= p <= 107:
            term.bg = ("bright", p - 100)
        elif p == 39:
            term.fg = None
        elif p == 49:
            term.bg = None
        elif p == 38 or p == 48:
            # extended color
            if i + 1 < len(params):
                mode = params[i + 1]
                if mode == 5 and i + 2 < len(params):
                    val = params[i + 2]
                    rgb = xterm256_to_rgb(val)
                    if p == 38:
                        term.fg = ("rgb", rgb)
                    else:
                        term.bg = ("rgb", rgb)
                    i += 2
                elif mode == 2 and i + 4 < len(params):
                    rgb = (params[i + 2], params[i + 3], params[i + 4])
                    if p == 38:
                        term.fg = ("rgb", rgb)
                    else:
                        term.bg = ("rgb", rgb)
                    i += 4
        i += 1


def xterm256_to_rgb(n):
    if n < 8:
        return COL[n]
    if n < 16:
        return BRIGHT[n - 8]
    if n >= 232:
        v = 8 + (n - 232) * 10
        return (v, v, v)
    n -= 16
    r = n // 36
    g = (n % 36) // 6
    b = n % 6
    levels = [47, 115, 155, 195, 235, 255]
    return (levels[r], levels[g], levels[b])


def resolve(style_key, default):
    if style_key is None:
        return default
    kind, val = style_key
    if kind == "std":
        return COL[val]
    if kind == "bright":
        return BRIGHT[val]
    if kind == "rgb":
        return val
    return default


CSI = "\x1b["
ESC = "\x1b"


def feed(term, data):
    """Process a chunk of terminal output, mutating term state."""
    i = 0
    n = len(data)
    while i < n:
        ch = data[i]
        if ch == ESC:
            # parse escape sequence
            if i + 1 < n and data[i + 1] == "[":
                # CSI
                j = i + 2
                params = []
                cur = ""
                while j < n:
                    c = data[j]
                    if c.isdigit():
                        cur += c
                        j += 1
                        continue
                    if c == ";":
                        params.append(int(cur) if cur else 0)
                        cur = ""
                        j += 1
                        continue
                    # final byte
                    if cur:
                        params.append(int(cur))
                        cur = ""
                    if c == "m":
                        if not params:
                            params = [0]
                        apply_sgr(term, params)
                        i = j + 1
                        break
                    elif c == "H" or c == "f":
                        row = params[0] if len(params) >= 1 else 1
                        col = params[1] if len(params) >= 2 else 1
                        term.move(col - 1, row - 1)
                        i = j + 1
                        break
                    elif c == "J":
                        mode = params[0] if params else 0
                        term.erase_display(mode)
                        i = j + 1
                        break
                    elif c == "K":
                        mode = params[0] if params else 0
                        term.erase_line(mode)
                        i = j + 1
                        break
                    elif c == "A":
                        n_ = params[0] if params else 1
                        term.cy = max(0, term.cy - n_)
                        i = j + 1
                        break
                    elif c == "B":
                        n_ = params[0] if params else 1
                        term.cy = min(term.h - 1, term.cy + n_)
                        i = j + 1
                        break
                    elif c == "C":
                        n_ = params[0] if params else 1
                        term.cx = min(term.w - 1, term.cx + n_)
                        i = j + 1
                        break
                    elif c == "D":
                        n_ = params[0] if params else 1
                        term.cx = max(0, term.cx - n_)
                        i = j + 1
                        break
                    elif c == "G":
                        col = params[0] if params else 1
                        term.cx = max(0, min(term.w - 1, col - 1))
                        i = j + 1
                        break
                    elif c == "d":
                        row = params[0] if params else 1
                        term.cy = max(0, min(term.h - 1, row - 1))
                        i = j + 1
                        break
                    elif c == "?":
                        # private modes (cursor visibility etc) — skip to final
                        j += 1
                        while j < n and not (0x40 <= ord(data[j]) <= 0x7e):
                            j += 1
                        i = j + 1
                        break
                    else:
                        # unknown CSI, skip
                        i = j + 1
                        break
                else:
                    # ran off end
                    i = n
                continue
            elif i + 1 < n and data[i + 1] == "]":
                # OSC — skip until BEL or ST
                j = i + 2
                while j < n and data[j] != "\x07" and not (
                    data[j] == ESC and j + 1 < n and data[j + 1] == "\\"
                ):
                    j += 1
                i = j + 1
                continue
            elif i + 1 < n and data[i + 1] == "(":
                # charset designation — skip 1 more
                i += 3
                continue
            else:
                # lone ESC, skip
                i += 1
                continue
        elif ch == "\r":
            term.carriage_return()
            i += 1
        elif ch == "\n":
            term.linefeed()
            i += 1
        elif ch == "\b":
            term.backspace()
            i += 1
        elif ch == "\t":
            term.cx = ((term.cx // 8) + 1) * 8
            if term.cx >= term.w:
                term.cx = 0
                term.linefeed()
            i += 1
        elif ch == "\x07":
            # bell — ignore
            i += 1
        elif ord(ch) < 32:
            i += 1
        else:
            term.put(ch)
            i += 1


def render_frame(term, font, font_bold, font_italic, char_w, char_h, pad, scale):
    """Render the terminal state to a PIL Image."""
    W = term.w * char_w + pad * 2
    H = term.h * char_h + pad * 2
    img = Image.new("RGB", (W, H), DEFAULT_BG)
    draw = ImageDraw.Draw(img)
    for y in range(term.h):
        for x in range(term.w):
            cell = term.cells[y][x]
            fg = resolve(cell.fg, DEFAULT_FG)
            bg = resolve(cell.bg, DEFAULT_BG)
            px = pad + x * char_w
            py = pad + y * char_h
            if bg != DEFAULT_BG:
                draw.rectangle([px, py, px + char_w - 1, py + char_h - 1], fill=bg)
            if cell.ch != " ":
                f = font
                if cell.bold and cell.italic and font_bold is not None:
                    f = font_bold
                elif cell.bold and font_bold is not None:
                    f = font_bold
                elif cell.italic and font_italic is not None:
                    f = font_italic
                draw.text((px, py), cell.ch, fill=fg, font=f)
    # cursor block
    if term.cur_visible and 0 <= term.cy < term.h and 0 <= term.cx < term.w:
        cx = pad + term.cx * char_w
        cy = pad + term.cy * char_h
        # draw a caret as a small block at bottom
        draw.rectangle([cx, cy + char_h - 3, cx + char_w - 1, cy + char_h - 1],
                       fill=DEFAULT_FG)
    if scale != 1:
        img = img.resize((W * scale, H * scale), Image.NEAREST)
    return img


def main():
    cast_path = sys.argv[1]
    gif_path = sys.argv[2]
    font_size = int(os.environ.get("FONT_SIZE", "18"))
    scale = int(os.environ.get("SCALE", "1"))
    # min ms a frame must last (coalesces typing bursts into one frame)
    min_ms = int(os.environ.get("MIN_MS", "80"))
    # cap ms for any single frame (idle pauses)
    max_ms = int(os.environ.get("MAX_MS", "600"))

    font = ImageFont.truetype(FONT_REGULAR, font_size)
    font_bold = ImageFont.truetype(FONT_BOLD, font_size)
    font_italic = ImageFont.truetype(FONT_ITALIC, font_size)

    # measure char size
    bbox = font.getbbox("M")
    char_w = bbox[2] - bbox[0]
    char_h = int(font.getmetrics()[1] + 2)

    with open(cast_path) as f:
        header = json.loads(f.readline())
        w = header["width"]
        h = header["height"]
        events = []
        for line in f:
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass

    term = Terminal(w, h)
    frames = []
    durations = []

    idle_limit = header.get("idle_time_limit", 9999)

    # Strategy: render one frame per output event, but coalesce events whose
    # inter-event gap is < min_ms into a single frame (so rapid typing chars
    # don't each produce a frame). The coalesced frame's duration is the time
    # from the first event in the group to the next group's first event.
    out_events = [(t, data) for (t, ev, data) in events if ev == "o"]

    groups = []  # list of (start_time, end_time)
    i = 0
    while i < len(out_events):
        t0 = out_events[i][0]
        j = i
        while j + 1 < len(out_events) and (out_events[j + 1][0] - out_events[j][0]) * 1000 < min_ms:
            j += 1
        groups.append((i, j, t0))
        i = j + 1

    last_rendered = -1
    for gi, (i0, i1, t0) in enumerate(groups):
        # apply all events in this group
        for k in range(i0, i1 + 1):
            feed(term, out_events[k][1])
        # duration: time until next group's first event
        if gi + 1 < len(groups):
            dur = (groups[gi + 1][2] - t0) * 1000
        else:
            dur = 400
        dur = max(min_ms, min(max_ms, int(dur)))
        frames.append(render_frame(term, font, font_bold, font_italic,
                                   char_w, char_h, 8, scale))
        durations.append(dur)

    if not frames:
        print("no frames generated", file=sys.stderr)
        sys.exit(1)

    print(f"rendered {len(frames)} frames, {sum(durations)/1000:.1f}s total",
          file=sys.stderr)
    frames[0].save(
        gif_path,
        save_all=True,
        append_images=frames[1:],
        duration=durations,
        loop=0,
        disposal=2,
        optimize=True,
    )
    print(f"saved {gif_path} ({os.path.getsize(gif_path)//1024} KB)", file=sys.stderr)


if __name__ == "__main__":
    main()