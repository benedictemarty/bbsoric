; ---------------------------------------------------------------------------
;  term.s - Terminal BBS autonome pour Oric (repertoire + dial AT + RX/TX)
;
;  Au demarrage  - affiche un REPERTOIRE (phonebook). L'utilisateur choisit une
;  entree (1-4) ; le terminal compose lui-meme la commande Hayes ATD<cible> vers
;  le modem (ACIA 6551), puis bascule en mode terminal :
;    RX  - flux serie -> VRAM ($BB80), attributs Teletexte seriels OASCII
;    TX  - scan matrice clavier (PSG-via-VIA) -> ACIA, avec echo local
;
;  Cible oric1-emu (ACIA 031C, VIA 0300). Tester avec  --serial modem
;  (mode commande Hayes). Assemblage xa. Chargement 1000.
;  Commentaires ASCII, sans deux-points (limitations de xa).
; ---------------------------------------------------------------------------

; --- ACIA 6551 ---
ACIA_DATA = $031C
ACIA_STAT = $031D
ACIA_CMD  = $031E
ACIA_CTL  = $031F
RDRF      = $08
TDRE      = $10

; --- VIA 6522 (acces PSG clavier) ---
VIA_ORB   = $0300
VIA_ORA   = $0301
VIA_DDRB  = $0302
VIA_DDRA  = $0303
VIA_PCR   = $030C

; --- Ecran TEXT 40x28 ---
SCREEN    = $BB80
SCREND    = $BFE0
LASTLINE  = $BFB8

; --- Page zero ---
SCRPTR    = $F0
COL       = $F2
SRC       = $F4
DST       = $F6
KCOL      = $F8
KROW      = $F9
LASTKEY   = $FA
PCRSAVE   = $FB
KTMP      = $FC
STRPTR    = $EE          ; pointeur de chaine (print/send)
TARGETLO  = $EC          ; adresse cible de numerotation
TARGETHI  = $ED

NUM_ENTRIES = 4

* = $1000

start:
        sei
        ; ACIA 9600 8N1, DTR on, IRQ off, TX on
        lda #$1E
        sta ACIA_CTL
        lda #$0B
        sta ACIA_CMD

        ; init clavier (VIA/PSG)
        lda VIA_PCR
        and #$11
        sta PCRSAVE
        lda #$FF
        sta VIA_DDRA
        lda #$F7
        sta VIA_DDRB
        lda #$7F
        ldy #7
        jsr psg_write
        lda #0
        sta LASTKEY

; ---------------------------------------------------------------------------
;  Repertoire (phonebook)
; ---------------------------------------------------------------------------
phonebook:
        jsr clear_screen
        jsr reset_cursor
        lda #<pb_text
        sta STRPTR
        lda #>pb_text
        sta STRPTR+1
        jsr print_string

pb_wait:
        jsr get_key              ; A = touche (bloquant)
        sta LASTKEY              ; anti-rebond - ne pas renvoyer ce choix au BBS
        sec
        sbc #'1'                 ; A = index 0..N-1
        bcc pb_wait              ; < '1' -> invalide
        cmp #NUM_ENTRIES
        bcs pb_wait              ; >= N -> invalide
        tax

        ; cible = dial[X]
        lda dial_lo,x
        sta TARGETLO
        lda dial_hi,x
        sta TARGETHI

        ; envoyer "ATD"
        lda #<at_atd
        sta STRPTR
        lda #>at_atd
        sta STRPTR+1
        jsr send_string
        ; envoyer la cible
        lda TARGETLO
        sta STRPTR
        lda TARGETHI
        sta STRPTR+1
        jsr send_string
        ; envoyer CR (declenche la numerotation)
        lda #$0D
        jsr acia_tx

        ; message local puis mode terminal
        lda #<msg_dial
        sta STRPTR
        lda #>msg_dial
        sta STRPTR+1
        jsr print_string

; ---------------------------------------------------------------------------
;  Mode terminal (RX ecran + TX clavier)
; ---------------------------------------------------------------------------
main:
        lda ACIA_STAT
        and #RDRF
        beq do_keyscan
        lda ACIA_DATA
        jsr putbyte
        jmp main

