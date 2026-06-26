; ---------------------------------------------------------------------------
;  sedoric.s - sauvegarde du buffer recu sur disquette via l'API Sedoric.
;  Concatene a term.s. Prerequis - Sedoric RESIDENT (Oric boote sur disquette
;  Sedoric, le terminal tourne sous Sedoric).
;  (NB xa scinde les commentaires sur le deux-points -> on n'en met pas.)
;
;  METHODE (API documentee "Sedoric a nu" + manuel desassemble SEDORIC 3.0)
;    Les routines API vivent dans la RAM OVERLAY ($C000-$FFFF), MASQUEE par
;    defaut (ROM Microdisc / BASIC). Recette langage machine (ANNEXE 15 du
;    manuel SEDORIC 3.0) -
;      1. JSR $04F2   basculer ROM -> RAM overlay (toggle)
;      2. poser les variables systeme (en RAM overlay $C0xx)
;      3. JSR XSAVEB ($DE9C)   sauver (entree directe = cible du vecteur $FF7C)
;      4. JSR $04F2   rebasculer RAM overlay -> ROM
;    $04F2 est la bascule overlay de SEDORIC 3.0 (V1.0/2.x = $0472).
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
;  VALIDATION - recette VALIDEE end-to-end dans l'emulateur sur SEDORIC V3.0 :
;  un fichier ("TESTML  BIN") ecrit et persiste dans la .dsk (entree catalogue +
;  write-back). XSAVEB ($DE9C) et la table $FF sont identiques V1.0/V3.0 ; seule
;  la bascule overlay change ($04F2 en V3.0). Voir docs/sedoric-api.md.
;  (NB xa scinde les commentaires sur le deux-points -> on n'en met pas.)
; ---------------------------------------------------------------------------

OVL_TOGGLE = $04F2          ; bascule ROM <-> RAM overlay SEDORIC 3.0 (toggle)
XSAVEB     = $DE9C          ; entree directe XSAVEB (sauve selon BUFNOM/VSALO0/...)
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
        ; --- garde PRE-bascule (RAM page 4 toujours mappee) ---
        ; Sedoric installe au boot une table de saut en $04F2/$04F5/$04F8
        ; (4C xx 04 = JMP $04xx). Sans Sedoric (terminal cassette sans disque),
        ; JSR $04F2 sauterait dans du code aleatoire -> on verifie d'abord.
        lda OVL_TOGGLE           ; $04F2 = 4C (JMP) ?
        cmp #$4C
        bne sed_ret
        lda OVL_TOGGLE+2         ; cible en page 4 ($04xx) ?
        cmp #$04
        bne sed_ret
        lda OVL_TOGGLE+3         ; $04F5 = 4C (JMP) ? (2e entree de la table)
        cmp #$4C
        bne sed_ret
        lda OVL_TOGGLE+5         ; cible en page 4 ?
        cmp #$04
        bne sed_ret
        ; --- bascule + confirmation overlay ---
        jsr OVL_TOGGLE           ; ROM -> RAM overlay (XSAVEB visible)
        lda XSAVEB               ; XSAVEB debute par SEI $78 ?
        cmp #$78
        beq sed_go
        jsr OVL_TOGGLE           ; pas attendu -> rebascule et abandonne
sed_ret:
        rts
sed_go:
        ; --- nom de fichier dans BUFNOM (dlname = nom recu du serveur) ---
        ldx #11
sed_nm:
        lda dlname,x
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

msg_saved:
        .byt $0D,$0A,$02,"SAUVE SUR DISQUETTE",$0D,$0A,$07,$00
