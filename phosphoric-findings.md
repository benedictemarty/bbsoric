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

**Bonne commande (LOCI + picowifi/BBS).** Retirer `--loci` (il sert aux opérations
carte SD/flash, pas au modem) ; pour le terminal BBS il suffit de mettre l'ACIA
série à `$03A0` :

```bash
~/Oric1/oric1-emu \
  -t client/term.tap -f -r ~/Oric1/roms/basic11b.rom \
  --serial picowifi --acia-addr 03A0 --serial-buffer 512
```

Dans le terminal : `2` (LOCI/`$03A0`) puis `1` (prod) → connexion.
À noter : sous `--loci` **sans** `--acia-addr`, Phosphoric place l'ACIA par défaut à
`$0380` (emplacement du vrai firmware LOCI, hors MIA) → pas de conflit non plus.

**Côté Phosphoric (v1.27.2).** `oric1-emu` avertit désormais quand `--loci` est
actif et que `--acia-addr` tombe dans `$03A0–$03BF` :
`WARNING: --acia-addr $03A0 tombe dans la MIA LOCI … casse le scan clavier (PSG) -> terminal fige.`

**Reste à faire (Phosphoric).** Faire **coexister** la MIA et la voie série à
`$03A0` comme le vrai LOCI (le modem USB-CDC y est exposé dans l'espace MIA), au
lieu de seulement avertir.