do_keyscan:
        jsr key_scan
        cmp #0
        beq ks_release
        cmp LASTKEY
        beq ks_ret
        sta LASTKEY
        jsr acia_tx
        jsr putbyte              ; echo local
        jmp main
ks_release:
        lda #0
        sta LASTKEY
ks_ret:
        jmp main

; ---------------------------------------------------------------------------
;  get_key - attend (bloquant) une touche, renvoie l'ASCII dans A
; ---------------------------------------------------------------------------
get_key:
        jsr key_scan
        cmp #0
        beq get_key
        rts

; ---------------------------------------------------------------------------
;  reset_cursor - SCRPTR = haut d'ecran, COL = 0
; ---------------------------------------------------------------------------
reset_cursor:
        lda #<SCREEN
        sta SCRPTR
        lda #>SCREEN
        sta SCRPTR+1
        lda #0
        sta COL
        rts

; ---------------------------------------------------------------------------
;  print_string - affiche la chaine terminee par 0 pointee par STRPTR
;                 (via putbyte ; STRPTR est detruit)
; ---------------------------------------------------------------------------
print_string:
        ldy #0
        lda (STRPTR),y
        beq ps_done
        jsr putbyte
        inc STRPTR
        bne print_string
        inc STRPTR+1
        jmp print_string
ps_done:
        rts

; ---------------------------------------------------------------------------
;  send_string - envoie la chaine terminee par 0 pointee par STRPTR via l'ACIA
; ---------------------------------------------------------------------------
send_string:
        ldy #0
        lda (STRPTR),y
        beq ss_done
        jsr acia_tx
        inc STRPTR
        bne send_string
        inc STRPTR+1
        jmp send_string
ss_done:
        rts

; ---------------------------------------------------------------------------
;  putbyte - affiche A (gere CR, LF+scroll, clamp 40 col)
; ---------------------------------------------------------------------------
putbyte:
        cmp #$0D
        beq pb_cr
        cmp #$0A
        beq pb_lf
        ldx COL
        cpx #40
        bcs pb_done
        ldy #0
        sta (SCRPTR),y
        inc SCRPTR
        bne pb_adv
        inc SCRPTR+1
pb_adv:
        inc COL
pb_done:
        rts
pb_cr:
        sec
        lda SCRPTR
        sbc COL
        sta SCRPTR
        lda SCRPTR+1
        sbc #0
        sta SCRPTR+1
        lda #0
        sta COL
        rts
pb_lf:
        lda #40
        sec
        sbc COL
        clc
        adc SCRPTR
        sta SCRPTR
        lda SCRPTR+1
        adc #0
        sta SCRPTR+1
        lda #0
        sta COL
        jmp check_scroll

; ---------------------------------------------------------------------------
;  check_scroll / scroll_up / clear_screen
; ---------------------------------------------------------------------------
check_scroll:
        lda SCRPTR+1
        cmp #>SCREND
        bcc cs_done
        bne cs_do
        lda SCRPTR
        cmp #<SCREND
        bcc cs_done
cs_do:
        jsr scroll_up
        lda #<LASTLINE
        sta SCRPTR
        lda #>LASTLINE
        sta SCRPTR+1
cs_done:
        rts

scroll_up:
        lda #<(SCREEN+40)
        sta SRC
        lda #>(SCREEN+40)
        sta SRC+1
        lda #<SCREEN
        sta DST
        lda #>SCREEN
        sta DST+1
        ldx #4
        ldy #0
su_page:
        lda (SRC),y
        sta (DST),y
        iny
        bne su_page
        inc SRC+1
        inc DST+1
        dex
        bne su_page
        ldy #0
su_rem:
        lda (SRC),y
        sta (DST),y
        iny
        cpy #$38
        bne su_rem
        ldy #0
        lda #$20
su_clr:
        sta LASTLINE,y
        iny
        cpy #40
        bne su_clr
        rts

clear_screen:
        lda #<SCREEN
        sta DST
        lda #>SCREEN
        sta DST+1
        ldx #4
        ldy #0
        lda #$20
clr_page:
        sta (DST),y
        iny
        bne clr_page
        inc DST+1
        dex
        bne clr_page
        ldy #0
clr_rem:
        sta (DST),y
        iny
        cpy #$60
        bne clr_rem
        rts

