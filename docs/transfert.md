# Transfert de fichiers — download / upload (XMODEM)

Le BBS Oric propose une **bibliothèque de fichiers** (la « mémoire de masse »
côté serveur) d'où l'on **télécharge** et vers laquelle on **téléverse**, via le
protocole historique **XMODEM**.

> **État.** Côté **serveur** : implémenté et testé (download/upload XMODEM,
> bibliothèque sur disque). Côté **terminal Oric** : le récepteur/émetteur XMODEM
> et l'écriture sur mémoire de masse (carte SD via LOCI, Microdisc, cassette)
> restent à faire dans `client/term.s` (cf. backlog **G1**). En attendant, les
> transferts se testent avec un **client XMODEM standard** (PC : `sx`/`rx`, ou un
> émulateur de terminal supportant XMODEM).

## Activer la bibliothèque

```
bbsd ... -files /var/lib/bbsoric/files -max-upload 65536
```

- `-files <dir>` : répertoire de la bibliothèque (créé si absent). Vide = transfert
  désactivé (les applets affichent « Bibliotheque indisponible »).
- `-max-upload <octets>` : taille max d'un téléversement (défaut 64 Ko ; 0 = illimité).

Les noms de fichiers sont **validés** (nom simple, pas de `/`, `\` ni `..`) pour
empêcher toute sortie du répertoire.

## Câbler les applets dans le contenu

Deux applets sont fournis : **`download`** et **`upload`**. On les branche comme
entrées de menu (type « ▶ applet », sélectionnables dans le studio) :

```jsonc
{ "title": "FICHIERS", "entries": [
  { "key": "T", "label": "Telecharger", "applet": "download", "next": "fichiers" },
  { "key": "E", "label": "Televerser",  "applet": "upload",   "next": "fichiers" },
  { "key": "R", "label": "Retour",      "target": "__back__" }
]}
```

- **`download`** : liste les fichiers (choix par chiffre 1–9), puis **envoie** le
  fichier au client par XMODEM (le client lance une **réception**).
- **`upload`** : demande un nom, puis **reçoit** le fichier par XMODEM (le client
  lance un **envoi**) et l'enregistre dans la bibliothèque.

## Détails techniques

- **Protocole** : `internal/xmodem` — blocs de 128 octets, somme de contrôle **ou**
  CRC-16 (imposé par le récepteur via `NAK`/`C`), ré-émission sur erreur. Le
  dernier bloc est complété par `SUB` (0x1A), élagué à la réception.
- **Canal brut** : pendant un transfert, l'applet utilise `Session.Raw()` qui
  court-circuite le filtrage telnet/ligne (lecture binaire). `Session.ClearDeadline()`
  rétablit ensuite le délai d'inactivité normal.
- **Limite XMODEM** : la taille exacte n'est pas transmise (padding `SUB`) — fidèle
  pour du texte ; pour un binaire finissant réellement par 0x1A, prévoir un format
  enveloppe (YMODEM) ultérieurement.

## Reste à faire (côté Oric)

- **Mode transfert** dans `client/term.s` : suspendre l'interprétation OASCII
  (octets 0–31 / plot) et router les octets vers le moteur XMODEM.
- **XMODEM 6502** : récepteur/émetteur.
- **Stockage** : écriture/lecture sur carte SD (LOCI), Microdisc ou cassette.
- **Telnet binaire** : privilégier un canal **raw** (le serveur filtre `IAC` en
  saisie), surtout pour le téléversement.

Voir aussi : `docs/agile/backlog.md` (G1), `docs/connexion-materielle.md`.
