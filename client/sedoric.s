; ---------------------------------------------------------------------------
;  sedoric.s - sauvegarde du buffer recu sur disquette via l'API Sedoric.
;  Concatene a term.s. Prerequis  - Sedoric RESIDENT (Oric boote sur disquette
;  Sedoric, puis le terminal est charge en RAM).
;
;  ATTENTION (24/06/2026) - approche par vecteurs $FF7x SUPERSEDED. Le reverse
;  (cf. docs/sedoric-api.md) a etabli :
;    - l'ecriture disquette est PROUVEE de bout en bout (SAVE Sedoric V3.0 ->
;      .dsk persistee) ; le faux blocage etait le flag emulateur --disk-writeback.
;    - les vecteurs $FF73.. du PDF ne sont PAS exposes par microdis.rom (page $FF
;      vide) -> le code XSAVEB ci-dessous ne s'execute jamais (garde de detection).
;    - le dispatch du SAVE est ENTRELACE avec la ROM BASIC ($F6xx-$F8xx) + de
;      nombreuses variables zero-page ; il n'existe pas d'entree ML isolable
;      triviale. Voie retenue : injection de commande via un mecanisme Sedoric
;      documente (a obtenir) - cf. docs/sedoric-api.md "Approches recommandees".
;
;  Le code ci-dessous est conserve SEULEMENT comme garde no-op sure (detection
;  $FF7C == 4C, fausse sur Microdisc -> ne fait rien, fichier reste en RAM $4000).
;  Il sera remplace par la routine d'injection une fois l'entree ML confirmee.
; ---------------------------------------------------------------------------

XDEFSA = $FF76
XSAVEB = $FF7C
B_BUFNOM = $C029
B_DESALO = $C052
B_FISALO = $C054

; ---------------------------------------------------------------------------
;  sed_save - sauve XSIZE octets de $4000 en fichier "BBSFILE.BIN".
; ---------------------------------------------------------------------------
sed_save:
        lda XSAVEB               ; Sedoric resident ? (vecteur = JMP $DE9C)
        cmp #$4C
        bne sed_ret
        lda XSAVEB+2
        cmp #$DE
        bne sed_ret

        cli                      ; autorise les IRQ (FDC Microdisc)
        lda #$00                 ; type SAVEO (ecrase si existe)
        jsr XDEFSA               ; defauts (A -> VSALO0, FTYPE, EXSALO=0)
        ldx #0
sed_nm:
        lda sed_fname,x
        sta B_BUFNOM,x
        inx
        cpx #12                  ; 9 nom + 3 ext
        bne sed_nm
        lda #$00                 ; DESALO = $4000
        sta B_DESALO
        lda #$40
        sta B_DESALO+1
        clc                      ; FISALO = $4000 + XSIZE
        lda #$00
        adc XSIZE
        sta B_FISALO
        lda #$40
        adc XSIZE+1
        sta B_FISALO+1
        jsr XSAVEB               ; sauve le fichier
        sei                      ; le terminal rebascule en IRQ off (clavier)
        lda #<msg_saved
        sta STRPTR
        lda #>msg_saved
        sta STRPTR+1
        jmp print_string         ; fait rts
sed_ret:
        rts

sed_fname:
        .byt "BBSFILE  BIN"      ; 9 (BBSFILE + 2 esp) + 3 (BIN)
msg_saved:
        .byt $0D,$0A,$02,"SAUVE SUR DISQUETTE",$0D,$0A,$07,$00
