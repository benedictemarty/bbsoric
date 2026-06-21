; ---------------------------------------------------------------------------
;  term.s - Terminal Oric pour le BBS Oric (RX ecran + TX clavier)
;
;  RX  - lit l'ACIA 6551 et ecrit DIRECTEMENT en VRAM (BB80) pour rendre les
;        attributs Teletexte seriels OASCII (octets 0-31).
;  TX  - scanne la matrice clavier (protocole PSG-via-VIA, repris de
;        'Oric asteroids/src/asm/input.s') et envoie la touche a l'ACIA ;
;        echo local a l'ecran.
;
;  Cible oric1-emu (ACIA 031C, VIA 0300). Assemblage xa. Chargement 1000.
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

* = $1000

start:
        sei
        ; ACIA 9600 8N1, DTR on, IRQ off, TX on
        lda #$1E
        sta ACIA_CTL
        lda #$0B
        sta ACIA_CMD

        jsr clear_screen
        lda #<SCREEN
        sta SCRPTR
        lda #>SCREEN
        sta SCRPTR+1
        lda #0
        sta COL

        ; --- init clavier (VIA/PSG) ---
        lda VIA_PCR
        and #$11                 ; preserver bits 0 (CA1) et 4 (CB1)
        sta PCRSAVE
        lda #$FF                 ; DDRA output (pour ecrire le PSG)
        sta VIA_DDRA
        lda #$F7                 ; DDRB - PB3 en entree, reste sortie
        sta VIA_DDRB
        lda #$7F                 ; PSG R7 - port A output, sons coupes
        ldy #7
        jsr psg_write
        lda #0
        sta LASTKEY

main:
        ; --- RX prioritaire (vidange) ---
        lda ACIA_STAT
        and #RDRF
        beq do_keyscan
        lda ACIA_DATA
        jsr putbyte
        jmp main

do_keyscan:
        jsr key_scan             ; A = ASCII (0 si rien)
        cmp #0
        beq ks_release
        cmp LASTKEY
        beq ks_ret               ; meme touche maintenue - pas de repetition
        sta LASTKEY
        jsr acia_tx              ; envoie au serveur (A preserve)
        jsr putbyte              ; echo local
        jmp main
ks_release:
        lda #0
        sta LASTKEY
ks_ret:
        jmp main

; ---------------------------------------------------------------------------
;  putbyte - affiche A a l'ecran (gere CR, LF+scroll, clamp 40 col)
; ---------------------------------------------------------------------------
putbyte:
        cmp #$0D
        beq pb_cr
        cmp #$0A
        beq pb_lf
        ldx COL
        cpx #40
        bcs pb_done              ; ligne pleine - ignore
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
        jmp check_scroll         ; check_scroll fait rts

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
;  acia_tx - envoie l'octet A via l'ACIA (attend TDRE). A preserve.
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
        ora #$EE                 ; latch address
        sta VIA_PCR
        lda PCRSAVE
        ora #$CC                 ; inactive
        sta VIA_PCR
        lda KTMP
        sta VIA_ORA
        lda PCRSAVE
        ora #$EC                 ; write data
        sta VIA_PCR
        lda PCRSAVE
        ora #$CC                 ; inactive
        sta VIA_PCR
        rts

; ---------------------------------------------------------------------------
;  key_scan - scanne la matrice 8x8, renvoie l'ASCII de la 1re touche pressee
;             (0 si aucune). Indep. de la ROM (IRQ deja masquee par SEI).
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
        and #$08                 ; PB3 - 1 si touche pressee
        beq ks_rownext
        lda KCOL
        asl
        asl
        asl
        ora KROW                 ; index = KCOL*8 + KROW
        tax
        lda asciitab,x
        cmp #0
        bne ks_found             ; ignore les modificateurs (ASCII 0)
ks_rownext:
        inc KROW
        lda KROW
        cmp #8
        bne ks_rowloop
        inc KCOL
        lda KCOL
        cmp #8
        bne ks_colloop
        lda #0                   ; rien trouve
ks_found:
        pha
        lda #$FF                 ; reg14 = toutes rangees inactives
        ldy #14
        jsr psg_write
        pla
        rts

; ---------------------------------------------------------------------------
;  Tables
; ---------------------------------------------------------------------------
; Masques R14 - un seul bit a 0 = rangee active (index = rangee)
rowmask:
        .byt $FE,$FD,$FB,$F7,$EF,$DF,$BF,$7F

; ASCII par position matrice, index = colonne*8 + rangee (0 = non mappe).
; Disposition reprise de src/io/keyboard.c (table QWERTY Oric-1).
asciitab:
        ; Col0  7    n    5    v    -    1    x    3
        .byt $37,$6E,$35,$76,$00,$31,$78,$33
        ; Col1  j    t    r    f    -    -    q    d
        .byt $6A,$74,$72,$66,$00,$00,$71,$64
        ; Col2  m    6    b    4    -    z    2    c
        .byt $6D,$36,$62,$34,$00,$7A,$32,$63
        ; Col3  k    9    ;    -    -    -    \    '
        .byt $6B,$39,$3B,$2D,$00,$00,$5C,$27
        ; Col4  SPC  ,    .    UP   LSH  LFT  DWN  RGT
        .byt $20,$2C,$2E,$00,$00,$00,$00,$00
        ; Col5  u    i    o    p    -    -    ]    [
        .byt $75,$69,$6F,$70,$00,$00,$5D,$5B
        ; Col6  y    h    g    e    -    a    s    w
        .byt $79,$68,$67,$65,$00,$61,$73,$77
        ; Col7  8    l    0    /    RSH  RET  -    =
        .byt $38,$6C,$30,$2F,$00,$0D,$00,$3D
