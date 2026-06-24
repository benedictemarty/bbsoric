# API Sedoric — sauvegarde/chargement de fichiers (voie B Microdisc)

> Extrait du désassemblage officiel **« Sedoric 3.0 à nu »**. Sert à écrire/lire
> un vrai **fichier Sedoric** (catalogue, nom) depuis le terminal Oric, pour le
> stockage Microdisc du transfert de fichiers (backlog **G1**, voie B).
>
> **Prérequis : Sedoric résident** (Oric booté depuis une disquette Sedoric ;
> ROM Microdisc active en overlay `$E000`, RAM Sedoric en `$C000+`, table de
> vecteurs en page `$FF`). Le terminal doit donc tourner **sous Sedoric**, pas en
> .tap cassette sur une machine sans disque.

## Table de vecteurs (API publique, page `$FF`)

| Vecteur | JMP | Routine | Rôle |
|---------|-----|---------|------|
| `$FF73` | `4C E5 E0` → `$E0E5` | **XLOADA** | charge le fichier de nom `BUFNOM` selon `VSALO0/1`, `DESALO` |
| `$FF76` | `4C 28 DE` → `$DE28` | **XDEFSA** | positionne les valeurs par défaut pour `XSAVEB` |
| `$FF79` | `4C E6 DF` → `$DFE6` | **XDEFLO** | positionne les valeurs par défaut pour `XLOADA` |
| `$FF7C` | `4C 9C DE` → `$DE9C` | **XSAVEB** | sauve le fichier `BUFNOM` selon `VSALO0/1`, `DESALO`, `FISALO`, `EXSALO` |

(Autres vecteurs utiles : `$FF70` charge selon POSNMX/POSNMP/POSNMS ; `$FF8B`
XWDESC écrit les descripteurs ; `$FF88` XLIBSE secteur libre ; etc.)

## Variables système (RAM Sedoric)

