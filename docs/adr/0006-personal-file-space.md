# ADR-0006 — Espace fichiers personnels (« Fichiers » = espace privé par utilisateur)

- **Status**: Accepted
- **Date**: 2026-09-02
- **Sprint / Backlog**: Epic L — items **L1** / **L2**
- **Deciders**: bmarty
- **Related to**: ADR-0001 (login/compte), ADR-0004 (DataWindow — catalogue public)

## Context

Le catalogue de téléchargement (Epic J) sert désormais la **bibliothèque publique** de fichiers
via une source DataWindow adossée au répertoire `-files` (touche `X` → XMODEM,
`server/internal/bbs/xfer.go:69 sendFileDownload`). Depuis la source tachibana (J5/J6), ce
`-files` contient ~1500 fichiers **publics**.

Or la section historique **« Fichiers »** du contenu de prod (`content/site.json`, page `fichiers`)
lance l'applet `download` qui **liste ce même `-files` partagé à plat, 9 fichiers maximum**
(`xfer.go:22 downloadApplet`). Deux problèmes :

1. **Redondance** : elle fait doublon avec le Catalogue (même `-files`), et son plafond de 9 entrées
   la rend inutilisable pour une vraie bibliothèque.
2. **Sémantique fausse** : « Fichiers » devrait être un **espace personnel** (les fichiers *de
   l'utilisateur*), pas une seconde vue de la bibliothèque publique.

Décision produit (02/09) : **« Fichiers » devient un espace privé par utilisateur**, distinct du
Catalogue public.

L'existant qui rend cela peu coûteux :
- Comptes `User` (pseudo unique, `Admin`) — `server/internal/user/user.go:41`. `ValidateHandle`
  (`user.go`) borne le pseudo à de l'**ASCII `[A-Za-z0-9_-]`** → **directement sûr comme nom de
  répertoire** (pas de `/`, pas de `..`).
- Distinction identifié / invité : `SessionState.User != nil` vs `SessionState.Guest`
  (`applet.go:24`, `LoggedIn()` / `IsAdmin()`).
- `files.Library` (`server/internal/files/files.go:16`) : `List/Read/Write`, `validName`
  (anti path-traversal, refuse sous-répertoires), écriture atomique, plafond par fichier.
- Injection de dépendances optionnelles (nil = désactivé) : `WelcomeHandler` → `SessionState` →
  `AppContext` (`welcome.go:27`, `applet.go:24/63`), et flags serveur `-files/-wall/-forum/...`
  (`server/cmd/bbsd/main.go:42`).

## Decision

### 1. Un store d'espaces personnels, un répertoire par compte
Nouveau package `server/internal/userfiles` avec un `Store` racine configuré par le flag
**`-userfiles <dir>`** (nil = fonctionnalité désactivée, comme `-files`). L'espace d'un utilisateur
est le sous-répertoire **`<dir>/<handle-minuscule>/`**. Le pseudo étant déjà `[A-Za-z0-9_-]`, il
sert **tel quel** (mis en minuscules, la clé d'unicité des comptes étant insensible à la casse) —
aucun encodage exotique, aucune collision, aucune traversée possible.

Le `Store` **réutilise `files.Library`** par utilisateur (un `Library` rooté sur le répertoire du
compte), ce qui hérite gratuitement de `validName`, de l'écriture atomique et de la lecture bornée.

### 2. Réservé aux comptes identifiés
L'espace personnel n'existe **que pour `SessionState.User != nil`**. Les **invités n'ont pas
d'espace** (leur identité `Invite-N` ne persiste pas). Tout applet personnel refuse proprement si
`ac.State.User == nil` ou si `ac.State.UserFiles == nil`.

### 3. Quota par utilisateur
Le `Store` porte un **quota** (deux bornes) vérifié à chaque écriture :
- **nombre de fichiers** max (`-userfiles-max-files`, défaut proposé **20**) ;
- **taille totale** max (`-userfiles-max-bytes`, défaut proposé **512 Ko**).
La taille **par fichier** reste bornée par `-max-upload` (64 Ko). Dépassement → **refus explicite**
(message clair), jamais d'écriture partielle. Cela protège le disque d'un serveur **public**.

### 4. Injection & câblage
- `WelcomeHandler.UserFiles *userfiles.Store` (nil = désactivé), propagé dans
  `SessionState.UserFiles` puis `AppContext.UserFiles`, sur le modèle exact des autres dépendances.
- Flags serveur : `-userfiles <dir>`, `-userfiles-max-files <n>`, `-userfiles-max-bytes <n>`.

### 5. Applets « Mes fichiers » et retrait du download public à plat
- Nouvel applet **`mesfichiers`** (enregistré par nom, cf. `Register`) : **liste** l'espace du
  compte, **télécharge** (numéro/`X` → XMODEM, même `sendFileDownload` sur le `Library` personnel) et
  **téléverse** (XMODEM → `Store.Write`, quota appliqué). Gate : compte identifié + `UserFiles != nil`.
- La page **`fichiers`** du contenu pointe désormais vers l'espace personnel (`applet: mesfichiers`).
  L'applet **`download` à plat sur le `-files` partagé est retiré** de cette page (superseded par le
  Catalogue pour le public). L'`upload` public partagé est retiré au profit de l'upload personnel.

## Consequences

**Positive**
- Séparation nette **public (Catalogue)** / **privé (Mes fichiers)** ; « Fichiers » retrouve un sens.
- Aucune fuite : un utilisateur ne voit **que** son répertoire ; les pseudos étant contraints, pas de
  traversée. Réutilise la validation éprouvée de `files.Library`.
- Quota → serveur public protégé contre le remplissage disque.
- Optionnel et rétrocompatible : sans `-userfiles`, rien ne change (applet gate sur nil).

**Negative / à surveiller**
- **Renommer un compte** casserait le lien répertoire↔pseudo. Aujourd'hui aucun renommage n'existe ;
  à documenter comme contrainte si cela apparaît (ou migration du dossier).
- Quota au niveau du store (nombre + octets) recalculé par `List()` à chaque écriture : coût
  négligeable aux volumes visés (dizaines de fichiers), à revoir si un jour massif.
- Persistance = système de fichiers (pas de base) : cohérent avec `-files`, sauvegarde = copie du
  répertoire `-userfiles`.

## Rejected alternatives

1. **Sous-répertoires par utilisateur DANS le `-files` public** : rejeté — mêle public et privé, et
   `validName` interdit justement les sous-répertoires (protection à ne pas affaiblir).
2. **Espace personnel pour les invités** : rejeté — identité `Invite-N` non persistante ; imposerait
   un nettoyage et n'apporte rien.
3. **Sans quota (borné seulement par `-max-upload`)** : rejeté — serveur public, risque de
   remplissage disque par abus.
4. **Base de données (blobs SQLite)** : rejeté pour L1 — le modèle « répertoire de fichiers » de
   `-files` suffit, se sauvegarde trivialement et réutilise `files.Library`.

## Increment plan (Epic L)

1. **L1 — MVP** ← cet incrément : package `userfiles` (Store + quota, tests), câblage (flags +
   injection), applet `mesfichiers` (liste + download + upload privés, gate compte identifié), et
   **retrait du `download`/`upload` public à plat** de la page `fichiers` (L2). Vérification en
   pilotant le serveur (compte → upload → réapparaît → download byte-exact ; invité → refusé ;
   quota → refus au-delà).
2. **L3 — gestion** (plus tard) : **supprimer / renommer** ses fichiers.
3. **L4 — passerelle Catalogue** (plus tard) : **copier** un fichier du Catalogue public vers son
   espace personnel.
