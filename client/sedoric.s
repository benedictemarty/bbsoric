; ---------------------------------------------------------------------------
;  sedoric.s - sauvegarde du buffer recu sur disquette via l'API Sedoric.
;  Concatene a term.s. Prerequis  - Sedoric RESIDENT (Oric boote sur disquette
;  Sedoric, puis le terminal est charge en RAM).
;
;  API (cf. docs/sedoric-api.md, desassemblage "Sedoric 3.0 a nu")  -
;    $FF76 XDEFSA  defauts de sauvegarde (A -> VSALO0)
;    $FF7C XSAVEB  sauve selon BUFNOM, DESALO, FISALO, EXSALO
;    BUFNOM $C029 (9 nom + 3 ext)  DESALO $C052 (debut)  FISALO $C054 (fin)
;
;  Detection  - si le vecteur XSAVEB ($FF7C) ne contient pas JMP $DE.. , Sedoric
;  est absent -> on ne fait rien (le fichier reste en RAM $4000).
;
;  Statut (24/06/2026)  - l'ecriture disquette est PROUVEE de bout en bout dans
;  l'emulateur (SAVE Sedoric V3.0 -> .dsk persistee, cf. docs/sedoric-api.md).
;  Le faux blocage etait le flag emulateur --disk-writeback, PAS les adresses.
;  Les vecteurs $FF73.. du PDF ne sont PAS exposes tels quels par microdis.rom :
;  l'entree d'appel machine reelle reste a tracer a partir du SAVE BASIC valide.
;  Ce code (vecteurs PDF) n'est donc PAS encore l'appel correct -> a recaler.
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
