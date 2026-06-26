# Message équipe terminal — l'option modem « LOCI » doit viser `$0380`, pas `$03A0`

*(Validé sur le firmware LOCI officiel `github.com/sodiumlb/loci-firmware`.)*

Salut,

Bug confirmé dans le menu modem de `term.tap` (`client/term.s`).

## Le problème

L'option « LOCI » initialise une ACIA à **`$03A0`**. Or `$03A0-$03BF` est le
**fichier de registres de la MIA** du LOCI (`src/mia/sys/mem.h` :
*« Oric address 0x03A0-0x03BF »*), pas le modem. Pire : `$03A2` est la **console
UART de la MIA** et `$03A3` est le registre **« ULA pattern match »**
(`src/mia/sys/mia.c`). Écrire l'« ACIA control » à `$03A3` perturbe l'ULA/PSG →
le scan clavier ne lit plus rien → **le terminal se fige sur l'annuaire**.

## La bonne base

Le modem WiFi du LOCI est exposé comme **ACIA 6551 à `$0380-$0383`**
(`src/mia/oric/acia.h`) :

```c
#define ACIA_IO_DATA 0x0380
#define ACIA_IO_STAT 0x0381
#define ACIA_IO_CMD  0x0382
#define ACIA_IO_CTRL 0x0383
```

## Correctif demandé (`client/term.s`)

Faire pointer l'option « LOCI » sur la base **`$0380`** (registres `$0380-$0383`)
au lieu de `$03A0`.

## Test (émulateur, modèle fidèle)

```bash
oric1-emu -t client/term.tap -f -r roms/basic11b.rom \
  --loci --serial picowifi --serial-buffer 512
```

(sans `--acia-addr` : l'ACIA va par défaut à `$0380`). Ne **pas** lancer avec
`--acia-addr 03A0`.

## Note

`$03A0` ne « marchait » que **sans** `--loci` (ACIA nue posée là — non fidèle au
matériel ; le picowifi n'a de sens qu'avec l'émulation LOCI). Phosphoric ≥ 1.27.3
avertit et pointe vers `$0380`. Détails techniques : `phosphoric-findings.md` (F1).
