; ---------------------------------------------------------------------------
;  loci.s - sauvegarde du buffer recu ($4000, XSIZE octets) sur la carte SD du
;  LOCI via l'API MIA (Mass Interface Adapter) a $03A0.
;  Concatene apres sedoric.s. Commentaires ASCII sans deux-points (contrainte xa).
;
;  PROTOCOLE MIA (cf. locifilemanager v1, RIA/picocomputer) -
;    un appel = lda #op ; sta MIA_OP ($03AF) ; clv ; jsr MIA_SPIN ($03B1).
;    Au retour A = resultat (octet bas), X = octet haut. Resultat < 0 (X bit7) =
;    erreur. Les arguments octet passent par la pile XSTACK ($03AC, on POUSSE en
;    ecrivant), les arguments 16 bits par AREG/XREG ($03B4/$03B6). Les chaines et
;    blocs se poussent a l'ENVERS (le firmware depile dans l'ordre direct).
;
;  Detection - le LOCI expose des opcodes fixes en $03B3/$03B5/$03B7 (A9/A2/60),
;  independants de l'init ROM (notre terminal boote en cassette, sans ROM LOCI).
; ---------------------------------------------------------------------------

MIA_XSTACK = $03AC          ; pousser un octet = ecrire ici
MIA_ERRLO  = $03AD          ; code d'erreur (octet bas)
MIA_OP     = $03AF          ; code d'operation (ecrire pour invoquer)
MIA_SPIN   = $03B1          ; JSR ici = executer/attendre l'operation
MIA_AREG   = $03B4          ; argument/resultat octet bas
MIA_XREG   = $03B6          ; argument/resultat octet haut
MIA_SIG3   = $03B3          ; A9 si LOCI present
MIA_SIG5   = $03B5          ; A2
MIA_SIG7   = $03B7          ; 60

OP_OPEN    = $14
OP_CLOSE   = $15
OP_WRXSTK  = $18            ; WRITE_XSTACK - ecrit les octets depiles dans le fd
OPEN_FLAGS = $32            ; O_WRONLY|O_CREAT|O_TRUNC

; ---------------------------------------------------------------------------
;  save_received - persiste le fichier recu vers le stockage disponible -
;  Sedoric en priorite (si resident), sinon carte SD LOCI. Si aucun, le fichier
;  reste en RAM ($4000) - "FICHIER RECU EN 4000" deja affiche par xmodem_recv.
; ---------------------------------------------------------------------------
save_received:
        jsr sed_save             ; A=1 si sauve sur Sedoric, A=0 sinon
        cmp #0
        bne sr_done
        jsr loci_save            ; sinon tente la carte SD LOCI (rts si absent)
sr_done:
        rts

; ---------------------------------------------------------------------------
;  loci_present - Z=1 (et A=0) si pas de LOCI ; Z=0 si present. A/X/Y ecrases.
; ---------------------------------------------------------------------------
loci_present:
        lda MIA_SIG3
        cmp #$A9
        bne lp_no
        lda MIA_SIG5
        cmp #$A2
        bne lp_no
        lda MIA_SIG7
        cmp #$60
        bne lp_no
        lda #1                   ; present
        rts
lp_no:
        lda #0                   ; absent (Z=1)
        rts

; ---------------------------------------------------------------------------
;  Primitives LOCI reutilisables (open / write / close) - partagees par la
;  sauvegarde bufferisee (loci_save) ET la reception en STREAMING (xmodem.s,
;  I2b) qui ecrit chaque bloc XMODEM directement sur la carte SD sans tout
;  bufferiser en RAM $4000 -> plus de plafond ~30 Ko.
; ---------------------------------------------------------------------------

