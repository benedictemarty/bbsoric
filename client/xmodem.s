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
        jsr xr_rx            ; numero de bloc
        sta KTMP
        jsr xr_rx            ; complement
        eor #$FF
        cmp KTMP
        bne xr_nak           ; en-tete corrompu
        lda #0
        sta XSUM
        ldy #0
xr_data:
        jsr xr_rx
        sta (XBUF),y
        clc
        adc XSUM
        sta XSUM
        iny
        cpy #128
        bne xr_data
        jsr xr_rx            ; somme de controle recue
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

; xr_rx - lit un octet (bloquant). A = octet. PRESERVE Y (ser_rx l'ecrase) car
; xr_data s'en sert comme index/compteur ; X est utilise comme tampon.
xr_rx:
        tya
        pha
xr_rx_w:
        jsr ser_rx_ready
        beq xr_rx_w
        jsr ser_rx           ; A = octet (Y ecrase)
        tax
        pla
        tay
        txa
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
        bne xs_waitc
        rts
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

msg_recu:
        .byt $0D,$0A,$02,"FICHIER RECU EN 4000",$0D,$0A,$07,$00
msg_envoye:
        .byt $0D,$0A,$02,"FICHIER ENVOYE",$0D,$0A,$07,$00
