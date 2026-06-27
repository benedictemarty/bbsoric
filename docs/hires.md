# Pages HIRES — graphisme haute résolution (240×200)

Une page **HIRES** présente un écran graphique Oric (mode haute résolution
**240×200**, VRAM `$A000`, 8000 octets) au lieu du mode TEXT 40×28. Les **3 dernières
lignes restent en TEXT** (`$BB80`) — idéal pour un menu/texte sous le décor (cf.
pattern « menu sur fond d'écran »). Voir `docs/adr/0005-hires-pages.md`.

Deux modèles, **combinables** dans une même page :

- **bitmap** (`background`) : un fond posé d'un bloc (image, logo) ;
- **primitives** (`draw`) : des commandes de tracé appliquées par-dessus.

## Déclarer une page HIRES

Dans une page du `site.json`, à la place de `lines`/`entries` :

```json
"logo": {
  "title": "LOGO",
  "hires": {
    "draw": [
      { "op": "ink",    "c": 3 },
      { "op": "curset", "x": 0,   "y": 0 },
      { "op": "box",    "x": 239, "y": 199 },
      { "op": "curset", "x": 120, "y": 100 },
      { "op": "circle", "r": 40 },
      { "op": "char",   "x": 100, "y": 96, "ch": "O" }
    ]
  }
}
```

Le **fond bitmap** (modèle « bitmap ») se déclare avec `background` : exactement
**8000 octets** de VRAM HIRES (base64 en JSON). On peut combiner `background` (posé
d'abord) **et** `draw` (par-dessus).

### Primitives (`draw`)

Le terminal maintient un **crayon** (pen). Chaque primitive :

| `op` | Champs | Effet |
|------|--------|-------|
| `ink` / `paper` | `c` (0-7) | couleur d'encre / de fond courante |
| `curset` | `x`,`y` | déplace le crayon (sans tracer) |
| `point` | `x`,`y` | allume le pixel (crayon ← x,y) |
| `line` | `x`,`y` | trace du crayon vers (x,y) (crayon ← x,y) |
| `box` | `x`,`y` | rectangle **vide** du crayon à (x,y) |
| `fillbox` | `x`,`y` | rectangle **plein** du crayon à (x,y) |
| `circle` | `r` | cercle de rayon r autour du crayon |
| `char` | `x`,`y`,`ch` | caractère ASCII tracé en (x,y) |

Bornes vérifiées au chargement (`Site.Validate`) : `x` ∈ [0,240[, `y` ∈ [0,200[,
couleurs 0-7, `background` = 8000 octets, `op` connu.

### Navigation

Une page HIRES **avec** `entries` route les touches comme un menu (les libellés sont
dessinés dans le décor ou les 3 lignes texte du bas) ; **sans** `entries`, une touche
suffit pour revenir.

## Protocole fil-de-fer (terminal)

Le serveur sérialise la page via `render.Hires` en un **flux de commandes** ouvert par
la sous-commande série **`1F FC`** (libre, hors plage colonnes ; les terminaux telnet
génériques l'ignorent). Suit une suite d'**opcodes** (`internal/oascii/hires.go`)
jusqu'à `HiEnd` :

| Opcode | Octet | Arguments |
|--------|-------|-----------|
| `HiEnd` | `00` | — (fin du flux → retour TEXT) |
| `HiOn` | `01` | — (bascule HIRES + efface) |
| `HiInk` / `HiPaper` | `02` / `03` | `c` |
| `HiCurset` | `10` | `x y` |
| `HiPoint` | `11` | `x y` |
| `HiLine` | `12` | `x y` |
| `HiBox` | `13` | `x y` |
| `HiFillBox` | `14` | `x y` |
| `HiCircle` | `15` | `r` |
| `HiChar` | `16` | `x y ch` |
| `HiBlit` | `20` | `off_lo off_hi len_lo len_hi <RLE>` |

Le **fond bitmap** est émis par `HiBlit` : `off`/`len` (octets décodés) puis un flux
**RLE** (paires *compteur 1-255 / valeur*) qui redécode exactement `len` octets en
`$A000+off`. Compact pour les grandes plages uniformes (un écran vide ≈ 32 paires au
lieu de 8000 octets).

## État d'avancement

- **Serveur** (modèle, validation, encodeur `render.Hires`, RLE, câblage moteur, tests) : **fait**.
- **Firmware terminal** (interpréteur HIRES `term.s`, primitives 6502, blit RLE, validation `oric1-emu`) : à venir.
- **Studio Forge** (aperçu 240×200, éditeur de primitives / import d'image) : à venir.
