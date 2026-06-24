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

## Voie retenue : injection de commande (décidé 24/06/2026)

Plutôt que d'appeler une routine SAVE interne avec une convention de paramètres
reverse-engineerée (fragile, spécifique à l'image), le terminal **injecte une
ligne de commande Sedoric et la fait exécuter** — exactement le chemin validé du
`SAVE` BASIC. Robuste et proche du matériel réel.

### Carte reverse établie (image `sedoric3.dsk`, `oric1-emu`)

Reverse par save-state au prompt + trace CPU ciblée + watchpoint mémoire
(`memory_set_trace`, type `MEM_READ`). Le `$` n'apparaissant que dans le
désassemblage (colonne cycle décimale), on isole les accès mémoire sans ambiguïté.

| Élément | Adresse | Comment validé |
|---|---|---|
| **Buffer ligne de commande** | **`$0035`** | dump RAM : la ligne `SAVE"…",A#…,E#…` y réside (buffer d'entrée BASIC Oric) |
| **Scanner de buffer (auto-modifiant)** | **`$00E2`–`$00ED`** | trace : l'opérande de `LDA $00E8` est l'octet `$E9/$EA`, incrémenté pour avancer ; saute les espaces (`CMP #$20`) |
| Lecture du buffer (absolu) | `LDA $0035`, `$0039`…`$0051` | trace : lectures byte-à-byte du buffer pendant le dispatch |
| Table de mots-clés Sedoric | ~`$CA6F` | dump RAM : `SAVE FIELD RSEC INIT INSTR…` ; matchée via pointeur `$DE/$DF`, séparateur quote `$22` |
| Helper compare-chaîne | `$D5B5` | trace : `LDA ($DE),Y` / `CMP $24/$25` |
| Routine de sauvegarde (cluster) | `$D33A`/`$D342`/`$D398`/`$D39E` | trace : JSR peu imbriqués juste avant l'écriture |
| **Primitive FDC write secteur** | **`$D075`** | trace : commandes Type II `$A8`/`$AC` sur `$0310` |
| Trampolines page 4 | `$04EF`→`JMP $C4A0`, `$0474`, `$0477` | trace : sauts indirects RAM ↔ Sedoric |

### Conclusion : le dispatch est entrelacé avec la ROM BASIC

Point décisif du reverse : **le `SAVE` n'est PAS dispatché par une entrée Sedoric
isolable**. Quand on tape la commande + Return, c'est la **ROM BASIC**
(`$F6xx`–`$F8xx`, routines d'entrée ligne) qui traite la ligne **puis** appelle le
scanner Sedoric ; `$C4A0` (cœur du prompt) n'est exécuté qu'**une fois en idle**,
pas sur le chemin du `SAVE`. Le dispatch dépend de nombreuses variables zéro-page
(`$A9` position/ligne-prête, `$24/$25`, `$DE/$DF`, `$E9/$EA`, `$2E`, `$0252`,
`$02F2`…) posées par la chaîne d'entrée BASIC.

**Conséquence** : appeler `SAVE` depuis du code machine **autonome** (le terminal)
ne se réduit pas à un `JSR <entrée>` avec la commande en `$0035`. Il faut soit
reproduire fidèlement le contexte d'entrée BASIC (fragile, spécifique à cette
image), soit utiliser un mécanisme documenté de Sedoric pour exécuter une commande
depuis l'ML.

### Approches recommandées (par robustesse décroissante)

1. **Mécanisme documenté Sedoric** — récupérer dans la doc « Sedoric à nu » (ou le
   manuel) l'**entrée officielle d'exécution de commande ML** (Sedoric en expose
   une : commande en buffer + appel d'un point d'entrée stable, conçu pour ça).
   C'est la seule voie *version-portable* et *matériel-réel*. À privilégier.
2. **Injection clavier (type-ahead)** — déposer la commande dans le tampon clavier
   et rendre la main au prompt Sedoric, qui l'exécute « comme tapée ». Robuste,
   mais suppose que le terminal puisse revenir proprement au prompt.
3. **Reproduction du contexte BASIC** — poser `$0035` + toutes les variables
   zéro-page ci-dessus et entrer dans la chaîne BASIC. **Déconseillé** : fragile,
   non portable, à revalider à chaque version Sedoric.

> Recette de trace rapide (save-state au prompt) pour itérer :
> `--load-state sed.state --type-keys '…SAVE…\n' --trace t.log --trace-max 4000000`
> puis grep des accès `$0035` (lecture buffer) et `$D075` (write FDC).

### Déploiement validé sans repackaging disque

Le terminal `client/term.s` est une **cassette** ; or `tap2sedoric` (outil
`oric1-emu`) est un **stub non implémenté** → pas de fabrication directe de `.dsk`.
Voie de déploiement **réaliste et testable** : booter Sedoric, puis **`CLOAD` le
terminal depuis la cassette** — Sedoric **reste résident**, le terminal tourne
avec l'API disque disponible. (Validé conceptuellement : Sedoric résident + tape.)

## API documentée confirmée + écart V1.0 doc / V3.0 image (24/06/2026)

La doc **« Sedoric à nu »** (`sednb3_0.pdf`) donne l'API officielle. Vérifications
faites contre l'image **sedoric3.dsk (SEDORIC V3.0)** de l'émulateur.

### ✅ Table de vecteurs publique — IDENTIQUE V1.0 et V3.0

En dumpant la **vue CPU `$C000-$FFFF` pendant un SAVE** (overlay mappé), on lit
sur V3.0 exactement les vecteurs du PDF (qui décrit SEDORIC 1.0) :

| Vecteur | Contenu (V3.0 mesuré) | = PDF V1.0 |
|---|---|---|
| `$FF7C` XSAVEB | `4C 9C DE` → JMP `$DE9C` | ✅ identique |
| `$FF76` XDEFSA | `4C 28 DE` → JMP `$DE28` | ✅ identique |

→ **La table `$FF43-$FFC6` est l'interface stable** (c'est son rôle). Les
variables système (`$C04D` VSALO0, `$C051` FTYPE, `$C052` DESALO, `$C054`
FISALO…) sont aux mêmes adresses. La séquence de `client/sedoric.s` (poser les
variables + `JSR $FF7C`) est donc **correcte**.

### Bascule overlay — spécifique à la version (V3.0 résolue)

Pour rendre les routines `$C000-$FFFF` visibles, il faut basculer sur la RAM
overlay. **L'adresse de la bascule change selon la version** :

| Version | Bascule overlay (toggle) | Source |
|---|---|---|
| Sedoric 1.0/2.x | `JSR $0472` | PDF « Sedoric à nu » |
| **Sedoric 3.0** | **`JSR $04F2`** | manuel désassemblé SEDORIC 3.0, ANNEXE 15 |

Sur V3.0, `$0472` n'est pas la bascule (page 4 réorganisée par la gestion de
**banques** de la 3.0) → `JSR $0472` **plante**. Le manuel 3.0 documente
explicitement : « en langage machine, faire un `JSR $04F2` pour accéder à la RAM
overlay, appeler les sous-programmes voulus, et terminer par un autre `JSR $04F2`
pour revenir ». L'écriture `$0314` brute plante (XSAVEB exige le contexte runtime).

