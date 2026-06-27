; ===========================================================================
;  hires.s - interpreteur HIRES du terminal (mode graphique 240x200)
; ---------------------------------------------------------------------------
;  Concatene a term.s par build.sh. Alimente par handle_rx  - la sous-commande
;  serie 1F FC ouvre un FLUX de commandes HIRES (cf. internal/oascii/hires.go).
;  Chaque octet recu pendant l'etat PLOTST=8 est passe a hires_feed, une machine
;  a etats qui execute les opcodes (HiOn/HiInk/ a ./HiBlit) jusqu'a HiEnd.
;
;  Modele ecran (verifie sur oric1-emu src/video/video.c)  -
;   - mode pose par un attribut seriel 0x18 ou mode ; HIRES = 0x1E (vid_mode et 4),
;     persistant ; on l'ecrit a SCREEN[0] ($BB80) pour amorcer la bascule.
;   - lignes 0-199 lues en $A000 + y*40 (40 octets/ligne) ; 6 pixels/octet.
;   - octet pixel  - bit6=1 (sinon collision attribut), bits5 a 0 = 6 pixels
;     (bit5 = pixel de gauche), bit7 = inverse. Ecran vide = $40.
;   - setpixel(x,y)  - $A000 + y*40 + x/6 , OR (1  decale  (5 - x mod 6)).
;
;  ZP empruntee (libre hors XMODEM/saisie, donc OK pendant handle_rx)  -
;   HPTR=SRC ($F4) pointeur pixel ; HDST=DST ($F6) pointeur blit.
; ===========================================================================

HVRAM     = $A000          ; base de la VRAM HIRES
HPTR      = SRC            ; $F4/$F5 - pointeur pixel
HDST      = DST            ; $F6/$F7 - pointeur destination blit

; --- Variables HIRES (bloc RAM dedie) ---
hpenx:    .byt 0           ; crayon (pen)
hpeny:    .byt 0
hx0:      .byt 0           ; segment courant
hy0:      .byt 0
hx1:      .byt 0
hy1:      .byt 0
hdx:      .byt 0           ;  ou dx ou ,  ou dy ou  (Bresenham)
hdy:      .byt 0
hsx:      .byt 0           ; signe x/y (+1 = $01, -1 = $FF)
hsy:      .byt 0
herr:     .byt 0           ; erreur (signee, 8 bits suffit  - dimensions  inf  240)
htmp:     .byt 0
hmask:    .byt 0
hcount:   .byt 0
hstate:   .byt 0           ; sous-etat du flux (0 = attend opcode)
hop:      .byt 0           ; opcode en cours d'arguments
har:       .byt 0          ; rayon (circle)
; blit
hblo:     .byt 0
hbhi:     .byt 0
hllo:     .byt 0
hlhi:     .byt 0
hrun:     .byt 0           ; octets restants de la run RLE courante
hval:     .byt 0           ; valeur de la run RLE courante
hink:     .byt 7           ; encre courante (0-7), blanc par defaut
hmlo:     .byt 0           ; scratch 16 bits (multiplication Y*40)
hmhi:     .byt 0
herrhi:   .byt 0           ; octet haut de l'erreur Bresenham (16 bits signee)
hch:      .byt 0           ; code du caractere a tracer (op char)
hcharrow: .byt 0           ; ligne (0-7) du glyphe en cours
hcharcol: .byt 0           ; colonne (0-5) du glyphe en cours
hcharbits: .byt 0          ; octet de police de la ligne courante
hcsaved:  .byt 0           ; 1 = charset deja sauvegarde en $9800

; ---------------------------------------------------------------------------
;  hires_feed - consomme un octet (A) du flux HIRES selon hstate.
;  Appele depuis handle_rx (etat PLOTST=8). Termine le flux (PLOTST=0) sur HiEnd.
; ---------------------------------------------------------------------------
hires_feed:
        ldx hstate
        bne hfd_n0
        jmp hf_opcode            ; etat 0 = attend un opcode
