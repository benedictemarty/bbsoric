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
XBUF  = $E0          ; pointeur buffer destination (2)
XBLK  = $E2          ; numero de bloc attendu
XSUM  = $FD          ; somme de controle calculee
XSIZE = $FE          ; taille recue en octets (2)

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

msg_recu:
        .byt $0D,$0A,$02,"FICHIER RECU EN 4000",$0D,$0A,$07,$00