### ✅ Recette V3.0 VALIDÉE end-to-end (24/06/2026)

Testée dans l'émulateur sur `sedoric3.dsk` (un fichier `TESTML  BIN` **écrit et
persisté** dans la `.dsk` — entrée catalogue + write-back, md5 modifié) :

```asm
        jsr $04F2          ; ROM -> RAM overlay (toggle V3.0)
        ; poser BUFNOM ($C029), VSALO0 ($C04D=#00), FTYPE ($C051=#40),
        ;   DESALO ($C052=$4000), FISALO ($C054=fin), LGSALO ($C04F=taille),
        ;   EXSALO ($C056=0), VSALO1 ($C04E=0)
        jsr $DE9C          ; XSAVEB (entree directe = cible du vecteur $FF7C)
        jsr $04F2          ; RAM overlay -> ROM
```

`client/sedoric.s` implémente cette recette (`OVL_TOGGLE = $04F2`,
`XSAVEB = $DE9C`, détection « XSAVEB débute par `SEI $78` »). Pour cibler la
1.x/2.x, mettre `OVL_TOGGLE = $0472`. **L'incertitude est levée** ; l'intégration
`term.s` → `sed_save` (après un download, taille en `XSIZE`) est déjà câblée.

### Garde de présence Sedoric (sûre sans disque)

`sed_save` vérifie d'abord, en **RAM page 4 toujours mappée** (avant tout
`JSR $04F2`), la **table de saut** que Sedoric installe au boot en `$04F2`/`$04F5`
(`4C xx 04` = `JMP $04xx`). Validé :

