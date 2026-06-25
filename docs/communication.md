# Communication — BBS Oric

Trace des annonces et publications externes du projet.

## Annonce alpha — forum Defence Force (2026-06-25)

- **Quoi** : annonce publique de la version **alpha** de l'écosystème BBS Oric
  (serveur Go + terminal Oric 6502 + studio « Forge »).
- **Où** : forum [Defence Force](https://forum.defence-force.org/) —
  fil de discussion : <https://forum.defence-force.org/viewtopic.php?t=2897>
- **Texte source** : `~/bbsoric-announce-defence-force.md` (Markdown) et
  `~/bbsoric-announce-defence-force.bbcode.txt` (BBCode publié).
- **Vidéo de démo** : <https://youtu.be/YRFBYkpsKMc>
  (boot → numérotation → invité → Fichiers → téléchargement d'Astéroric → « FICHIER RECU »).
- **Dépôt** : rendu **public** le 2026-06-25 —
  <https://github.com/benedictemarty/bbsoric>
  (historique réécrit au préalable pour purger les IP internes, cf. `deploy/caddy-tls.md`).

### Appel à contribution (test sur matériel réel)

L'annonce sollicite des retours sur trois points non encore validés sur fer :

1. Le terminal se connecte-t-il et rend-il correctement sur matériel réel
   (timing WiFiModem réel, ACIA réelle) ?
2. Le transfert XMODEM survit-il au timing série réel ?
3. La sauvegarde Sedoric fonctionne-t-elle sur un lecteur physique
   (Microdisc/LOCI) ?

> Suivi des retours : à consigner ici au fil des réponses du forum.