; ---------------------------------------------------------------------------
;  acia_tx - envoie A via l'ACIA (attend TDRE). A preserve.
; ---------------------------------------------------------------------------
acia_tx:
        pha
tx_wait:
        lda ACIA_STAT
        and #TDRE
        beq tx_wait
        pla
        sta ACIA_DATA
        rts

; ---------------------------------------------------------------------------
;  psg_write - ecrit A dans le registre PSG Y (protocole BDIR/BC1 via VIA)
; ---------------------------------------------------------------------------
psg_write:
        sta KTMP
        tya
        sta VIA_ORA
        lda PCRSAVE
        ora #$EE
        sta VIA_PCR
        lda PCRSAVE
        ora #$CC
        sta VIA_PCR
        lda KTMP
        sta VIA_ORA
        lda PCRSAVE
        ora #$EC
        sta VIA_PCR
        lda PCRSAVE
        ora #$CC
        sta VIA_PCR
        rts

; ---------------------------------------------------------------------------
;  key_scan - scanne la matrice 8x8, renvoie l'ASCII de la 1re touche (0 sinon)
; ---------------------------------------------------------------------------
key_scan:
        lda #0
        sta KCOL
ks_colloop:
        lda VIA_ORB
        and #$F8
        ora KCOL
        sta VIA_ORB
        lda #0
        sta KROW
ks_rowloop:
        ldx KROW
        lda rowmask,x
        ldy #14
        jsr psg_write
        nop
        nop
        lda VIA_ORB
        and #$08
        beq ks_rownext
        lda KCOL
        asl
        asl
        asl
        ora KROW
        tax
        lda asciitab,x
        cmp #0
        bne ks_found
ks_rownext:
        inc KROW
        lda KROW
        cmp #8
        bne ks_rowloop
        inc KCOL
        lda KCOL
        cmp #8
        bne ks_colloop
        lda #0
ks_found:
        pha
        lda #$FF
        ldy #14
        jsr psg_write
        pla
        rts

; ---------------------------------------------------------------------------
;  Donnees
; ---------------------------------------------------------------------------
; Masques R14 - un seul bit a 0 = rangee active (index = rangee)
rowmask:
        .byt $FE,$FD,$FB,$F7,$EF,$DF,$BF,$7F

; ASCII par position matrice, index = colonne*8 + rangee (0 = non mappe).
asciitab:
        .byt $37,$6E,$35,$76,$00,$31,$78,$33
        .byt $6A,$74,$72,$66,$00,$00,$71,$64
        .byt $6D,$36,$62,$34,$00,$7A,$32,$63
        .byt $6B,$39,$3B,$2D,$00,$00,$5C,$27
        .byt $20,$2C,$2E,$00,$00,$00,$00,$00
        .byt $75,$69,$6F,$70,$00,$00,$5D,$5B
        .byt $79,$68,$67,$65,$00,$61,$73,$77
        .byt $38,$6C,$30,$2F,$00,$0D,$00,$3D

; Commande de numerotation
at_atd:
        .byt "ATD", $00
msg_dial:
        .byt $0D,$0A,$02,"Numerotation en cours...",$0D,$0A,$07,$00

; Repertoire affiche ($03=jaune $06=cyan $07=blanc $02=vert)
pb_text:
        .byt "========================================",$0D,$0A
        .byt $03,"          REPERTOIRE BBS ORIC",$0D,$0A
        .byt "========================================",$0D,$0A,$0D,$0A
        .byt $06," 1  ",$07,"BBS Oric (prod)  pavi.3617.fr",$0D,$0A
        .byt $06," 2  ",$07,"ParticlesBBS",$0D,$0A
        .byt $06," 3  ",$07,"Altair",$0D,$0A
        .byt $06," 4  ",$07,"Heatwave",$0D,$0A,$0D,$0A
        .byt $02,"Votre choix (1-4) > ",$07,$00

; Table d'adresses des cibles + chaines de numerotation
dial_lo:
        .byt <dial0,<dial1,<dial2,<dial3
dial_hi:
        .byt >dial0,>dial1,>dial2,>dial3
dial0:
        .byt "pavi.3617.fr:6502",$00
dial1:
        .byt "particlesbbs.dyndns.org:6400",$00
dial2:
        .byt "altair.virtualaltair.com:4667",$00
dial3:
        .byt "heatwave.ddns.net:9640",$00
