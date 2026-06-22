# ADR-0002 — Modèle de saisie : terminal en mode caractère, ReadKey + ReadLine

- **Statut** : Accepté
- **Date** : 2026-06-22
- **Sprint / Backlog** : Sprint 2 — item **C4** (login) et confort de navigation
- **Décideurs** : bmarty
- **Lié à** : ADR-0001 (login)

## Contexte

Les BBS historiques « snappy » réagissent à une **seule frappe sans RETURN** pour la
navigation dans les menus (cf. `readKey` de `petscii-bbs`), tout en exigeant une **ligne
terminée par RETURN** pour les champs de saisie (pseudo, mot de passe, futurs messages).

Aujourd'hui le terminal Oric (`oric-client/term.s`) **bufferise une ligne et ne l'émet
qu'au RETURN** (CR), et le serveur ne propose que `Session.ReadLine`. Conséquence : appuyer
sur `1` dans un menu ne fait rien tant qu'on n'a pas tapé RETURN — pas le *feel* BBS.

On veut **les deux comportements**, choisis selon le contexte :
- **menus / « appuyez sur une touche »** → réaction immédiate à une frappe ;
- **champs texte** (pseudo, mot de passe) → saisie multi-caractères validée par RETURN.

## Décision

### 1. Le terminal Oric est déjà en *mode caractère* (vérifié — aucune modif requise)

**Constat (2026-06-22)** : à la relecture, `oric-client/term.s` **émet déjà chaque frappe
immédiatement**. La boucle terminal `main` fait `key_scan` → (si touche nouvelle) `ser_tx`
de l'octet → écho local `putbyte`, **sans tampon de ligne** ; le buffer `input_line` ne
sert qu'à la saisie manuelle host/port (avant connexion), pas à la session BBS. Le terminal
se comporte donc comme un terminal série classique en mode caractère, et conserve l'écho
local. **Aucune modification de `term.s` n'est nécessaire** pour `ReadKey`/`ReadLine` côté
serveur (l'hypothèse initiale d'un terminal bufferisé était erronée).

### 2. Le serveur expose deux primitives

- **`ReadKey() (byte, error)`** *(nouveau)* — lit **un octet** significatif : filtre les
  séquences telnet IAC, ignore les `CR`/`LF`/`NUL` résiduels, renvoie la première vraie
  touche. Utilisé pour les **choix de menu** et les écrans « appuyez sur une touche ».
- **`ReadLine() (string, error)`** *(existant)* — **accumule** les octets jusqu'au CR.
  Utilisé pour les **champs texte** (pseudo, mot de passe). Lisant déjà octet par octet, il
  fonctionne sans modification que le client émette en rafale ou caractère par caractère.

### 3. Qui utilise quoi

| Écran | Primitive |
|-------|-----------|
| Menu (choix d'une entrée) | `ReadKey` |
| Page de contenu (« une touche pour revenir ») | `ReadKey` |
| Composant login/inscription (pseudo, mot de passe) | `ReadLine` |

## Conséquences

**Positives**
- Navigation réactive façon BBS (une frappe = une action), saisie texte robuste par ligne.
- `ReadLine` inchangé fonctionne avec le terminal en mode caractère (lecture octet/octet).
- Séparation nette : la couche `server` fournit les primitives, l'`engine`/les composants
  choisissent selon le contexte.

**Négatives / à surveiller**
- ~~`term.s` doit être modifié~~ → **non** : le terminal est déjà en mode caractère (cf.
  Décision 1). La validation **bout-en-bout dans l'émulateur** du nouvel écran de login
  reste à faire : le backend modem émulé compose les noms d'hôtes réels du répertoire et la
  synchro `--type-keys` est fragile → prévoir une entrée locale dans la config picowifi ou
  un test sur matériel réel. Le serveur est validé via `nc` + tests d'intégration.
- Avec un client « bête » (`nc`) qui envoie `1\r\n`, `ReadKey` consomme `1` et laisse
  `\r\n` ; le `ReadKey` suivant ignore ces `CR`/`LF` résiduels (d'où le skip explicite).
  Le vrai terminal en mode caractère n'émet pas de CR après une touche de menu.
- L'écho local affiche le mot de passe (déjà acté en ADR-0001, TLS couvre le transport).

## Alternatives écartées

1. **Tout en ligne + RETURN** (statu quo) : simple mais navigation lourde, non conforme au
   *feel* BBS demandé.
2. **Tout en touche unique** : impossible pour les champs texte multi-caractères (pseudo,
   mot de passe).
3. **Négociation telnet (mode caractère via IAC)** pour piloter le mode à distance : le
   terminal Oric maison n'implémente pas la négociation ; on choisit un terminal qui émet
   en mode caractère par construction, plus simple.