- **Sous Sedoric** : `$04F2 = $4C`, `$04F4 = $04` → garde OK, fichier sauvé
  (`TESTG4 BIN` écrit dans la `.dsk`).
- **Sans disque** (terminal cassette, Atmos seul) : `$04F2 = $55` (motif RAM) →
  garde refuse, **aucun** `JSR $04F2`, pas de plantage.

Ainsi le même terminal est sûr en cassette **et** sous Sedoric ; la sauvegarde ne
s'active que si Sedoric est réellement résident. Reste le **déploiement** du
terminal sous Sedoric résident (booté disquette ou `CLOAD`).

> Détail validation : le harnais `--type-keys` perd parfois le 1er caractère
> d'une ligne (n° de ligne BASIC, sans incidence sur les valeurs DATA) — prévoir
> un `\n` de purge en tête. Confirmé d'abord par l'exemple « HELLO ANDRE » de
> l'ANNEXE 15 (`JSR $04F2`/`JSR $D637`/`JSR $04F2`), puis par XSAVEB.

## ✅ Disquette bootable du terminal (déploiement voie B)

`client/build-disk.sh` fabrique une disquette Sedoric contenant le terminal, de
façon **reproductible** (validé dans l'émulateur) :

1. assemble `term.bin` (`build.sh`) ;
2. fabrique une cassette **non-autorun** du terminal (octet autorun `$C7` → `$00`) ;
3. pilote `oric1-emu` : boot Sedoric (master) + **fast-load** de la cassette — le
   terminal est injecté en RAM `$1000` à ~3M cycles (phase 1 du fast-load, *sans
   CLOAD*) et **survit au boot Sedoric** ; au prompt, `SAVE"TERM",A#1000,E#1E26`
   écrit **TERM.COM** sur une copie du master ;
4. `--disk-writeback` persiste → `client/term-boot.dsk`.

**Lancement du terminal** depuis la disquette (validé — le menu modem s'affiche) :

```
LOAD"TERM":CALL#1000
```

> Au menu « TYPE DE MODEM », choisir **LOCI `$03A0`** si un Microdisc est présent :
> l'ACIA `$031C` chevauche la plage I/O Microdisc `$0310-$031F`. Le terminal gère
> les deux adresses au runtime (`ACIAPTR`), aucune variante de build n'est requise.

**Notes de mise au point** :
- Le terminal **tourne** sous Sedoric (≈2,6 M instructions exécutées, menu affiché) ;
  le `BREAK ON BYTE #1000` initial venait de l'option Sedoric `,J` (`LOAD"TERM",J`),
  **pas** d'un conflit runtime → utiliser `LOAD` + `CALL`.
- Auto-démarrage *hands-free* : mécanisme identifié — au boot, Sedoric cherche
  **`BOOTUP.COM`** et exécute `!BOOTUP` (manuel SEDORIC 3.0, désassemblage
  `; found BOOTUPCOM ? executes !BOOTUP`). **Mais** sur `sedoric3.dsk` (master/
  outils) le menu « WELCOME TO SEDORIC DOS V3.0 » n'est **pas** un fichier
  directory remplaçable : `DESTROY"BOOTUP.COM"` répond *FILE NOT FOUND* → le menu
  est **intégré au système** du master (sur le master, `DESTROY"BOOTUP.COM"` =
  *FILE NOT FOUND* alors que le menu tourne quand même).
- **Blocage émulateur pour le hands-free** : créer une disquette Sedoric vierge
  où poser `BOOTUP.COM` exige `INIT` (formatage) → *Write Track* du FDC. Or, dans
  `oric1-emu`, `FDC_OP_WRITE_TRACK` est positionné (`src/storage/disk.c`) mais
  **sans handler de données** = **no-op** : le formatage n'écrit rien. `INIT` ne
  peut donc pas produire de disquette bootable dans l'émulateur.
- **Conséquence** : l'auto-démarrage hands-free **n'est pas validable dans
  l'émulateur**. Sur **matériel réel** (où `INIT` formate normalement), la voie
  est : `INIT` une disquette Sedoric minimale → y copier `TERM.COM` → créer
  `BOOTUP.COM` = lanceur (`LOAD"TERM":CALL#1000`). En l'état (émulateur **et**
  master), **une commande** lance le terminal (`LOAD"TERM":CALL#1000`).

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
