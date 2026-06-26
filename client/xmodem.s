; ---------------------------------------------------------------------------
;  xmodem.s - recepteur XMODEM (mode somme de controle) en RAM.
;  Concatene a term.s. Recoit un fichier dans BUFFER (4000), taille -> XSIZE.
;  Utilise ser_rx_ready / ser_rx / ser_tx (term.s) via ACIAPTR.
;  Declenche par la sequence serveur 1F FE (voir handle_rx).
;  Commentaires ASCII sans deux-points (contrainte xa).
; ---------------------------------------------------------------------------

; Octets de controle XMODEM
SOH  = $01
EOT  = $04
ACK  = $06
NAK  = $15
CAN  = $18

; Page zero (cases libres ; SRC/KTMP reutilises hors key_scan)
XBUF  = $E0          ; pointeur buffer (2)
XBLK  = $E2          ; numero de bloc
XSUM  = $FD          ; somme de controle (reception)
XSIZE = $FE          ; taille du fichier en octets (2)
XCRC  = $E6          ; CRC-16 (emission) (2)
XREM  = $EC          ; octets restant a envoyer (2)
XSAVY = $FC          ; sauvegarde de Y autour de ser_tx (= KTMP)
CRCCHR = $43         ; 'C' (demarrage CRC)

XBUFADDR = $4000     ; adresse du buffer de reception

; --- Jauge de progression (barre %). Le serveur envoie le total de blocs apres
;     1F FE (recus dans XTOTAL). On remplit une barre de BARLEN segments par un
;     comptage de Bresenham (XACC) - pas de multiplication/division 16 bits.
;     XTOTAL/XACC/XSEG sont definies dans term.s (utilisees par handle_rx).
BARLEN = 20          ; longueur de la barre (segments) ; 100% / 20 = 5% par segment
GAUGEROW = 25        ; ligne ecran de la barre (0..27)

; ---------------------------------------------------------------------------
;  xmodem_recv - recoit un fichier en RAM (somme de controle). A preserve nul.
; ---------------------------------------------------------------------------
xmodem_recv:
        lda #<XBUFADDR
        sta XBUF
        lda #>XBUFADDR
        sta XBUF+1
        lda #1
        sta XBLK
        lda #0
        sta XSIZE
        sta XSIZE+1
        sta XACC             ; init jauge (A=0)
        sta XACC+1
        sta XSEG
        jsr xr_gauge_draw    ; barre vide au demarrage (si XTOTAL non nul)
xr_start:
        lda #NAK             ; demarrer / relancer en mode somme de controle
        jsr ser_tx
xr_wait:
        jsr xr_rx_timeout    ; carry=1 et A=octet, ou carry=0 si timeout
        bcc xr_start
        cmp #SOH
        beq xr_block
        cmp #EOT
        beq xr_eot
        cmp #CAN
        beq xr_done
        jmp xr_wait
