; ---------------------------------------------------------------------------
;  sedoric.s - sauvegarde du buffer recu sur disquette via l'API Sedoric.
;  Concatene a term.s. Prerequis - Sedoric RESIDENT (Oric boote sur disquette
;  Sedoric, le terminal tourne sous Sedoric).
;  (NB xa scinde les commentaires sur le deux-points -> on n'en met pas.)
;
;  METHODE (API documentee "Sedoric a nu", F.BROCHE/D.SEBBAG)
;    Les vecteurs API vivent dans la RAM OVERLAY ($C000-$FFFF), MASQUEE par
;    defaut (ROM Microdisc / BASIC). Il faut donc
;      1. JSR $0472   basculer ROM -> RAM overlay (la table $FF.. devient visible)
;      2. poser les variables systeme (en RAM overlay $C0xx)
;      3. JSR XSAVEB ($FF7C)   sauver
;      4. JSR $0472   rebasculer RAM overlay -> ROM
;    $0472 est une bascule (toggle) ; un 2e JSR $0472 revient sur la ROM.
;
;  Variables RAM overlay (cf. desassemblage XSAVEB $DE9C)
;    BUFNOM $C029 (9 nom + 3 ext, espaces), drive en $C028
;    VSALO0 $C04D   type de SAVE (#00 SAVEO ecrase, #80 SAVE, #C0 SAVEU .BAK)
;    VSALO1 $C04E
;    LGSALO $C04F/$C050   longueur (FISALO - DESALO)
;    FTYPE  $C051   type fichier (#40 bloc de donnees)
;    DESALO $C052/$C053   adresse debut (source du SAVE)
;    FISALO $C054/$C055   adresse fin
;    EXSALO $C056/$C057   adresse d'execution (0 non executable)
;
;  PORTEE / VALIDATION - recette de l'API documentee (Sedoric 1.x/2.x). L'image
;  de test de l'emulateur est SEDORIC V3.0 dont les adresses de page 4 (dont la
;  bascule overlay) DIFFERENT (page 4 dynamique/auto-modifiante). La validation
;  end-to-end requiert une disquette Sedoric 1.x ou du materiel reel. Voir
;  docs/sedoric-api.md (section "Ecart V1.0 doc / V3.0 image").
; ---------------------------------------------------------------------------

OVL_TOGGLE = $0472          ; bascule ROM <-> RAM overlay (toggle)
XSAVEB     = $FF7C          ; JMP $DE9C, sauve selon BUFNOM/VSALO0/DESALO/FISALO
B_DRIVE    = $C028
B_BUFNOM   = $C029
V_VSALO0   = $C04D
V_VSALO1   = $C04E
V_LGSALO   = $C04F
V_FTYPE    = $C051
V_DESALO   = $C052
V_FISALO   = $C054
V_EXSALO   = $C056

; ---------------------------------------------------------------------------
;  sed_save - sauve XSIZE octets de $4000 en fichier "BBSFILE.BIN".
;  XSIZE (mot, zero-page) = taille recue ; defini par term.s/xmodem.s.
; ---------------------------------------------------------------------------
sed_save:
        jsr OVL_TOGGLE           ; ROM -> RAM overlay (table $FF visible)
        lda XSAVEB               ; Sedoric resident ? (vecteur = JMP ..)
        cmp #$4C
        beq sed_go
        jsr OVL_TOGGLE           ; pas Sedoric -> rebascule et abandonne
        rts
sed_go:
        ; --- nom de fichier dans BUFNOM ---
        ldx #11
sed_nm:
        lda sed_fname,x
        sta B_BUFNOM,x
        dex
        bpl sed_nm
        ; --- type et flags ---
        lda #$00                 ; SAVEO, ecrase sans creer de .BAK
        sta V_VSALO0
        sta V_VSALO1
        sta V_EXSALO             ; EXSALO = 0000 (non executable)
        sta V_EXSALO+1
        lda #$40                 ; FTYPE = bloc de donnees
        sta V_FTYPE
        ; --- DESALO = $4000 ---
        lda #$00
        sta V_DESALO
        lda #$40
        sta V_DESALO+1
        ; --- FISALO = $4000 + XSIZE ---
        clc
        lda #$00
        adc XSIZE
        sta V_FISALO
        lda #$40
        adc XSIZE+1
        sta V_FISALO+1
        ; --- LGSALO = XSIZE ---
        lda XSIZE
        sta V_LGSALO
        lda XSIZE+1
        sta V_LGSALO+1
        ; --- sauvegarde ---
        jsr XSAVEB               ; XSAVEB ecrit le fichier sur disquette
        jsr OVL_TOGGLE           ; RAM overlay -> ROM
        lda #<msg_saved
        sta STRPTR
        lda #>msg_saved
        sta STRPTR+1
        jmp print_string         ; fait rts

sed_fname:
        .byt "BBSFILE  BIN"      ; 9 (BBSFILE + 2 esp) + 3 (BIN)
msg_saved:
        .byt $0D,$0A,$02,"SAUVE SUR DISQUETTE",$0D,$0A,$07,$00