hfd_n0:
        cpx #1
        bne hfd_n1
        jmp hf_ink
hfd_n1:
        cpx #2
        bne hfd_n2
        jmp hf_paper
hfd_n2:
        cpx #3
        bne hfd_n3
        jmp hf_arg_x
hfd_n3:
        cpx #4
        bne hfd_n4
        jmp hf_arg_y
hfd_n4:
        cpx #5
        bne hfd_n5
        jmp hf_radius
hfd_n5:
        cpx #6
        bne hfd_n6
        jmp hf_blit
hfd_n6:
        cpx #7
        bne hfd_n7
        jmp hf_chx
hfd_n7:
        cpx #8
        bne hfd_n8
        jmp hf_chy
hfd_n8:
        jmp hf_chc               ; etat 9 = char ch (3e argument)

; --- attend un opcode ---
hf_opcode:
        cmp #$00                 ; HiEnd
        bne hfo_n0
        lda #0
        sta PLOTST               ; fin du flux  vers  retour traitement normal
        sta hstate
        rts
hfo_n0:
        cmp #$01                 ; HiOn
        bne hfo_n1
        jsr hires_on
        rts
hfo_n1:
        cmp #$02                 ; HiInk
        bne hfo_n2
        lda #1
        sta hstate
        rts
hfo_n2:
        cmp #$03                 ; HiPaper
        bne hfo_n3
        lda #2
        sta hstate
        rts
hfo_n3:
        cmp #$15                 ; HiCircle
        bne hfo_n4
        lda #5
        sta hstate
        rts
hfo_n4:
        cmp #$16                 ; HiChar
        bne hfo_n5
        lda #7
        sta hstate
        rts
hfo_n5:
        cmp #$20                 ; HiBlit
        bne hfo_n6
        lda #0
        sta hcount               ; compteur d'octets d'entete (off/len)  vers  0
        lda #6
        sta hstate
        rts
hfo_n6:
        ; opcodes a 2 args (x,y)  - Curset/Point/Line/Box/FillBox (0x10 a 0x14)
        cmp #$10
        bcc hf_ignore            ;  inf  0x10 et non gere  vers  ignore
        cmp #$15
        bcs hf_ignore            ;  sup  0x15 deja traite
        sta hop                  ; memorise l'opcode
        lda #3
        sta hstate               ; attend x
        rts
hf_ignore:
        rts                      ; opcode inconnu  - ignore (reste etat 0)

; --- HiInk / HiPaper  - 1 argument couleur ---
hf_ink:
        and #7
        sta hink
        jmp hf_reset_state
