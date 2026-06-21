# Pipeline de test avec l'émulateur Oric local

Ce projet bénéficie d'un environnement de test **100% local, sans matériel ni réseau externe**.

> ⚠️ **Émulateur de référence : `/home/bmarty/Oric1/oric1-emu` UNIQUEMENT.**
> C'est le seul binaire à utiliser pour tester le BBS. Ne PAS utiliser `oric2/Phosphoric`
> (sources/tests d'un autre projet) ni d'autres copies.

## Ressources locales disponibles

| Ressource | Emplacement | Rôle |
|-----------|-------------|------|
| **oric1-emu** (Phosphoric v1.21.x) | `/home/bmarty/Oric1/oric1-emu` | **Émulateur de référence** : Oric-1 avec ACIA + série configurable |
| picowifi modem | `~/.phosphoric_picowifi.cfg` | Modem WiFi émulé, déjà configuré en telnet (carnet d'appels) |

## Support série de oric1-emu (Phosphoric)

```
--serial TYPE     loopback | tcp:H:P | pty | modem:H:P |
                  com:B,D,P,S,DEV | file:IN[:OUT] | picowifi[:SSID[:PASS]]
--serial-v23      mode V23 1200/75 (Minitel/Prestel)
--serial-baud N   timing réaliste ACIA 6551
--serial-trace F  trace TX/RX horodatée (debug)
--acia-addr ADDR  base ACIA en hex (défaut 031C)
--loci            LOCI MIA à $03A0-$03BF
```

## Pipeline de test recommandé

### 1. Démarrer le serveur BBS
```bash
cd /home/bmarty/bbsoric
go run ./cmd/bbsd -addr 127.0.0.1:6502
```

### 2a. Connexion directe ACIA → BBS (le plus simple)
L'émulateur relie son ACIA à notre serveur via une socket TCP :
```bash
cd /home/bmarty/Oric1
./oric1-emu --serial tcp:127.0.0.1:6502 --acia-addr 031C
```
Côté Oric, un programme terminal lit/écrit l'ACIA à `$031C`. La trace `--serial-trace`
aide à diagnostiquer le rendu (octets d'attribut Téletexte).

### 2b. Connexion via modem émulé (proche du réel)
```bash
./oric1-emu --serial picowifi
# puis, depuis l'Oric : ATD 127.0.0.1:6502
```
ou `--serial modem:127.0.0.1:6502` selon le scénario.

### 3. Test serveur seul (sans émulateur)
```bash
# bannière + commandes
printf 'HELP\r\nQUIT\r\n' | nc 127.0.0.1 6502
```

## Intégration au modem picowifi réel

`~/.phosphoric_picowifi.cfg` contient déjà un carnet d'appels (`dial0..2`) vers des BBS publics.
Pour notre serveur, ajouter une entrée :
```
dialN=<host-vps>:6502,bbsoric
```
> ⚠️ Le picowifi est configuré `tty_type=ansi, tty_w=80, tty_h=24`. L'Oric en mode TEXT fait
> **40 colonnes** : la couche OASCII (Sprint 1) devra produire un rendu adapté à 40 colonnes,
> indépendamment des réglages ANSI du modem.

## Note sur l'adressage ACIA
- **oric1-emu / Telestrat** : ACIA à `$031C` par défaut (`--acia-addr` pour changer).
- **LOCI** : MIA à `$03A0-$03BF` (côté émulateur) ; la doc Raxiss mentionne aussi l'exposition
  modem USB-CDC. Le client Oric devra cibler la bonne base selon le montage.
