; ---------------------------------------------------------------------------
;  term.s - Terminal Oric minimal pour le BBS Oric
;
;  Recoit le flux serie de l'ACIA 6551 et l'ecrit DIRECTEMENT en memoire
;  ecran ($BB80), pour que les octets de controle 0-31 deviennent de vrais
;  attributs Teletexte seriels Oric (encre/fond/clignotement), au lieu d'etre
;  interpretes par les routines ROM.
;
;  Cible  - oric1-emu (ACIA @ $031C). Assemblage  - xa. Chargement  - $1000.
;
;  CR ($0D) -> debut de ligne ; LF ($0A) -> ligne suivante + scroll.
;  Caracteres et attributs  - ecrits a la position courante (clamp a 40 col).
;  TX clavier omis (le serveur emet la banniere a la connexion).
;  (Commentaires en ASCII  - xa ne supporte pas l'UTF-8.)
; ---------------------------------------------------------------------------

; Registres ACIA 6551 (oric1-emu, base $031C)
ACIA_DATA = $031C        ; R - RDR / W - TDR
ACIA_STAT = $031D        ; R - status
ACIA_CMD  = $031E        ; command
ACIA_CTL  = $031F        ; control
RDRF      = $08          ; status bit3  - Receiver Data Register Full
TDRE      = $10          ; status bit4  - Transmit Data Register Empty

; Ecran TEXT Oric  - 40x28 a $BB80
SCREEN    = $BB80
SCREND    = $BFE0        ; fin exclusive = $BB80 + 28*40
LASTLINE  = $BFB8        ; $BB80 + 27*40

; Variables page zero (IRQ masquees, ROM non appelee -> ZP libre)
SCRPTR    = $F0          ; pointeur ecran courant (2 octets)
COL       = $F2          ; colonne courante (0..40)
SRC       = $F4          ; pointeur source scroll (2 octets)
DST       = $F6          ; pointeur destination (2 octets)

* = $1000

start:
        sei                      ; on prend la main (pas d'IRQ ROM)
        ; init ACIA  - 9600 8N1, DTR on, IRQ off, TX on
        lda #$1E                 ; control  - 9600, 8 bits, 1 stop, horloge interne
        sta ACIA_CTL
        lda #$0B                 ; command  - DTR=1, RX-IRQ off, RTS low, sans parite
        sta ACIA_CMD

        jsr clear_screen
        ; SCRPTR = SCREEN ; COL = 0
        lda #<SCREEN
        sta SCRPTR
        lda #>SCREEN
        sta SCRPTR+1
        lda #0
        sta COL

main:
        lda ACIA_STAT
        and #RDRF
        beq main                 ; rien recu
        lda ACIA_DATA            ; octet recu (acquitte RDRF)

        cmp #$0D
        beq do_cr
        cmp #$0A
        beq do_lf

        ; caractere imprimable ou octet d'attribut  - ecrire au curseur
        ldx COL
        cpx #40
        bcs main                 ; ligne pleine  - ignore jusqu'a CR/LF (clamp)
        ldy #0
        sta (SCRPTR),y
        inc SCRPTR
        bne adv_col
        inc SCRPTR+1
adv_col:
        inc COL
        jmp main

do_cr:
        ; SCRPTR -= COL ; COL = 0 (retour debut de ligne)
        sec
        lda SCRPTR
        sbc COL
        sta SCRPTR
        lda SCRPTR+1
        sbc #0
        sta SCRPTR+1
        lda #0
        sta COL
        jmp main

do_lf:
        ; SCRPTR += (40 - COL) ; COL = 0 ; verifier scroll
        lda #40
        sec
        sbc COL                  ; A = 40 - COL
        clc
        adc SCRPTR
        sta SCRPTR
        lda SCRPTR+1
        adc #0
        sta SCRPTR+1
        lda #0
        sta COL
        jsr check_scroll
        jmp main

; --- Scroll si le curseur a depasse le bas de l'ecran ---
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

; --- Remonte l'ecran d'une ligne (1080 octets) + efface la derniere ---
scroll_up:
        lda #<(SCREEN+40)
        sta SRC
        lda #>(SCREEN+40)
        sta SRC+1
        lda #<SCREEN
        sta DST
        lda #>SCREEN
        sta DST+1
        ldx #4                   ; 4 pages pleines (1024 octets)
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
        ; reste  - 1080 - 1024 = 56 octets ($38)
        ldy #0
su_rem:
        lda (SRC),y
        sta (DST),y
        iny
        cpy #$38
        bne su_rem
        ; efface la derniere ligne (40 octets) avec des espaces
        ldy #0
        lda #$20
su_clr:
        sta LASTLINE,y
        iny
        cpy #40
        bne su_clr
        rts

; --- Efface tout l'ecran avec des espaces (1120 octets) ---
clear_screen:
        lda #<SCREEN
        sta DST
        lda #>SCREEN
        sta DST+1
        ldx #4                   ; 4 pages (1024) + reste 96
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
        cpy #$60                 ; 96 octets restants
        bne clr_rem
        rts