| Adresse | Nom | Rôle |
|---------|-----|------|
| `$C029`–`$C034` | **BUFNOM** | nom du fichier : 9 octets nom + 3 octets extension (complétés d'espaces) ; `$C028` = n° de drive |
| `$C04A` | flag point d'entrée | b7 = mode d'appel |
| `$C050` | VSALO0 | code « type de SAVE » (#00 = SAVEO, #40 = SAVEM…) |
| `$C051` | VSALO1 | flag |
| `$C052`/`$C053` | **DESALO** | adresse de début (source pour SAVE, cible pour LOAD) |
| `$C054`/`$C055` | **FISALO** | adresse de fin (SAVE) |
| `$C056`… | LGSALO / FTYPE | longueur (FISALO-DESALO) / type de fichier |
| EXSALO | EXSALO | adresse d'exécution (0000 si non exécutable) |

## Séquence d'appel — SAUVER un buffer

```asm
; sauve LGSALO octets de $4000 dans un fichier Sedoric nommé.
; 1) remplir BUFNOM ($C029..) avec "NOM      EXT" (9+3, espaces)
; 2) valeurs par défaut
        jsr $FF76          ; XDEFSA
; 3) override de la zone à sauver
        lda #$00           ; DESALO = $4000
        sta $C052
        lda #$40
        sta $C053
        clc                ; FISALO = $4000 + taille
        lda #$00
        adc XSIZE
        sta $C054
        lda #$40
        adc XSIZE+1
        sta $C055
; 4) sauvegarde
        jsr $FF7C          ; XSAVEB
```

## Séquence d'appel — CHARGER un fichier

```asm
; 1) remplir BUFNOM avec le nom
        jsr $FF79          ; XDEFLO (défauts)
; 2) DESALO = adresse cible ($4000) si chargement à adresse imposée
        lda #$00
        sta $C052
        lda #$40
        sta $C053
        jsr $FF73          ; XLOADA  (taille lue dans LGSALO)
```

## Points à valider sur environnement Sedoric (non testés ici)

- **Format exact de `BUFNOM`** (drive en `$C028`, nom/ext justifiés à gauche,
  octets de statut `pstt` en fin selon la version — cf. `$C029..$C038`).
- **Rôle précis de `VSALO0/VSALO1`** après `XDEFSA` (type de SAVE, exécutable).
- **Contexte d'appel** : interruptions, page de travail `$04`, bitmap (BUF2) — la
  sauvegarde met à jour le catalogue + la bitmap ; vérifier qu'aucune init
  préalable n'est requise au-delà du boot Sedoric.
- **Gestion d'erreur** : `XSAVEB`/`XLOADA` peuvent lever DISK_FULL, etc.

## ✅ Écriture disquette VALIDÉE dans l'émulateur (24/06/2026)

La chaîne d'écriture a été **prouvée de bout en bout** dans `oric1-emu`. Le
« blocage » précédent (vecteurs `$FF73` introuvables) était un **faux problème** :
ce n'était ni les adresses API, ni le mapping ROMDIS, mais un **flag de l'émulateur**.

### Cause racine : le write-back est opt-in

`oric1-emu` n'écrit les modifications dans le fichier `.dsk` hôte **que si on passe
`--disk-writeback`** (désactivé par défaut pour ne jamais écraser une `.dsk` par
accident — `src/main.c:3630`, gate `disk_writeback`). Sans ce flag, le `SAVE`
Sedoric s'exécute, écrit bien les secteurs dans l'image **en mémoire** (primitive
FDC en `$D075`, commandes Type II `$A8`/`$AC` sur `$0310`), mais **rien n'est
persisté** → d'où le faux constat « ça ne marche pas ».

### Recette de validation (reproductible)

```sh
cd ~/Oric1
cp disks/sedoric3.dsk /tmp/sedtest.dsk
KEYS='13000000:\n\p1POKE#4000,65:POKE#4001,66\n\p1SAVE"TEST.BIN",A#4000,E#4002\n\p8'
./oric1-emu -n -r roms/basic11b.rom --disk-rom roms/microdis.rom -d /tmp/sedtest.dsk \
    --disk-writeback -c 32000000 --type-keys "$KEYS" --screenshot /tmp/sedok.ppm
# -> log "Disk write-back: drive A", md5 de la .dsk change,
#    entrée catalogue "TEST     BIN" écrite. Boot = "SEDORIC V3.0".
```

Faits établis :
- **Boot Sedoric V3.0 résident** : `-r basic11b.rom --disk-rom microdis.rom -d <dsk>`
  amène au prompt `Ready` (Sedoric installé). Pour booter, `-r` est **obligatoire**.
- **`SAVE"NOM.EXT",A#deb,E#fin`** depuis le prompt écrit un **vrai fichier**
  (catalogue + données + bitmap), persisté avec `--disk-writeback`.
- `microdis.rom` est `Oric DOS V0.6` : sa **page `$FF` est vide** (seuls les
  vecteurs CPU `$FFFA-$FFFF`). Les vecteurs API du PDF (`$FF73`…) n'y sont donc
  pas — l'API Sedoric V3 est installée **en RAM overlay** par le boot.

### Conséquences pour le projet

1. **Test storage** : tout test émulateur de stockage disquette doit passer
   `--disk-writeback`, sinon l'écriture ne persiste pas (piège silencieux).
2. **Terminal `client/sedoric.s`** : l'objectif n'exige **pas** de recaler les
   `$FF73` du PDF. La voie fiable est d'invoquer Sedoric comme le fait le `SAVE`
   BASIC. Reste à déterminer l'**entrée d'appel machine** (interpréteur de
   commande Sedoric, ou primitive de sauvegarde) à appeler depuis le terminal —
   à tracer à partir du chemin du `SAVE` validé ci-dessus (`$D075` est la
   primitive FDC d'écriture secteur ; l'entrée haut niveau est au-dessus).
3. **Déploiement** : le terminal devra tourner **sous Sedoric résident** (booté
   depuis une `.dsk` Sedoric), cf. « Le mur du déploiement » plus bas.

## Le mur du déploiement

Le terminal `client/term.s` est aujourd'hui une **cassette autorun** (`$1000`) sur
une machine **sans disque**. Pour appeler ces vecteurs, il faut que Sedoric soit
**résident**. Deux options à trancher :

1. **Terminal sur disquette Sedoric** : fabriquer une `.dsk` Sedoric contenant le
   terminal, booter Sedoric, le `LOAD`+`RUN`. Le terminal a alors accès à l'API.
2. **Terminal chargé après boot Sedoric** (cassette) : booter Sedoric, `CLOAD` le
   terminal — Sedoric reste résident.

Le **test** complet nécessitera une image `.dsk` Sedoric de travail + un cycle de
debug dans l'émulateur (`--disk-rom microdis.rom --disk sedoric.dsk`).

Voir aussi : `docs/transfert.md`, `docs/agile/backlog.md` (G1).
