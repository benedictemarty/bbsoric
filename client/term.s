; ---------------------------------------------------------------------------
;  term.s - Terminal BBS autonome pour Oric
;
;  Demarrage  -> menu modem -> repertoire (ou saisie manuelle) -> numerotation
;  AT autonome -> mode terminal (RX ecran couleur OASCII + TX clavier).
;
;  E/S serie abstraites via ACIAPTR (pointeur ZP sur la base de l'ACIA 6551)  -
;    backend 1 = ACIA 6551 direct  ($031C)
;    backend 2 = LOCI / Pico W      ($0380)   (meme interface 6551, autre base)
;  (DTL2000 = V23/Minitel, hors sujet ; TLS = role du modem, voir docs.)
;
;  Cible oric1-emu. Tester ACIA $031C avec --serial modem ; LOCI $0380 avec
;  --loci --serial picowifi (SANS --acia-addr - l'ACIA va par defaut a $0380).
;  Assemblage xa. Chargement 1000. Commentaires ASCII sans deux-points.
; ---------------------------------------------------------------------------

; --- Bits de statut ACIA 6551 (offset 1 depuis la base) ---
RDRF      = $08          ; Receiver Data Register Full
TDRE      = $10          ; Transmit Data Register Empty

; --- Commande de positionnement curseur (plot X,Y) - 1F col row ---
PLOTCMD   = $1F

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
STRPTR    = $EE
TARGETLO  = $EC
TARGETHI  = $ED
ACIAPTR   = $EA          ; base de l'ACIA (2 octets)
BUFPTR    = $E8          ; cible de saisie (2 octets)
INLEN     = $E7
INMAX     = $E6
PROTO     = $E5          ; 0 = telnet/raw, 1 = TLS
PLOTST    = $E3          ; etat reception plot (0=normal 1=attend col 2=attend row)
PLOTX     = $E4          ; colonne memorisee pendant la reception du plot
SHIFTF    = $F3          ; etat SHIFT ($80 si LSHIFT/RSHIFT enfonce, 0 sinon)
; Jauge XMODEM (definie ici car utilisee par handle_rx ; details dans xmodem.s).
; Aliasees sur des cases INACTIVES pendant un transfert (page zero pleine).
XTOTAL    = $E8          ; total de blocs attendus (alias BUFPTR - saisie)
XACC      = $E3          ; accumulateur Bresenham (alias PLOTST/PLOTX - plot)
XSEG      = $E5          ; segments dessines (alias PROTO - dial)

; --- Carte zero-page (source unique ; xmodem.s/sedoric.s reutilisent SRC/DST/
;     KTMP/STRPTR HORS de leurs fenetres d'usage clavier - ne pas reallouer sans
;     verifier les collisions). Libres connus - $F3 utilise ci-dessus pour SHIFTF.
;     xmodem.s - XBUF $E0 XBLK $E2 XCRC $E6 XREM $EC XSUM $FD XSIZE $FE XSAVY $FC.

NUM_ENTRIES = 6

* = $1000

start:
        sei                      ; IRQ off pour toute la session - le terminal
                                 ; pilote lui-meme le clavier (scan matrice) et
                                 ; la serie (polling ACIA), sans la ROM. Sedoric
                                 ; (XSAVEB) re-SEI de son cote, donc OK.
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
        sta PLOTST
        ; charge la police BBS dans le charset alternatif ($B800)
        jsr load_altcharset

; ---------------------------------------------------------------------------
;  Menu modem -> choisit ACIAPTR puis initialise l'ACIA
; ---------------------------------------------------------------------------
modem_menu:
        jsr clear_screen
        jsr reset_cursor
        lda #<mm_text
        sta STRPTR
        lda #>mm_text
        sta STRPTR+1
        jsr print_string
mm_wait:
        jsr get_key
        sta LASTKEY
        cmp #'1'
        beq mm_6551
        cmp #'2'
        beq mm_loci
        jmp mm_wait
mm_6551:
        lda #$1C
        sta ACIAPTR
        lda #$03
        sta ACIAPTR+1
        jmp mm_init
mm_loci:
        lda #$80                 ; LOCI - modem WiFi expose en ACIA a $0380
        sta ACIAPTR              ; ($03A0-$03BF = espace MIA du LOCI, PAS le modem)
        lda #$03
        sta ACIAPTR+1
mm_init:
        ; ACIA 9600 8N1, DTR on, IRQ off, TX on
        lda #$1E
        ldy #3                   ; control
        sta (ACIAPTR),y
        lda #$0B
        ldy #2                   ; command
        sta (ACIAPTR),y
        jsr wait_release

; ---------------------------------------------------------------------------
;  Repertoire (phonebook) + option M (saisie manuelle)
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
        jsr get_key
        sta LASTKEY
        cmp #'M'
        beq manual_entry
        cmp #'m'
        beq manual_entry
        sec
        sbc #'1'
        bcc pb_wait
        cmp #NUM_ENTRIES
        bcs pb_wait
        tax
        ; cible = dial[X]
        lda dial_lo,x
        sta TARGETLO
        lda dial_hi,x
        sta TARGETHI
        ; prefixe selon proto_tbl[X] (0=telnet ATD, 1=TLS ATDT#)
        lda proto_tbl,x
        beq pb_telnet
        lda #<at_atdts
        sta STRPTR
        lda #>at_atdts
        sta STRPTR+1
        jsr send_string
        jmp pb_target
pb_telnet:
        jsr send_atd
pb_target:
        lda TARGETLO
        sta STRPTR
        lda TARGETHI
        sta STRPTR+1
        jsr send_string
        jsr send_cr
        jmp connecting

; ---------------------------------------------------------------------------
;  Saisie manuelle host / port / protocole
; ---------------------------------------------------------------------------
manual_entry:
        jsr wait_release
        jsr clear_screen
        jsr reset_cursor
        lda #<me_host
        sta STRPTR
        lda #>me_host
        sta STRPTR+1
        jsr print_string
        ; saisir l'hote dans hostbuf (max 40)
        lda #<hostbuf
        sta BUFPTR
        lda #>hostbuf
        sta BUFPTR+1
        lda #40
        sta INMAX
        jsr input_line

        lda #<me_port
        sta STRPTR
        lda #>me_port
        sta STRPTR+1
        jsr print_string
        lda #<portbuf
        sta BUFPTR
        lda #>portbuf
        sta BUFPTR+1
        lda #6
        sta INMAX
        jsr input_line

        ; protocole 1=telnet 2=TLS
        lda #<me_proto
        sta STRPTR
        lda #>me_proto
        sta STRPTR+1
        jsr print_string
mp_wait:
        jsr get_key
        sta LASTKEY
        cmp #'1'
        beq mp_telnet
        cmp #'2'
        beq mp_tls
        jmp mp_wait
mp_telnet:
        lda #0
        sta PROTO
        jmp mp_dial
mp_tls:
        lda #1
        sta PROTO
        lda #<me_tlsnote
        sta STRPTR
        lda #>me_tlsnote
        sta STRPTR+1
        jsr print_string
mp_dial:
        jsr wait_release
        ; prefixe de numerotation selon le protocole
        lda PROTO
        beq md_telnet
        ; TLS - ATDT#  (le modem Pico W termine le TLS, l'Oric recoit du clair)
        lda #<at_atdts
        sta STRPTR
        lda #>at_atdts
        sta STRPTR+1
        jsr send_string
        jmp md_hostport
md_telnet:
        jsr send_atd             ; telnet/raw - ATD
md_hostport:
        lda #<hostbuf
        sta STRPTR
        lda #>hostbuf
        sta STRPTR+1
        jsr send_string
        lda #$3A                 ; " -"
        jsr ser_tx
        lda #<portbuf
        sta STRPTR
        lda #>portbuf
        sta STRPTR+1
        jsr send_string
        jsr send_cr

connecting:
        lda #<msg_dial
        sta STRPTR
        lda #>msg_dial
        sta STRPTR+1
        jsr print_string

; ---------------------------------------------------------------------------
;  Mode terminal (RX ecran + TX clavier)
; ---------------------------------------------------------------------------
main:
        jsr ser_rx_ready
        beq do_keyscan
        jsr ser_rx
        jsr handle_rx
        jmp main
do_keyscan:
        jsr key_scan
        cmp #0
        beq ks_release
        cmp LASTKEY
        beq ks_ret
        sta LASTKEY
        jsr ser_tx
        jsr putbyte              ; echo local
        jmp main
ks_release:
        lda #0
        sta LASTKEY
ks_ret:
        jmp main

; ---------------------------------------------------------------------------
;  handle_rx - traite un octet recu - commande plot (1F col row) ou affichage.
;  A = octet recu.
; ---------------------------------------------------------------------------
; Etats PLOTST - 0 normal, 1 apres 1F, 2 attend ligne (plot), 3/4 attend le
; total de blocs (lo/hi) annonce par le serveur apres 1F FE (pour la jauge).
; NB la jauge alias XACC sur PLOTST/PLOTX -> on REINITIALISE PLOTST apres un
; transfert (sinon la machine a etats reste desynchronisee).
handle_rx:
        ldx PLOTST
        beq hr_normal
        cpx #1
        beq hr_state1
        cpx #2
        beq hr_row
        cpx #3
        beq hr_totlo
        sta XTOTAL+1             ; etat 4 - total blocs (hi) puis reception
        jsr xmodem_recv          ; recoit en RAM ($4000) avec barre de progression
        jsr sed_save             ; sauve sur disquette si Sedoric resident
        lda #0
        sta PLOTST               ; XACC a ecrase PLOTST -> reinit
        rts
hr_normal:
        cmp #PLOTCMD
        beq hr_begin
        jmp putbyte              ; octet normal -> affichage (putbyte fait rts)
hr_begin:
        lda #1
        sta PLOTST
        rts
hr_state1:
        cmp #$FE                 ; 1F FE = "recevoir un fichier"
        beq hr_recv
        cmp #$FD                 ; 1F FD = "envoyer un fichier"
        beq hr_send
        sta PLOTX                ; sinon 1er octet = colonne (plot)
        lda #2
        sta PLOTST
        rts
hr_recv:
        lda #3                   ; 1F FE -> attendre le total de blocs (lo, hi)
        sta PLOTST
        rts
hr_send:
        lda #0
        sta PLOTST
        jsr xmodem_send          ; envoie le buffer RAM
        lda #0
        sta PLOTST               ; XACC a ecrase PLOTST -> reinit
        rts
hr_totlo:
        sta XTOTAL               ; etat 3 - total blocs (lo)
        lda #4
        sta PLOTST
        rts
hr_row:
        jsr set_cursor_xy        ; A = ligne, PLOTX = colonne
        lda #0
        sta PLOTST
        rts

; ---------------------------------------------------------------------------
;  set_cursor_xy - SCRPTR = SCREEN + row*40 + col ; COL = col.  A = row (0-27).
; ---------------------------------------------------------------------------
set_cursor_xy:
        cmp #28                  ; clamp ligne 0..27 (coords reseau non fiables)
        bcc scxy_rowok
        lda #27
scxy_rowok:
        tax                      ; X = row (compteur)
        lda PLOTX
        cmp #40                  ; clamp colonne 0..39
        bcc scxy_colok
        lda #39
        sta PLOTX
scxy_colok:
        lda #<SCREEN
        sta SCRPTR
        lda #>SCREEN
        sta SCRPTR+1
scxy_row:
        cpx #0
        beq scxy_col
        clc
        lda SCRPTR
        adc #40
        sta SCRPTR
        lda SCRPTR+1
        adc #0
        sta SCRPTR+1
        dex
        jmp scxy_row
scxy_col:
        clc
        lda SCRPTR
        adc PLOTX
        sta SCRPTR
        lda SCRPTR+1
        adc #0
        sta SCRPTR+1
        lda PLOTX
        sta COL
        rts

; ---------------------------------------------------------------------------
;  Primitives serie (via ACIAPTR  - offset 0=data 1=status 2=cmd 3=ctrl)
; ---------------------------------------------------------------------------
ser_tx:                          ; A = octet a envoyer (A preserve)
        pha
stx_wait:
        ldy #1
        lda (ACIAPTR),y
        and #TDRE
        beq stx_wait
        pla
        ldy #0
        sta (ACIAPTR),y
        rts

ser_rx_ready:                    ; renvoie A = status & RDRF (Z=1 si rien)
        ldy #1
        lda (ACIAPTR),y
        and #RDRF
        rts

ser_rx:                          ; A = octet recu
        ldy #0
        lda (ACIAPTR),y
        rts

send_atd:                        ; envoie "ATD"
        lda #<at_atd
        sta STRPTR
        lda #>at_atd
        sta STRPTR+1
        jmp send_string          ; send_string fait rts

send_cr:                         ; envoie CR (declenche la numerotation)
        lda #$0D
        jmp ser_tx               ; ser_tx fait rts

; ---------------------------------------------------------------------------
;  input_line - lit une ligne dans (BUFPTR), max INMAX, echo, RETURN termine
; ---------------------------------------------------------------------------
input_line:
        lda #0
        sta INLEN
il_loop:
        jsr get_key
        cmp #$0D
        beq il_done
        cmp #$08                 ; BACKSPACE (touche DEL Oric)
        beq il_back
        cmp #$20
        bcc il_skip              ; ignore controle < espace
        ldx INLEN
        cpx INMAX
        bcs il_skip              ; plein
        ldy INLEN
        sta (BUFPTR),y
        inc INLEN
        jsr putbyte              ; echo (A preserve)
il_back:
        ldx INLEN
        beq il_skip              ; rien tape -> ignore
        dec INLEN
        lda #$08
        jsr putbyte             ; efface a l'ecran (putbyte gere $08)
il_skip:
        jsr wait_release
        jmp il_loop
il_done:
        ldy INLEN
        lda #0
        sta (BUFPTR),y           ; terminer la chaine
        jsr wait_release
        lda #$0D
        jsr putbyte
        lda #$0A
        jsr putbyte
        rts

; ---------------------------------------------------------------------------
;  get_key / wait_release
; ---------------------------------------------------------------------------
get_key:
        jsr key_scan
        cmp #0
        beq get_key
        rts

wait_release:
        jsr key_scan
        cmp #0
        bne wait_release
        rts

; ---------------------------------------------------------------------------
;  load_altcharset - copie la police BBS (1024 o) dans le charset alt $B800.
;  Donnees fournies par altcharset.s (label altcharset_data), concatene au build.
; ---------------------------------------------------------------------------
load_altcharset:
        lda #<altcharset_data
        sta SRC
        lda #>altcharset_data
        sta SRC+1
        lda #$00
        sta DST
        lda #$B8
        sta DST+1
        ldx #4                   ; 4 pages de 256 octets = 1024
        ldy #0
lac_copy:
        lda (SRC),y
        sta (DST),y
        iny
        bne lac_copy
        inc SRC+1
        inc DST+1
        dex
        bne lac_copy
        rts

; ---------------------------------------------------------------------------
;  reset_cursor
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
;  print_string / send_string (chaine terminee par 0 pointee par STRPTR)
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

send_string:
        ldy #0
        lda (STRPTR),y
        beq ss_done
        jsr ser_tx
        inc STRPTR
        bne send_string
        inc STRPTR+1
        jmp send_string
ss_done:
        rts

; ---------------------------------------------------------------------------
;  putbyte - affiche A (CR, LF+scroll, clamp 40 col). A preserve (chemin char).
; ---------------------------------------------------------------------------
putbyte:
        cmp #$0D
        beq pb_cr
        cmp #$0A
        beq pb_lf
        cmp #$08
        beq pb_bs
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
pb_bs:                           ; backspace destructif (recule, efface, recule)
        lda COL
        beq pb_done              ; deja en colonne 0 - rien a effacer
        dec COL
        sec
        lda SCRPTR
        sbc #1
        sta SCRPTR
        lda SCRPTR+1
        sbc #0
        sta SCRPTR+1
        lda #$20
        ldy #0
        sta (SCRPTR),y
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
;  psg_write / key_scan (clavier, inchanges)
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

key_scan:
        jsr scan_shift           ; met SHIFTF selon LSHIFT/RSHIFT
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
        cmp #$61                 ; lettre minuscule a..z ?
        bcc ksf_done
        cmp #$7B
        bcs ksf_done
        bit SHIFTF               ; SHIFT enfonce -> majuscule
        bpl ksf_done
        sec
        sbc #$20                 ; 'a'..'z' -> 'A'..'Z'
ksf_done:
        pha
        lda #$FF
        ldy #14
        jsr psg_write
        pla
        rts

; scan_shift - SHIFTF = $80 si LSHIFT (col4 row4) ou RSHIFT (col7 row4) enfonce,
; sinon 0. rowmask ligne 4 = $EF. Appele en debut de key_scan.
scan_shift:
        lda #0
        sta SHIFTF
        lda VIA_ORB              ; colonne 4 (LSHIFT)
        and #$F8
        ora #4
        sta VIA_ORB
        lda #$EF
        ldy #14
        jsr psg_write
        nop
        nop
        lda VIA_ORB
        and #$08
        bne ss_yes
        lda VIA_ORB              ; colonne 7 (RSHIFT)
        and #$F8
        ora #7
        sta VIA_ORB
        lda #$EF
        ldy #14
        jsr psg_write
        nop
        nop
        lda VIA_ORB
        and #$08
        beq ss_no
ss_yes:
        lda #$80
        sta SHIFTF
ss_no:
        lda #$FF                 ; relacher les lignes
        ldy #14
        jsr psg_write
        rts

; ---------------------------------------------------------------------------
;  Donnees
; ---------------------------------------------------------------------------
rowmask:
        .byt $FE,$FD,$FB,$F7,$EF,$DF,$BF,$7F
asciitab:
        .byt $37,$6E,$35,$76,$00,$31,$78,$33
        .byt $6A,$74,$72,$66,$00,$00,$71,$64
        .byt $6D,$36,$62,$34,$00,$7A,$32,$63
        .byt $6B,$39,$3B,$2D,$00,$00,$5C,$27
        .byt $20,$2C,$2E,$00,$00,$00,$00,$00
        .byt $75,$69,$6F,$70,$00,$08,$5D,$5B   ; ...FUNCT(44) DEL=$08(45) ] [
        .byt $79,$68,$67,$65,$00,$61,$73,$77
        .byt $38,$6C,$30,$2F,$00,$0D,$00,$3D

at_atd:
        .byt "ATD",$00
at_atdts:
        .byt "ATDT#",$00          ; dial securise TLS (picowifi v0.2.0)
msg_dial:
        .byt $0D,$0A,$02,"Numerotation en cours...",$0D,$0A,$07,$00

mm_text:
        .byt "========================================",$0D,$0A
        .byt $03,"           TYPE DE MODEM",$0D,$0A
        .byt "========================================",$0D,$0A,$0D,$0A
        .byt $06," 1  ",$07,"ACIA 6551 direct  (031C)",$0D,$0A
        .byt $06," 2  ",$07,"LOCI / Pico W     (0380)",$0D,$0A,$0D,$0A
        .byt $02,"Votre choix (1-2) > ",$07,$00

pb_text:
        .byt "========================================",$0D,$0A
        .byt $03,"          REPERTOIRE BBS ORIC",$0D,$0A
        .byt "========================================",$0D,$0A,$0D,$0A
        .byt $06," 1  ",$07,"BBS Oric (prod)  pavi.3617.fr",$0D,$0A
        .byt $06," 2  ",$07,"ParticlesBBS",$0D,$0A
        .byt $06," 3  ",$07,"Altair",$0D,$0A
        .byt $06," 4  ",$07,"Heatwave",$0D,$0A
        .byt $06," 5  ",$07,"BBS Oric TLS  pavi.3617.fr:6992",$0D,$0A
        .byt $06," 6  ",$07,"IDreamIn8Bits  bbs.idi8b.com",$0D,$0A
        .byt $06," M  ",$07,"Saisie manuelle",$0D,$0A,$0D,$0A
        .byt $02,"Choix (1-6, M) > ",$07,$00

me_host:
        .byt "========================================",$0D,$0A
        .byt $03,"           SAISIE MANUELLE",$0D,$0A
        .byt "========================================",$0D,$0A,$0D,$0A
        .byt $07,"Hote > ",$00
me_port:
        .byt $07,"Port > ",$00
me_proto:
        .byt $0D,$0A,$07,"Protocole  1=telnet  2=TLS > ",$00
me_tlsnote:
        .byt $0D,$0A,$01,"TLS (ATDT#) termine par le modem.",$0D,$0A,$07,$00

dial_lo:
        .byt <dial0,<dial1,<dial2,<dial3,<dial4,<dial5
dial_hi:
        .byt >dial0,>dial1,>dial2,>dial3,>dial4,>dial5
; protocole par entree - 0 = telnet (ATD), 1 = TLS (ATDT#)
proto_tbl:
        .byt 0,0,0,0,1,0
dial0:
        .byt "pavi.3617.fr:6502",$00
dial1:
        .byt "particlesbbs.dyndns.org:6400",$00
dial2:
        .byt "altair.virtualaltair.com:4667",$00
dial3:
        .byt "heatwave.ddns.net:9640",$00
dial4:
        .byt "pavi.3617.fr:6992",$00
dial5:
        .byt "bbs.idreamtin8bits.com:6500",$00

; Tampons de saisie
hostbuf:
        .dsb 41,0
portbuf:
        .dsb 7,0