hf_paper:
        ; PAPER non gere finement (HIRES monochrome pour l'instant)  - ignore
        jmp hf_reset_state

; --- opcodes a coordonnees  - x puis y ---
hf_arg_x:
        sta hx1                  ; x cible
        lda #4
        sta hstate               ; attend y
        rts
hf_arg_y:
        sta hy1                  ; y cible
        jsr hires_exec_xy        ; execute selon hop
        jmp hf_reset_state

; --- HiCircle  - 1 argument rayon ---
hf_radius:
        sta har
        jsr hires_circle
        jmp hf_reset_state

; --- HiChar  - x (etat 7), y (etat 8), ch (etat 9) puis trace ---
hf_chx:
        sta hx1                  ; x du caractere
        lda #8
        sta hstate
        rts
hf_chy:
        sta hy1                  ; y du caractere
        lda #9
        sta hstate
        rts
hf_chc:
        sta hch                  ; code caractere
        jsr hires_char
        jmp hf_reset_state

; --- HiBlit  - entete (off_lo,off_hi,len_lo,len_hi) puis flux RLE ---
hf_blit:
        ldx hcount
        cpx #0
        bne hfb_n0
        sta hblo
        inc hcount
        rts
hfb_n0:
        cpx #1
        bne hfb_n1
        sta hbhi
        inc hcount
        rts
hfb_n1:
        cpx #2
        bne hfb_n2
        sta hllo
        inc hcount
        rts
hfb_n2:
        cpx #3
        bne hfb_n3
        sta hlhi
        ; HDST = HVRAM + offset
        clc
        lda #<HVRAM
        adc hblo
        sta HDST
        lda #>HVRAM
        adc hbhi
        sta HDST+1
        lda #0
        sta hrun                 ; pas de run en cours
        inc hcount               ; etat 4 = compteur RLE (attend un 'count')
        rts
hfb_n3:
        ; corps RLE  - alternance (count, value). A = octet recu (NE PAS l'ecraser).
        ; hrun=0  vers  A est un 'count' ; hrun!=0  vers  A est une 'value'.
        ldx hrun                 ; teste hrun SANS toucher A
        bne hfb_value
        ; A = count
        cmp #0
        beq hfb_done_check       ; count 0 = anormal  vers  verifie fin
        sta hrun
        rts
hfb_value:
        ; A = value  - ecrire 'hrun' fois a HDST, decrementer len
        sta hval
hfb_loop:
        ; len atteint 0 ?
        lda hllo
        ora hlhi
        beq hfb_finish
        lda hval
        ldy #0
        sta (HDST),y
        inc HDST
        bne hfb_nohi
        inc HDST+1
hfb_nohi:
        ; len--
        lda hllo
        bne hfb_declo
        dec hlhi
hfb_declo:
        dec hllo
        dec hrun
        bne hfb_loop
        rts                      ; run finie  vers  prochain octet = count
hfb_done_check:
hfb_finish:
        ; longueur epuisee  vers  fin du blit, retour attente opcode
        jmp hf_reset_state

hf_reset_state:
        lda #0
        sta hstate
        rts

; ---------------------------------------------------------------------------
;  hires_on - bascule en HIRES (attribut 0x1E a SCREEN[0]) et efface $A000 (0x40)
; ---------------------------------------------------------------------------
hires_on:
        ; Sauve UNE FOIS le charset $B400 vers $9800 (hors bitmap, jamais efface),
        ; tant que $B400 est encore valide (l'effacement VRAM ci-dessous le recouvre).
        ; hires_char lit ensuite la police depuis $9800.
        lda hcsaved
        bne hon_clr
        lda #$00
        sta HPTR
        lda #$B4
        sta HPTR+1               ; src $B400
        lda #$00
        sta HDST
        lda #$98
        sta HDST+1               ; dst $9800
        ldx #4                   ; 4 pages (1024 octets)
        ldy #0
hon_cpy:
        lda (HPTR),y
        sta (HDST),y
        iny
        bne hon_cpy
        inc HPTR+1
        inc HDST+1
        dex
        bne hon_cpy
        lda #1
        sta hcsaved
hon_clr:
        ; efface la VRAM HIRES  - 8000 octets a $40 (pixel vide, bit6 pose)
        lda #<HVRAM
        sta HPTR
        lda #>HVRAM
        sta HPTR+1
        ldx #>8000               ; 8000 = $1F40  vers  31 pages + 64 octets
        ldy #0
        lda #$40
hon_page:
        sta (HPTR),y
        iny
        bne hon_page
        inc HPTR+1
        dex
        bne hon_page
        ; reste $40 octets (la derniere page partielle)
        ldy #0
hon_tail:
        cpy #$40
        beq hon_mode
        sta (HPTR),y
        iny
        bne hon_tail
hon_mode:
        ; efface les 3 lignes texte du bas ($BF40-$BFDF, hors bitmap HIRES) en
        ; espaces, sinon le texte residuel de l'ecran precedent y reste affiche.
        lda #$40
        sta HPTR
        lda #$BF
        sta HPTR+1
        ldy #0
        lda #$20
hon_btm:
        sta (HPTR),y
        iny
        cpy #$A0                 ; $BF40 + $A0 = $BFE0 (fin ecran texte)
        bne hon_btm
        ; amorce la bascule  - attribut video HIRES (0x1E) en haut de l'ecran texte
        lda #$1E
        sta SCREEN               ; $BB80  - decode au balayage  vers  vid_mode HIRES
        ; crayon a l'origine
        lda #0
        sta hpenx
        sta hpeny
        lda #7
        sta hink                 ; encre par defaut = blanc (non utilise encore)
        rts

; ---------------------------------------------------------------------------
;  hires_exec_xy - execute l'opcode hop avec (hx1,hy1) cible et le crayon source.
; ---------------------------------------------------------------------------
hires_exec_xy:
        lda hop
        cmp #$10                 ; Curset  - deplace le crayon, ne trace pas
        bne hex_n10
        lda hx1
        sta hpenx
        lda hy1
        sta hpeny
        rts
hex_n10:
        cmp #$11                 ; Point  - pixel a (hx1,hy1)
        bne hex_n11
        lda hx1
        sta hpenx
        lda hy1
        sta hpeny
        ldx hx1
        ldy hy1
        jmp setpixel_xy
hex_n11:
        cmp #$12                 ; Line  - crayon  vers  (hx1,hy1)
        bne hex_n12
        jsr hires_line
        lda hx1
        sta hpenx
        lda hy1
        sta hpeny
        rts
hex_n12:
        cmp #$13                 ; Box  - rectangle vide crayon a (hx1,hy1)
        bne hex_n13
        jmp hires_box
hex_n13:
        ; FillBox (0x14)  - rectangle plein
        jmp hires_fillbox

; ---------------------------------------------------------------------------
;  setpixel_xy - allume le pixel (X=col 0-239, Y=row 0-199).
;  Calcule HPTR = $A000 + Y*40 + X/6 puis OR le bit (1  decale  (5 - X mod 6)).
;  Clampe hors champ (securite reseau).
; ---------------------------------------------------------------------------
setpixel_xy:
        cpy #200
        bcs sp_ret               ; y  sup  200  vers  ignore
        cpx #240
        bcs sp_ret               ; x  sup  240  vers  ignore
        ; (hmlo,hmhi) = Y*8 puis HPTR = Y*8 ; (hmlo,hmhi) = Y*32 ; HPTR += -> Y*40
        lda #0
        sta hmhi
        tya
        asl                      ; Y*2
        rol hmhi
        asl                      ; Y*4
        rol hmhi
        asl                      ; Y*8
        rol hmhi
        sta hmlo                 ; (hmlo,hmhi) = Y*8
        sta HPTR
        lda hmhi
        sta HPTR+1               ; HPTR = Y*8
        asl hmlo                 ; Y*16
        rol hmhi
        asl hmlo                 ; Y*32
        rol hmhi
        clc                      ; HPTR = Y*8 + Y*32 = Y*40
        lda HPTR
        adc hmlo
        sta HPTR
        lda HPTR+1
        adc hmhi
        sta HPTR+1
        ; + base $A000
        lda HPTR+1
        clc
        adc #>HVRAM
        sta HPTR+1
        ; + X/6 (octet dans la ligne) ; reste = X mod 6 dans htmp
        txa
        jsr div6                 ; A = X/6, htmp = X mod 6
        clc
        adc HPTR
        sta HPTR
        bcc sp_nocar2
        inc HPTR+1
sp_nocar2:
        ; bit = 1  decale  (5 - (X mod 6))
        lda #5
        sec
        sbc htmp                 ; A = 5 - reste
        tax
        lda #1
sp_shift:
        cpx #0
        beq sp_set
        asl
        dex
        jmp sp_shift
sp_set:
        ; OR le bit dans l'octet (en preservant bit6 deja pose par CLS)
        ldy #0
        ora (HPTR),y
        sta (HPTR),y
sp_ret:
        rts

; div6  - A = quotient A/6, reste dans htmp. A  inf  240 (max 39 quotient).
div6:
        ldx #0
        stx htmp
d6_loop:
        cmp #6
        bcc d6_done
        sbc #6
        inx
        jmp d6_loop
d6_done:
        sta htmp                 ; reste
        txa                      ; quotient
        rts

; ---------------------------------------------------------------------------
;  hires_line - segment de (hpenx,hpeny) a (hx1,hy1) (Bresenham,  ou pente ou  qcq).
; ---------------------------------------------------------------------------
hires_line:
        ; copie points de travail
        lda hpenx
        sta hx0
        lda hpeny
        sta hy0
        ; dx =  ou x1-x0 ou , sx = signe
        lda hx1
        sec
        sbc hx0
        bcs hl_dxpos
        eor #$FF
        clc
        adc #1                   ; abs
        sta hdx
        lda #$FF
        sta hsx
        jmp hl_dy
hl_dxpos:
        sta hdx
        lda #$01
        sta hsx
hl_dy:
        lda hy1
        sec
        sbc hy0
        bcs hl_dypos
        eor #$FF
        clc
        adc #1
        sta hdy
        lda #$FF
        sta hsy
        jmp hl_err
hl_dypos:
        sta hdy
        lda #$01
        sta hsy
hl_err:
        ; Choix de l'axe majeur (erreur 16 bits signee, pas de debordement).
        lda hdx
        cmp hdy
        bcs hl_xmajor            ; dx >= dy -> x-major
        jmp hl_ymajor
; x-major, un pixel par colonne
hl_xmajor:
        lda hdx
        lsr                      ; err = dx/2 (positif)
        sta herr
        lda #0
        sta herrhi
hlx_loop:
        ldx hx0
        ldy hy0
        jsr setpixel_xy
        lda hx0
        cmp hx1
        bne hlx_step             ; pas fini -> continue
        rts                      ; x0 == x1 -> fini
hlx_step:
        lda hx0
        clc
        adc hsx
        sta hx0                  ; x0 += sx
        lda herr                 ; err -= dy (16 bits)
        sec
        sbc hdy
        sta herr
        lda herrhi
        sbc #0
        sta herrhi
        bpl hlx_loop             ; err >= 0 -> pas de pas en y
        lda hy0                  ; y0 += sy
        clc
        adc hsy
        sta hy0
        lda herr                 ; err += dx
        clc
        adc hdx
        sta herr
        lda herrhi
        adc #0
        sta herrhi
        jmp hlx_loop
; y-major, un pixel par ligne
hl_ymajor:
        lda hdy
        lsr                      ; err = dy/2
        sta herr
        lda #0
        sta herrhi
hly_loop:
        ldx hx0
        ldy hy0
        jsr setpixel_xy
        lda hy0
        cmp hy1
        bne hly_step
        rts                      ; y0 == y1 -> fini
hly_step:
        lda hy0
        clc
        adc hsy
        sta hy0                  ; y0 += sy
        lda herr                 ; err -= dx
        sec
        sbc hdx
        sta herr
        lda herrhi
        sbc #0
        sta herrhi
        bpl hly_loop
        lda hx0                  ; x0 += sx
        clc
        adc hsx
        sta hx0
        lda herr                 ; err += dy
        clc
        adc hdy
        sta herr
        lda herrhi
        adc #0
        sta herrhi
        jmp hly_loop

; ---------------------------------------------------------------------------
;  hires_box - rectangle vide entre le crayon et (hx1,hy1) (4 segments).
; ---------------------------------------------------------------------------
hires_box:
        ; Coins  A=(ax,ay)=(penx,peny)  C=(cx,cy)=(hx1,hy1). seg_pen_to ECRASE
        ; hx1/hy1, donc on sauve d'abord LES DEUX coins dans des vars dediees.
        lda hpenx
        sta hx0bk                ; ax
        lda hpeny
        sta hy0bk                ; ay
        lda hx1
        sta bx1bk                ; cx
        lda hy1
        sta by1bk                ; cy
        ; AB  (ax,ay) vers (cx,ay)
        lda hx0bk
        sta hpenx
        lda hy0bk
        sta hpeny
        lda bx1bk
        sta bxx
        lda hy0bk
        sta bxy
        jsr seg_pen_to
        ; BC  (cx,ay) vers (cx,cy)
        lda bx1bk
        sta hpenx
        lda hy0bk
        sta hpeny
        lda bx1bk
        sta bxx
        lda by1bk
        sta bxy
        jsr seg_pen_to
        ; CD  (cx,cy) vers (ax,cy)
        lda bx1bk
        sta hpenx
        lda by1bk
        sta hpeny
        lda hx0bk
        sta bxx
        lda by1bk
        sta bxy
        jsr seg_pen_to
        ; DA  (ax,cy) vers (ax,ay)
        lda hx0bk
        sta hpenx
        lda by1bk
        sta hpeny
        lda hx0bk
        sta bxx
        lda hy0bk
        sta bxy
        jsr seg_pen_to
        ; crayon final = coin oppose C
        lda bx1bk
        sta hpenx
        lda by1bk
        sta hpeny
        rts

hx0bk:    .byt 0
hy0bk:    .byt 0
bx1bk:    .byt 0
by1bk:    .byt 0
bxx:      .byt 0
bxy:      .byt 0

; seg_pen_to  - trace de (hpenx,hpeny) a (bxx,bxy) sans changer le crayon appelant
seg_pen_to:
        lda bxx
        sta hx1
        lda bxy
        sta hy1
        jmp hires_line

; ---------------------------------------------------------------------------
;  hires_fillbox - rectangle plein  - lignes horizontales de peny a y1.
; ---------------------------------------------------------------------------
hires_fillbox:
        ; normalise y  - ymin a ymax
        lda hpeny
        sta hy0
        lda hy1
        sta hy1
        ; boucle de hpeny vers hy1 inclus, une hires_line horizontale par ligne
        ; crayon x reste penx ; cible x1 reste hx1 ; on fait varier y0 vers hy1
        lda hpeny
        sta fbcur
fb_loop:
        ; ligne horizontale  - (penx,fbcur) vers (x1,fbcur)
        lda hpenx
        sta hx0
        lda fbcur
        sta hy0
        ; trace via setpixel direct (horizontale) pour la vitesse/robustesse
        jsr fb_hline
        ; fin ?
        lda fbcur
        cmp hy1
        beq fb_done
        ; avance vers hy1
        lda hpeny
        cmp hy1
        bcc fb_inc               ; peny  inf  y1  vers  incremente
        dec fbcur
        jmp fb_loop
fb_inc:
        inc fbcur
        jmp fb_loop
fb_done:
        lda hx1
        sta hpenx
        lda hy1
        sta hpeny
        rts

fbcur:    .byt 0

; fb_hline  - ligne horizontale (hx0,hy0) vers (hx1,hy0)
fb_hline:
        ; x de min(hx0,hx1) a max
        lda hx0
        cmp hx1
        bcc fbh_ok
        ; swap
        ldx hx1
        lda hx0
        sta hx1
        stx hx0
fbh_ok:
        lda hx0
        sta fbx
fbh_loop:
        ldx fbx
        ldy hy0
        jsr setpixel_xy
        lda fbx
        cmp hx1
        beq fbh_end
        inc fbx
        jmp fbh_loop
fbh_end:
        rts

fbx:      .byt 0

; ---------------------------------------------------------------------------
;  hires_circle - cercle de rayon har autour du crayon (midpoint, 8 octants).
; ---------------------------------------------------------------------------
hires_circle:
        lda har
        beq hc_ret               ; rayon 0  vers  rien
        ; cx=hpenx, cy=hpeny ; x=har, y=0 ; err = 1 - har
        lda har
        sta hcx
        lda #0
        sta hcy
        lda #1
        sec
        sbc har
        sta herr                 ; err = 1 - r (signe)
hc_loop:
        jsr circ_points          ; 8 symetriques de (hcx,hcy)
        ; y++
        inc hcy
        ; if err  inf = 0  - err += 2*y+1
        lda herr
        bmi hc_erradd
        beq hc_erradd
        ; err  sup  0  - x-- ; err += 2*(y-x)+1
        dec hcx
        ; err += 2*y - 2*x + 1
        lda hcy
        asl
        sec
        sbc hcx
        sbc hcx
        ; A = 2y - 2x ; +1
        clc
        adc #1
        clc
        adc herr
        sta herr
        jmp hc_test
hc_erradd:
        lda hcy
        asl
        clc
        adc #1
        clc
        adc herr
        sta herr
hc_test:
        ; continue tant que hcx  sup  hcy
        lda hcx
        cmp hcy
        bcs hc_loop
hc_ret:
        rts

hcx:      .byt 0
hcy:      .byt 0

; circ_points  - trace les 8 points symetriques (centre hpenx,hpeny ; off hcx,hcy)
circ_points:
        ; (cx+x, cy+y)
        lda hpenx
        clc
        adc hcx
        tax
        lda hpeny
        clc
        adc hcy
        tay
        jsr setpixel_xy
        ; (cx-x, cy+y)
        lda hpenx
        sec
        sbc hcx
        tax
        lda hpeny
        clc
        adc hcy
        tay
        jsr setpixel_xy
        ; (cx+x, cy-y)
        lda hpenx
        clc
        adc hcx
        tax
        lda hpeny
        sec
        sbc hcy
        tay
        jsr setpixel_xy
        ; (cx-x, cy-y)
        lda hpenx
        sec
        sbc hcx
        tax
        lda hpeny
        sec
        sbc hcy
        tay
        jsr setpixel_xy
        ; (cx+y, cy+x)
        lda hpenx
        clc
        adc hcy
        tax
        lda hpeny
        clc
        adc hcx
        tay
        jsr setpixel_xy
        ; (cx-y, cy+x)
        lda hpenx
        sec
        sbc hcy
        tax
        lda hpeny
        clc
        adc hcx
        tay
        jsr setpixel_xy
        ; (cx+y, cy-x)
        lda hpenx
        clc
        adc hcy
        tax
        lda hpeny
        sec
        sbc hcx
        tay
        jsr setpixel_xy
        ; (cx-y, cy-x)
        lda hpenx
        sec
        sbc hcy
        tax
        lda hpeny
        sec
        sbc hcx
        tay
        jmp setpixel_xy
hc_done2:
        rts

; ---------------------------------------------------------------------------
;  hires_char - trace le glyphe hch (6x8) en (hx1,hy1), lu dans le charset ROM
;  $B400 (96 caracteres a partir de $20, 8 octets chacun, bits 5..0 = pixels).
; ---------------------------------------------------------------------------
hires_char:
        ; HDST = $9800 + (hch et $7F)*8  (charset indexe par le code complet)
        lda hch
        and #$7F
        sta hmlo
        lda #0
        sta hmhi
        asl hmlo
        rol hmhi
        asl hmlo
        rol hmhi
        asl hmlo
        rol hmhi                 ; (hch-$20)*8 dans (hmlo,hmhi)
        clc
        lda #$00                 ; bas de $9800 (copie du charset)
        adc hmlo
        sta HDST
        lda #$98                 ; haut de $9800
        adc hmhi
        sta HDST+1
        lda #0
        sta hcharrow
hch_row:
        ldy hcharrow
        lda (HDST),y             ; octet de police de la ligne
        sta hcharbits
        lda #0
        sta hcharcol
        lda #$20                 ; masque = bit5 (pixel de gauche)
        sta hmask
hch_col:
        lda hcharbits
        and hmask
        beq hch_skip             ; bit eteint  vers  pas de pixel
        clc                      ; setpixel(hx1+col, hy1+row)
        lda hx1
        adc hcharcol
        tax
        clc
        lda hy1
        adc hcharrow
        tay
        jsr setpixel_xy
hch_skip:
        lsr hmask                ; masque vers la droite (bit suivant)
        inc hcharcol
        lda hcharcol
        cmp #6
        bne hch_col
        inc hcharrow
        lda hcharrow
        cmp #8
        bne hch_row
        rts