; loci_open - detecte le LOCI, construit le chemin depuis dlname, ouvre le
; fichier en ecriture (O_WRONLY|O_CREAT|O_TRUNC). A=1 (fd dans lcfd) si ouvert,
; A=0 sinon (LOCI absent, nom vide ou erreur d'ouverture).
loci_open:
        jsr loci_present
        bne lo_go
        rts                      ; A=0, pas de LOCI
lo_go:
        jsr loci_build_path      ; pathbuf / pathlen depuis dlname
        ldx lcpathlen
        bne lo_pathok            ; nom present -> continuer
        lda #0                   ; nom vide
        rts
lo_pathok:
        ; --- open(path, OPEN_FLAGS) --- pousser le NUL puis le chemin a l'envers.
        ; Le firmware depile en ordre direct (pop_zstring lit jusqu'au NUL) ; le
        ; NUL pousse en premier est lu en dernier -> terminateur de chaine. Requis
        ; par le vrai LOCI (l'emulateur tolere son absence, pas le firmware reel).
        lda #0
        sta MIA_XSTACK
lo_pushpath:
        dex
        lda lcpathbuf,x
        sta MIA_XSTACK
        txa
        bne lo_pushpath
        lda #0                   ; xreg (haut) = 0
        sta MIA_XREG
        lda #OPEN_FLAGS
        sta MIA_AREG
        lda #OP_OPEN
        sta MIA_OP
        clv
        jsr MIA_SPIN
        sta lcfd                 ; A=fd bas, X=fd haut
        stx lcfd+1
        txa
        bpl lo_ok                ; fd>=0 -> ok
        lda #0                   ; fd<0 -> erreur
        rts
lo_ok:
        lda #1
        rts

; loci_write_chunk - ecrit lcnb octets (1..128) depuis (SRC) dans lcfd via
; WRITE_XSTACK. A=1 si ok, A=0 sinon. PRESERVE SRC. lcnb doit etre non nul
; (garanti par appelant ; lcnb=0 pousserait 256 octets par sous-passage de Y).
loci_write_chunk:
        ldy lcnb                 ; pousser nb octets a l'envers (index decroissant)
        dey
lw_push:
        lda (SRC),y
        sta MIA_XSTACK
        dey
        bpl lw_push
        lda lcfd+1               ; ax = fd
        sta MIA_XREG
        lda lcfd
        sta MIA_AREG
        lda #OP_WRXSTK
        sta MIA_OP
        clv
        jsr MIA_SPIN
        txa
        bpl lw_ok                ; ecriture ok
        lda #0                   ; ecriture < 0 -> erreur
        rts
lw_ok:
        lda #1
        rts

; loci_close - ferme lcfd (evite la fuite de descripteur - le LOCI n'en a que 16).
loci_close:
        lda lcfd+1
        sta MIA_XREG
        lda lcfd
        sta MIA_AREG
        lda #OP_CLOSE
        sta MIA_OP
        clv
        jsr MIA_SPIN
        rts

; ---------------------------------------------------------------------------
;  loci_save - ecrit XSIZE octets de $4000 dans un fichier nomme d'apres dlname.
;  Renvoie A=1 si sauve, A=0 sinon (LOCI absent ou erreur). Dernier bloc partiel
;  tronque a la taille reelle (XSIZE, sans padding XMODEM).
; ---------------------------------------------------------------------------
loci_save:
        jsr loci_open            ; A=1 si ouvert (present + nom + open ok)
        bne lsv_open
        lda #0                   ; pas de LOCI ou echec -> non sauve
        rts
lsv_open:
        lda #$00
        sta SRC
        lda #$40
        sta SRC+1                ; SRC = $4000
        lda XSIZE
        sta lcrem
        lda XSIZE+1
        sta lcrem+1
lsv_loop:
        lda lcrem
        ora lcrem+1
        beq lsv_close            ; plus rien -> fermer
        lda lcrem+1
        bne lsv_full             ; rem >= 256 -> bloc plein
        lda lcrem
        cmp #128
        bcs lsv_full             ; rem.lo >= 128 -> bloc plein
        sta lcnb                 ; rem.lo < 128 (et > 0) -> bloc partiel
        jmp lsv_have
lsv_full:
        lda #128
        sta lcnb
lsv_have:
        jsr loci_write_chunk
        beq lsv_wfail            ; A=0 -> echec ecriture
        clc                      ; SRC += nb
        lda SRC
        adc lcnb
        sta SRC
        bcc lsv_noinc
        inc SRC+1
lsv_noinc:
        sec                      ; rem -= nb
        lda lcrem
        sbc lcnb
        sta lcrem
        bcs lsv_loop
        dec lcrem+1
        jmp lsv_loop
lsv_close:
        jsr loci_close
        lda #<msg_loci_ok
        sta STRPTR
        lda #>msg_loci_ok
        sta STRPTR+1
        jsr print_string
        lda #1                   ; sauve
        rts
lsv_wfail:
        jsr loci_close           ; fermer le fd avant d'abandonner
        lda #<msg_loci_ko
        sta STRPTR
        lda #>msg_loci_ko
        sta STRPTR+1
        jsr print_string
        lda #0
        rts

; ---------------------------------------------------------------------------
;  loci_build_path - construit "NOM.EXT" (sans NUL) dans lcpathbuf depuis dlname
;  (12 o Sedoric = 9 nom + 3 ext, completes d'espaces). lcpathlen = longueur.
; ---------------------------------------------------------------------------
loci_build_path:
        ldx #0                   ; index dlname (nom)
        ldy #0                   ; index pathbuf
lbp_name:
        lda dlname,x
        cmp #$20
        beq lbp_name_done        ; espace -> fin du nom (pas d'espace interne)
        sta lcpathbuf,y
        iny
        inx
        cpx #9
        bne lbp_name
lbp_name_done:
        lda dlname+9             ; extension presente ?
        cmp #$20
        beq lbp_fin              ; ext vide -> pas de point
        lda #$2E                 ; '.'
        sta lcpathbuf,y
        iny
        ldx #9
lbp_ext:
        lda dlname,x
        cmp #$20
        beq lbp_fin
        sta lcpathbuf,y
        iny
        inx
        cpx #12
        bne lbp_ext
lbp_fin:
        sty lcpathlen
        rts

; --- donnees ---
lcfd:
        .byt 0,0
lcrem:
        .byt 0,0
lcnb:
        .byt 0
lcpathlen:
        .byt 0
lcpathbuf:
        .dsb 16,0
msg_loci_ok:
        .byt $0D,$0A,$02,"SAUVE SUR CARTE SD",$0D,$0A,$07,$00
msg_loci_ko:
        .byt $0D,$0A,$01,"ECHEC SAUVEGARDE LOCI",$0D,$0A,$07,$00