xr_block:
        jsr xr_rx_t          ; numero de bloc (avec timeout)
        bcc xr_start         ; bloc incomplet (gigue reseau) -> re-NAK rapide
        sta KTMP
        jsr xr_rx_t          ; complement
        bcc xr_start
        eor #$FF
        cmp KTMP
        bne xr_nak           ; en-tete corrompu
        lda XBUF+1           ; tampon plein ($4000..$B7FF avant l'ecran $BB80) ?
        cmp #$B8
        bcc xr_sizeok        ; place pour 128 octets -> continuer
        jmp xr_overflow      ; fichier trop gros -> annuler (protege la RAM)
xr_sizeok:
        lda #0
        sta XSUM
        ldy #0
xr_data:
        jsr xr_rx_t
        bcc xr_start         ; octet manquant -> re-NAK (le serveur renvoie le bloc)
        sta (XBUF),y
        clc
        adc XSUM
        sta XSUM
        iny
        cpy #128
        bne xr_data
        jsr xr_rx_t          ; somme de controle recue
        bcc xr_start
        cmp XSUM
        bne xr_nak
        lda KTMP
        cmp XBLK
        bne xr_dup           ; bloc deja recu -> ACK sans avancer
        clc                  ; bloc valide -> avancer le buffer
        lda XBUF
        adc #128
        sta XBUF
        bcc xr_nocarry
        inc XBUF+1
xr_nocarry:
        inc XBLK
        clc                  ; XSIZE += 128
        lda XSIZE
        adc #128
        sta XSIZE
        bcc xr_ack
        inc XSIZE+1
xr_ack:
        lda #ACK
        jsr ser_tx
        jsr xr_gauge         ; avance la barre (apres l'ACK, pendant le RTT)
        jmp xr_wait
xr_dup:
        lda #ACK
        jsr ser_tx
        jmp xr_wait
xr_nak:
        lda #NAK
        jsr ser_tx
        jmp xr_wait
xr_eot:
        lda #ACK
        jsr ser_tx
xr_done:
        lda #<msg_recu
        sta STRPTR
        lda #>msg_recu
        sta STRPTR+1
        jmp print_string     ; print_string fait rts
xr_overflow:
        lda #CAN             ; annule la transmission cote serveur
        jsr ser_tx
        jsr ser_tx
        lda #<msg_full
        sta STRPTR
        lda #>msg_full
        sta STRPTR+1
        jmp print_string     ; print_string fait rts

; xr_rx_t - lit un octet d'un bloc AVEC timeout (~1.3 s). PRESERVE Y (ser_rx
; l'ecrase) car xr_data s'en sert comme index/compteur ; X sert de tampon.
; carry=1 + A = octet ; carry=0 si timeout (bloc fige -> re-NAK rapide au lieu
; de bloquer indefiniment sur une gigue reseau).
xr_rx_t:
        tya
        pha                  ; sauve Y
        lda #0
        sta SRC
        sta SRC+1
xrt2_loop:
        jsr ser_rx_ready
        bne xrt2_got
        inc SRC
        bne xrt2_loop
        inc SRC+1
        bne xrt2_loop
        pla                  ; timeout - restaure Y, carry=0
        tay
        clc
        rts
xrt2_got:
        jsr ser_rx           ; A = octet (Y ecrase)
        tax
        pla
        tay                  ; restaure Y
        txa
        sec
        rts

; xr_rx_timeout - attend un octet avec timeout (~1.3 s). carry=1 + A, ou carry=0.
xr_rx_timeout:
        lda #0
        sta SRC
        sta SRC+1
xrt_loop:
        jsr ser_rx_ready
        bne xrt_got
        inc SRC
        bne xrt_loop
        inc SRC+1
        bne xrt_loop
        clc
        rts
xrt_got:
        jsr ser_rx
        sec
        rts

; ---------------------------------------------------------------------------
;  xmodem_send - envoie XSIZE octets depuis BUFFER (4000) en XMODEM CRC.
;  XSIZE suppose multiple de 128 (cas d'un fichier recu prealablement).
; ---------------------------------------------------------------------------
xmodem_send:
        lda #<XBUFADDR
        sta XBUF
        lda #>XBUFADDR
        sta XBUF+1
        lda #1
        sta XBLK
        lda XSIZE
        sta XREM
        lda XSIZE+1
        sta XREM+1
        ora XREM             ; rien a envoyer ?
        bne xs_init_gauge
        rts
xs_init_gauge:
        lda XSIZE            ; jauge - total blocs = XSIZE / 128
        asl                  ; C = bit7 de l'octet bas
        lda XSIZE+1
        rol                  ; A = (hi<<1) | bit7 = nb de blocs (<32 Ko -> 1 octet)
        sta XTOTAL
        lda #0
        sta XTOTAL+1
        sta XACC
        sta XACC+1
        sta XSEG
        jsr xr_gauge_draw    ; barre vide
xs_waitc:
        jsr xr_rx_timeout    ; attendre 'C' (mode CRC) du recepteur
        bcc xs_waitc
        cmp #CRCCHR
        bne xs_waitc
xs_loop:
        lda XREM
        ora XREM+1
        beq xs_eot
xs_send_block:
        lda #SOH
        jsr ser_tx
        lda XBLK
        jsr ser_tx
        lda XBLK
        eor #$FF
        jsr ser_tx
        lda #0
        sta XCRC
        sta XCRC+1
        ldy #0
xs_data:
        lda (XBUF),y
        sty XSAVY
        pha
        jsr ser_tx           ; envoie l'octet (Y ecrase)
        pla
        jsr crc_update       ; XCRC integre l'octet (A,X ecrases)
        ldy XSAVY
        iny
        cpy #128
        bne xs_data
        lda XCRC+1           ; CRC haut puis bas
        jsr ser_tx
        lda XCRC
        jsr ser_tx
        jsr xr_rx_timeout    ; attendre ACK
        bcc xs_send_block    ; timeout -> renvoyer le bloc
        cmp #ACK
        bne xs_send_block    ; NAK/autre -> renvoyer
        clc                  ; bloc accepte -> avancer
        lda XBUF
        adc #128
        sta XBUF
        bcc xs_nc
        inc XBUF+1
xs_nc:
        inc XBLK
        jsr xr_gauge         ; avance la barre (bloc envoye et acquitte)
        sec                  ; XREM -= 128
        lda XREM
        sbc #128
        sta XREM
        bcs xs_loop
        dec XREM+1
        jmp xs_loop
xs_eot:
        lda #EOT
        jsr ser_tx
        jsr xr_rx_timeout    ; attendre ACK de l'EOT
        bcc xs_eot
        lda #<msg_envoye
        sta STRPTR
        lda #>msg_envoye
        sta STRPTR+1
        jmp print_string

; crc_update - integre l'octet A dans XCRC (CRC-16 XMODEM, poly 1021).
crc_update:
        eor XCRC+1
        sta XCRC+1
        ldx #8
cu_loop:
        asl XCRC
        rol XCRC+1
        bcc cu_skip
        lda XCRC
        eor #$21
        sta XCRC
        lda XCRC+1
        eor #$10
        sta XCRC+1
cu_skip:
        dex
        bne cu_loop
        rts

; ---------------------------------------------------------------------------
;  xr_gauge - avance la barre d'un bloc (Bresenham) puis la redessine.
;  xr_gauge_draw - redessine la barre a XSEG segments (sans avancer).
;  Sans effet si XTOTAL = 0 (serveur sans annonce de taille).
; ---------------------------------------------------------------------------
xr_gauge:
        lda XTOTAL
        ora XTOTAL+1
        beq xg_skip          ; pas de total -> pas de jauge
        clc                  ; XACC += BARLEN
        lda XACC
        adc #BARLEN
        sta XACC
        bcc xg_chk
        inc XACC+1
xg_chk:
        lda XACC+1           ; XACC >= XTOTAL ?
        cmp XTOTAL+1
        bcc xr_gauge_draw    ; XACC < XTOTAL -> dessiner
        bne xg_step
        lda XACC
        cmp XTOTAL
        bcc xr_gauge_draw
xg_step:
        sec                  ; XACC -= XTOTAL
        lda XACC
        sbc XTOTAL
        sta XACC
        lda XACC+1
        sbc XTOTAL+1
        sta XACC+1
        lda XSEG             ; XSEG++ (plafonne a BARLEN)
        cmp #BARLEN
        bcs xg_chk
        inc XSEG
        jmp xg_chk
xg_skip:
        rts

xr_gauge_draw:
        lda XTOTAL
        ora XTOTAL+1
        beq xg_skip          ; pas de total -> rien
        lda #<(SCREEN+GAUGEROW*40)   ; ligne fixe -> adresse ecran constante
        sta SCRPTR
        lda #>(SCREEN+GAUGEROW*40)
        sta SCRPTR+1
        lda #0
        sta COL
        lda #'['
        jsr putbyte
        ldx #0
xg_bar:
        cpx XSEG
        bcs xg_empty         ; X >= XSEG -> segment vide
        lda #'#'
        bne xg_putc          ; '#' != 0 -> branche toujours prise
xg_empty:
        lda #'-'
xg_putc:
        jsr putbyte
        inx
        cpx #BARLEN
        bne xg_bar
        lda #']'
        jsr putbyte
        lda #$20
        jsr putbyte
        lda XSEG             ; pourcentage = XSEG * 5
        asl
        asl
        clc
        adc XSEG
        jsr print_dec_byte
        lda #'%'
        jmp putbyte          ; putbyte fait rts

; print_dec_byte - affiche A (0..100) en decimal, largeur 3, espaces de tete.
; X sert d'indicateur "un chiffre significatif a deja ete affiche".
print_dec_byte:
        ldx #0
        ldy #'0'-1           ; centaines
pdc_h:
        iny
        sec
        sbc #100
        bcs pdc_h
        adc #100
        sta XSUM             ; reste 0..99 (XSUM libre hors data-loop)
        cpy #'0'
        beq pdc_hsp
        tya
        jsr putbyte
        ldx #1               ; centaine affichee
        jmp pdc_t0
pdc_hsp:
        lda #$20             ; centaine 0 -> espace
        jsr putbyte
pdc_t0:
        lda XSUM             ; dizaines
        ldy #'0'-1
pdc_t:
        iny
        sec
        sbc #10
        bcs pdc_t
        adc #10
        sta XSUM             ; unites
        cpy #'0'
        bne pdc_tp           ; dizaine non nulle -> chiffre
        cpx #1               ; centaine deja affichee -> chiffre 0
        beq pdc_tp
        lda #$20             ; sinon espace de tete
        jsr putbyte
        jmp pdc_u
pdc_tp:
        tya
        jsr putbyte
pdc_u:
        lda XSUM
        ora #'0'
        jmp putbyte          ; putbyte fait rts

msg_recu:
        .byt $0D,$0A,$02,"FICHIER RECU EN 4000",$0D,$0A,$07,$00
msg_full:
        .byt $0D,$0A,$01,"FICHIER TROP GROS - ANNULE",$0D,$0A,$07,$00
msg_envoye:
        .byt $0D,$0A,$02,"FICHIER ENVOYE",$0D,$0A,$07,$00
