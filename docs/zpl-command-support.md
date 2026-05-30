# ZPL Command Support

This document lists all known ZPL (Zebra Programming Language) commands and their support status in Zebrash.

Zebrash is a **rendering library** — its goal is to reproduce what a real Zebra printer would print. Hardware-only commands (RFID, network configuration, media calibration, serial port settings) are intentionally out of scope and are not listed here.

---

## Supported Commands

These 40+ commands are fully parsed and rendered by Zebrash.

### Label Format

| Command | Description |
|---------|-------------|
| `^XA` | Start label format |
| `^XZ` | End label format |
| `^CC` / `~CT` | Change caret / tilde prefix characters (handled inline) |
| `^LH` | Label home — shift the X,Y origin |
| `^LR` | Label reverse print — invert entire label |
| `^PO` | Print orientation — rotate label 180° |
| `^PW` | Print width |

### Fonts & Text

| Command | Description |
|---------|-------------|
| `^A` | Select scalable/bitmapped font |
| `^CF` | Change default font |
| `^CI` | Change character set / encoding |
| `^CW` | Assign alias to a font |

### Fields

| Command | Description |
|---------|-------------|
| `^FB` | Field block — multi-line text with word wrap |
| `^FD` | Field data — text content |
| `^FH` | Field hex indicator — embed hex characters via `_` prefix |
| `^FN` | Field number — named template placeholder |
| `^FO` | Field origin — top-left X,Y anchor |
| `^FR` | Field reverse print — invert this field |
| `^FS` | Field separator — end of field |
| `^FT` | Field typeset — baseline-anchored X,Y position |
| `^FV` | Field variable — variable data for template fields |
| `^FW` | Field orientation — default rotation for subsequent fields |

### Barcodes

| Command | Description |
|---------|-------------|
| `^B2` | Interleaved 2 of 5 (ITF) |
| `^B3` | Code 39 |
| `^B7` | PDF417 |
| `^BC` | Code 128 |
| `^BD` | UPS MaxiCode |
| `^BE` | EAN-13 |
| `^BO` | Aztec |
| `^BQ` | QR Code |
| `^BX` | Data Matrix |
| `^BY` | Barcode field defaults — module width, ratio, height |

### Graphics

| Command | Description |
|---------|-------------|
| `^GB` | Graphic box |
| `^GC` | Graphic circle |
| `^GD` | Graphic diagonal line |
| `^GF` | Graphic field — embedded compressed bitmap |
| `^GS` | Graphic symbol — sizing for built-in symbols |

### Images & Storage

| Command | Description |
|---------|-------------|
| `^DF` | Download format — store a label template in memory |
| `^IL` | Image load — recall a stored graphic by name |
| `^XF` | Recall format — instantiate a stored template |
| `^XG` | Recall graphic — place a stored graphic |
| `~DG` | Download graphics — store a bitmap in printer memory |
| `~DU` | Download unbounded TTF — embed a custom TrueType font |

---

## Not Supported — Rendering Gaps

These commands affect what gets rendered but are not yet implemented. Contributions welcome.

### Fonts & Text

| Command | Description | Notes |
|---------|-------------|-------|
| `^A@` | Use font by name | Alternative font-selection syntax |
| `^FL` | Font linking | Fallback glyph chain across fonts |
| `^PA` | Advanced text properties | Spacing, line height overrides |
| `^TB` | Text block | Like `^FB` but also constrains height |

### Fields

| Command | Description | Notes |
|---------|-------------|-------|
| `^FC` | Field clock | Format real-time clock output |
| `^FE` | Field end / substring | Extract substrings from `^FN` fields |
| `^FM` | Multiple field origins | Set several origin points at once |
| `^FP` | Field parameter | Vertical stacking / reverse text modes |
| `^SF` | Serialize standard field | Auto-increment `^FD` string per copy |
| `^SN` | Serialization data | Auto-increment numeric field per copy |

### Barcodes

| Command | Description |
|---------|-------------|
| `^B1` | Code 11 |
| `^B4` | Code 49 |
| `^B5` | Planet Code |
| `^B8` | EAN-8 |
| `^B9` | UPC-E |
| `^BA` | Code 93 |
| `^BB` | CODABLOCK |
| `^BF` | MicroPDF417 |
| `^BI` | Industrial 2 of 5 |
| `^BJ` | Standard 2 of 5 |
| `^BK` | ANSI Codabar |
| `^BL` | LOGMARS |
| `^BM` | MSI |
| `^BP` | Plessey |
| `^BR` | GS1 DataBar |
| `^BS` | UPC/EAN extensions |
| `^BT` | TLC39 |
| `^BU` | UPC-A |
| `^BZ` | USPS POSTNET / POSTBAR |

### Graphics

| Command | Description |
|---------|-------------|
| `^GE` | Graphic ellipse |

### Label Control

| Command | Description | Notes |
|---------|-------------|-------|
| `^LL` | Label length | Sets label height in dots |
| `^LS` | Label shift | Horizontal shift for all fields |
| `^LT` | Label top | Vertical shift for all fields |
| `^PM` | Print mirror image | Horizontal mirror of entire label |

---

## Out of Scope — Hardware / Device Commands

These commands configure printer hardware and have no visual effect on the rendered output. They are not implemented and are not planned.

| Category | Commands |
|----------|----------|
| Media & print engine | `^MD`, `^MM`, `^MN`, `^MT`, `^MF`, `^ML`, `^MU`, `^MW` |
| Print job control | `^PQ`, `^PR`, `^PH`, `^PF`, `^PN`, `^PP`, `^SP` |
| Sensor & calibration | `^JS`, `^JM`, `^JT`, `^JU`, `^JW`, `^JZ`, `^JB`, `^JH` |
| Serial / config | `^SC`, `^SD`, `^SE`, `^SI`, `^SO`, `^SR`, `^SS`, `^ST`, `^SX`, `^SZ` |
| Object / memory management | `^ID`, `^IM`, `^IS`, `^TO`, `^CM`, `^CN`, `^CO`, `^CP`, `^CV`, `^KD`, `^KL`, `^KN`, `^KP`, `^KV` |
| Host / network | `^HF`, `^HG`, `^HH`, `^HT`, `^HV`, `^HW`, `^HY`, `^HZ`, `^NC`, `^ND`, `^NI`, `^NN`, `^NP`, `^NS`, `^NT`, `^NW`, `^NB`, `^KC` |
| Wireless | `^WA`, `^WE`, `^WI`, `^WL`, `^WP`, `^WR`, `^WS`, `^WX` |
| RFID | `^HL`, `^HR`, `^RA`–`^RZ`, `^WF`, `^WT`, `^WV` |
| Diagnostics (`~`) | `~DG`, `~DS`, `~EG`, `~HB`, `~HD`, `~HI`, `~HQ`, `~HS`, `~HU`, `~JA`–`~JR`, `~RO`, `~SD`, `~TA`, `~WC`, `~WL`, `~PS` |

> Note: `~DG` and `~DU` are exceptions — they store data used during rendering and are supported.
