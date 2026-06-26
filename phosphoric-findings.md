# Findings Phosphoric (émulateur Oric) rencontrés depuis bbsoric

Journal des défauts / pièges de l'émulateur **Phosphoric** (`~/Oric1`) repérés en
testant le terminal BBS. À remonter à l'équipe Phosphoric (et déjà corrigés pour
certains).

---

## F1 — `--loci` + `--acia-addr 03A0` fige le clavier (annuaire « gelé »)

**Statut : diagnostiqué + reproduit. Garde-fou ajouté côté Phosphoric (v1.27.2).**

| Commande | Résultat |
|----------|----------|
| `--serial picowifi --acia-addr 03A0` (sans `--loci`) | ✅ connecte (CONNECT 9600 + bannière) |
| `--loci --serial picowifi --acia-addr 03A0` | ❌ **figé sur l'annuaire** |

**Cause.** `--loci` place la MIA LOCI sur `$03A0–$03BF` et `--acia-addr 03A0` y
force aussi l'ACIA → **double mappage** au même endroit. Dans les callbacks I/O de
Phosphoric, la MIA est routée **avant** l'ACIA → elle **masque** l'ACIA. Or la MIA
pilote le **PSG (AY)**, et le `key_scan` du terminal scanne le clavier via ce même
PSG. Résultat : le scan lit du vide → `get_key` boucle indéfiniment → l'annuaire
paraît figé (il attend une touche qu'il ne verra jamais). Confirmé par le log :
`LOCI: pre-seeded PSG R7=$7F … + LOCI MIA enabled at $03A0-$03BF`.

**⚠️ Correction (le picowifi EST le modem du LOCI).** Recommander de « retirer
`--loci` » était une erreur : le picowifi n'a de sens **qu'avec** l'émulation LOCI.
Le vrai LOCI expose son modem comme **ACIA à `$0380`** (confirmé par le firmware de
référence `~/picowifi/PicoWiFiModemUSB` : « programme Oric de référence (via LOCI,
ACIA `$0380`) »). Le `$03A0` est l'**espace MIA** (pas le modem) ; il ne « marchait »
que **sans** `--loci`, en posant une ACIA nue à `$03A0` — un contournement non fidèle.

**Bonne commande (fidèle au matériel).** Garder `--loci`, **sans** `--acia-addr`
(l'ACIA va par défaut à `$0380`) :

```bash
~/Oric1/oric1-emu \
  -t client/term.tap -f -r ~/Oric1/roms/basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

Le terminal doit alors **adresser le modem à `$0380`**, pas `$03A0`.

**Validé sur le firmware LOCI officiel** (`github.com/sodiumlb/loci-firmware`) :

| Fait | Source |
|------|--------|
| Modem/série = **ACIA `$0380-$0383`** | `src/mia/oric/acia.h` : `ACIA_IO_DATA 0x0380` / `STAT 0x0381` / `CMD 0x0382` / `CTRL 0x0383` |
| `$03A0-$03BF` = **registres MIA** | `src/mia/sys/mem.h` : `.equ regs, 0x200400A0  // Oric address 0x03A0-0x03BF` |
| `$03A0`/`$03A2` = **console UART de la MIA** (≠ modem) | `src/mia/sys/mia.c` (0x03A0 « UART Tx/Rx flow control », 0x03A2 « UART Rx ») |
| `$03A3` = **« ULA pattern match »** | `src/mia/sys/mia.c` `CASE_WRITE(0x03A3)` → explique pourquoi écrire l'« ACIA control » à `$03A3` casse l'ULA/PSG/clavier |

**À corriger côté terminal (bbsoric).** L'option « `2` = LOCI / `$03A0` » du menu
modem vise la mauvaise base : pour le vrai LOCI il faut **`$0380`** (registres
`$0380-$0383`). → ajuster `client/term.s` (cf. `MESSAGE-equipe-terminal-LOCI.md`).

**Côté Phosphoric (v1.27.2 → 1.27.3).** `oric1-emu` avertit quand `--loci` est actif
et que `--acia-addr` tombe dans `$03A0–$03BF`, et **pointe désormais vers `$0380`** :
`WARNING: … Le modem LOCI (picowifi) est exposé à $0380 … laissez --loci SANS --acia-addr et adressez $0380.`

**Reste à faire (Phosphoric).** Faire **coexister** la MIA et une voie série dans
l'espace MIA comme le vrai LOCI (le modem USB-CDC y est exposé via le protocole MIA),
pour les logiciels qui passeraient par la MIA plutôt que par l'ACIA `$0380`.
