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

## ⚠️ Découvertes émulateur (mapping mémoire) — à intégrer

Tests dans `oric1-emu` (ROM `microdis.rom` + `sedoric3.dsk`, sorti vers BASIC) :

- **Les vecteurs page `$FF` sont MASQUÉS** par la **ROM Microdisc** (overlay
  `$E000-$FFFF`) : `$FF7C` lit `20 F5 F9` (ROM microdis), pas `4C 9C DE`. On ne
  peut donc **pas** appeler `XSAVEB` via `$FF7C` sans gérer le mapping (ROMDIS,
  registre `$0314`).
- **`$C000-$DFFF` est de la RAM Sedoric** (accessible) ; les routines en `$C0xx`/
  `$DExx` y sont, mais celles en `$E0xx` (ex `XLOADA $E0E5`) sont masquées.
- **Les adresses du PDF ne collent pas à cette image** : `$DE9C` contient
  `D0 84 DF 60…` (pas le début attendu de XSAVEB). Le désassemblage « à nu »
  correspond à une **version/un mapping différents** — les adresses doivent être
  **recalées sur la version Sedoric cible**.

**Conséquence** : l'appel API n'est pas un simple `JSR $FF7C`. Il faut (1) recaler
les adresses sur l'image Sedoric utilisée, et (2) gérer le bascule ROMDIS pour
exposer la RAM Sedoric à l'appel. C'est un travail de reverse spécifique à la
version, mieux mené avec l'image cible (et idéalement validé sur matériel réel).

> **Statut du code** : `client/sedoric.s` (`sed_save`) est assemblé et **protégé
> par une détection** (`$FF7C == 4C…`) : si le mapping ne l'expose pas, il **ne
> fait rien** (le fichier reste en RAM `$4000`, pas de plantage). Il n'est donc
> **pas encore opérationnel** sur la config Microdisc émulée — voir ci-dessus.

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
