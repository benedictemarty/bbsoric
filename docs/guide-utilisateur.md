# Guide utilisateur — se connecter au BBS Oric

Bienvenue sur le **BBS Oric** ! Ce serveur de messagerie rétro est accessible
**24h/24** sur Internet. Ce guide explique comment s'y connecter et le parcourir,
que vous ayez un **vrai Oric** ou un simple ordinateur moderne.

## Coordonnées

| | Adresse | Port | Protocole |
|--|---------|------|-----------|
| **Telnet (clair)** | `pavi.3617.fr` | `6502` | telnet / raw |
| **TLS (chiffré)** | `pavi.3617.fr` | `6992` | TLS (terminé par le modem) |

> Le port **6502** est un clin d'œil au microprocesseur de l'Oric.

---

## A. Depuis un Oric réel (Oric-1 / Atmos)

C'est l'usage prévu : un Oric équipé d'une **interface série** (carte ACIA ou
**LOCI**) et d'un **modem WiFi**.

1. Branchez l'interface série et le modem WiFi (modem associé à votre WiFi).
2. Chargez le terminal : `CLOAD"TERM"` (le programme `term.tap` démarre seul).
3. **Menu modem** : tapez `1` (ACIA `$031C`) ou `2` (LOCI `$03A0`) selon la carte.
4. **Répertoire** : tapez `1` pour *BBS Oric (prod)*, ou `M` pour saisir une
   adresse à la main.
5. Le terminal compose tout seul l'appel et affiche la **bannière couleur** du BBS.

Détails de câblage, commandes AT, dépannage : voir **`connexion-materielle.md`**.

Pour les curieux, l'appel composé manuellement depuis n'importe quel modem Hayes :

```
ATD pavi.3617.fr:6502
```

---

## B. Depuis un ordinateur moderne (pour tester)

Aucun matériel Oric n'est nécessaire pour **essayer** le BBS — n'importe quel
client telnet fait l'affaire. Les couleurs Oric (attributs Téletexte sériels)
n'apparaîtront pas correctement sur un terminal PC, mais la navigation fonctionne.

### Linux / macOS

```console
# avec netcat (recommandé : pas de négociation telnet parasite)
nc pavi.3617.fr 6502

# ou avec telnet
telnet pavi.3617.fr 6502
```

### Windows

- Activez le client Telnet (« Fonctionnalités Windows » → *Client Telnet*) puis :
  `telnet pavi.3617.fr 6502`
- Ou utilisez **PuTTY** : type de connexion *Telnet* ? non — choisissez *Raw*,
  hôte `pavi.3617.fr`, port `6502`.

### Connexion chiffrée (TLS) pour tester

```console
openssl s_client -connect pavi.3617.fr:6992 -quiet
```

---

## C. Naviguer dans le BBS

À la connexion, le BBS affiche une **bannière** puis le **menu principal**. La
navigation est pensée pour le clavier d'un Oric :

- **Menus** : une seule touche suffit (pas besoin d'appuyer sur Entrée).
  Exemple : `1` ouvre « Informations système ».
- **Champs de saisie** (login, etc.) : tapez votre texte puis **Entrée** (RETURN).
- **Revenir / continuer** : appuyez sur une touche quand l'invite « une touche »
  apparaît.
- **Quitter** : `Q` au menu principal (le BBS répond « A bientot »).

### Comptes utilisateurs

Le BBS propose (selon le contenu en ligne) :

- **Invité** : accès immédiat sans compte.
- **Connexion** : identifiant + mot de passe pour un accès personnalisé.
- **Inscription** : créer un compte (mot de passe stocké haché, jamais en clair).

---

## D. Problèmes fréquents

| Symptôme | Solution |
|----------|----------|
| « Serveur sature, reessayez plus tard » | limite de connexions atteinte ; réessayez dans un moment. |
| « Trop de connexions depuis votre adresse » | vous avez déjà plusieurs sessions ouvertes depuis la même IP ; fermez-en. |
| Déconnexion après quelques minutes d'inactivité | normal : délai d'inactivité de 5 min. Reconnectez-vous. |
| Texte coloré illisible sur PC | attendu : les couleurs sont des attributs Oric, rendus uniquement par un Oric. |
| Connexion impossible | vérifiez l'adresse/port ; le serveur est peut-être en maintenance. |

---

## Voir aussi

- `connexion-materielle.md` — branchement et configuration depuis un Oric réel.
- `oascii.md` — comment l'Oric affiche les couleurs (attributs Téletexte).
- `README.md` (racine) — présentation générale du projet.
