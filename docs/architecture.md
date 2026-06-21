# Architecture technique — BBS Oric

## 1. Vue d'ensemble

```
┌─────────────┐   telnet/TCP   ┌──────────────────────┐
│  Oric-1 /   │  (ACIA série)  │   Serveur BBS Oric    │
│  Atmos      │◀──────────────▶│  (PC / Raspberry Pi)  │
│             │                │                       │
│ LOCI +      │                │  ┌─────────────────┐  │
│ WiFiModem   │   AT / Hayes   │  │ Couche réseau   │  │  TCP, telnet (IAC),
│ (ACIA 0x380)│                │  │ (1 tâche/conn.) │  │  timeout
└─────────────┘                │  ├─────────────────┤  │
       ▲                       │  │ Moteur BBS      │  │  menus, sessions, login
       │ test                  │  ├─────────────────┤  │
┌─────────────┐                │  │ Couche OASCII   │  │  rendu Téletexte Oric
│ Oricutron / │  ACIA modem    │  │ (rendu écran)   │  │  (attributs sériels)
│ Phosphoror  │◀──────────────▶│  └─────────────────┘  │
└─────────────┘                └──────────────────────┘
```

## 2. Couches

### 2.1 Couche réseau
- Serveur TCP, **1 connexion = 1 tâche** (thread ou coroutine asyncio).
- Négociation telnet minimale (IAC) ou « fake telnet » selon décision (cf. ROADMAP §Décisions).
- Timeout d'inactivité, fermeture propre, journalisation des sessions.

### 2.2 Moteur BBS
- Boucle de session (à la `doLoop()` de petscii-bbs) : afficher écran → lire saisie → router.
- Pile de menus / navigation, écran d'accueil, login optionnel.
- Persistance (utilisateurs, messages) — format à définir au Sprint 2.

### 2.3 Couche OASCII (cœur technique — Sprint 1)
Encapsule les spécificités d'affichage Oric pour que le moteur BBS reste agnostique.

**Mode TEXT Oric :** 40 colonnes × 28 lignes, type **Téletexte**. Les attributs sont **sériels** :
un code de contrôle (valeur < 32) posé dans une case écran change le rendu **à partir de cette case
jusqu'à la fin de la ligne** (ou jusqu'au prochain code). Conséquences :

- Poser une couleur **consomme une colonne** → prévoir la place dans la mise en page.
- Les attributs ne « traversent » pas les fins de ligne (réinitialisés à chaque ligne).

**Codes d'attribut TEXT (à confirmer/compléter en implémentation) :**

| Plage | Effet |
|-------|-------|
| `0`–`7`   | Couleur d'encre (encre 0..7) |
| `8`–`15`  | Attributs texte (clignotement, double hauteur, jeux de caractères standard/alternatif) |
| `16`–`23` | Couleur de fond (papier 0..7) |
| `24`–`31` | Attributs (mode, etc.) |

> ⚠️ Ces plages doivent être **vérifiées sur matériel/émulateur** au Sprint 1 (table d'attributs Oric
> exacte) avant d'être figées.

**API cible (langage-agnostique) :**
```
cls()                  efface l'écran
at(x, y)               positionne le curseur
ink(c)                 couleur d'encre (0..7)  → émet le code d'attribut
paper(c)               couleur de fond (0..7)
print(text)            écrit du texte
println(text)          écrit + retour ligne
flush()                envoie le buffer
read_key() / read_line()  lecture clavier
```

## 3. Pipeline de test (sans matériel)

1. Lancer le serveur BBS sur `localhost:<port>`.
2. Oricutron, backend ACIA `modem` : `ATD 127.0.0.1:<port>` (ou `ATS0=1` pour écouter).
   - Adresse ACIA Oricutron : `#31C`.
3. Variante simple : `loopback` pour tester l'ACIA seule ; client `nc`/SyncTerm pour tester le serveur seul.

## 4. Matériel réel (Sprint 4)

- Oric-1/Atmos + **LOCI** + WiFiModem USB. Adressage : MIA LOCI à **`$03A0-$03BF`** (cf. oric1-emu
  `--loci`) ; ACIA « standard » à **`$031C`** (Telestrat / défaut oric1-emu).
- Le client Oric devra cibler la bonne base ACIA selon le montage.
- Pipeline de test local complet via les émulateurs : voir [`test-emulateurs.md`](test-emulateurs.md).
- Commandes Hayes AT pour établir la connexion telnet vers le serveur.

## 5. Exposition Internet (contrainte de premier ordre)

Le BBS est un **serveur Internet public** : il écoute sur `0.0.0.0:<port>` et est joignable depuis
n'importe quel Oric connecté via son WiFiModem. Conséquences à intégrer dès le départ :

- **Port public** : **`6502`** retenu (clin d'œil au CPU de l'Oric ; évite le port 23 très scanné et
  souvent bloqué en sortie par les FAI). À configurer côté client dans le `ATD <host>:6502`.
- **Hébergement** : **VPS cloud avec IP fixe** (service public 24/7, exposition directe sans DNS dynamique).
- **Pas de chiffrement** : les clients Oric ne font pas de TLS → le flux telnet est **en clair**.
  Donc : jamais de secret sensible côté utilisateur, mots de passe BBS traités comme non confidentiels,
  hachage côté serveur quand même.
- **Surface d'attaque** : un port ouvert sur Internet est scanné en permanence.
  - Binding maîtrisé, **rate limiting** par IP, **limite de connexions simultanées**.
  - Lecture défensive des entrées (jamais d'`eval`, tailles bornées, timeouts agressifs).
  - Journalisation des connexions (IP, horodatage) + rotation des logs.
  - Isolation du process (utilisateur dédié non privilégié / conteneur).
- **Disponibilité** : service `systemd` ou conteneur avec redémarrage auto sur le VPS.

> Ces points remontent la sécurité et le déploiement comme préoccupations **transverses**, pas comme un
> sprint final. Voir la ROADMAP mise à jour.

## 6. Décisions d'architecture (ADR) — à formaliser
Voir `ROADMAP.md` §« Décisions ouvertes ». Les ADR seront versionnés dans `docs/adr/`.
