# ADR-0001 — Login : composant interactif appelé par une page, persistance hachée

- **Statut** : Accepté
- **Date** : 2026-06-22
- **Sprint / Backlog** : Sprint 2 — item **C4** (« je veux m'identifier et retrouver mon profil »)
- **Décideurs** : bmarty
- **Remplace / complète** : aucun (premier ADR formalisé du dépôt)

## Contexte

Le moteur BBS (`internal/bbs/engine.go`) déroule un **flux de pages piloté par JSON**
(`internal/content`) : un `Site` contient des `Page` de type `menu` ou `page`, et la
navigation se fait par des `Entry` dont la `Target` est soit un identifiant de page, soit
une **cible spéciale** (`__quit__`, `__back__`, `__home__`). Le contenu est éditable et
**rechargé à chaud** ; aucune logique n'est codée dans le JSON.

Il faut introduire l'identification des utilisateurs sans casser ces propriétés. Trois
contraintes structurent la décision :

1. **Écho local du terminal Oric** (`oric-client/term.s`) : le caractère tapé est affiché
   par l'Oric avant l'envoi → le masquage serveur du mot de passe par `*` est inopérant.
2. **Saisie ligne par ligne** (`server.Session.ReadLine`) : pas de lecture caractère par
   caractère ni de contrôle du curseur distant.
3. **Philosophie « zéro dépendance »** : `go.mod` ne déclare aucune dépendance externe ;
   on souhaite le rester.

## Décision

### 1. Le login est un *applet* lancé par une *page de type `applet`*, en porte au CONNECT

**Révision 2 (2026-06-22)** : conformément aux BBS historiques, l'identification se fait
**dès la connexion, avant le menu principal**. On généralise le patron « page pure →
comportement Go » avec un **3ᵉ type de page : `applet`**. La page reste du **JSON/texte**,
elle déclare juste *quel applet* exécuter ; l'**applet** est une petite unité Go spécifique
(login, inscription, invité… puis jeux, sondages), enregistrée par son nom dans un
**registry**. On n'ajoute donc **pas** de cibles spéciales par fonction.

- Nouveau type de page `applet` avec deux champs : `applet` (nom enregistré) et `next`
  (page où aller **après succès**, ex. `main`). Une page `applet` peut aussi porter des
  `lines` (texte d'intro affiché avant de lancer l'applet).
- La **page de départ** (`site.Start`) est un menu d'auth en JSON pur dont les entrées
  pointent vers des pages `applet` (`login`/`register`/`guest`) comme vers n'importe quelle
  page. Tant que l'utilisateur n'est ni connecté ni invité, il reste sur cette porte.
- **Registry d'applets** (package `bbs`) : `Register(nom, Applet)`. Un applet a la
  signature `func(ctx, *server.Session, *AppContext) Outcome` ; il fait son propre rendu
  OASCII et sa propre saisie, **ne connaît pas** le flux de pages → testable isolément.
  `AppContext` injecte les dépendances (`*user.Store`, état de session) ; `Outcome` indique
  au moteur la suite (succès → `next`, annulation → retour, quitter).
- Le **moteur** (`engine.go`), en arrivant sur une page `applet`, résout l'applet par son
  nom, l'exécute, puis applique l'`Outcome` (navigue vers `next` si succès).

**Ajouter un applet** = écrire une petite fonction Go + l'enregistrer ; le **placer** dans
le BBS = éditer le JSON. Aucune modification de la navigation.

### 2. État de session

`runSite` reçoit un état de session minimal portant l'utilisateur courant
(`user *user.User`, `nil` si invité), afin que les écrans suivants personnalisent
l'affichage (« Bonjour {pseudo}, appel n°{n} ») et, plus tard, restreignent l'accès.

### 3. Persistance et hachage

- Modèle `user.User` : `Handle`, `PassHash`, `Created`, `LastLogin`, `Calls`.
- Store fichier **`users.json`** avec **verrou** (écritures concurrentes) et **écriture
  atomique** (fichier temporaire + `rename`). Symétrique au choix JSON déjà retenu pour le
  contenu, mais en lecture **et** écriture.
- Mots de passe **jamais en clair** : hachage **PBKDF2-HMAC-SHA256** (`crypto/pbkdf2`,
  **stdlib** Go 1.24+), sel aléatoire par compte (`crypto/rand`). Format encodé
  auto-descriptif : `pbkdf2$sha256$<iter>$<sel_b64>$<hash_b64>`.

### 4. Mot de passe en clair à l'écran : assumé pour l'instant

Faute d'écho contrôlable, la saisie du mot de passe est **visible** à l'écran de l'Oric.
On l'assume (avertissement affiché), la confidentialité du **transport** étant déjà
couverte par le **TLS sur `:6992`**. Le masquage réel (négociation `IAC WILL ECHO` ou
mode « no-echo » côté `term.s`) est repoussé à un incrément ultérieur.

## Conséquences

**Positives**
- Le flux reste 100 % piloté par JSON et rechargeable à chaud ; aucune page figée.
- Le composant login est isolé, testable sans réseau, et le patron « cible spéciale →
  composant » est réutilisable (futur : poster un message, jouer, etc.).
- Aucune dépendance externe ajoutée.

**Négatives / à surveiller**
- Le mot de passe transite en clair **à l'écran** (pas sur le réseau si TLS) tant que le
  no-echo n'est pas fait.
- Le login au CONNECT supprime le besoin d'entrées de menu conditionnelles au départ
  (l'utilisateur passe la porte avant d'atteindre le menu principal). Cacher dynamiquement
  des entrées selon le rôle reste un incrément ultérieur (règles de visibilité JSON).
- L'écriture concurrente de `users.json` impose verrou + écriture atomique (pris en compte).
- La saisie (touche unique pour les menus, ligne+RETURN pour les champs texte) fait l'objet
  d'un ADR dédié : voir **ADR-0002**.

## Alternatives écartées

1. **Page de login codée en dur (en Go) avant le flux** : assure bien le login au CONNECT
   mais casse l'uniformité « tout est piloté par le JSON » et n'est pas réutilisable. On
   préfère faire de la **page de départ JSON** la porte d'auth (même résultat, sans figer
   l'écran dans le code).
2. **Login comme un `type` de page JSON** (ex. `"type":"login"`) : mélange données et
   comportement interactif dans le contenu ; la cible spéciale est plus simple et cohérente
   avec `__quit__`/`__back__`/`__home__`.
3. **bcrypt/argon2 via `golang.org/x/crypto`** : meilleur état de l'art, mais ajoute une
   dépendance externe ; PBKDF2 stdlib est suffisant pour l'usage et préserve le « zéro
   dépendance ».

## Plan d'incréments (Sprint 2 / C4)

1. **`internal/user`** : modèle + store haché atomique + tests unitaires (sans réseau). ← *cet incrément*
2. Composant `RunLogin` / `RunRegister` / accès invité inséré via cibles spéciales (test émulateur).
3. État de session + accueil personnalisé.
4. (Ultérieur) Entrées conditionnelles, no-echo du mot de passe.
